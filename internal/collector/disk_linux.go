//go:build linux

package collector

import "github.com/shirou/gopsutil/v3/disk"

func CollectDisk() float64 {
	dUsage, err := disk.Usage("/")
	if err == nil {
		return dUsage.UsedPercent
	}
	return 0
}
