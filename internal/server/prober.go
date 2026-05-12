package server

import (
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// ProbeResult — результаты TCP-проб для одного узла
type ProbeResult struct {
	SSHReachable   bool
	RDPReachable   bool
	SMBReachable   bool
	HTTPReachable  bool
	WinRMReachable bool
	SSHMs          float64
	RDPMs          float64
	SMBMs          float64
	HTTPMs         float64
	WinRMMs        float64
}

var (
	probeCache   = map[string]ProbeResult{}
	probeCacheMu sync.RWMutex

	customProbeCache   = map[string][]CustomServiceResult{}
	customProbeCacheMu sync.RWMutex
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

// GetProbe возвращает последние результаты стандартных проб для IP-адреса.
func GetProbe(ip string) ProbeResult {
	probeCacheMu.RLock()
	defer probeCacheMu.RUnlock()
	return probeCache[ip]
}

// GetCustomProbe возвращает последние результаты проб пользовательских сервисов для IP-адреса.
func GetCustomProbe(ip string) []CustomServiceResult {
	customProbeCacheMu.RLock()
	defer customProbeCacheMu.RUnlock()
	r := customProbeCache[ip]
	if r == nil {
		return []CustomServiceResult{}
	}
	return r
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

func probeHTTP(ip string, port int) (bool, float64) {
	client := &http.Client{Timeout: 2 * time.Second}
	start := time.Now()
	resp, err := client.Get(fmt.Sprintf("http://%s:%d/", ip, port))
	ms := float64(time.Since(start).Nanoseconds()) / 1e6
	if err != nil {
		return false, 0
	}
	resp.Body.Close()
	return resp.StatusCode < 500, ms
}

type nodeTarget struct {
	ip   string
	name string
}

func runProbes(db *sql.DB) {
	rows, err := db.Query(`
		SELECT m.ip, m.node_name FROM metrics m
		INNER JOIN (
			SELECT ip, MAX(timestamp) AS mt FROM metrics
			WHERE ip != '' AND ip != '127.0.0.1'
			GROUP BY ip
		) latest ON m.ip = latest.ip AND m.timestamp = latest.mt`)
	if err != nil {
		return
	}
	var targets []nodeTarget
	for rows.Next() {
		var t nodeTarget
		if rows.Scan(&t.ip, &t.name) == nil && t.ip != "" {
			targets = append(targets, t)
		}
	}
	rows.Close()

	globalPorts := GetPortSettings(db)
	customs := GetCustomServices(db)

	var wg sync.WaitGroup
	for _, t := range targets {
		wg.Add(1)
		go func(t nodeTarget) {
			defer wg.Done()
			override := GetNodePortOverride(db, t.name)
			ports := EffectivePortSettings(globalPorts, override)

			result := ProbeResult{}
			result.SSHReachable, result.SSHMs = probeTCP(t.ip, fmt.Sprintf("%d", ports.SSHPort))
			result.RDPReachable, result.RDPMs = probeTCP(t.ip, fmt.Sprintf("%d", ports.RDPPort))
			result.SMBReachable, result.SMBMs = probeTCP(t.ip, fmt.Sprintf("%d", ports.SMBPort))
			result.HTTPReachable, result.HTTPMs = probeHTTP(t.ip, ports.HTTPPort)
			result.WinRMReachable, result.WinRMMs = probeTCP(t.ip, fmt.Sprintf("%d", ports.WinRMPort))

			probeCacheMu.Lock()
			probeCache[t.ip] = result
			probeCacheMu.Unlock()

			// Пробы пользовательских сервисов
			var customResults []CustomServiceResult
			for _, svc := range customs {
				ok, ms := probeTCP(t.ip, fmt.Sprintf("%d", svc.Port))
				customResults = append(customResults, CustomServiceResult{
					ID: svc.ID, Name: svc.Name, Port: svc.Port,
					Reachable: ok, Ms: ms,
				})
			}
			customProbeCacheMu.Lock()
			customProbeCache[t.ip] = customResults
			customProbeCacheMu.Unlock()
		}(t)
	}
	wg.Wait()
}
