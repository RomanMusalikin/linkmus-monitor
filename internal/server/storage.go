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

	MigrateDB(db)

	log.Println("✅ База данных инициализирована")
	return db
}

// MigrateDB добавляет новые колонки к существующей таблице metrics.
// Ошибки игнорируются — колонка может уже существовать.
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
	}
	for _, stmt := range cols {
		db.Exec(stmt)
	}
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
		process_count, processes_json, top_mem_json, fsrm_json
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.NodeName, p.OS, p.IP, p.Uptime, p.BootTime, p.Timestamp, p.LoggedUsers,
		p.CPUUsage, p.CPUUser, p.CPUSystem, p.CPUIOwait, p.CPUSteal, p.CPUTemp,
		p.CPUModel, p.CPUFreqMHz, p.CPUCoresJSON,
		p.LoadAvg1, p.LoadAvg5, p.LoadAvg15,
		p.RAMUsage, p.RAMTotal, p.RAMCached, p.RAMBuffers, p.SwapUsed, p.SwapTotal,
		p.DiskUsage, p.DiskReadSec, p.DiskWriteSec, p.DiskQueue, p.DisksJSON,
		p.RDPRunning, p.SMBRunning,
		p.NetInterface, p.NetBytesRecv, p.NetBytesSent, p.AllIfacesJSON,
		p.TCPTotal, p.TCPEstablished, p.TCPTimeWait,
		p.ProcessCount, p.ProcessesJSON, p.TopMemJSON, p.FSRMJson,
	)
	return err
}

func GetLatestNodes(db *sql.DB) ([]NodeSummary, error) {
	rows, err := db.Query(`SELECT DISTINCT node_name FROM metrics`)
	if err != nil {
		return nil, err
	}
	var nodeNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			nodeNames = append(nodeNames, name)
		}
	}
	rows.Close()

	var nodes []NodeSummary

	for _, name := range nodeNames {
		var (
			lastTimestamp                       string
			cpuUsage                            float64
			cpuUser, cpuSystem                  sql.NullFloat64
			cpuIOwait, cpuSteal, cpuTemp        sql.NullFloat64
			cpuModel                            sql.NullString
			cpuFreq                             sql.NullFloat64
			cpuCoresJSON                        sql.NullString
			la1, la5, la15                      sql.NullFloat64
			ramUsage, ramTotal                  float64
			ramCached, ramBuffers               sql.NullFloat64
			swapUsed, swapTotal                 sql.NullFloat64
			diskUsage                           float64
			diskReadSec, diskWriteSec           sql.NullFloat64
			diskQueue                           sql.NullFloat64
			disksJSON                           sql.NullString
			fsrmJSON                            sql.NullString
			rdpRunning, smbRunning              bool
			os_, ip, uptime                     string
			bootTime                            sql.NullString
			netIface                            sql.NullString
			netRecv, netSent                    sql.NullFloat64
			allIfacesJSON                       sql.NullString
			tcpTotal, tcpEstab, tcpTW           sql.NullInt64
			processCount, loggedUsers           sql.NullInt64
			procsJSON, topMemJSON               sql.NullString
		)

		err := db.QueryRow(`
			SELECT
				timestamp,
				cpu_usage, cpu_user, cpu_system, cpu_iowait, cpu_steal, cpu_temp,
				cpu_model, cpu_freq_mhz, cpu_cores_json,
				load_avg_1, load_avg_5, load_avg_15,
				ram_usage, ram_total, ram_cached, ram_buffers, swap_used, swap_total,
				disk_usage, disk_read_sec, disk_write_sec, COALESCE(disk_queue,0), disks_json,
				rdp_running, smb_running,
				os, ip, uptime, boot_time,
				net_interface, net_bytes_recv, net_bytes_sent, all_ifaces_json,
				tcp_total, tcp_established, tcp_timewait,
				process_count, logged_users,
				processes_json, top_mem_json, COALESCE(fsrm_json,'[]')
			FROM metrics
			WHERE node_name = ?
			ORDER BY timestamp DESC LIMIT 1`, name).Scan(
			&lastTimestamp,
			&cpuUsage, &cpuUser, &cpuSystem, &cpuIOwait, &cpuSteal, &cpuTemp,
			&cpuModel, &cpuFreq, &cpuCoresJSON,
			&la1, &la5, &la15,
			&ramUsage, &ramTotal, &ramCached, &ramBuffers, &swapUsed, &swapTotal,
			&diskUsage, &diskReadSec, &diskWriteSec, &diskQueue, &disksJSON,
			&rdpRunning, &smbRunning,
			&os_, &ip, &uptime, &bootTime,
			&netIface, &netRecv, &netSent, &allIfacesJSON,
			&tcpTotal, &tcpEstab, &tcpTW,
			&processCount, &loggedUsers,
			&procsJSON, &topMemJSON, &fsrmJSON,
		)
		if err != nil {
			log.Printf("Ошибка получения метрики для %s: %v", name, err)
			continue
		}

		// Онлайн-статус: последняя метрика не старше 30 секунд
		online := false
		lastSeen := lastTimestamp
		if t, err := time.Parse(time.RFC3339, lastTimestamp); err == nil {
			online = time.Since(t) < 30*time.Second
			lastSeen = t.Local().Format("02.01 15:04:05")
		}

		// Парсим JSON-поля
		var cpuCores []float64
		if cpuCoresJSON.Valid && cpuCoresJSON.String != "" && cpuCoresJSON.String != "null" {
			json.Unmarshal([]byte(cpuCoresJSON.String), &cpuCores)
		}

		var disks []DiskInfo
		if disksJSON.Valid && disksJSON.String != "" && disksJSON.String != "null" {
			json.Unmarshal([]byte(disksJSON.String), &disks)
		}

		var allIfaces []NetIfaceInfo
		if allIfacesJSON.Valid && allIfacesJSON.String != "" && allIfacesJSON.String != "null" {
			json.Unmarshal([]byte(allIfacesJSON.String), &allIfaces)
		}

		var processes []ProcessInfo
		if procsJSON.Valid && procsJSON.String != "" && procsJSON.String != "null" {
			json.Unmarshal([]byte(procsJSON.String), &processes)
		}

		var topMemProcesses []ProcessInfo
		if topMemJSON.Valid && topMemJSON.String != "" && topMemJSON.String != "null" {
			json.Unmarshal([]byte(topMemJSON.String), &topMemProcesses)
		}

		var fsrmList []FSRMInfo
		if fsrmJSON.Valid && fsrmJSON.String != "" && fsrmJSON.String != "null" {
			json.Unmarshal([]byte(fsrmJSON.String), &fsrmList)
		}

		probe := GetProbe(ip)
		snmp := GetSNMP(ip)
		summary := NodeSummary{
			Name:         name,
			OS:           os_,
			IP:           ip,
			Online:       online,
			LastSeen:     lastSeen,
			Uptime:       uptime,
			BootTime:     bootTime.String,

			CPU:          int(cpuUsage),
			CPUUser:      cpuUser.Float64,
			CPUSystem:    cpuSystem.Float64,
			CPUIOwait:    cpuIOwait.Float64,
			CPUSteal:     cpuSteal.Float64,
			CPUTemp:      cpuTemp.Float64,
			CPUModel:     cpuModel.String,
			CPUFreqMHz:   cpuFreq.Float64,
			CPUCores:     cpuCores,
			LoadAvg1:     la1.Float64,
			LoadAvg5:     la5.Float64,
			LoadAvg15:    la15.Float64,
			RAMUsed:      ramUsage,
			RAMTotal:     ramTotal,
			RAMCached:    ramCached.Float64,
			RAMBuffers:   ramBuffers.Float64,
			SwapUsed:     swapUsed.Float64,
			SwapTotal:    swapTotal.Float64,
			DiskUsage:    diskUsage,
			DiskReadSec:  diskReadSec.Float64,
			DiskWriteSec: diskWriteSec.Float64,
			DiskQueue:    diskQueue.Float64,
			Disks:        disks,
			RDPRunning:   rdpRunning,
			SMBRunning:   smbRunning,
			NetInterface: netIface.String,
			NetRecvSec:   netRecv.Float64,
			NetSentSec:   netSent.Float64,
			AllIfaces:    allIfaces,
			TCPTotal:     int(tcpTotal.Int64),
			TCPEstablished: int(tcpEstab.Int64),
			TCPTimeWait:  int(tcpTW.Int64),
			ProcessCount: int(processCount.Int64),
			LoggedUsers:  int(loggedUsers.Int64),
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
			CPUHistory:     queryCPUHistory(db, name),
			RAMHistory:     queryRAMHistory(db, name),
			NetHistory:     queryNetHistory(db, name),
		}

		nodes = append(nodes, summary)
	}

	return nodes, nil
}

