//go:build windows

package collector

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
)

func CollectCPU() float64 {
	percent, err := cpu.Percent(time.Second, false)
	if err == nil && len(percent) > 0 {
		return percent[0]
	}
	return 0
}
