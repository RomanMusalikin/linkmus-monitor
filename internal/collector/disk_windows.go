//go:build windows

package collector

import (
	"time"

	"github.com/shirou/gopsutil/v3/disk"
)

var (
	prevDiskCounters map[string]disk.IOCountersStat
	prevDiskTime     time.Time
)

// CollectDisk — обёртка для обратной совместимости (только C:\)
func CollectDisk() float64 {
	u, err := disk.Usage(`C:\`)
	if err == nil {
		return u.UsedPercent
	}
	return 0
}

// CollectAllDisks возвращает список всех смонтированных разделов с их заполненностью.
func CollectAllDisks() []DiskInfo {
	parts, err := disk.Partitions(false)
	if err != nil {
		return nil
	}
	var result []DiskInfo
	for _, p := range parts {
		u, err := disk.Usage(p.Mountpoint)
		if err != nil || u.Total < 50*1024*1024 { // пропускаем разделы < 50 МБ
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
