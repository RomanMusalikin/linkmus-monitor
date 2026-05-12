package server

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

// HourlyPoint — одна точка долгосрочной истории (nil = нет данных за этот интервал)
type HourlyPoint struct {
	Time      string   `json:"time"`
	CPU       *float64 `json:"cpu"`
	RAM       *float64 `json:"ram"`
	RAMTotal  *float64 `json:"ramTotal"`
	Disk      *float64 `json:"disk"`
	NetRecv   *float64 `json:"netRecv"`
	NetSent   *float64 `json:"netSent"`
	DiskRead  *float64 `json:"diskRead"`
	DiskWrite *float64 `json:"diskWrite"`
}

// StartHourlyAggregator агрегирует метрики в metrics_hourly раз в час.
func StartHourlyAggregator(db *sql.DB) {
	run := func() {
		if err := aggregateLastHour(db); err != nil {
			log.Printf("⚠️  hourly aggregator: %v", err)
		}
	}
	go func() {
		run()
		for {
			now := time.Now()
			next := now.Truncate(time.Hour).Add(time.Hour + time.Minute)
			time.Sleep(time.Until(next))
			run()
		}
	}()
}

// StartHalfHourlyAggregator агрегирует метрики в metrics_30min каждые 30 минут.
func StartHalfHourlyAggregator(db *sql.DB) {
	run := func() {
		if err := aggregateHalfHour(db); err != nil {
			log.Printf("⚠️  30min aggregator: %v", err)
		}
	}
	go func() {
		run()
		for {
			now := time.Now()
			halfHour := now.Truncate(30 * time.Minute).Add(31 * time.Minute)
			time.Sleep(time.Until(halfHour))
			run()
		}
	}()
}

// StartFifteenMinAggregator агрегирует метрики в metrics_15min каждые 15 минут.
func StartFifteenMinAggregator(db *sql.DB) {
	run := func() {
		if err := aggregateFifteenMin(db); err != nil {
			log.Printf("⚠️  15min aggregator: %v", err)
		}
	}
	go func() {
		run()
		for {
			now := time.Now()
			next := now.Truncate(15 * time.Minute).Add(16 * time.Minute)
			time.Sleep(time.Until(next))
			run()
		}
	}()
}

func aggregateFifteenMin(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT
			node_name,
			strftime('%Y-%m-%dT%H:', timestamp) ||
				CASE
					WHEN CAST(strftime('%M', timestamp) AS INTEGER) < 15 THEN '00:00Z'
					WHEN CAST(strftime('%M', timestamp) AS INTEGER) < 30 THEN '15:00Z'
					WHEN CAST(strftime('%M', timestamp) AS INTEGER) < 45 THEN '30:00Z'
					ELSE '45:00Z'
				END AS bucket,
			AVG(cpu_usage),
			AVG(ram_usage),
			AVG(ram_total),
			AVG(disk_usage),
			AVG(COALESCE(net_bytes_recv, 0)),
			AVG(COALESCE(net_bytes_sent, 0)),
			AVG(COALESCE(disk_read_sec, 0)),
			AVG(COALESCE(disk_write_sec, 0))
		FROM metrics
		WHERE timestamp < strftime('%Y-%m-%dT%H:', 'now') ||
			CASE
				WHEN CAST(strftime('%M', 'now') AS INTEGER) < 15 THEN '00:00Z'
				WHEN CAST(strftime('%M', 'now') AS INTEGER) < 30 THEN '15:00Z'
				WHEN CAST(strftime('%M', 'now') AS INTEGER) < 45 THEN '30:00Z'
				ELSE '45:00Z'
			END
		GROUP BY node_name, bucket
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type row struct {
		name, bucket                                            string
		cpu, ram, ramTotal, disk, recv, sent, dRead, dWrite float64
	}
	var agg []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.name, &r.bucket, &r.cpu, &r.ram, &r.ramTotal, &r.disk, &r.recv, &r.sent, &r.dRead, &r.dWrite); err == nil {
			agg = append(agg, r)
		}
	}
	rows.Close()

	for _, r := range agg {
		_, err := db.Exec(`
			INSERT INTO metrics_15min(node_name, bucket, avg_cpu, avg_ram, avg_ram_total, avg_disk, avg_net_recv, avg_net_sent, avg_disk_read, avg_disk_write)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(node_name, bucket) DO UPDATE SET
				avg_cpu=excluded.avg_cpu,
				avg_ram=excluded.avg_ram,
				avg_ram_total=excluded.avg_ram_total,
				avg_disk=excluded.avg_disk,
				avg_net_recv=excluded.avg_net_recv,
				avg_net_sent=excluded.avg_net_sent,
				avg_disk_read=excluded.avg_disk_read,
				avg_disk_write=excluded.avg_disk_write
		`, r.name, r.bucket, r.cpu, r.ram, r.ramTotal, r.disk, r.recv, r.sent, r.dRead, r.dWrite)
		if err != nil {
			log.Printf("⚠️  15min upsert %s %s: %v", r.name, r.bucket, err)
		}
	}

	db.Exec(`DELETE FROM metrics_15min WHERE bucket < strftime('%Y-%m-%dT%H:00:00Z', 'now', '-90 days')`)

	return nil
}

