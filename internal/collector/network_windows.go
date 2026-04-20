//go:build windows

package collector

import (
	"strings"
	"time"

	psnet "github.com/shirou/gopsutil/v3/net"
)

// NetworkInfo содержит данные о трафике основного сетевого интерфейса
type NetworkInfo struct {
	Interface    string
	BytesRecvSec float64
	BytesSentSec float64
}

var (
	prevNetCounters    map[string]psnet.IOCountersStat
	prevNetTime        time.Time
	prevAllNetCounters map[string]psnet.IOCountersStat
	prevAllNetTime     time.Time
)

// CollectNetwork возвращает имя и скорость трафика основного интерфейса (байт/сек).
func CollectNetwork(outboundIP string) NetworkInfo {
	counters, err := psnet.IOCounters(true)
	if err != nil || len(counters) == 0 {
		return NetworkInfo{}
	}

	ifaceName := findInterfaceByIP(outboundIP)
	var curr psnet.IOCountersStat
	found := false
	for _, c := range counters {
		if c.Name == ifaceName {
			curr = c
			found = true
			break
		}
	}
	if !found {
		for _, c := range counters {
			if c.BytesRecv > curr.BytesRecv {
				curr = c
			}
		}
	}

	now := time.Now()
	result := NetworkInfo{Interface: curr.Name}

	if !prevNetTime.IsZero() && prevNetCounters != nil {
		dt := now.Sub(prevNetTime).Seconds()
		if dt > 0 {
			if prev, ok := prevNetCounters[curr.Name]; ok {
				r := float64(curr.BytesRecv-prev.BytesRecv) / dt
				s := float64(curr.BytesSent-prev.BytesSent) / dt
				if r >= 0 {
					result.BytesRecvSec = r
				}
				if s >= 0 {
					result.BytesSentSec = s
				}
			}
		}
	}

	if prevNetCounters == nil {
		prevNetCounters = make(map[string]psnet.IOCountersStat)
	}
	prevNetCounters[curr.Name] = curr
	prevNetTime = now
	return result
}

// CollectAllInterfaces возвращает трафик по каждому активному сетевому интерфейсу.
func CollectAllInterfaces() []NetIfaceInfo {
	counters, err := psnet.IOCounters(true)
	if err != nil || len(counters) == 0 {
		return nil
	}

	now := time.Now()
	var result []NetIfaceInfo

	for _, curr := range counters {
		// Пропускаем loopback и интерфейсы без трафика
		if strings.EqualFold(curr.Name, "lo") || strings.HasPrefix(curr.Name, "Loopback") {
			continue
		}
		if curr.BytesRecv == 0 && curr.BytesSent == 0 {
			continue
		}

		info := NetIfaceInfo{Name: curr.Name}

		if !prevAllNetTime.IsZero() && prevAllNetCounters != nil {
			dt := now.Sub(prevAllNetTime).Seconds()
			if dt > 0 {
				if prev, ok := prevAllNetCounters[curr.Name]; ok {
					r := float64(curr.BytesRecv-prev.BytesRecv) / dt
					s := float64(curr.BytesSent-prev.BytesSent) / dt
					if r >= 0 {
						info.BytesRecvSec = r
					}
					if s >= 0 {
						info.BytesSentSec = s
					}
				}
			}
		}
		result = append(result, info)
	}

	if prevAllNetCounters == nil {
		prevAllNetCounters = make(map[string]psnet.IOCountersStat)
	}
	for _, c := range counters {
		prevAllNetCounters[c.Name] = c
	}
	prevAllNetTime = now
	return result
}

func findInterfaceByIP(ip string) string {
	ifaces, err := psnet.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		for _, addr := range iface.Addrs {
			if strings.HasPrefix(addr.Addr, ip+"/") || addr.Addr == ip {
				return iface.Name
			}
		}
	}
	return ""
}
