package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// Version задаётся через ldflags при сборке: -X 'linkmus-monitor/internal/server.Version=v1.2.0'
var Version = "unknown"

// MetricPayload — структура входящего JSON от агента (должна совпадать с agent.MetricPayload)
type MetricPayload struct {
	NodeName    string  `json:"node_name"`
	OS          string  `json:"os"`
	IP          string  `json:"ip"`
	Uptime      string  `json:"uptime"`
	BootTime    string  `json:"boot_time"`
	Timestamp   string  `json:"timestamp"`
	LoggedUsers int     `json:"logged_users"`

	CPUUsage     float64 `json:"cpu_usage"`
	CPUUser      float64 `json:"cpu_user"`
	CPUSystem    float64 `json:"cpu_system"`
	CPUIOwait    float64 `json:"cpu_iowait"`
	CPUSteal     float64 `json:"cpu_steal"`
	CPUTemp      float64 `json:"cpu_temp"`
	CPUModel     string  `json:"cpu_model"`
	CPUFreqMHz   float64 `json:"cpu_freq_mhz"`
	CPUCoresJSON string  `json:"cpu_cores_json"`

	LoadAvg1  float64 `json:"load_avg_1"`
	LoadAvg5  float64 `json:"load_avg_5"`
	LoadAvg15 float64 `json:"load_avg_15"`

	RAMUsage   float64 `json:"ram_usage"`
	RAMTotal   float64 `json:"ram_total"`
	RAMCached  float64 `json:"ram_cached"`
	RAMBuffers float64 `json:"ram_buffers"`
	SwapUsed   float64 `json:"swap_used"`
	SwapTotal  float64 `json:"swap_total"`

	DiskUsage    float64 `json:"disk_usage"`
	DiskReadSec  float64 `json:"disk_read_sec"`
	DiskWriteSec float64 `json:"disk_write_sec"`
	DiskQueue    float64 `json:"disk_queue"`
	DisksJSON    string  `json:"disks_json"`
	FSRMJson     string  `json:"fsrm_json"`

	RDPRunning bool `json:"rdp_running"`
	SMBRunning bool `json:"smb_running"`

	NetInterface  string  `json:"net_interface"`
	NetBytesRecv  float64 `json:"net_bytes_recv"`
	NetBytesSent  float64 `json:"net_bytes_sent"`
	AllIfacesJSON string  `json:"all_ifaces_json"`

	TCPTotal       int    `json:"tcp_total"`
	TCPEstablished int    `json:"tcp_established"`
	TCPTimeWait    int    `json:"tcp_timewait"`
	ProcessCount   int    `json:"process_count"`
	ProcessesJSON  string `json:"processes_json"`
	TopMemJSON     string `json:"top_mem_json"`
	AgentVersion   string `json:"agent_version"`
}

var dbConn *sql.DB

// corsHeaders добавляет заголовки CORS (нужно для dev-режима с Vite-прокси)
func corsHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
}

// requireAuth — middleware, проверяет Bearer-токен в заголовке Authorization
func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		corsHeaders(w)
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		authHeader := r.Header.Get("Authorization")
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if !ValidateSession(dbConn, token) {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func Run() {
	dbConn = InitDB("monitor.db")
	defer dbConn.Close()

	StartProber(dbConn)
	StartSNMPPoller(dbConn)
	StartNodesCache(dbConn)
	StartDataCleanup(dbConn)
	StartHourlyAggregator(dbConn)

	// Агент — без авторизации
	http.HandleFunc("/api/metrics", handleMetrics)

	// Версия сервера — без авторизации
	http.HandleFunc("/api/version", handleVersion)

	// Данные узлов — только авторизованным
	http.HandleFunc("/api/nodes", requireAuth(HandleNodes))
	http.HandleFunc("/api/nodes/", requireAuth(HandleNodeDelete))
	http.HandleFunc("/api/history/", requireAuth(HandleNodeHistory))

	// Auth-эндпоинты
	http.HandleFunc("/api/auth/setup", handleAuthSetup)
	http.HandleFunc("/api/auth/register", handleAuthRegister)
	http.HandleFunc("/api/auth/login", handleAuthLogin)
	http.HandleFunc("/api/auth/logout", handleAuthLogout)

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

	fmt.Printf("💾 [%s] CPU:%.1f%%(iow:%.1f%%) | RAM:%.1f/%.1fGB | Disk:%.1f%% | Net↓%.0fB/s↑%.0fB/s | TCP:%d | Procs:%d\n",
		payload.NodeName, payload.CPUUsage, payload.CPUIOwait,
		payload.RAMUsage, payload.RAMTotal,
		payload.DiskUsage,
		payload.NetBytesRecv, payload.NetBytesSent,
		payload.TCPTotal, payload.ProcessCount,
	)

	w.WriteHeader(http.StatusOK)
}

func handleVersion(w http.ResponseWriter, r *http.Request) {
	corsHeaders(w)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"version": Version})
}

// handleAuthSetup — GET /api/auth/setup
// Возвращает {"needSetup": true} если пользователей ещё нет
func handleAuthSetup(w http.ResponseWriter, r *http.Request) {
	corsHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	has, err := HasUsers(dbConn)
	if err != nil {
		http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(map[string]bool{"needSetup": !has})
}

// handleAuthRegister — POST /api/auth/register
// Разрешает регистрацию только если пользователей нет (первый запуск)
func handleAuthRegister(w http.ResponseWriter, r *http.Request) {
	corsHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	has, err := HasUsers(dbConn)
	if err != nil || has {
		http.Error(w, `{"error":"registration closed"}`, http.StatusForbidden)
		return
	}

	var body struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Login == "" || body.Password == "" {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	userID, err := RegisterUser(dbConn, body.Login, body.Password)
	if err != nil {
		log.Printf("❌ Ошибка регистрации: %v", err)
		http.Error(w, `{"error":"registration failed"}`, http.StatusInternalServerError)
		return
	}

	token, err := CreateSession(dbConn, userID)
	if err != nil {
		http.Error(w, `{"error":"session failed"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// handleAuthLogin — POST /api/auth/login
func handleAuthLogin(w http.ResponseWriter, r *http.Request) {
	corsHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")

	var body struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	userID, err := AuthenticateUser(dbConn, body.Login, body.Password)
	if err != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	token, err := CreateSession(dbConn, userID)
	if err != nil {
		http.Error(w, `{"error":"session failed"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"token": token})
}

// handleAuthLogout — POST /api/auth/logout
func handleAuthLogout(w http.ResponseWriter, r *http.Request) {
	corsHeaders(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	DeleteSession(dbConn, token)
	w.WriteHeader(http.StatusOK)
}
