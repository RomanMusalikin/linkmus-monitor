package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// MetricPayload — структура входящего JSON от агента (должна совпадать с agent.MetricPayload)
type MetricPayload struct {
	NodeName      string  `json:"node_name"`
	OS            string  `json:"os"`
	IP            string  `json:"ip"`
	Uptime        string  `json:"uptime"`
	Timestamp     string  `json:"timestamp"`
	CPUUsage      float64 `json:"cpu_usage"`
	CPUUser       float64 `json:"cpu_user"`
	CPUSystem     float64 `json:"cpu_system"`
	LoadAvg1      float64 `json:"load_avg_1"`
	LoadAvg5      float64 `json:"load_avg_5"`
	LoadAvg15     float64 `json:"load_avg_15"`
	RAMUsage      float64 `json:"ram_usage"`
	RAMTotal      float64 `json:"ram_total"`
	RAMCached     float64 `json:"ram_cached"`
	RAMBuffers    float64 `json:"ram_buffers"`
	SwapUsed      float64 `json:"swap_used"`
	SwapTotal     float64 `json:"swap_total"`
	DiskUsage     float64 `json:"disk_usage"`
	RDPRunning    bool    `json:"rdp_running"`
	SMBRunning    bool    `json:"smb_running"`
	NetInterface  string  `json:"net_interface"`
	NetBytesRecv  float64 `json:"net_bytes_recv"`
	NetBytesSent  float64 `json:"net_bytes_sent"`
	ProcessesJSON string  `json:"processes_json"`
}

var dbConn *sql.DB

func Run() {
	dbConn = InitDB("monitor.db")
	defer dbConn.Close()

	http.HandleFunc("/api/metrics", handleMetrics)
	http.HandleFunc("/api/nodes", HandleNodes)

	port := ":8080"
	log.Printf("🚀 Мастер-сервер запущен на порту %s", port)

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

func handleMetrics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Ожидается метод POST", http.StatusMethodNotAllowed)
		return
	}

	var payload MetricPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Ошибка чтения JSON", http.StatusBadRequest)
		return
	}

	if err := SaveMetric(dbConn, payload); err != nil {
		log.Printf("❌ Ошибка записи в БД: %v", err)
		http.Error(w, "Ошибка сохранения данных", http.StatusInternalServerError)
		return
	}

	fmt.Printf("💾 [%s] CPU:%.1f%% | RAM:%.1fGB/%.1fGB | Disk:%.1f%% | Net↓%.0fB/s↑%.0fB/s\n",
		payload.NodeName, payload.CPUUsage,
		payload.RAMUsage, payload.RAMTotal,
		payload.DiskUsage,
		payload.NetBytesRecv, payload.NetBytesSent,
	)

	w.WriteHeader(http.StatusOK)
}
