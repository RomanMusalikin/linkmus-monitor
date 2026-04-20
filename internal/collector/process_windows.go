//go:build windows

package collector

import (
	"sort"

	"github.com/shirou/gopsutil/v3/process"
)

// ProcessInfo описывает один процесс
type ProcessInfo struct {
	PID  int32   `json:"pid"`
	Name string  `json:"name"`
	CPU  float64 `json:"cpu"`
	RAM  float64 `json:"ram"` // МБ
	User string  `json:"user"`
}

// CollectProcesses возвращает топ-10 процессов по загрузке CPU.
func CollectProcesses() []ProcessInfo {
	return collectTop10(func(a, b ProcessInfo) bool { return a.CPU > b.CPU })
}

// CollectTopMemProcesses возвращает топ-10 процессов по потреблению RAM.
func CollectTopMemProcesses() []ProcessInfo {
	return collectTop10(func(a, b ProcessInfo) bool { return a.RAM > b.RAM })
}

// CollectProcessCount возвращает общее количество запущенных процессов.
func CollectProcessCount() int {
	procs, err := process.Processes()
	if err != nil {
		return 0
	}
	return len(procs)
}

func collectTop10(less func(a, b ProcessInfo) bool) []ProcessInfo {
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
		if mi, err := p.MemoryInfo(); err == nil && mi != nil {
			memMB = float64(mi.RSS) / 1024 / 1024
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

	sort.Slice(result, func(i, j int) bool { return less(result[i], result[j]) })
	if len(result) > 10 {
		result = result[:10]
	}
	return result
}
