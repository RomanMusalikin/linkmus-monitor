package server

import (
	"database/sql"
	"encoding/json"
	"log"
	"time"

	_ "modernc.org/sqlite"
)

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite", filepath)
	if err != nil {
		log.Fatalf("❌ Ошибка открытия БД: %v", err)
	}

	// SQLite WAL: несколько читателей работают параллельно без блокировок.
	// Запись сериализуется самим SQLite, лимит на соединения не нужен.
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(0)

	pragmas := []string{
		`PRAGMA journal_mode=WAL`,
		`PRAGMA synchronous=NORMAL`,
		`PRAGMA cache_size=-32000`,
		`PRAGMA temp_store=MEMORY`,
		`PRAGMA mmap_size=268435456`,
	}
	for _, p := range pragmas {
		db.Exec(p)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS users (
		id       INTEGER PRIMARY KEY AUTOINCREMENT,
		login    TEXT UNIQUE NOT NULL,
		password TEXT NOT NULL
	)`)
	if err != nil {
		log.Fatalf("❌ Ошибка создания таблицы users: %v", err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS sessions (
		token      TEXT PRIMARY KEY,
		user_id    INTEGER NOT NULL REFERENCES users(id),
		created_at DATETIME NOT NULL,
		expires_at DATETIME NOT NULL
	)`)
	if err != nil {
		log.Fatalf("❌ Ошибка создания таблицы sessions: %v", err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS metrics (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		node_name   TEXT,
		os          TEXT,
		ip          TEXT,
		uptime      TEXT,
		timestamp   DATETIME,
		cpu_usage   REAL,
		ram_usage   REAL,
		ram_total   REAL,
		disk_usage  REAL,
		rdp_running BOOLEAN,
		smb_running BOOLEAN
	)`)
	if err != nil {
		log.Fatalf("❌ Ошибка создания таблицы: %v", err)
	}

	_, err = db.Exec(`
	CREATE TABLE IF NOT EXISTS node_aliases (
		node_name TEXT PRIMARY KEY,
		alias     TEXT NOT NULL
	)`)
	if err != nil {
		log.Fatalf("❌ Ошибка создания таблицы node_aliases: %v", err)
	}

	MigrateDB(db)

	log.Println("✅ База данных инициализирована")
	return db
}

// MigrateDB добавляет новые колонки и индексы.
// Ошибки ALTER TABLE игнорируются — колонка может уже существовать.
func MigrateDB(db *sql.DB) {
	cols := []string{
		`ALTER TABLE metrics ADD COLUMN cpu_user        REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN cpu_system      REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN cpu_model       TEXT    DEFAULT ''`,
		`ALTER TABLE metrics ADD COLUMN cpu_freq_mhz    REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN cpu_cores_json  TEXT    DEFAULT '[]'`,
		`ALTER TABLE metrics ADD COLUMN load_avg_1      REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN load_avg_5      REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN load_avg_15     REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN ram_cached      REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN ram_buffers     REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN swap_used       REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN swap_total      REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN disk_read_sec   REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN disk_write_sec  REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN disks_json      TEXT    DEFAULT '[]'`,
		`ALTER TABLE metrics ADD COLUMN net_interface   TEXT    DEFAULT ''`,
		`ALTER TABLE metrics ADD COLUMN net_bytes_recv  REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN net_bytes_sent  REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN all_ifaces_json TEXT    DEFAULT '[]'`,
		`ALTER TABLE metrics ADD COLUMN process_count   INTEGER DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN logged_users    INTEGER DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN boot_time       TEXT    DEFAULT ''`,
		`ALTER TABLE metrics ADD COLUMN processes_json  TEXT    DEFAULT '[]'`,
		`ALTER TABLE metrics ADD COLUMN top_mem_json    TEXT    DEFAULT '[]'`,
		`ALTER TABLE metrics ADD COLUMN cpu_iowait      REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN cpu_steal       REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN cpu_temp        REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN tcp_total       INTEGER DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN tcp_established INTEGER DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN tcp_timewait    INTEGER DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN disk_queue      REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN fsrm_json       TEXT    DEFAULT '[]'`,
		`ALTER TABLE metrics ADD COLUMN agent_version   TEXT    DEFAULT ''`,
	}
	for _, stmt := range cols {
		db.Exec(stmt)
	}

	// Индексы: критически важны для производительности при большом числе узлов
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_metrics_node_ts ON metrics(node_name, timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_ip      ON metrics(ip)`,
		`CREATE INDEX IF NOT EXISTS idx_metrics_ts      ON metrics(timestamp)`,
	}
	for _, stmt := range indexes {
		if _, err := db.Exec(stmt); err != nil {
			log.Printf("⚠️  Индекс: %v", err)
		}
	}
}

