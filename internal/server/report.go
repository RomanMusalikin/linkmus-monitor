package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// ReportHistoryItem — запись в истории отчётов (без полного текста для списка)
type ReportHistoryItem struct {
	ID        int    `json:"id"`
	CreatedAt string `json:"createdAt"`
	Period    string `json:"period"`
	FromDate  string `json:"fromDate"`
	ToDate    string `json:"toDate"`
	Nodes     []string `json:"nodes"`
	Preview   string `json:"preview"` // первые 120 символов отчёта
}

// ReportHistoryFull — полная запись с текстом отчёта
type ReportHistoryFull struct {
	ReportHistoryItem
	Report string `json:"report"`
}

func saveReportHistory(db *sql.DB, period, from, to string, nodes []string, report string) {
	nodesJSON, _ := json.Marshal(nodes)
	db.Exec(`INSERT INTO report_history(created_at, period, from_date, to_date, nodes, report)
		VALUES(?,?,?,?,?,?)`,
		time.Now().UTC().Format(time.RFC3339),
		period, from, to, string(nodesJSON), report)
	// Оставляем не более 100 последних отчётов
	db.Exec(`DELETE FROM report_history WHERE id NOT IN (
		SELECT id FROM report_history ORDER BY id DESC LIMIT 100)`)
}

func getReportHistory(db *sql.DB) []ReportHistoryItem {
	rows, err := db.Query(`SELECT id, created_at, period, from_date, to_date, nodes,
		substr(report, 1, 120) FROM report_history ORDER BY id DESC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var result []ReportHistoryItem
	for rows.Next() {
		var item ReportHistoryItem
		var nodesJSON string
		if err := rows.Scan(&item.ID, &item.CreatedAt, &item.Period, &item.FromDate, &item.ToDate, &nodesJSON, &item.Preview); err == nil {
			json.Unmarshal([]byte(nodesJSON), &item.Nodes)
			result = append(result, item)
		}
	}
	return result
}

func getReportByID(db *sql.DB, id int) (ReportHistoryFull, bool) {
	var item ReportHistoryFull
	var nodesJSON string
	err := db.QueryRow(`SELECT id, created_at, period, from_date, to_date, nodes, report
		FROM report_history WHERE id=?`, id).
		Scan(&item.ID, &item.CreatedAt, &item.Period, &item.FromDate, &item.ToDate, &nodesJSON, &item.Report)
	if err != nil {
		return item, false
	}
	json.Unmarshal([]byte(nodesJSON), &item.Nodes)
	return item, true
}

// HandleReportHistory — GET /api/reports  |  GET /api/reports/{id}  |  DELETE /api/reports/{id}
func HandleReportHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	corsHeaders(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/reports")
	idStr = strings.Trim(idStr, "/")

	// GET /api/reports — список
	if idStr == "" && r.Method == http.MethodGet {
		items := getReportHistory(dbConn)
		if items == nil {
			items = []ReportHistoryItem{}
		}
		json.NewEncoder(w).Encode(items)
		return
	}

	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		http.Error(w, `{"error":"invalid id"}`, http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		item, ok := getReportByID(dbConn, id)
		if !ok {
			http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(item)
	case http.MethodDelete:
		dbConn.Exec(`DELETE FROM report_history WHERE id=?`, id)
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

// ReportRequest — тело запроса на генерацию отчёта.
// Либо указывается Period (относительный), либо From+To (произвольный диапазон в формате "2006-01-02").
type ReportRequest struct {
	Nodes  []string `json:"nodes"`
	Period string   `json:"period"` // "1h","6h","12h","24h","7d","30d" — или пусто при кастомном диапазоне
	From   string   `json:"from"`   // "2006-01-02", опционально
	To     string   `json:"to"`     // "2006-01-02", опционально
}

type nodeStats struct {
	Name       string
	DisplayName string
	OS         string
	IP         string
	Online     bool
	LastSeen   string

	AvgCPU  float64
	MaxCPU  float64
	MinCPU  float64
	AvgRAM  float64
	MaxRAM  float64
	RAMTotal float64
	AvgDisk float64
	MaxDisk float64
	AvgNetRecv float64
	AvgNetSent float64
	Samples int
}

// HandleReport — POST /api/report
func HandleReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	corsHeaders(w)

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req ReportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || len(req.Nodes) == 0 {
		http.Error(w, `{"error":"укажите хотя бы один узел"}`, http.StatusBadRequest)
		return
	}
	// Нормализуем: если задан диапазон дат — period игнорируется
	if req.From != "" && req.To != "" {
		req.Period = "custom"
	} else if req.Period == "" {
		req.Period = "24h"
	}

	settings := GetGigachatSettings(dbConn)
	if settings.ClientID == "" || settings.ClientSecret == "" {
		http.Error(w, `{"error":"GigaChat не настроен — укажите Client ID и Client Secret в настройках"}`, http.StatusBadRequest)
		return
	}

	stats := gatherStats(dbConn, req.Nodes, req.Period, req.From, req.To)
	if len(stats) == 0 {
		http.Error(w, `{"error":"нет данных для выбранных узлов за указанный период"}`, http.StatusNotFound)
		return
	}

	prompt := buildPrompt(stats, req.Period, req.From, req.To)

	token, err := GigachatGetToken(settings)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"ошибка авторизации GigaChat: %s"}`, jsonEscape(err.Error())), http.StatusBadGateway)
		return
	}

	report, err := GigachatChat(token, prompt)
	if err != nil {
		// сбрасываем кеш токена при ошибке
		gcTokenMu.Lock()
		gcToken = ""
		gcTokenMu.Unlock()
		http.Error(w, fmt.Sprintf(`{"error":"ошибка GigaChat: %s"}`, jsonEscape(err.Error())), http.StatusBadGateway)
		return
	}

	saveReportHistory(dbConn, req.Period, req.From, req.To, req.Nodes, report)
	json.NewEncoder(w).Encode(map[string]string{"report": report})
}

