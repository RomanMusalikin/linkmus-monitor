package server

import (
	"database/sql"
	"log"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"
)

// SNMPResult — метрики, собранные по SNMP с узла
type SNMPResult struct {
	SysUpTimeSec uint32
	SysName      string
	CPULoad      int    // hrProcessorLoad (%)
	IfCount      int    // кол-во интерфейсов
	Collected    bool
	CollectedAt  time.Time
}

var (
	snmpCache   = map[string]SNMPResult{}
	snmpCacheMu sync.RWMutex
)

// StartSNMPPoller запускает фоновый цикл SNMP-опроса каждые 30 секунд.
func StartSNMPPoller(db *sql.DB) {
	go func() {
		for {
			pollAllSNMP(db)
			time.Sleep(30 * time.Second)
		}
	}()
}

// GetSNMP возвращает последние SNMP-метрики для IP.
func GetSNMP(ip string) SNMPResult {
	snmpCacheMu.RLock()
	defer snmpCacheMu.RUnlock()
	return snmpCache[ip]
}

func pollAllSNMP(db *sql.DB) {
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
			result := pollSNMP(ip)
			snmpCacheMu.Lock()
			snmpCache[ip] = result
			snmpCacheMu.Unlock()
		}(ip)
	}
	wg.Wait()
}

func pollSNMP(ip string) SNMPResult {
	g := &gosnmp.GoSNMP{
		Target:    ip,
		Port:      161,
		Community: "public",
		Version:   gosnmp.Version2c,
		Timeout:   2 * time.Second,
		Retries:   1,
	}
	if err := g.Connect(); err != nil {
		return SNMPResult{}
	}
	defer g.Conn.Close()

	oids := []string{
		".1.3.6.1.2.1.1.3.0",     // sysUpTime
		".1.3.6.1.2.1.1.5.0",     // sysName
		".1.3.6.1.2.1.25.3.3.1.2.1", // hrProcessorLoad (первый процессор)
	}

	result, err := g.Get(oids)
	if err != nil {
		log.Printf("SNMP get %s: %v", ip, err)
		return SNMPResult{}
	}

	var res SNMPResult
	res.Collected = true
	res.CollectedAt = time.Now()

	for _, pdu := range result.Variables {
		switch pdu.Name {
		case ".1.3.6.1.2.1.1.3.0":
			if v, ok := pdu.Value.(uint32); ok {
				res.SysUpTimeSec = v / 100 // TimeTicks -> секунды
			}
		case ".1.3.6.1.2.1.1.5.0":
			switch v := pdu.Value.(type) {
			case string:
				res.SysName = v
			case []byte:
				res.SysName = string(v)
			}
		case ".1.3.6.1.2.1.25.3.3.1.2.1":
			if v := gosnmp.ToBigInt(pdu.Value); v != nil {
				res.CPULoad = int(v.Int64())
			}
		}
	}

	// Получаем кол-во интерфейсов — ifNumber
	ifResult, err := g.Get([]string{".1.3.6.1.2.1.2.1.0"})
	if err == nil && len(ifResult.Variables) > 0 {
		if v := gosnmp.ToBigInt(ifResult.Variables[0].Value); v != nil {
			res.IfCount = int(v.Int64())
		}
	}

	return res
}
