package server

import (
	"database/sql"
	"encoding/json"
	"log"

	_ "modernc.org/sqlite"
)

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite", filepath)
	if err != nil {
		log.Fatalf("❌ Ошибка открытия БД: %v", err)
	}

	// Создаём таблицу (старые колонки)
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

	// Миграция: добавляем новые колонки (ошибки игнорируем — колонка уже может существовать)
	MigrateDB(db)

	log.Println("✅ База данных инициализирована")
	return db
}

// MigrateDB добавляет новые колонки к существующей таблице metrics.
// Если колонка уже есть — ошибка игнорируется.
func MigrateDB(db *sql.DB) {
	cols := []string{
		`ALTER TABLE metrics ADD COLUMN cpu_user       REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN cpu_system     REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN load_avg_1     REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN load_avg_5     REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN load_avg_15    REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN ram_cached     REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN ram_buffers    REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN swap_used      REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN swap_total     REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN net_interface  TEXT    DEFAULT ''`,
		`ALTER TABLE metrics ADD COLUMN net_bytes_recv REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN net_bytes_sent REAL    DEFAULT 0`,
		`ALTER TABLE metrics ADD COLUMN processes_json TEXT    DEFAULT '[]'`,
	}
	for _, stmt := range cols {
		db.Exec(stmt)
	}
}