// gatherStats собирает агрегированную статистику по узлам за период.
func gatherStats(db *sql.DB, nodes []string, period, from, to string) []nodeStats {
	since, until := periodToRange(period, from, to)

	aliases := GetAllAliases(db)

	// Текущее состояние из кеша
	cached, _ := getCachedNodes()
	onlineMap := make(map[string]NodeSummary)
	for _, n := range cached {
		onlineMap[n.Name] = n
	}

	var result []nodeStats
	for _, name := range nodes {
		var s nodeStats
		s.Name = name
		if a, ok := aliases[name]; ok && a != "" {
			s.DisplayName = a
		} else {
			s.DisplayName = name
		}
		if n, ok := onlineMap[name]; ok {
			s.OS = n.OS
			s.IP = n.IP
			s.Online = n.Online
			s.LastSeen = n.LastSeen
		}

		useHourly := period == "7d" || period == "30d" || period == "custom"
		if useHourly {
			row := db.QueryRow(`
				SELECT
					AVG(avg_cpu), MAX(avg_cpu), MIN(avg_cpu),
					AVG(avg_ram), MAX(avg_ram), AVG(avg_ram_total),
					AVG(avg_disk), MAX(avg_disk),
					AVG(avg_net_recv), AVG(avg_net_sent),
					COUNT(*)
				FROM metrics_hourly
				WHERE node_name = ? AND hour >= ? AND hour <= ?
			`, name, since, until)
			row.Scan(
				&s.AvgCPU, &s.MaxCPU, &s.MinCPU,
				&s.AvgRAM, &s.MaxRAM, &s.RAMTotal,
				&s.AvgDisk, &s.MaxDisk,
				&s.AvgNetRecv, &s.AvgNetSent,
				&s.Samples,
			)
		} else {
			row := db.QueryRow(`
				SELECT
					AVG(cpu_usage), MAX(cpu_usage), MIN(cpu_usage),
					AVG(ram_usage), MAX(ram_usage), AVG(ram_total),
					AVG(disk_usage), MAX(disk_usage),
					AVG(COALESCE(net_bytes_recv,0)), AVG(COALESCE(net_bytes_sent,0)),
					COUNT(*)
				FROM metrics
				WHERE node_name = ? AND timestamp >= ? AND timestamp <= ?
			`, name, since, until)
			row.Scan(
				&s.AvgCPU, &s.MaxCPU, &s.MinCPU,
				&s.AvgRAM, &s.MaxRAM, &s.RAMTotal,
				&s.AvgDisk, &s.MaxDisk,
				&s.AvgNetRecv, &s.AvgNetSent,
				&s.Samples,
			)
		}

		if s.Samples == 0 {
			continue
		}
		result = append(result, s)
	}
	return result
}

