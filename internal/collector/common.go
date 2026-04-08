// internal/collector/common.go
package collector

// SystemMetrics — общая структура для всех платформ
type SystemMetrics struct {
	CPUUsage   float64
	RAMUsage   float64
	DiskUsage  float64
	RDPRunning bool // Работает ли удаленный рабочий стол (TermService)
	SMBRunning bool // Работают ли общие папки (LanmanServer)
}

type Collector interface {
	Collect() (SystemMetrics, error)
}
