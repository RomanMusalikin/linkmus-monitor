//go:build windows

package collector

import (
	"sort"

	"github.com/shirou/gopsutil/v3/process"
)

// ProcessInfo описывает один процесс для отправки на сервер
type ProcessInfo struct {
	PID  int32   `json:"pid"`
	Name string  `json:"name"`
	CPU  float64 `json:"cpu"`
	RAM  float64 `json:"ram"` // МБ
	User string  `json:"user"`
}

// CollectProcesses возвращает топ-10 процессов по загрузке CPU.
// Первый вызов вернёт CPU=0 для всех (gopsutil требует две точки для расчёта дельты).
func CollectProcesses() []ProcessInfo {
	procs, err := process.Processes()
	if err != nil {
		return nil
	}

	var result []ProcessInfo
	for _, p := range procs {
		name, err := p.Name()
		if err != nil || name == "" {
			continue
		}

		cpuPct, _ := p.CPUPercent()

		var memMB float64
		if memInfo, err := p.MemoryInfo(); err == nil && memInfo != nil {
			memMB = float64(memInfo.RSS) / 1024 / 1024
		}

		user, _ := p.Username()

		result = append(result, ProcessInfo{
			PID:  p.Pid,
			Name: name,
			CPU:  cpuPct,
			RAM:  memMB,
			User: user,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].CPU > result[j].CPU
	})

	if len(result) > 10 {
		result = result[:10]
	}
	return result
}
