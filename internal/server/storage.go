// internal/server/storage.go
package server

import (
	"database/sql"
	"log"

	// Импортируем pure-Go драйвер SQLite
	_ "modernc.org/sqlite"
)

// InitDB создает базу данных и нужные таблицы, если их нет
func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite", filepath)
	if err != nil {
		log.Fatalf("❌ Ошибка открытия БД: %v", err)
	}

	// Создаем таблицу для метрик
	query := `
	CREATE TABLE IF NOT EXISTS metrics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		node_name TEXT,
		timestamp DATETIME,
		cpu_usage REAL,
		ram_usage REAL,
		disk_usage REAL
	);`

	_, err = db.Exec(query)
	if err != nil {
		log.Fatalf("❌ Ошибка создания таблицы: %v", err)
	}

	log.Println("✅ База данных SQLite успешно инициализирована")
	return db
}

// SaveMetric принимает подключение к БД и структуру с метриками, а затем сохраняет их
func SaveMetric(db *sql.DB, payload MetricPayload) error {
	// Используем плейсхолдеры (?) вместо прямой подстановки строк.
	// Это стандартный паттерн для защиты от SQL-инъекций.
	query := `
	INSERT INTO metrics (node_name, timestamp, cpu_usage, ram_usage, disk_usage)
	VALUES (?, ?, ?, ?, ?)`

	// Выполняем запрос, передавая данные из структуры на места вопросов
	_, err := db.Exec(query,
		payload.NodeName,
		payload.Timestamp,
		payload.CPUUsage,
		payload.RAMUsage,
		payload.DiskUsage,
	)

	return err
}
