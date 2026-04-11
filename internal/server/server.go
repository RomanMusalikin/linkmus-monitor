package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type MetricPayload struct {
	NodeName   string  `json:"node_name"`
	Timestamp  string  `json:"timestamp"`
	CPUUsage   float64 `json:"cpu_usage"`
	RAMUsage   float64 `json:"ram_usage"`
	DiskUsage  float64 `json:"disk_usage"`
	RDPRunning bool    `json:"rdp_running"` // Добавили RDP
	SMBRunning bool    `json:"smb_running"` // Добавили SMB
}

// dbConn хранит активное подключение к нашей SQLite
var dbConn *sql.DB

func Run() {
	// 1. Инициализируем БД
	dbConn = InitDB("monitor.db")
	defer dbConn.Close()

	// 2. РЕГИСТРИРУЕМ ВСЕ МАРШРУТЫ ЗДЕСЬ (один раз при старте!)
	http.HandleFunc("/api/metrics", handleMetrics) // Сюда стучатся агенты (POST)
	http.HandleFunc("/api/nodes", HandleNodes)     // Сюда стучится React-фронтенд (GET) - эта функция лежит в api.go!

	port := ":8080"
	log.Printf("🚀 Мастер-сервер запущен. Слушаю порт %s...", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

// Эта функция срабатывает каждый раз, когда агент шлет POST на /api/metrics
func handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Должен быть строго POST
	if r.Method != http.MethodPost {
		http.Error(w, "Ожидается метод POST", http.StatusMethodNotAllowed)
		return // <-- Ошибочная строка удалена отсюда!
	}

	var payload MetricPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Ошибка чтения JSON", http.StatusBadRequest)
		return
	}

	// Вызываем функцию сохранения в БД
	err = SaveMetric(dbConn, payload)
	if err != nil {
		log.Printf("❌ Ошибка записи в БД: %v", err)
		http.Error(w, "Ошибка сохранения данных", http.StatusInternalServerError)
		return
	}

	// Выводим красивый лог
	fmt.Printf("\n💾 Сохранено в БД: [%s] CPU: %.1f%% | RAM: %.1f%% | Disk: %.1f%%\n",
		payload.NodeName, payload.CPUUsage, payload.RAMUsage, payload.DiskUsage)

	w.WriteHeader(http.StatusOK)
}
