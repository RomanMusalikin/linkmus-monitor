//go:build windows

package collector

import "github.com/shirou/gopsutil/v3/mem"

func CollectMemory() float64 {
	vMem, err := mem.VirtualMemory()
	if err == nil {
		return vMem.UsedPercent
	}
	return 0
}
