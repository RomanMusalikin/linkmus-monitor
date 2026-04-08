package server

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite", filepath)
	if err != nil {
		log.Fatalf("❌ Ошибка открытия БД: %v", err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		node_name TEXT,
		timestamp DATETIME,
		cpu_usage REAL,
		ram_usage REAL,
		disk_usage REAL,
		rdp_running BOOLEAN,
		smb_running BOOLEAN
	);`

	_, err = db.Exec(query)
	if err != nil {
		log.Fatalf("❌ Ошибка создания таблицы: %v", err)
	}

	log.Println("✅ База данных SQLite успешно инициализирована")
	return db
}

func SaveMetric(db *sql.DB, payload MetricPayload) error {
	query := `
	INSERT INTO metrics (node_name, timestamp, cpu_usage, ram_usage, disk_usage, rdp_running, smb_running)
	VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err := db.Exec(query,
		payload.NodeName, payload.Timestamp, payload.CPUUsage,
		payload.RAMUsage, payload.DiskUsage, payload.RDPRunning, payload.SMBRunning,
	)
	return err
}

func GetRecentMetrics(db *sql.DB, limit int) ([]MetricPayload, error) {
	query := `
	SELECT node_name, timestamp, cpu_usage, ram_usage, disk_usage, rdp_running, smb_running
	FROM metrics ORDER BY timestamp DESC LIMIT ?`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var metrics []MetricPayload
	for rows.Next() {
		var m MetricPayload
		err := rows.Scan(&m.NodeName, &m.Timestamp, &m.CPUUsage, &m.RAMUsage, &m.DiskUsage, &m.RDPRunning, &m.SMBRunning)
		if err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}
	return metrics, nil
}
