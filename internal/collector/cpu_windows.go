//go:build windows

package collector

import "github.com/shirou/gopsutil/v3/cpu"

var (
	prevCPUTimes      []cpu.TimesStat
	prevCPUCoresTimes []cpu.TimesStat
)

// CollectCPU — обёртка для обратной совместимости
func CollectCPU() float64 {
	_, _, total := CollectCPUBreakdown()
	return total
}

// CollectCPUBreakdown возвращает user%, system%, total% через дельту между вызовами.
func CollectCPUBreakdown() (user, system, total float64) {
	curr, err := cpu.Times(false)
	if err != nil || len(curr) == 0 {
		return
	}
	if len(prevCPUTimes) > 0 {
		dt := curr[0].Total() - prevCPUTimes[0].Total()
		if dt > 0 {
			idle := (curr[0].Idle - prevCPUTimes[0].Idle) / dt * 100
			total = 100 - idle
			user = (curr[0].User - prevCPUTimes[0].User) / dt * 100
			system = (curr[0].System - prevCPUTimes[0].System) / dt * 100
			if total < 0 {
				total = 0
			}
			if total > 100 {
				total = 100
			}
			if user < 0 {
				user = 0
			}
			if system < 0 {
				system = 0
			}
		}
	}
	prevCPUTimes = curr
	return
}

// CollectCPUPerCore возвращает загрузку каждого ядра в процентах.
// Первый вызов вернёт нули (нет базовой точки).
func CollectCPUPerCore() []float64 {
	curr, err := cpu.Times(true)
	if err != nil || len(curr) == 0 {
		return nil
	}
	result := make([]float64, len(curr))
	if len(prevCPUCoresTimes) == len(curr) {
		for i := range curr {
			dt := curr[i].Total() - prevCPUCoresTimes[i].Total()
			if dt > 0 {
				idle := (curr[i].Idle - prevCPUCoresTimes[i].Idle) / dt * 100
				pct := 100 - idle
				if pct < 0 {
					pct = 0
				}
				if pct > 100 {
					pct = 100
				}
				result[i] = pct
			}
		}
	}
	prevCPUCoresTimes = curr
	return result
}

// CollectCPUInfo возвращает модель процессора и текущую частоту в МГц.
func CollectCPUInfo() (model string, freqMHz float64) {
	infos, err := cpu.Info()
	if err != nil || len(infos) == 0 {
		return "", 0
	}
	return infos[0].ModelName, infos[0].Mhz
}