func aggregateHalfHour(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT
			node_name,
			strftime('%Y-%m-%dT%H:', timestamp) ||
				CASE WHEN CAST(strftime('%M', timestamp) AS INTEGER) < 30
					THEN '00:00Z' ELSE '30:00Z' END AS bucket,
			AVG(cpu_usage),
			AVG(ram_usage),
			AVG(ram_total),
			AVG(disk_usage),
			AVG(COALESCE(net_bytes_recv, 0)),
			AVG(COALESCE(net_bytes_sent, 0)),
			AVG(COALESCE(disk_read_sec, 0)),
			AVG(COALESCE(disk_write_sec, 0))
		FROM metrics
		WHERE timestamp < strftime('%Y-%m-%dT%H:', 'now') ||
			CASE WHEN CAST(strftime('%M', 'now') AS INTEGER) < 30
				THEN '00:00Z' ELSE '30:00Z' END
		GROUP BY node_name, bucket
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type row struct {
		name, bucket                                            string
		cpu, ram, ramTotal, disk, recv, sent, dRead, dWrite float64
	}
	var agg []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.name, &r.bucket, &r.cpu, &r.ram, &r.ramTotal, &r.disk, &r.recv, &r.sent, &r.dRead, &r.dWrite); err == nil {
			agg = append(agg, r)
		}
	}
	rows.Close()

	for _, r := range agg {
		_, err := db.Exec(`
			INSERT INTO metrics_30min(node_name, bucket, avg_cpu, avg_ram, avg_ram_total, avg_disk, avg_net_recv, avg_net_sent, avg_disk_read, avg_disk_write)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(node_name, bucket) DO UPDATE SET
				avg_cpu=excluded.avg_cpu,
				avg_ram=excluded.avg_ram,
				avg_ram_total=excluded.avg_ram_total,
				avg_disk=excluded.avg_disk,
				avg_net_recv=excluded.avg_net_recv,
				avg_net_sent=excluded.avg_net_sent,
				avg_disk_read=excluded.avg_disk_read,
				avg_disk_write=excluded.avg_disk_write
		`, r.name, r.bucket, r.cpu, r.ram, r.ramTotal, r.disk, r.recv, r.sent, r.dRead, r.dWrite)
		if err != nil {
			log.Printf("⚠️  30min upsert %s %s: %v", r.name, r.bucket, err)
		}
	}

	db.Exec(`DELETE FROM metrics_30min WHERE bucket < strftime('%Y-%m-%dT%H:00:00Z', 'now', '-90 days')`)

	return nil
}