// periodToRange возвращает (since, until) в формате RFC3339.
// При period="custom" парсит from/to как "2006-01-02T15:04" или "2006-01-02".
func periodToRange(period, from, to string) (string, string) {
	now := time.Now().UTC()
	until := now.Format(time.RFC3339)

	if period == "custom" && from != "" && to != "" {
		loc := time.Local
		// пробуем datetime-local формат, затем date-only
		parseCustom := func(s string) (time.Time, error) {
			if t, err := time.ParseInLocation("2006-01-02T15:04", s, loc); err == nil {
				return t, nil
			}
			return time.ParseInLocation("2006-01-02", s, loc)
		}
		f, err1 := parseCustom(from)
		t, err2 := parseCustom(to)
		if err1 == nil && err2 == nil {
			since := f.UTC().Format(time.RFC3339)
			// если to задан только датой — берём конец дня, иначе точное время
			var untilCustom string
			if len(to) == 10 {
				untilCustom = t.Add(24*time.Hour - time.Second).UTC().Format(time.RFC3339)
			} else {
				untilCustom = t.UTC().Format(time.RFC3339)
			}
			return since, untilCustom
		}
	}

	var d time.Duration
	switch period {
	case "1h":
		d = time.Hour
	case "6h":
		d = 6 * time.Hour
	case "12h":
		d = 12 * time.Hour
	case "7d":
		d = 7 * 24 * time.Hour
	case "30d":
		d = 30 * 24 * time.Hour
	default: // "24h"
		d = 24 * time.Hour
	}
	return now.Add(-d).Format(time.RFC3339), until
}

// periodLabel возвращает человекочитаемое название периода.
func periodLabel(period, from, to string) string {
	if period == "custom" && from != "" && to != "" {
		fLabel := strings.ReplaceAll(from, "T", " ")
		tLabel := strings.ReplaceAll(to, "T", " ")
		return fmt.Sprintf("с %s по %s", fLabel, tLabel)
	}
	switch period {
	case "1h":
		return "последний час"
	case "6h":
		return "последние 6 часов"
	case "12h":
		return "последние 12 часов"
	case "7d":
		return "последние 7 дней"
	case "30d":
		return "последние 30 дней"
	default:
		return "последние 24 часа"
	}
}

