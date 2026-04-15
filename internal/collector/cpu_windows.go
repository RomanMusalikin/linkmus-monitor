//go:build windows

package collector

import "github.com/shirou/gopsutil/v3/cpu"

var prevCPUTimes []cpu.TimesStat

// CollectCPU — обёртка для обратной совместимости
func CollectCPU() float64 {
	_, _, total := CollectCPUBreakdown()
	return total
}

// CollectCPUBreakdown возвращает user%, system%, total% на основе дельты между вызовами.
// Первый вызов всегда вернёт 0 (нет базовой точки), со второго — реальные значения.
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
