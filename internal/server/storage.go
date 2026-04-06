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
