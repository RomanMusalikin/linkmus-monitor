//go:build linux

package collector

import (
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
)

var (
	prevDiskCounters map[string]disk.IOCountersStat
	prevDiskTime     time.Time
)

// CollectDisk — обёртка для обратной совместимости (только /)
func CollectDisk() float64 {
	u, err := disk.Usage("/")
	if err == nil {
		return u.UsedPercent
	}
	return 0
}

// CollectAllDisks возвращает список всех смонтированных разделов с их заполненностью.
// Фильтрует виртуальные файловые системы (tmpfs, devtmpfs и т.п.).
func CollectAllDisks() []DiskInfo {
	parts, err := disk.Partitions(false)
	if err != nil {
		return nil
	}
	var result []DiskInfo
	for _, p := range parts {
		// Пропускаем виртуальные, системные и snap-ФС
		fs := strings.ToLower(p.Fstype)
		if fs == "tmpfs" || fs == "devtmpfs" || fs == "sysfs" ||
			fs == "proc" || fs == "cgroup" || fs == "cgroup2" ||
			fs == "pstore" || fs == "securityfs" || fs == "debugfs" ||
			fs == "squashfs" || fs == "overlay" || fs == "fuse" ||
			strings.HasPrefix(p.Mountpoint, "/sys") ||
			strings.HasPrefix(p.Mountpoint, "/proc") ||
			strings.HasPrefix(p.Mountpoint, "/dev") ||
			strings.HasPrefix(p.Mountpoint, "/snap") ||
			strings.HasPrefix(p.Mountpoint, "/run") {
			continue
		}
		u, err := disk.Usage(p.Mountpoint)
		if err != nil || u.Total < 50*1024*1024 {
			continue
		}
		result = append(result, DiskInfo{
			Mount:       p.Mountpoint,
			FSType:      p.Fstype,
			TotalGB:     float64(u.Total) / 1024 / 1024 / 1024,
			UsedGB:      float64(u.Used) / 1024 / 1024 / 1024,
			UsedPercent: u.UsedPercent,
		})
	}
	return result
}

// CollectDiskIO возвращает суммарную скорость чтения/записи (байт/сек) и среднюю длину очереди.
func CollectDiskIO() (readSec, writeSec, queueLen float64) {
	counters, err := disk.IOCounters()
	if err != nil {
		return
	}
	now := time.Now()

	if !prevDiskTime.IsZero() && prevDiskCounters != nil {
		dt := now.Sub(prevDiskTime).Seconds()
		if dt > 0 {
			for name, curr := range counters {
				if prev, ok := prevDiskCounters[name]; ok {
					r := float64(curr.ReadBytes-prev.ReadBytes) / dt
					w := float64(curr.WriteBytes-prev.WriteBytes) / dt
					if r >= 0 {
						readSec += r
					}
					if w >= 0 {
						writeSec += w
					}
				}
			}
		}
	}

	// IopsInProgress — мгновенная длина очереди IO из /proc/diskstats
	var qSum float64
	var qCount int
	for _, c := range counters {
		if c.ReadCount+c.WriteCount > 0 {
			qSum += float64(c.IopsInProgress)
			qCount++
		}
	}
	if qCount > 0 {
		queueLen = qSum / float64(qCount)
	}

	prevDiskCounters = counters
	prevDiskTime = now
	return
}