func aggregateLastHour(db *sql.DB) error {
	rows, err := db.Query(`
		SELECT
			node_name,
			strftime('%Y-%m-%dT%H:00:00Z', timestamp) AS hour,
			AVG(cpu_usage),
			AVG(ram_usage),
			AVG(ram_total),
			AVG(disk_usage),
			AVG(COALESCE(net_bytes_recv, 0)),
			AVG(COALESCE(net_bytes_sent, 0)),
			AVG(COALESCE(disk_read_sec, 0)),
			AVG(COALESCE(disk_write_sec, 0))
		FROM metrics
		WHERE timestamp < strftime('%Y-%m-%dT%H:00:00Z', 'now')
		GROUP BY node_name, hour
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type row struct {
		name, hour                                          string
		cpu, ram, ramTotal, disk, recv, sent, dRead, dWrite float64
	}
	var agg []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.name, &r.hour, &r.cpu, &r.ram, &r.ramTotal, &r.disk, &r.recv, &r.sent, &r.dRead, &r.dWrite); err == nil {
			agg = append(agg, r)
		}
	}
	rows.Close()

	for _, r := range agg {
		_, err := db.Exec(`
			INSERT INTO metrics_hourly(node_name, hour, avg_cpu, avg_ram, avg_ram_total, avg_disk, avg_net_recv, avg_net_sent, avg_disk_read, avg_disk_write)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(node_name, hour) DO UPDATE SET
				avg_cpu=excluded.avg_cpu,
				avg_ram=excluded.avg_ram,
				avg_ram_total=excluded.avg_ram_total,
				avg_disk=excluded.avg_disk,
				avg_net_recv=excluded.avg_net_recv,
				avg_net_sent=excluded.avg_net_sent,
				avg_disk_read=excluded.avg_disk_read,
				avg_disk_write=excluded.avg_disk_write
		`, r.name, r.hour, r.cpu, r.ram, r.ramTotal, r.disk, r.recv, r.sent, r.dRead, r.dWrite)
		if err != nil {
			log.Printf("⚠️  hourly upsert %s %s: %v", r.name, r.hour, err)
		}
	}

	db.Exec(`DELETE FROM metrics_hourly WHERE hour < strftime('%Y-%m-%dT%H:00:00Z', 'now', '-90 days')`)

	return nil
}

// HandleNodeHistory — GET /api/history/{name}?range=1h|24h|7d|14d|30d
func HandleNodeHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	name := strings.TrimPrefix(r.URL.Path, "/api/history/")
	if name == "" {
		http.Error(w, `{"error":"node name required"}`, http.StatusBadRequest)
		return
	}

	rangeParam := r.URL.Query().Get("range")

	// Режим 1h — сырые метрики, интервал 10 секунд
	if rangeParam == "1h" {
		since := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
		rows, err := dbConn.Query(`
			SELECT
				strftime('%H:%M:', datetime(timestamp, 'localtime')) ||
					printf('%02d', (CAST(strftime('%S', datetime(timestamp, 'localtime')) AS INTEGER) / 10) * 10) AS bucket,
				AVG(cpu_usage),
				AVG(ram_usage),
				AVG(ram_total),
				AVG(disk_usage),
				AVG(COALESCE(net_bytes_recv, 0)),
				AVG(COALESCE(net_bytes_sent, 0)),
				AVG(COALESCE(disk_read_sec, 0)),
				AVG(COALESCE(disk_write_sec, 0))
			FROM metrics
			WHERE node_name = ? AND datetime(timestamp) >= datetime(?)
			GROUP BY strftime('%Y-%m-%dT%H:%M:', datetime(timestamp, 'localtime')) ||
				printf('%02d', (CAST(strftime('%S', datetime(timestamp, 'localtime')) AS INTEGER) / 10) * 10)
			ORDER BY MIN(datetime(timestamp)) ASC
		`, name, since)
		if err != nil {
			http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var points []HourlyPoint
		for rows.Next() {
			var label string
			var cpu, ram, ramTotal, disk, recv, sent, dRead, dWrite float64
			if err := rows.Scan(&label, &cpu, &ram, &ramTotal, &disk, &recv, &sent, &dRead, &dWrite); err == nil {
				c, ra, rt, d, rc, s, dr, dw := cpu, ram, ramTotal, disk, recv, sent, dRead, dWrite
				points = append(points, HourlyPoint{Time: label, CPU: &c, RAM: &ra, RAMTotal: &rt, Disk: &d, NetRecv: &rc, NetSent: &s, DiskRead: &dr, DiskWrite: &dw})
			}
		}
		if points == nil {
			points = []HourlyPoint{}
		}
		json.NewEncoder(w).Encode(points)
		return
	}

	// Режим 24h — агрегация по 3 минуты из сырых метрик
	if rangeParam == "24h" {
		since := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
		rows, err := dbConn.Query(`
			SELECT
				strftime('%d.%m %H:', datetime(timestamp, 'localtime')) ||
					printf('%02d', (CAST(strftime('%M', datetime(timestamp, 'localtime')) AS INTEGER) / 3) * 3) AS bucket,
				AVG(cpu_usage),
				AVG(ram_usage),
				AVG(ram_total),
				AVG(disk_usage),
				AVG(COALESCE(net_bytes_recv, 0)),
				AVG(COALESCE(net_bytes_sent, 0)),
				AVG(COALESCE(disk_read_sec, 0)),
				AVG(COALESCE(disk_write_sec, 0))
			FROM metrics
			WHERE node_name = ? AND datetime(timestamp) >= datetime(?)
			GROUP BY strftime('%Y-%m-%dT%H:', datetime(timestamp, 'localtime')) ||
				printf('%02d', (CAST(strftime('%M', datetime(timestamp, 'localtime')) AS INTEGER) / 3) * 3)
			ORDER BY MIN(datetime(timestamp)) ASC
		`, name, since)
		if err != nil {
			http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		var points []HourlyPoint
		for rows.Next() {
			var label string
			var cpu, ram, ramTotal, disk, recv, sent, dRead, dWrite float64
			if err := rows.Scan(&label, &cpu, &ram, &ramTotal, &disk, &recv, &sent, &dRead, &dWrite); err == nil {
				c, ra, rt, d, rc, s, dr, dw := cpu, ram, ramTotal, disk, recv, sent, dRead, dWrite
				points = append(points, HourlyPoint{Time: label, CPU: &c, RAM: &ra, RAMTotal: &rt, Disk: &d, NetRecv: &rc, NetSent: &s, DiskRead: &dr, DiskWrite: &dw})
			}
		}
		if points == nil {
			points = []HourlyPoint{}
		}
		json.NewEncoder(w).Encode(points)
		return
	}

	var days int
	var step time.Duration
	var use15min, use30min bool

	switch rangeParam {
	case "7d":
		days = 7
		step = 15 * time.Minute
		use15min = true
	case "14d":
		days = 14
		step = 30 * time.Minute
		use30min = true
	case "30d":
		days = 30
		step = time.Hour
	default:
		days = 7
		step = 15 * time.Minute
		use15min = true
	}

	since := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour).Truncate(step)

	type rowData struct {
		cpu, ram, ramTotal, disk, recv, sent, dRead, dWrite float64
	}
	dataMap := make(map[time.Time]rowData)

	switch {
	case use15min:
		rows, err := dbConn.Query(`
			SELECT bucket, avg_cpu, avg_ram, avg_ram_total, avg_disk, avg_net_recv, avg_net_sent,
			       COALESCE(avg_disk_read, 0), COALESCE(avg_disk_write, 0)
			FROM metrics_15min
			WHERE node_name = ? AND bucket >= ?
			ORDER BY bucket ASC
		`, name, since.Format(time.RFC3339))
		if err != nil {
			http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var bucketStr string
			var d rowData
			if err := rows.Scan(&bucketStr, &d.cpu, &d.ram, &d.ramTotal, &d.disk, &d.recv, &d.sent, &d.dRead, &d.dWrite); err == nil {
				if t, err := time.Parse(time.RFC3339, bucketStr); err == nil {
					dataMap[t.Truncate(step)] = d
				}
			}
		}
	case use30min:
		rows, err := dbConn.Query(`
			SELECT bucket, avg_cpu, avg_ram, avg_ram_total, avg_disk, avg_net_recv, avg_net_sent,
			       COALESCE(avg_disk_read, 0), COALESCE(avg_disk_write, 0)
			FROM metrics_30min
			WHERE node_name = ? AND bucket >= ?
			ORDER BY bucket ASC
		`, name, since.Format(time.RFC3339))
		if err != nil {
			http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var bucketStr string
			var d rowData
			if err := rows.Scan(&bucketStr, &d.cpu, &d.ram, &d.ramTotal, &d.disk, &d.recv, &d.sent, &d.dRead, &d.dWrite); err == nil {
				if t, err := time.Parse(time.RFC3339, bucketStr); err == nil {
					dataMap[t.Truncate(step)] = d
				}
			}
		}
	default:
		rows, err := dbConn.Query(`
			SELECT hour, avg_cpu, avg_ram, avg_ram_total, avg_disk, avg_net_recv, avg_net_sent,
			       COALESCE(avg_disk_read, 0), COALESCE(avg_disk_write, 0)
			FROM metrics_hourly
			WHERE node_name = ? AND hour >= ?
			ORDER BY hour ASC
		`, name, since.Format(time.RFC3339))
		if err != nil {
			http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
			return
		}
		defer rows.Close()
		for rows.Next() {
			var hourStr string
			var d rowData
			if err := rows.Scan(&hourStr, &d.cpu, &d.ram, &d.ramTotal, &d.disk, &d.recv, &d.sent, &d.dRead, &d.dWrite); err == nil {
				if t, err := time.Parse(time.RFC3339, hourStr); err == nil {
					dataMap[t.Truncate(time.Hour)] = d
				}
			}
		}
	}

	now := time.Now().UTC().Truncate(step)
	var points []HourlyPoint
	for t := since; !t.After(now); t = t.Add(step) {
		label := t.Local().Format("02.01 15:04")
		if d, ok := dataMap[t]; ok {
			cpu, ram, ramTotal, disk, recv, sent, dr, dw := d.cpu, d.ram, d.ramTotal, d.disk, d.recv, d.sent, d.dRead, d.dWrite
			points = append(points, HourlyPoint{
				Time: label, CPU: &cpu, RAM: &ram, RAMTotal: &ramTotal,
				Disk: &disk, NetRecv: &recv, NetSent: &sent, DiskRead: &dr, DiskWrite: &dw,
			})
		} else {
			points = append(points, HourlyPoint{Time: label})
		}
	}

	if points == nil {
		points = []HourlyPoint{}
	}
	json.NewEncoder(w).Encode(points)
}
