//go:build linux

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
	prevNetCounters map[string]psnet.IOCountersStat
	prevNetTime     time.Time
)

// CollectNetwork возвращает имя интерфейса и скорость трафика (байт/сек).
// outboundIP — IP-адрес, с которого агент выходит наружу (для определения основного интерфейса).
func CollectNetwork(outboundIP string) NetworkInfo {
	counters, err := psnet.IOCounters(true)
	if err != nil || len(counters) == 0 {
		return NetworkInfo{}
	}

	// Определяем имя основного интерфейса по IP
	ifaceName := findInterfaceByIP(outboundIP)

	// Если не нашли по IP — берём интерфейс с максимальным трафиком
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
				recv := float64(curr.BytesRecv-prev.BytesRecv) / dt
				sent := float64(curr.BytesSent-prev.BytesSent) / dt
				if recv >= 0 {
					result.BytesRecvSec = recv
				}
				if sent >= 0 {
					result.BytesSentSec = sent
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
