//go:build linux

package collector

// FSRMInfo — заглушка для Linux (FSRM — только Windows/SRMSVC)
type FSRMInfo struct {
	QuotaPath        string  `json:"quotaPath"`
	QuotaLimitBytes  int64   `json:"quotaLimitBytes"`
	QuotaUsedBytes   int64   `json:"quotaUsedBytes"`
	QuotaUsedPercent float64 `json:"quotaUsedPercent"`
	Violations24h    int     `json:"violations24h"`
	LastViolationTime string `json:"lastViolationTime"`
	LastViolationType string `json:"lastViolationType"`
}

func CollectFSRM() []FSRMInfo { return nil }
