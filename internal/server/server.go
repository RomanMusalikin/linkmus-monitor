package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
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
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "monitor.db"
	}
	dbConn = InitDB(dbPath)
	defer dbConn.Close()

	StartProber(dbConn)
	StartSNMPPoller(dbConn)
	StartNodesCache(dbConn)
	StartDataCleanup(dbConn)
	StartHourlyAggregator(dbConn)
	StartHalfHourlyAggregator(dbConn)
	StartFifteenMinAggregator(dbConn)
	StartAlertChecker(dbConn)

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
	http.HandleFunc("/api/auth/users", requireAuth(handleCreateUser))

	// Настройки алертов
	http.HandleFunc("/api/settings/alerts", requireAuth(handleAlertSettings))
	http.HandleFunc("/api/settings/alerts/test-telegram", requireAuth(handleTestTelegram))

	// Настройки портов сервисов
	http.HandleFunc("/api/settings/ports", requireAuth(handlePortSettings))

	// Пользовательские сервисы
	http.HandleFunc("/api/settings/services", requireAuth(handleCustomServices))
	http.HandleFunc("/api/settings/services/", requireAuth(handleCustomServices))

	// Настройки GigaChat
	http.HandleFunc("/api/settings/gigachat", requireAuth(handleGigachatSettings))

	// Генерация отчёта через GigaChat
	http.HandleFunc("/api/report", requireAuth(HandleReport))

	// История отчётов
	http.HandleFunc("/api/reports", requireAuth(HandleReportHistory))
	http.HandleFunc("/api/reports/", requireAuth(HandleReportHistory))

	// Фронтенд — статические файлы с SPA-fallback
	webPath := os.Getenv("WEB_PATH")
	if webPath == "" {
		webPath = "/opt/linkmus-monitor/web"
	}
	fs := http.FileServer(http.Dir(webPath))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Если файл существует — отдаём его, иначе index.html (SPA)
		if r.URL.Path != "/" {
			if _, err := os.Stat(webPath + r.URL.Path); err == nil {
				fs.ServeHTTP(w, r)
				return
			}
		}
		http.ServeFile(w, r, webPath+"/index.html")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("🚀 Мастер-сервер запущен на порту %s", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
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
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Login == "" || body.Password == "" {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}

	userID, err := RegisterUser(dbConn, body.Login, body.Password, body.Email)
	if err != nil {
		log.Printf("❌ Ошибка регистрации: %v", err)
		http.Error(w, `{"error":"registration failed"}`, http.StatusInternalServerError)
		return
	}

	// Подставляем email регистрации в настройки уведомлений если там пусто
	if body.Email != "" {
		existing := GetAlertSettings(dbConn)
		if existing.ToEmail == "" {
			existing.ToEmail = body.Email
			SaveAlertSettings(dbConn, existing)
		}
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

// handleCreateUser — POST /api/auth/users (требует авторизации)
// Позволяет создать нового пользователя системы
func handleCreateUser(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Login    string `json:"login"`
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Login == "" || body.Password == "" {
		http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
		return
	}
	if _, err := RegisterUser(dbConn, body.Login, body.Password, body.Email); err != nil {
		log.Printf("❌ Ошибка создания пользователя: %v", err)
		http.Error(w, `{"error":"user already exists or db error"}`, http.StatusConflict)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

// handleTestTelegram — POST /api/settings/alerts/test-telegram
func handleTestTelegram(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	corsHeaders(w)
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	s := GetAlertSettings(dbConn)
	if err := SendTestTelegram(s); err != nil {
		http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadGateway)
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
}

// handleAlertSettings — GET/PUT /api/settings/alerts
func handleAlertSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	corsHeaders(w)
	switch r.Method {
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
	case http.MethodGet:
		json.NewEncoder(w).Encode(GetAlertSettings(dbConn))
	case http.MethodPut:
		var s AlertSettings
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
			return
		}
		if err := SaveAlertSettings(dbConn, s); err != nil {
			http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	case http.MethodPost:
		// POST /api/settings/alerts/test
		s := GetAlertSettings(dbConn)
		if err := SendTestEmail(s); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":%q}`, err.Error()), http.StatusBadGateway)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "sent"})
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// handleGigachatSettings — GET/PUT /api/settings/gigachat
func handleGigachatSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	corsHeaders(w)
	switch r.Method {
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
	case http.MethodGet:
		json.NewEncoder(w).Encode(GetGigachatSettings(dbConn))
	case http.MethodPut:
		var s GigachatSettings
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
			return
		}
		if err := SaveGigachatSettings(dbConn, s); err != nil {
			http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
			return
		}
		// сбрасываем кеш токена при смене настроек
		gcTokenMu.Lock()
		gcToken = ""
		gcTokenMu.Unlock()
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// handleCustomServices — GET/POST /api/settings/services  |  DELETE /api/settings/services/{id}
func handleCustomServices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	corsHeaders(w)

	switch r.Method {
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
	case http.MethodGet:
		svcs := GetCustomServices(dbConn)
		if svcs == nil {
			svcs = []CustomService{}
		}
		json.NewEncoder(w).Encode(svcs)
	case http.MethodPost:
		var body struct {
			Name string `json:"name"`
			Port int    `json:"port"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" || body.Port <= 0 {
			http.Error(w, `{"error":"name and port required"}`, http.StatusBadRequest)
			return
		}
		svc, err := CreateCustomService(dbConn, strings.TrimSpace(body.Name), body.Port)
		if err != nil {
			http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
			return
		}
		AddCustomServiceToProbeCache(dbConn, svc)
		InvalidateNodesCache(dbConn)
		json.NewEncoder(w).Encode(svc)
	case http.MethodDelete:
		// /api/settings/services/{id}
		idStr := strings.TrimPrefix(r.URL.Path, "/api/settings/services/")
		var id int
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil || id <= 0 {
			http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
			return
		}
		if err := DeleteCustomService(dbConn, id); err != nil {
			http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
			return
		}
		RemoveCustomServiceFromProbeCache(id)
		InvalidateNodesCache(dbConn)
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// handlePortSettings — GET/PUT /api/settings/ports
func handlePortSettings(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	corsHeaders(w)
	switch r.Method {
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
	case http.MethodGet:
		json.NewEncoder(w).Encode(GetPortSettings(dbConn))
	case http.MethodPut:
		var s PortSettings
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			http.Error(w, `{"error":"invalid body"}`, http.StatusBadRequest)
			return
		}
		if err := SavePortSettings(dbConn, s); err != nil {
			http.Error(w, `{"error":"db error"}`, http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
