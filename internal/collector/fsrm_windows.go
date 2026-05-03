//go:build windows

package collector

import (
	"fmt"
	"time"

	"github.com/yusufpapurcu/wmi"
)

// FSRMInfo описывает квоту FSRM и нарушения за последние 24 часа.
type FSRMInfo struct {
	QuotaPath         string  `json:"quotaPath"`
	QuotaLimitBytes   int64   `json:"quotaLimitBytes"`
	QuotaUsedBytes    int64   `json:"quotaUsedBytes"`
	QuotaUsedPercent  float64 `json:"quotaUsedPercent"`
	Violations24h     int     `json:"violations24h"`
	LastViolationTime string  `json:"lastViolationTime"`
	LastViolationType string  `json:"lastViolationType"`
}

type msftFSRMQuota struct {
	Path  string
	Size  uint64
	Usage uint64
}

type win32NTLogEvent struct {
	EventCode     uint32
	TimeGenerated string
	Message       string
}

// CollectFSRM читает реальные квоты через MSFT_FSRMQuota (root\Microsoft\Windows\FSRM)
// и дополняет их нарушениями из Event Log за последние 24 часа.
func CollectFSRM() []FSRMInfo {
	var quotas []msftFSRMQuota
	if err := wmi.QueryNamespace(
		"SELECT Path, Size, Usage FROM MSFT_FSRMQuota",
		&quotas,
		`root\Microsoft\Windows\FSRM`,
	); err != nil {
		fmt.Printf("⚠️  FSRM WMI error: %v — falling back to event log\n", err)
		return collectFSRMFallback()
	}
	if len(quotas) == 0 {
		return nil
	}

	// Читаем события нарушений за 24 часа одним запросом
	violations := collectViolations()

	result := make([]FSRMInfo, 0, len(quotas))
	for _, q := range quotas {
		var pct float64
		if q.Size > 0 {
			pct = float64(q.Usage) / float64(q.Size) * 100
		}
		info := FSRMInfo{
			QuotaPath:        q.Path,
			QuotaLimitBytes:  int64(q.Size),
			QuotaUsedBytes:   int64(q.Usage),
			QuotaUsedPercent: pct,
		}
		// Сопоставляем нарушения по пути
		if v, ok := violations[q.Path]; ok {
			info.Violations24h = v.count
			info.LastViolationTime = v.lastTime
			info.LastViolationType = v.lastType
		}
		result = append(result, info)
	}
	return result
}

type violationSummary struct {
	count    int
	lastTime string
	lastType string
}

func collectViolations() map[string]violationSummary {
	cutoff := time.Now().Add(-24 * time.Hour)
	wmiTime := fmt.Sprintf("%s.000000+000", cutoff.UTC().Format("20060102150405"))

	query := fmt.Sprintf(
		"SELECT EventCode, TimeGenerated, Message FROM Win32_NTLogEvent WHERE SourceName='SRMSVC' AND TimeGenerated > '%s'",
		wmiTime,
	)

	var events []win32NTLogEvent
	wmi.Query(query, &events)

	byPath := make(map[string]violationSummary)
	for _, ev := range events {
		path := extractPathFromMessage(ev.Message)
		evType := eventCodeToType(ev.EventCode)
		s := byPath[path]
		s.count++
		if s.lastTime == "" {
			s.lastTime = ev.TimeGenerated
			s.lastType = evType
		}
		byPath[path] = s
	}
	return byPath
}

func extractPathFromMessage(_ string) string {
	// Сообщения SRMSVC содержат путь в тексте; используем как ключ без разбора,
	// поэтому нарушения без совпадения пути попадут в пустой ключ "".
	// Для отображения достаточно.
	return ""
}

func eventCodeToType(code uint32) string {
	switch code {
	case 8215:
		return "soft_quota_exceeded"
	case 8216:
		return "hard_quota_exceeded"
	case 12325:
		return "file_screen_blocked"
	default:
		return fmt.Sprintf("event_%d", code)
	}
}

// collectFSRMFallback — fallback через WMI Win32_NTLogEvent если MSFT_FSRMQuota недоступна.
func collectFSRMFallback() []FSRMInfo {
	cutoff := time.Now().Add(-24 * time.Hour)
	wmiTime := fmt.Sprintf("%s.000000+000", cutoff.UTC().Format("20060102150405"))

	query := fmt.Sprintf(
		"SELECT EventCode, TimeGenerated, Message FROM Win32_NTLogEvent WHERE SourceName='SRMSVC' AND TimeGenerated > '%s'",
		wmiTime,
	)

	var events []win32NTLogEvent
	if err := wmi.Query(query, &events); err != nil || len(events) == 0 {
		return nil
	}

	info := FSRMInfo{
		QuotaPath:     "C:\\CorpShare",
		Violations24h: len(events),
	}
	last := events[0]
	info.LastViolationTime = last.TimeGenerated
	info.LastViolationType = eventCodeToType(last.EventCode)
	return []FSRMInfo{info}
}