// StartDataCleanup запускает фоновую очистку метрик старше 25 часов.
func StartDataCleanup(db *sql.DB) {
	go func() {
		for {
			time.Sleep(time.Hour)
			res, err := db.Exec(`DELETE FROM metrics WHERE timestamp < datetime('now', '-25 hours')`)
			if err != nil {
				log.Printf("⚠️  Очистка данных: %v", err)
				continue
			}
			if n, _ := res.RowsAffected(); n > 0 {
				log.Printf("🗑️  Очистка: удалено %d устаревших записей", n)
			}
		}
	}()
}

func GetAllAliases(db *sql.DB) map[string]string {
	rows, err := db.Query(`SELECT node_name, alias FROM node_aliases`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	m := make(map[string]string)
	for rows.Next() {
		var name, alias string
		if rows.Scan(&name, &alias) == nil {
			m[name] = alias
		}
	}
	return m
}

func SetNodeAlias(db *sql.DB, name, alias string) error {
	if alias == "" {
		_, err := db.Exec(`DELETE FROM node_aliases WHERE node_name = ?`, name)
		return err
	}
	_, err := db.Exec(
		`INSERT INTO node_aliases(node_name, alias) VALUES(?,?) ON CONFLICT(node_name) DO UPDATE SET alias=excluded.alias`,
		name, alias,
	)
	return err
}

func SaveMetric(db *sql.DB, p MetricPayload) error {
	_, err := db.Exec(`
	INSERT INTO metrics (
		node_name, os, ip, uptime, boot_time, timestamp, logged_users,
		cpu_usage, cpu_user, cpu_system, cpu_iowait, cpu_steal, cpu_temp,
		cpu_model, cpu_freq_mhz, cpu_cores_json,
		load_avg_1, load_avg_5, load_avg_15,
		ram_usage, ram_total, ram_cached, ram_buffers, swap_used, swap_total,
		disk_usage, disk_read_sec, disk_write_sec, disk_queue, disks_json,
		rdp_running, smb_running,
		net_interface, net_bytes_recv, net_bytes_sent, all_ifaces_json,
		tcp_total, tcp_established, tcp_timewait,
		process_count, processes_json, top_mem_json, fsrm_json, agent_version
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.NodeName, p.OS, p.IP, p.Uptime, p.BootTime, p.Timestamp, p.LoggedUsers,
		p.CPUUsage, p.CPUUser, p.CPUSystem, p.CPUIOwait, p.CPUSteal, p.CPUTemp,
		p.CPUModel, p.CPUFreqMHz, p.CPUCoresJSON,
		p.LoadAvg1, p.LoadAvg5, p.LoadAvg15,
		p.RAMUsage, p.RAMTotal, p.RAMCached, p.RAMBuffers, p.SwapUsed, p.SwapTotal,
		p.DiskUsage, p.DiskReadSec, p.DiskWriteSec, p.DiskQueue, p.DisksJSON,
		p.RDPRunning, p.SMBRunning,
		p.NetInterface, p.NetBytesRecv, p.NetBytesSent, p.AllIfacesJSON,
		p.TCPTotal, p.TCPEstablished, p.TCPTimeWait,
		p.ProcessCount, p.ProcessesJSON, p.TopMemJSON, p.FSRMJson, p.AgentVersion,
	)
	return err
}

// rawNodeRow — промежуточная структура для чтения строки из JOIN-запроса.
// Нужна чтобы закрыть cursor ДО запуска history-запросов (иначе дедлок при лимите соединений).
type rawNodeRow struct {
	name, os_, ip, uptime, bootTime, lastTimestamp string
	loggedUsers                                     int
	cpuUsage, cpuUser, cpuSystem                   float64
	cpuIOwait, cpuSteal, cpuTemp                   float64
	cpuModel                                        string
	cpuFreq                                         float64
	cpuCoresJSON                                    string
	la1, la5, la15                                  float64
	ramUsage, ramTotal, ramCached, ramBuffers       float64
	swapUsed, swapTotal                             float64
	diskUsage, diskReadSec, diskWriteSec, diskQueue float64
	disksJSON                                       string
	rdpRunning, smbRunning                          bool
	netIface                                        string
	netRecv, netSent                                float64
	allIfacesJSON                                   string
	tcpTotal, tcpEstab, tcpTW                       int
	processCount                                    int
	procsJSON, topMemJSON, fsrmJSON                 string
	agentVersion                                    string
}

// GetLatestNodes возвращает последние метрики для всех узлов.
// Использует один JOIN-запрос вместо N+1 для получения актуальных строк.
func GetLatestNodes(db *sql.DB, full bool) ([]NodeSummary, error) {
	rows, err := db.Query(`
		SELECT
			m.node_name, m.os, m.ip, m.uptime, COALESCE(m.boot_time,''), m.timestamp,
			COALESCE(m.logged_users,0),
			m.cpu_usage, COALESCE(m.cpu_user,0), COALESCE(m.cpu_system,0),
			COALESCE(m.cpu_iowait,0), COALESCE(m.cpu_steal,0), COALESCE(m.cpu_temp,0),
			COALESCE(m.cpu_model,''), COALESCE(m.cpu_freq_mhz,0), COALESCE(m.cpu_cores_json,'[]'),
			COALESCE(m.load_avg_1,0), COALESCE(m.load_avg_5,0), COALESCE(m.load_avg_15,0),
			m.ram_usage, m.ram_total, COALESCE(m.ram_cached,0), COALESCE(m.ram_buffers,0),
			COALESCE(m.swap_used,0), COALESCE(m.swap_total,0),
			m.disk_usage, COALESCE(m.disk_read_sec,0), COALESCE(m.disk_write_sec,0),
			COALESCE(m.disk_queue,0), COALESCE(m.disks_json,'[]'),
			m.rdp_running, m.smb_running,
			COALESCE(m.net_interface,''), COALESCE(m.net_bytes_recv,0), COALESCE(m.net_bytes_sent,0),
			COALESCE(m.all_ifaces_json,'[]'),
			COALESCE(m.tcp_total,0), COALESCE(m.tcp_established,0), COALESCE(m.tcp_timewait,0),
			COALESCE(m.process_count,0),
			COALESCE(m.processes_json,'[]'), COALESCE(m.top_mem_json,'[]'),
			COALESCE(m.fsrm_json,'[]'),
			COALESCE(m.agent_version,'')
		FROM metrics m
		INNER JOIN (
			SELECT node_name, MAX(timestamp) AS ts
			FROM metrics
			GROUP BY node_name
		) latest ON m.node_name = latest.node_name AND m.timestamp = latest.ts
		ORDER BY m.node_name
	`)
	if err != nil {
		return nil, err
	}

	// Читаем все строки в память и сразу закрываем cursor.
	// History-запросы запускаются только после этого, чтобы не было дедлока.
	var raws []rawNodeRow
	for rows.Next() {
		var r rawNodeRow
		if err := rows.Scan(
			&r.name, &r.os_, &r.ip, &r.uptime, &r.bootTime, &r.lastTimestamp,
			&r.loggedUsers,
			&r.cpuUsage, &r.cpuUser, &r.cpuSystem,
			&r.cpuIOwait, &r.cpuSteal, &r.cpuTemp,
			&r.cpuModel, &r.cpuFreq, &r.cpuCoresJSON,
			&r.la1, &r.la5, &r.la15,
			&r.ramUsage, &r.ramTotal, &r.ramCached, &r.ramBuffers,
			&r.swapUsed, &r.swapTotal,
			&r.diskUsage, &r.diskReadSec, &r.diskWriteSec, &r.diskQueue, &r.disksJSON,
			&r.rdpRunning, &r.smbRunning,
			&r.netIface, &r.netRecv, &r.netSent, &r.allIfacesJSON,
			&r.tcpTotal, &r.tcpEstab, &r.tcpTW,
			&r.processCount,
			&r.procsJSON, &r.topMemJSON, &r.fsrmJSON,
			&r.agentVersion,
		); err != nil {
			log.Printf("Ошибка сканирования метрики: %v", err)
			continue
		}
		raws = append(raws, r)
	}
	rows.Close() // явно закрываем до history-запросов

	aliases := GetAllAliases(db)

	var nodes []NodeSummary
	for _, r := range raws {
		online := false
		lastSeen := r.lastTimestamp
		if t, err := time.Parse(time.RFC3339, r.lastTimestamp); err == nil {
			online = time.Since(t) < 30*time.Second
			lastSeen = t.Local().Format("02.01 15:04:05")
		}

		var cpuCores []float64
		json.Unmarshal([]byte(r.cpuCoresJSON), &cpuCores)
		var disks []DiskInfo
		json.Unmarshal([]byte(r.disksJSON), &disks)
		var allIfaces []NetIfaceInfo
		json.Unmarshal([]byte(r.allIfacesJSON), &allIfaces)
		var processes []ProcessInfo
		json.Unmarshal([]byte(r.procsJSON), &processes)
		var topMemProcesses []ProcessInfo
		json.Unmarshal([]byte(r.topMemJSON), &topMemProcesses)
		var fsrmList []FSRMInfo
		json.Unmarshal([]byte(r.fsrmJSON), &fsrmList)

		probe := GetProbe(r.ip)
		snmp := GetSNMP(r.ip)

		displayName := r.name
		if a, ok := aliases[r.name]; ok && a != "" {
			displayName = a
		}

		nodes = append(nodes, NodeSummary{
			Name:        r.name,
			DisplayName: displayName,
			OS:          r.os_,
			IP:       r.ip,
			Online:   online,
			LastSeen: lastSeen,
			Uptime:   r.uptime,
			BootTime: r.bootTime,

			CPU:          int(r.cpuUsage),
			CPUUser:      r.cpuUser,
			CPUSystem:    r.cpuSystem,
			CPUIOwait:    r.cpuIOwait,
			CPUSteal:     r.cpuSteal,
			CPUTemp:      r.cpuTemp,
			CPUModel:     r.cpuModel,
			CPUFreqMHz:   r.cpuFreq,
			CPUCores:     cpuCores,
			LoadAvg1:     r.la1,
			LoadAvg5:     r.la5,
			LoadAvg15:    r.la15,
			RAMUsed:      r.ramUsage,
			RAMTotal:     r.ramTotal,
			RAMCached:    r.ramCached,
			RAMBuffers:   r.ramBuffers,
			SwapUsed:     r.swapUsed,
			SwapTotal:    r.swapTotal,
			DiskUsage:    r.diskUsage,
			DiskReadSec:  r.diskReadSec,
			DiskWriteSec: r.diskWriteSec,
			DiskQueue:    r.diskQueue,
			Disks:        disks,
			RDPRunning:   r.rdpRunning,
			SMBRunning:   r.smbRunning,
			NetInterface: r.netIface,
			NetRecvSec:   r.netRecv,
			NetSentSec:   r.netSent,
			AllIfaces:    allIfaces,
			TCPTotal:     r.tcpTotal,
			TCPEstablished: r.tcpEstab,
			TCPTimeWait:  r.tcpTW,
			ProcessCount: r.processCount,
			LoggedUsers:  r.loggedUsers,
			Processes:    processes,
			TopMemProcesses: topMemProcesses,
			SSHReachable:   probe.SSHReachable,
			RDPReachable:   probe.RDPReachable,
			SMBReachable:   probe.SMBReachable,
			HTTPReachable:  probe.HTTPReachable,
			WinRMReachable: probe.WinRMReachable,
			DNSReachable:   probe.DNSReachable,
			SSHMs:          probe.SSHMs,
			RDPMs:          probe.RDPMs,
			SMBMs:          probe.SMBMs,
			HTTPMs:         probe.HTTPMs,
			WinRMMs:        probe.WinRMMs,
			DNSMs:          probe.DNSMs,
			SNMPCollected:  snmp.Collected,
			SNMPSysUpTime:  snmp.SysUpTimeSec,
			SNMPSysName:    snmp.SysName,
			SNMPCPULoad:    snmp.CPULoad,
			SNMPIfCount:    snmp.IfCount,
			FSRM:           fsrmList,
			AgentVersion:   r.agentVersion,
			CPUHistory:  queryCPUHistory(db, r.name, full),
			RAMHistory:  queryRAMHistory(db, r.name, full),
			NetHistory:  queryNetHistory(db, r.name, full),
			DiskHistory: queryDiskHistory(db, r.name, full),
		})
	}

	return nodes, nil
}

func formatHistoryTime(ts string) string {
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t.Local().Format("15:04:05")
	}
	return ts
}

const dayBucketDur = 10 * time.Minute

func queryCPUHistory(db *sql.DB, name string, full bool) []CpuPoint {
	if !full {
		rows, err := db.Query(
			`SELECT cpu_usage, timestamp FROM metrics WHERE node_name = ? ORDER BY timestamp DESC LIMIT 20`, name)
		if err != nil {
			return nil
		}
		defer rows.Close()
		var temp []struct {
			v  int
			ts string
		}
		for rows.Next() {
			var v float64
			var ts string
			if err := rows.Scan(&v, &ts); err == nil {
				temp = append(temp, struct {
					v  int
					ts string
				}{int(v), ts})
			}
		}
		result := make([]CpuPoint, len(temp))
		for i, r := range temp {
			v := r.v
			result[len(temp)-1-i] = CpuPoint{Value: &v, Time: formatHistoryTime(r.ts)}
		}
		return result
	}

	rows, err := db.Query(
		`SELECT cpu_usage, timestamp FROM metrics WHERE node_name = ? ORDER BY timestamp ASC LIMIT 8640`, name)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var vals []float64
	var times []time.Time
	for rows.Next() {
		var v float64
		var ts string
		if err := rows.Scan(&v, &ts); err == nil {
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				vals = append(vals, v)
				times = append(times, t)
			}
		}
	}
	if len(vals) == 0 {
		return nil
	}

	now := time.Now().Local()
	start := now.Add(-24 * time.Hour).Truncate(dayBucketDur)
	type bucket struct{ sum float64; n int }
	grid := make(map[time.Time]*bucket)
	for i, t := range times {
		key := t.Local().Truncate(dayBucketDur)
		if key.Before(start) {
			continue
		}
		if grid[key] == nil {
			grid[key] = &bucket{}
		}
		grid[key].sum += vals[i]
		grid[key].n++
	}

	var result []CpuPoint
	for t := start; !t.After(now); t = t.Add(dayBucketDur) {
		label := t.Format("15:04")
		if b := grid[t]; b != nil {
			avg := int(b.sum / float64(b.n))
			result = append(result, CpuPoint{Value: &avg, Time: label})
		} else {
			result = append(result, CpuPoint{Value: nil, Time: label})
		}
	}
	return result
}

func queryRAMHistory(db *sql.DB, name string, full bool) []RamPoint {
	if !full {
		rows, err := db.Query(
			`SELECT ram_usage, ram_total, timestamp FROM metrics WHERE node_name = ? ORDER BY timestamp DESC LIMIT 20`, name)
		if err != nil {
			return nil
		}
		defer rows.Close()
		var temp []struct {
			pct int
			ts  string
		}
		for rows.Next() {
			var used, total float64
			var ts string
			if err := rows.Scan(&used, &total, &ts); err == nil {
				pct := 0
				if total > 0 {
					pct = int(used / total * 100)
				}
				temp = append(temp, struct {
					pct int
					ts  string
				}{pct, ts})
			}
		}
		result := make([]RamPoint, len(temp))
		for i, r := range temp {
			v := r.pct
			result[len(temp)-1-i] = RamPoint{Value: &v, Time: formatHistoryTime(r.ts)}
		}
		return result
	}

	rows, err := db.Query(
		`SELECT ram_usage, ram_total, timestamp FROM metrics WHERE node_name = ? ORDER BY timestamp ASC LIMIT 8640`, name)
	if err != nil {
		return nil
	}
	defer rows.Close()

	type rec struct {
		pct float64
		t   time.Time
	}
	var all []rec
	for rows.Next() {
		var used, total float64
		var ts string
		if err := rows.Scan(&used, &total, &ts); err == nil {
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				pct := 0.0
				if total > 0 {
					pct = used / total * 100
				}
				all = append(all, rec{pct, t})
			}
		}
	}
	if len(all) == 0 {
		return nil
	}

	now := time.Now().Local()
	start := now.Add(-24 * time.Hour).Truncate(dayBucketDur)
	type bucket struct{ sum float64; n int }
	grid := make(map[time.Time]*bucket)
	for _, r := range all {
		key := r.t.Local().Truncate(dayBucketDur)
		if key.Before(start) {
			continue
		}
		if grid[key] == nil {
			grid[key] = &bucket{}
		}
		grid[key].sum += r.pct
		grid[key].n++
	}

	var result []RamPoint
	for t := start; !t.After(now); t = t.Add(dayBucketDur) {
		label := t.Format("15:04")
		if b := grid[t]; b != nil {
			avg := int(b.sum / float64(b.n))
			result = append(result, RamPoint{Value: &avg, Time: label})
		} else {
			result = append(result, RamPoint{Value: nil, Time: label})
		}
	}
	return result
}

func queryNetHistory(db *sql.DB, name string, full bool) []NetPoint {
	if !full {
		rows, err := db.Query(`
			SELECT COALESCE(net_bytes_recv,0), COALESCE(net_bytes_sent,0), timestamp
			FROM metrics WHERE node_name = ? ORDER BY timestamp DESC LIMIT 20`, name)
		if err != nil {
			return nil
		}
		defer rows.Close()
		var temp []NetPoint
		for rows.Next() {
			var recv, sent float64
			var ts string
			if err := rows.Scan(&recv, &sent, &ts); err == nil {
				r, s := recv, sent
				temp = append(temp, NetPoint{Recv: &r, Sent: &s, Time: formatHistoryTime(ts)})
			}
		}
		result := make([]NetPoint, len(temp))
		for i, v := range temp {
			result[len(temp)-1-i] = v
		}
		return result
	}

	rows, err := db.Query(`
		SELECT COALESCE(net_bytes_recv,0), COALESCE(net_bytes_sent,0), timestamp
		FROM metrics WHERE node_name = ? ORDER BY timestamp ASC LIMIT 8640`, name)
	if err != nil {
		return nil
	}
	defer rows.Close()

	type rec struct {
		recv, sent float64
		t          time.Time
	}
	var all []rec
	for rows.Next() {
		var recv, sent float64
		var ts string
		if err := rows.Scan(&recv, &sent, &ts); err == nil {
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				all = append(all, rec{recv, sent, t})
			}
		}
	}
	if len(all) == 0 {
		return nil
	}

	now := time.Now().Local()
	start := now.Add(-24 * time.Hour).Truncate(dayBucketDur)
	type bucket struct{ sumR, sumS float64; n int }
	grid := make(map[time.Time]*bucket)
	for _, r := range all {
		key := r.t.Local().Truncate(dayBucketDur)
		if key.Before(start) {
			continue
		}
		if grid[key] == nil {
			grid[key] = &bucket{}
		}
		grid[key].sumR += r.recv
		grid[key].sumS += r.sent
		grid[key].n++
	}

	var result []NetPoint
	for t := start; !t.After(now); t = t.Add(dayBucketDur) {
		label := t.Format("15:04")
		if b := grid[t]; b != nil {
			n := float64(b.n)
			r, s := b.sumR/n, b.sumS/n
			result = append(result, NetPoint{Recv: &r, Sent: &s, Time: label})
		} else {
			result = append(result, NetPoint{Recv: nil, Sent: nil, Time: label})
		}
	}
	return result
}

func queryDiskHistory(db *sql.DB, name string, full bool) []DiskPoint {
	if !full {
		rows, err := db.Query(
			`SELECT disk_usage, timestamp FROM metrics WHERE node_name = ? ORDER BY timestamp DESC LIMIT 20`, name)
		if err != nil {
			return nil
		}
		defer rows.Close()
		var temp []struct {
			v  int
			ts string
		}
		for rows.Next() {
			var v float64
			var ts string
			if err := rows.Scan(&v, &ts); err == nil {
				temp = append(temp, struct {
					v  int
					ts string
				}{int(v), ts})
			}
		}
		result := make([]DiskPoint, len(temp))
		for i, r := range temp {
			v := r.v
			result[len(temp)-1-i] = DiskPoint{Value: &v, Time: formatHistoryTime(r.ts)}
		}
		return result
	}

	rows, err := db.Query(
		`SELECT disk_usage, timestamp FROM metrics WHERE node_name = ? ORDER BY timestamp ASC LIMIT 8640`, name)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var vals []float64
	var times []time.Time
	for rows.Next() {
		var v float64
		var ts string
		if err := rows.Scan(&v, &ts); err == nil {
			if t, err := time.Parse(time.RFC3339, ts); err == nil {
				vals = append(vals, v)
				times = append(times, t)
			}
		}
	}
	if len(vals) == 0 {
		return nil
	}

	now := time.Now().Local()
	start := now.Add(-24 * time.Hour).Truncate(dayBucketDur)
	type bucket struct{ sum float64; n int }
	grid := make(map[time.Time]*bucket)
	for i, t := range times {
		key := t.Local().Truncate(dayBucketDur)
		if key.Before(start) {
			continue
		}
		if grid[key] == nil {
			grid[key] = &bucket{}
		}
		grid[key].sum += vals[i]
		grid[key].n++
	}

	var result []DiskPoint
	for t := start; !t.After(now); t = t.Add(dayBucketDur) {
		label := t.Format("15:04")
		if b := grid[t]; b != nil {
			avg := int(b.sum / float64(b.n))
			result = append(result, DiskPoint{Value: &avg, Time: label})
		} else {
			result = append(result, DiskPoint{Value: nil, Time: label})
		}
	}
	return result
}
