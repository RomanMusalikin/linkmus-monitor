package server

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"
)

// SNMPIfaceResult — трафик одного интерфейса (байт/с)
type SNMPIfaceResult struct {
	Index       int
	Name        string
	SpeedBps    uint64  // ifSpeed (бит/с)
	RecvByteSec float64 // входящий трафик байт/с
	SentByteSec float64 // исходящий трафик байт/с
}

// SNMPResult — метрики, собранные по SNMP с узла
type SNMPResult struct {
	SysUpTimeSec uint32
	SysName      string
	CPULoad      int    // hrProcessorLoad (%)
	IfCount      int    // кол-во интерфейсов
	Ifaces       []SNMPIfaceResult
	Collected    bool
	CollectedAt  time.Time
}

// ifCounters — снапшот счётчиков ifInOctets/ifOutOctets для расчёта дельты
type ifCounters struct {
	InOctets  map[int]uint64
	OutOctets map[int]uint64
	At        time.Time
}

var (
	snmpCache   = map[string]SNMPResult{}
	snmpCacheMu sync.RWMutex

	snmpPrevCounters   = map[string]ifCounters{}
	snmpPrevCountersMu sync.Mutex
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

	// Walk ifTable: ifDescr, ifSpeed, ifInOctets, ifOutOctets
	res.Ifaces = pollIfaceTraffic(g, ip)

	return res
}

// pollIfaceTraffic выполняет BulkWalk по ifTable и возвращает трафик по интерфейсам.
func pollIfaceTraffic(g *gosnmp.GoSNMP, ip string) []SNMPIfaceResult {
	baseOIDs := map[string]string{
		".1.3.6.1.2.1.2.2.1.2":  "descr",     // ifDescr
		".1.3.6.1.2.1.2.2.1.5":  "speed",     // ifSpeed
		".1.3.6.1.2.1.2.2.1.10": "inOctets",  // ifInOctets
		".1.3.6.1.2.1.2.2.1.16": "outOctets", // ifOutOctets
	}

	type ifRow struct {
		Descr     string
		Speed     uint64
		InOctets  uint64
		OutOctets uint64
	}
	rows := map[int]*ifRow{}

	for oid, kind := range baseOIDs {
		pdus, err := g.BulkWalkAll(oid)
		if err != nil {
			continue
		}
		for _, pdu := range pdus {
			// OID вида .1.3.6.1.2.1.2.2.1.2.3 — последнее число = ifIndex
			parts := strings.Split(strings.TrimPrefix(pdu.Name, "."), ".")
			if len(parts) == 0 {
				continue
			}
			idx := 0
			fmt.Sscanf(parts[len(parts)-1], "%d", &idx)
			if idx == 0 {
				continue
			}
			if rows[idx] == nil {
				rows[idx] = &ifRow{}
			}
			switch kind {
			case "descr":
				switch v := pdu.Value.(type) {
				case string:
					rows[idx].Descr = v
				case []byte:
					rows[idx].Descr = string(v)
				}
			case "speed":
				if v := gosnmp.ToBigInt(pdu.Value); v != nil {
					rows[idx].Speed = v.Uint64()
				}
			case "inOctets":
				if v := gosnmp.ToBigInt(pdu.Value); v != nil {
					rows[idx].InOctets = v.Uint64()
				}
			case "outOctets":
				if v := gosnmp.ToBigInt(pdu.Value); v != nil {
					rows[idx].OutOctets = v.Uint64()
				}
			}
		}
	}

	now := time.Now()

	snmpPrevCountersMu.Lock()
	prev, hasPrev := snmpPrevCounters[ip]
	// Сохраняем текущий снапшот
	cur := ifCounters{
		InOctets:  make(map[int]uint64, len(rows)),
		OutOctets: make(map[int]uint64, len(rows)),
		At:        now,
	}
	for idx, r := range rows {
		cur.InOctets[idx] = r.InOctets
		cur.OutOctets[idx] = r.OutOctets
	}
	snmpPrevCounters[ip] = cur
	snmpPrevCountersMu.Unlock()

	var elapsed float64
	if hasPrev && !prev.At.IsZero() {
		elapsed = now.Sub(prev.At).Seconds()
	}

	var result []SNMPIfaceResult
	for idx, r := range rows {
		iface := SNMPIfaceResult{
			Index:    idx,
			Name:     r.Descr,
			SpeedBps: r.Speed,
		}
		if elapsed > 0 && hasPrev {
			if prevIn, ok := prev.InOctets[idx]; ok && r.InOctets >= prevIn {
				iface.RecvByteSec = float64(r.InOctets-prevIn) / elapsed
			}
			if prevOut, ok := prev.OutOctets[idx]; ok && r.OutOctets >= prevOut {
				iface.SentByteSec = float64(r.OutOctets-prevOut) / elapsed
			}
		}
		result = append(result, iface)
	}

	// Сортируем по индексу
	for i := 1; i < len(result); i++ {
		for j := i; j > 0 && result[j].Index < result[j-1].Index; j-- {
			result[j], result[j-1] = result[j-1], result[j]
		}
	}

	return result
}