func SaveMetric(db *sql.DB, p MetricPayload) error {
	_, err := db.Exec(`
	INSERT INTO metrics (
		node_name, os, ip, uptime, timestamp,
		cpu_usage, cpu_user, cpu_system,
		load_avg_1, load_avg_5, load_avg_15,
		ram_usage, ram_total, ram_cached, ram_buffers,
		swap_used, swap_total,
		disk_usage, rdp_running, smb_running,
		net_interface, net_bytes_recv, net_bytes_sent,
		processes_json
	) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		p.NodeName, p.OS, p.IP, p.Uptime, p.Timestamp,
		p.CPUUsage, p.CPUUser, p.CPUSystem,
		p.LoadAvg1, p.LoadAvg5, p.LoadAvg15,
		p.RAMUsage, p.RAMTotal, p.RAMCached, p.RAMBuffers,
		p.SwapUsed, p.SwapTotal,
		p.DiskUsage, p.RDPRunning, p.SMBRunning,
		p.NetInterface, p.NetBytesRecv, p.NetBytesSent,
		p.ProcessesJSON,
	)
	return err
}

func GetLatestNodes(db *sql.DB) ([]NodeSummary, error) {
	// 1. Список всех уникальных узлов
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
		// 2. Последняя метрика узла
		var last MetricPayload
		var (
			cpuUser, cpuSystem        sql.NullFloat64
			la1, la5, la15            sql.NullFloat64
			ramCached, ramBuffers     sql.NullFloat64
			swapUsed, swapTotal       sql.NullFloat64
			netIface                  sql.NullString
			netRecv, netSent          sql.NullFloat64
			procsJSON                 sql.NullString
		)

		err := db.QueryRow(`
			SELECT
				timestamp, cpu_usage, cpu_user, cpu_system,
				load_avg_1, load_avg_5, load_avg_15,
				ram_usage, ram_total, ram_cached, ram_buffers,
				swap_used, swap_total,
				disk_usage, rdp_running, smb_running,
				os, ip, uptime,
				net_interface, net_bytes_recv, net_bytes_sent,
				processes_json
			FROM metrics
			WHERE node_name = ?
			ORDER BY timestamp DESC LIMIT 1`, name).Scan(
			&last.Timestamp, &last.CPUUsage, &cpuUser, &cpuSystem,
			&la1, &la5, &la15,
			&last.RAMUsage, &last.RAMTotal, &ramCached, &ramBuffers,
			&swapUsed, &swapTotal,
			&last.DiskUsage, &last.RDPRunning, &last.SMBRunning,
			&last.OS, &last.IP, &last.Uptime,
			&netIface, &netRecv, &netSent,
			&procsJSON,
		)
		if err != nil {
			log.Printf("Ошибка получения метрики для %s: %v", name, err)
			continue
		}

		// 3. История CPU (20 точек, от старых к новым)
		cpuHistory := queryCPUHistory(db, name)

		// 4. История RAM % (20 точек)
		ramHistory := queryRAMHistory(db, name)

		// 5. История сети (20 точек)
		netHistory := queryNetHistory(db, name)

		// 6. Парсим процессы из JSON
		var processes []ProcessInfo
		if procsJSON.Valid && procsJSON.String != "" && procsJSON.String != "null" {
			json.Unmarshal([]byte(procsJSON.String), &processes)
		}

		summary := NodeSummary{
			Name:         name,
			OS:           last.OS,
			IP:           last.IP,
			Online:       true,
			CPU:          int(last.CPUUsage),
			CPUUser:      cpuUser.Float64,
			CPUSystem:    cpuSystem.Float64,
			LoadAvg1:     la1.Float64,
			LoadAvg5:     la5.Float64,
			LoadAvg15:    la15.Float64,
			RAMUsed:      last.RAMUsage,
			RAMTotal:     last.RAMTotal,
			RAMCached:    ramCached.Float64,
			RAMBuffers:   ramBuffers.Float64,
			SwapUsed:     swapUsed.Float64,
			SwapTotal:    swapTotal.Float64,
			DiskUsage:    last.DiskUsage,
			RDPRunning:   last.RDPRunning,
			SMBRunning:   last.SMBRunning,
			Uptime:       last.Uptime,
			Ping:         1,
			NetInterface: netIface.String,
			NetRecvSec:   netRecv.Float64,
			NetSentSec:   netSent.Float64,
			CPUHistory:   cpuHistory,
			RAMHistory:   ramHistory,
			NetHistory:   netHistory,
			Processes:    processes,
		}

		nodes = append(nodes, summary)
	}

	return nodes, nil
}

func queryCPUHistory(db *sql.DB, name string) []CpuPoint {
	rows, err := db.Query(
		`SELECT cpu_usage FROM metrics WHERE node_name = ? ORDER BY timestamp DESC LIMIT 20`, name)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var temp []int
	for rows.Next() {
		var v float64
		if err := rows.Scan(&v); err == nil {
			temp = append(temp, int(v))
		}
	}
	// Разворачиваем: от старых к новым
	result := make([]CpuPoint, len(temp))
	for i, v := range temp {
		result[len(temp)-1-i] = CpuPoint{Value: v}
	}
	return result
}

func queryRAMHistory(db *sql.DB, name string) []RamPoint {
	rows, err := db.Query(
		`SELECT ram_usage, ram_total FROM metrics WHERE node_name = ? ORDER BY timestamp DESC LIMIT 20`, name)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var temp []int
	for rows.Next() {
		var used, total float64
		if err := rows.Scan(&used, &total); err == nil {
			pct := 0
			if total > 0 {
				pct = int(used / total * 100)
			}
			temp = append(temp, pct)
		}
	}
	result := make([]RamPoint, len(temp))
	for i, v := range temp {
		result[len(temp)-1-i] = RamPoint{Value: v}
	}
	return result
}

func queryNetHistory(db *sql.DB, name string) []NetPoint {
	rows, err := db.Query(`
		SELECT COALESCE(net_bytes_recv, 0), COALESCE(net_bytes_sent, 0)
		FROM metrics WHERE node_name = ? ORDER BY timestamp DESC LIMIT 20`, name)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var temp []NetPoint
	for rows.Next() {
		var recv, sent float64
		if err := rows.Scan(&recv, &sent); err == nil {
			temp = append(temp, NetPoint{Recv: recv, Sent: sent})
		}
	}
	result := make([]NetPoint, len(temp))
	for i, v := range temp {
		result[len(temp)-1-i] = v
	}
	return result
}
