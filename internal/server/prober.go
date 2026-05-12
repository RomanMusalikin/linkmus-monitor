package server

import (
	"crypto/tls"
	"database/sql"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"
)

// ProbeResult — результаты TCP-проб для одного узла
type ProbeResult struct {
	SSHReachable    bool
	RDPReachable    bool
	SMBReachable    bool
	HTTPReachable   bool
	HTTPSReachable  bool
	WinRMReachable  bool
	SSHMs           float64
	RDPMs           float64
	SMBMs           float64
	HTTPMs          float64
	HTTPSMs         float64
	WinRMMs         float64
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

// RemoveCustomServiceFromProbeCache синхронно удаляет сервис из кэша проб всех IP.
func RemoveCustomServiceFromProbeCache(serviceID int) {
	customProbeCacheMu.Lock()
	defer customProbeCacheMu.Unlock()
	for ip, results := range customProbeCache {
		filtered := results[:0]
		for _, r := range results {
			if r.ID != serviceID {
				filtered = append(filtered, r)
			}
		}
		customProbeCache[ip] = filtered
	}
}

// AddCustomServiceToProbeCache немедленно добавляет новый сервис в кэш проб всех IP
// (reachable=false) и запускает фоновую пробу для получения реального статуса.
func AddCustomServiceToProbeCache(db *sql.DB, svc CustomService) {
	customProbeCacheMu.Lock()
	for ip := range customProbeCache {
		customProbeCache[ip] = append(customProbeCache[ip], CustomServiceResult{
			ID: svc.ID, Name: svc.Name, Port: svc.Port,
		})
	}
	customProbeCacheMu.Unlock()

	go func() {
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
				reachable, ms := probeTCP(ip, fmt.Sprintf("%d", svc.Port))
				customProbeCacheMu.Lock()
				for i, r := range customProbeCache[ip] {
					if r.ID == svc.ID {
						customProbeCache[ip][i].Reachable = reachable
						customProbeCache[ip][i].Ms = ms
						break
					}
				}
				customProbeCacheMu.Unlock()
			}(ip)
		}
		wg.Wait()
		InvalidateNodesCache(db)
	}()
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

func probeHTTPS(ip string, port int) (bool, float64) {
	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	start := time.Now()
	resp, err := client.Get(fmt.Sprintf("https://%s:%d/", ip, port))
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
			if ports.HTTPSPort > 0 {
				result.HTTPSReachable, result.HTTPSMs = probeHTTPS(t.ip, ports.HTTPSPort)
			}
			result.WinRMReachable, result.WinRMMs = probeTCP(t.ip, fmt.Sprintf("%d", ports.WinRMPort))

			probeCacheMu.Lock()
			probeCache[t.ip] = result
			probeCacheMu.Unlock()

			// Пробы пользовательских сервисов (с учётом per-node переопределений портов)
			customPortOverrides := GetNodeCustomServicePorts(db, t.name)
			var customResults []CustomServiceResult
			for _, svc := range customs {
				port := svc.Port
				if p, ok := customPortOverrides[svc.ID]; ok {
					port = p
				}
				reachable, ms := probeTCP(t.ip, fmt.Sprintf("%d", port))
				customResults = append(customResults, CustomServiceResult{
					ID: svc.ID, Name: svc.Name, Port: port,
					Reachable: reachable, Ms: ms,
				})
			}
			customProbeCacheMu.Lock()
			customProbeCache[t.ip] = customResults
			customProbeCacheMu.Unlock()
		}(t)
	}
	wg.Wait()
}
