package server

import (
	"context"
	"database/sql"
	"net"
	"net/http"
	"sync"
	"time"
)

// ProbeResult — результаты TCP/DNS-проб для одного узла
type ProbeResult struct {
	SSHReachable   bool
	RDPReachable   bool
	SMBReachable   bool
	HTTPReachable  bool
	WinRMReachable bool
	DNSReachable   bool
	SSHMs          float64
	RDPMs          float64
	SMBMs          float64
	HTTPMs         float64
	WinRMMs        float64
	DNSMs          float64
}

var (
	probeCache   = map[string]ProbeResult{}
	probeCacheMu sync.RWMutex
)

// StartProber запускает фоновый цикл TCP-проб каждые 15 секунд.
func StartProber(db *sql.DB) {
	go func() {
		for {
			runProbes(db)
			time.Sleep(15 * time.Second)
		}
	}()
}

// GetProbe возвращает последние результаты проб для IP-адреса.
func GetProbe(ip string) ProbeResult {
	probeCacheMu.RLock()
	defer probeCacheMu.RUnlock()
	return probeCache[ip]
}

func probeTCP(host, port string) (bool, float64) {
	start := time.Now()
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), 2*time.Second)
	ms := float64(time.Since(start).Nanoseconds()) / 1e6
	if err != nil {
		return false, 0
	}
	conn.Close()
	return true, ms
}

func probeHTTP(ip string) (bool, float64) {
	client := &http.Client{Timeout: 2 * time.Second}
	start := time.Now()
	resp, err := client.Get("http://" + ip + "/")
	ms := float64(time.Since(start).Nanoseconds()) / 1e6
	if err != nil {
		return false, 0
	}
	resp.Body.Close()
	return resp.StatusCode < 500, ms
}

func probeDNS(ip string) (bool, float64) {
	resolver := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 2 * time.Second}).DialContext(ctx, "udp", net.JoinHostPort(ip, "53"))
		},
	}
	start := time.Now()
	_, err := resolver.LookupHost(context.Background(), "srv-mon-01.local")
	ms := float64(time.Since(start).Nanoseconds()) / 1e6
	if err != nil {
		// Пробуем просто TCP:53 как fallback
		ok, ms2 := probeTCP(ip, "53")
		return ok, ms2
	}
	return true, ms
}

func runProbes(db *sql.DB) {
	rows, err := db.Query(`SELECT DISTINCT ip FROM metrics WHERE ip != '' AND ip != '127.0.0.1'`)
	if err != nil {
		return
	}
	var ips []string
	for rows.Next() {
		var ip string
		if rows.Scan(&ip) == nil && ip != "" {
			ips = append(ips, ip)
		}
	}
	rows.Close()

	var wg sync.WaitGroup
	for _, ip := range ips {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			result := ProbeResult{}
			result.SSHReachable, result.SSHMs = probeTCP(ip, "22")
			result.RDPReachable, result.RDPMs = probeTCP(ip, "3389")
			result.SMBReachable, result.SMBMs = probeTCP(ip, "445")
			result.HTTPReachable, result.HTTPMs = probeHTTP(ip)
			result.WinRMReachable, result.WinRMMs = probeTCP(ip, "5985")
			result.DNSReachable, result.DNSMs = probeDNS(ip)

			probeCacheMu.Lock()
			probeCache[ip] = result
			probeCacheMu.Unlock()
		}(ip)
	}
	wg.Wait()
}
