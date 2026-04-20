//go:build windows

package collector

import psnet "github.com/shirou/gopsutil/v3/net"

// CollectTCPConnections возвращает общее кол-во TCP-соединений, established и time_wait.
func CollectTCPConnections() (total, established, timeWait int) {
	conns, err := psnet.Connections("tcp")
	if err != nil {
		return
	}
	total = len(conns)
	for _, c := range conns {
		switch c.Status {
		case "ESTABLISHED":
			established++
		case "TIME_WAIT":
			timeWait++
		}
	}
	return
}