func formatHistoryTime(ts string) string {
	if t, err := time.Parse(time.RFC3339, ts); err == nil {
		return t.Local().Format("15:04:05")
	}
	return ts
}

func queryCPUHistory(db *sql.DB, name string) []CpuPoint {
	rows, err := db.Query(
		`SELECT cpu_usage, timestamp FROM metrics WHERE node_name = ? ORDER BY timestamp DESC LIMIT 20`, name)
	if err != nil {
		return nil
	}
	defer rows.Close()

	type row struct {
		v  int
		ts string
	}
	var temp []row
	for rows.Next() {
		var v float64
		var ts string
		if err := rows.Scan(&v, &ts); err == nil {
			temp = append(temp, row{int(v), ts})
		}
	}
	result := make([]CpuPoint, len(temp))
	for i, r := range temp {
		result[len(temp)-1-i] = CpuPoint{Value: r.v, Time: formatHistoryTime(r.ts)}
	}
	return result
}

func queryRAMHistory(db *sql.DB, name string) []RamPoint {
	rows, err := db.Query(
		`SELECT ram_usage, ram_total, timestamp FROM metrics WHERE node_name = ? ORDER BY timestamp DESC LIMIT 20`, name)
	if err != nil {
		return nil
	}
	defer rows.Close()

	type row struct {
		pct int
		ts  string
	}
	var temp []row
	for rows.Next() {
		var used, total float64
		var ts string
		if err := rows.Scan(&used, &total, &ts); err == nil {
			pct := 0
			if total > 0 {
				pct = int(used / total * 100)
			}
			temp = append(temp, row{pct, ts})
		}
	}
	result := make([]RamPoint, len(temp))
	for i, r := range temp {
		result[len(temp)-1-i] = RamPoint{Value: r.pct, Time: formatHistoryTime(r.ts)}
	}
	return result
}

func queryNetHistory(db *sql.DB, name string) []NetPoint {
	rows, err := db.Query(`
		SELECT COALESCE(net_bytes_recv, 0), COALESCE(net_bytes_sent, 0), timestamp
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
			temp = append(temp, NetPoint{Recv: recv, Sent: sent, Time: formatHistoryTime(ts)})
		}
	}
	result := make([]NetPoint, len(temp))
	for i, v := range temp {
		result[len(temp)-1-i] = v
	}
	return result
}
