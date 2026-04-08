// internal/server/server.go
package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type MetricPayload struct {
	NodeName  string  `json:"node_name"`
	Timestamp string  `json:"timestamp"`
	CPUUsage  float64 `json:"cpu_usage"`
	RAMUsage  float64 `json:"ram_usage"`
	DiskUsage float64 `json:"disk_usage"`
}

// dbConn хранит активное подключение к нашей SQLite
var dbConn *sql.DB

func Run() {
	// 1. Инициализируем БД (файл monitor.db появится в корне проекта)
	dbConn = InitDB("monitor.db")
	defer dbConn.Close() // Закроем БД только при полной остановке сервера

	http.HandleFunc("/api/metrics", handleMetrics)

	port := ":8080"
	log.Printf("🚀 Мастер-сервер запущен. Слушаю порт %s...", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

// Эта функция срабатывает каждый раз, когда кто-то стучится на /api/metrics
func handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Ожидается метод POST", http.StatusMethodNotAllowed)
		http.HandleFunc("/api/history", handleGetHistory)
		return
	}

	var payload MetricPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Ошибка чтения JSON", http.StatusBadRequest)
		return
	}

	// 2. Вызываем функцию сохранения в БД
	err = SaveMetric(dbConn, payload)
	if err != nil {
		log.Printf("❌ Ошибка записи в БД: %v", err)
		http.Error(w, "Ошибка сохранения данных", http.StatusInternalServerError)
		return
	}

	// 3. Выводим красивый лог об успешной записи
	fmt.Printf("\n💾 Сохранено в БД: [%s] CPU: %.1f%% | RAM: %.1f%% | Disk: %.1f%%\n",
		payload.NodeName, payload.CPUUsage, payload.RAMUsage, payload.DiskUsage)

	// Отвечаем агенту, что всё хорошо
	w.WriteHeader(http.StatusOK)
}
