package server

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"
)

// HourlyPoint — одна точка долгосрочной истории (nil = нет данных за этот час)
type HourlyPoint struct {
	Time     string   `json:"time"`
	CPU      *float64 `json:"cpu"`
	RAM      *float64 `json:"ram"`
	RAMTotal *float64 `json:"ramTotal"`
	Disk     *float64 `json:"disk"`
	NetRecv  *float64 `json:"netRecv"`
	NetSent  *float64 `json:"netSent"`
}

// StartHourlyAggregator агрегирует метрики в metrics_hourly раз в час.
func StartHourlyAggregator(db *sql.DB) {
	run := func() {
		if err := aggregateLastHour(db); err != nil {
			log.Printf("⚠️  hourly aggregator: %v", err)
		}
	}
	go func() {
		// первый запуск сразу при старте — заполняем историю из уже имеющихся данных
		run()
		for {
			// ждём до начала следующего часа + 1 минута
			now := time.Now()
			next := now.Truncate(time.Hour).Add(time.Hour + time.Minute)
			time.Sleep(time.Until(next))
			run()
		}
	}()
}

func aggregateLastHour(db *sql.DB) error {
	// Агрегируем все завершённые часы которых ещё нет в metrics_hourly
	rows, err := db.Query(`
		SELECT
			node_name,
			strftime('%Y-%m-%dT%H:00:00Z', timestamp) AS hour,
			AVG(cpu_usage),
			AVG(ram_usage),
			AVG(ram_total),
			AVG(disk_usage),
			AVG(COALESCE(net_bytes_recv, 0)),
			AVG(COALESCE(net_bytes_sent, 0))
		FROM metrics
		WHERE timestamp < strftime('%Y-%m-%dT%H:00:00Z', 'now')
		GROUP BY node_name, hour
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	type row struct {
		name, hour                            string
		cpu, ram, ramTotal, disk, recv, sent float64
	}
	var agg []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.name, &r.hour, &r.cpu, &r.ram, &r.ramTotal, &r.disk, &r.recv, &r.sent); err == nil {
			agg = append(agg, r)
		}
	}
	rows.Close()

	for _, r := range agg {
		_, err := db.Exec(`
			INSERT INTO metrics_hourly(node_name, hour, avg_cpu, avg_ram, avg_ram_total, avg_disk, avg_net_recv, avg_net_sent)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT(node_name, hour) DO UPDATE SET
				avg_cpu=excluded.avg_cpu,
				avg_ram=excluded.avg_ram,
				avg_ram_total=excluded.avg_ram_total,
				avg_disk=excluded.avg_disk,
				avg_net_recv=excluded.avg_net_recv,
				avg_net_sent=excluded.avg_net_sent
		`, r.name, r.hour, r.cpu, r.ram, r.ramTotal, r.disk, r.recv, r.sent)
		if err != nil {
			log.Printf("⚠️  hourly upsert %s %s: %v", r.name, r.hour, err)
		}
	}

	// Удаляем агрегаты старше 90 дней
	db.Exec(`DELETE FROM metrics_hourly WHERE hour < strftime('%Y-%m-%dT%H:00:00Z', 'now', '-90 days')`)

	return nil
}

// HandleNodeHistory — GET /api/history/{name}?range=7d|30d|90d
func HandleNodeHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	name := strings.TrimPrefix(r.URL.Path, "/api/history/")
	if name == "" {
		http.Error(w, `{"error":"node name required"}`, http.StatusBadRequest)
		return
	}

	rangeParam := r.URL.Query().Get("range")
	var days int
	switch rangeParam {
	case "14d":
		days = 14
	case "30d":
		days = 30
	default: // 7d
		days = 7
	}

	since := time.Now().UTC().Add(-time.Duration(days) * 24 * time.Hour).Truncate(time.Hour)

	rows, err := dbConn.Query(`
		SELECT hour, avg_cpu, avg_ram, avg_ram_total, avg_disk, avg_net_recv, avg_net_sent
		FROM metrics_hourly
		WHERE node_name = ? AND hour >= ?
		ORDER BY hour ASC
	`, name, since.Format(time.RFC3339))
	if err != nil {
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	// Читаем все строки в карту hour → данные
	type rowData struct {
		cpu, ram, ramTotal, disk, recv, sent float64
	}
	dataMap := make(map[time.Time]rowData)
	for rows.Next() {
		var hourStr string
		var d rowData
		if err := rows.Scan(&hourStr, &d.cpu, &d.ram, &d.ramTotal, &d.disk, &d.recv, &d.sent); err == nil {
			if t, err := time.Parse(time.RFC3339, hourStr); err == nil {
				dataMap[t.Truncate(time.Hour)] = d
			}
		}
	}

	// Заполняем все часы в диапазоне, вставляем null для отсутствующих
	now := time.Now().UTC().Truncate(time.Hour)
	var points []HourlyPoint
	for t := since; !t.After(now); t = t.Add(time.Hour) {
		label := t.Local().Format("02.01 15:04")
		if d, ok := dataMap[t]; ok {
			cpu := d.cpu
			ram := d.ram
			ramTotal := d.ramTotal
			disk := d.disk
			recv := d.recv
			sent := d.sent
			points = append(points, HourlyPoint{
				Time: label, CPU: &cpu, RAM: &ram, RAMTotal: &ramTotal,
				Disk: &disk, NetRecv: &recv, NetSent: &sent,
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
