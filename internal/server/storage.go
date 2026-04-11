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

func GetLatestNodes(db *sql.DB) ([]NodeSummary, error) {
	// Шаг 1: Получаем список всех уникальных серверов, которые вообще когда-либо присылали данные
	rows, err := db.Query(`SELECT DISTINCT node_name FROM metrics`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []NodeSummary
	var nodeNames []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			nodeNames = append(nodeNames, name)
		}
	}

	// Шаг 2: Для каждого сервера достаем его последнюю метрику и историю CPU
	for _, name := range nodeNames {
		// Получаем самые свежие данные для конкретного узла
		var lastMetric MetricPayload
		err := db.QueryRow(`
			SELECT timestamp, cpu_usage, ram_usage, disk_usage, rdp_running, smb_running
			FROM metrics 
			WHERE node_name = ? 
			ORDER BY timestamp DESC LIMIT 1`, name).Scan(
			&lastMetric.Timestamp, &lastMetric.CPUUsage, &lastMetric.RAMUsage,
			&lastMetric.DiskUsage, &lastMetric.RDPRunning, &lastMetric.SMBRunning,
		)

		if err != nil {
			log.Printf("Ошибка получения последней метрики для %s: %v", name, err)
			continue
		}

		// Получаем последние 20 записей CPU для графика
		histRows, err := db.Query(`
			SELECT cpu_usage 
			FROM metrics 
			WHERE node_name = ? 
			ORDER BY timestamp DESC LIMIT 20`, name)

		var cpuHistory []CpuPoint
		if err == nil {
			defer histRows.Close()
			// SQL возвращает записи от новых к старым (DESC), а графику нужно от старых к новым.
			// Поэтому мы сначала сохраним их во временный срез
			var tempHistory []int
			for histRows.Next() {
				var cpu float64
				if err := histRows.Scan(&cpu); err == nil {
					tempHistory = append(tempHistory, int(cpu))
				}
			}
			// Разворачиваем срез (от старых к новым)
			for i := len(tempHistory) - 1; i >= 0; i-- {
				cpuHistory = append(cpuHistory, CpuPoint{Value: tempHistory[i]})
			}
		}

		// Формируем объект для отправки на фронтенд
		// (В реальном проекте OS и IP мы бы брали из отдельной таблицы конфигурации серверов,
		// но для курсовой мы можем захардкодить их или отдавать пустыми)
		summary := NodeSummary{
			Name:       name,
			OS:         "Linux/Windows", // Заглушка, т.к. агент пока не присылает ОС
			IP:         "10.10.x.x",     // Заглушка
			Online:     true,            // Если есть записи, считаем, что онлайн
			CPU:        int(lastMetric.CPUUsage),
			RAMUsed:    int(lastMetric.RAMUsage), // Твой агент должен присылать RAM в МБ или ГБ
			RAMTotal:   4096,                     // Заглушка (предположим 4ГБ). В идеале агент тоже должен это присылать
			Uptime:     "Active",
			Ping:       1,
			CPUHistory: cpuHistory,
		}

		nodes = append(nodes, summary)
	}

	return nodes, nil
}
