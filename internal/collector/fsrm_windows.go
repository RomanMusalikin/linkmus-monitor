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

type win32NTLogEvent struct {
	EventCode   uint32
	TimeGenerated string
	Message     string
}

// CollectFSRM читает события SRMSVC из Windows Event Log за последние 24 часа.
func CollectFSRM() []FSRMInfo {
	cutoff := time.Now().Add(-24 * time.Hour)
	// WMI формат времени: YYYYMMDDHHmmss.000000+000
	wmiTime := fmt.Sprintf("%s.000000+000", cutoff.UTC().Format("20060102150405"))

	query := fmt.Sprintf(
		"SELECT EventCode, TimeGenerated, Message FROM Win32_NTLogEvent WHERE SourceName='SRMSVC' AND TimeGenerated > '%s'",
		wmiTime,
	)

	var events []win32NTLogEvent
	if err := wmi.Query(query, &events); err != nil {
		return nil
	}
	if len(events) == 0 {
		return nil
	}

	// Агрегируем нарушения в одну сводку
	info := FSRMInfo{
		QuotaPath:     "C:\\CorpShare",
		Violations24h: len(events),
	}
	if len(events) > 0 {
		last := events[0]
		info.LastViolationTime = last.TimeGenerated
		switch last.EventCode {
		case 8215:
			info.LastViolationType = "soft_quota_exceeded"
		case 8216:
			info.LastViolationType = "hard_quota_exceeded"
		case 12325:
			info.LastViolationType = "file_screen_blocked"
		default:
			info.LastViolationType = fmt.Sprintf("event_%d", last.EventCode)
		}
	}

	return []FSRMInfo{info}
}