// buildPrompt формирует текст запроса к GigaChat.
func buildPrompt(stats []nodeStats, period, from, to string) string {
	var sb strings.Builder

	sb.WriteString("Ты — система анализа серверной инфраструктуры. ")
	sb.WriteString("Составь профессиональный отчёт на русском языке о состоянии узлов сети за указанный период.\n\n")
	sb.WriteString(fmt.Sprintf("Период анализа: %s\n", periodLabel(period, from, to)))
	sb.WriteString(fmt.Sprintf("Количество узлов: %d\n\n", len(stats)))
	sb.WriteString("=== ДАННЫЕ МОНИТОРИНГА ===\n\n")

	for _, s := range stats {
		sb.WriteString(fmt.Sprintf("Узел: %s", s.DisplayName))
		if s.DisplayName != s.Name {
			sb.WriteString(fmt.Sprintf(" (%s)", s.Name))
		}
		sb.WriteString("\n")
		if s.OS != "" {
			sb.WriteString(fmt.Sprintf("  ОС: %s\n", s.OS))
		}
		if s.IP != "" {
			sb.WriteString(fmt.Sprintf("  IP: %s\n", s.IP))
		}
		if s.Online {
			sb.WriteString("  Статус: ОНЛАЙН\n")
		} else {
			sb.WriteString(fmt.Sprintf("  Статус: ОФЛАЙН (последний раз онлайн: %s)\n", s.LastSeen))
		}
		sb.WriteString(fmt.Sprintf("  CPU: среднее %.1f%%, максимум %.1f%%, минимум %.1f%%\n",
			s.AvgCPU, s.MaxCPU, s.MinCPU))

		ramPct := 0.0
		maxRamPct := 0.0
		if s.RAMTotal > 0 {
			ramPct = s.AvgRAM / s.RAMTotal * 100
			maxRamPct = s.MaxRAM / s.RAMTotal * 100
		}
		sb.WriteString(fmt.Sprintf("  RAM: среднее %.2f / %.2f ГБ (%.1f%%), пиковое %.2f ГБ (%.1f%%)\n",
			s.AvgRAM, s.RAMTotal, ramPct, s.MaxRAM, maxRamPct))
		sb.WriteString(fmt.Sprintf("  Диск: среднее %.1f%%, максимум %.1f%%\n",
			s.AvgDisk, s.MaxDisk))
		sb.WriteString(fmt.Sprintf("  Сеть: среднее получение %.1f КБ/с, отправка %.1f КБ/с\n",
			s.AvgNetRecv/1024, s.AvgNetSent/1024))
		sb.WriteString(fmt.Sprintf("  Точек данных: %d\n\n", s.Samples))
	}

	sb.WriteString("=== ПОРОГОВЫЕ ЗНАЧЕНИЯ ДЛЯ ОЦЕНКИ ===\n\n")
	sb.WriteString("Используй строго эти пороги при анализе — не считай нормальные показатели проблемой:\n\n")
	sb.WriteString("CPU:\n")
	sb.WriteString("  - до 60% среднее — норма, не требует комментария\n")
	sb.WriteString("  - 60–80% среднее — умеренная нагрузка, упомяни вскользь\n")
	sb.WriteString("  - выше 80% среднее — высокая нагрузка, требует внимания\n")
	sb.WriteString("  - пиковые значения до 90% допустимы даже при низком среднем — не считай их проблемой\n\n")
	sb.WriteString("RAM:\n")
	sb.WriteString("  - до 75% — норма\n")
	sb.WriteString("  - 75–90% — умеренное использование\n")
	sb.WriteString("  - выше 90% — критично\n\n")
	sb.WriteString("Диск:\n")
	sb.WriteString("  - до 80% — норма\n")
	sb.WriteString("  - 80–90% — следует упомянуть\n")
	sb.WriteString("  - выше 90% — критично\n\n")
	sb.WriteString("=== ЗАДАНИЕ ===\n\n")
	sb.WriteString("Составь структурированный отчёт со следующими разделами:\n")
	sb.WriteString("1. Общая сводка — краткое резюме состояния инфраструктуры\n")
	sb.WriteString("2. Анализ узлов — подробный разбор каждого узла с оценкой показателей по указанным порогам\n")
	sb.WriteString("3. Выявленные проблемы — только реальные проблемы согласно порогам выше; если проблем нет — так и напиши\n")
	sb.WriteString("4. Рекомендации — конкретные действия только при наличии реальных проблем\n\n")
	sb.WriteString("Строгие правила форматирования:\n")
	sb.WriteString("- НИКАКИХ таблиц, ни в каком виде\n")
	sb.WriteString("- НИКАКИХ символов # для заголовков — разделы обозначай только заглавными буквами или цифрами (например: «1. ОБЩАЯ СВОДКА»)\n")
	sb.WriteString("- Не используй markdown-разметку вообще: ни ##, ни **, ни __, ни другие специальные символы\n")
	sb.WriteString("- Не драматизируй нормальные показатели, не используй слова «опасно», «критично», «тревожно» для метрик в пределах нормы\n")
	sb.WriteString("- Будь конкретен в цифрах, используй профессиональный технический язык\n")
	sb.WriteString("- Пиши на русском языке")

	return sb.String()
}

// jsonEscape экранирует строку для безопасной вставки в JSON-литерал.
func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return strings.Trim(string(b), `"`)
}
