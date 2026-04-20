// internal/collector/common.go
package collector

// SystemMetrics — общая структура для всех платформ (legacy)
type SystemMetrics struct {
	CPUUsage   float64
	RAMUsage   float64
	DiskUsage  float64
	RDPRunning bool
	SMBRunning bool
}

type Collector interface {
	Collect() (SystemMetrics, error)
}

// DiskInfo — информация об одном дисковом разделе
type DiskInfo struct {
	Mount       string  `json:"mount"`
	FSType      string  `json:"fstype"`
	TotalGB     float64 `json:"totalGB"`
	UsedGB      float64 `json:"usedGB"`
	UsedPercent float64 `json:"usedPercent"`
}

// NetIfaceInfo — трафик одного сетевого интерфейса (байт/сек)
type NetIfaceInfo struct {
	Name         string  `json:"name"`
	BytesRecvSec float64 `json:"bytesRecvSec"`
	BytesSentSec float64 `json:"bytesSentSec"`
}
