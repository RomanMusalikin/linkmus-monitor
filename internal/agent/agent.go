// internal/agent/agent.go
package agent

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

// MetricPayload — это структура нашего сообщения для сервера
type MetricPayload struct {
	NodeName  string  `json:"node_name"`
	Timestamp string  `json:"timestamp"`
	CPUUsage  float64 `json:"cpu_usage"`
	RAMUsage  float64 `json:"ram_usage"`
	DiskUsage float64 `json:"disk_usage"`
}

func Run() {
	// 1. Читаем конфиг из файла
	cfg, err := LoadConfig("configs/agent-config.yaml")
	if err != nil {
		log.Fatalf("❌ Ошибка загрузки конфига: %v", err)
	}

	log.Printf("Агент запущен. Сервер: %s, Интервал: %s", cfg.Server.URL, cfg.Server.Interval)

	// 2. Используем интервал из конфига
	ticker := time.NewTicker(cfg.Server.Interval)
	defer ticker.Stop()

	for t := range ticker.C {
		// 3. Передаем URL сервера из конфига в функцию
		collectAndSend(t, cfg.Server.URL)
	}
}

// Добавили параметр serverURL
func collectAndSend(t time.Time, serverURL string) {
	// 1. Собираем данные
	cpuPercent, _ := cpu.Percent(time.Second, false)
	vMem, _ := mem.VirtualMemory()
	dUsage, _ := disk.Usage("/")
	hostname, _ := os.Hostname()

	// 2. Заполняем наш "бланк" (структуру)
	payload := MetricPayload{
		NodeName:  hostname,
		Timestamp: t.Format(time.RFC3339),
		RAMUsage:  vMem.UsedPercent,
		DiskUsage: dUsage.UsedPercent,
	}

	if len(cpuPercent) > 0 {
		payload.CPUUsage = cpuPercent[0]
	}

	// 3. Превращаем структуру в компактный JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("❌ Ошибка упаковки JSON: %v", err)
		return
	}

	// 4. Отправляем данные на Мастер-сервер по URL из параметров!
	// Мы удалили жестко зашитый http://127.0.0.1:8080/api/metrics
	resp, err := http.Post(serverURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("❌ Ошибка связи с сервером: %v", err)
		return
	}
	defer resp.Body.Close()

	// 5. Выводим статус отправки
	if resp.StatusCode == http.StatusOK {
		log.Printf("✅ Метрики успешно отправлены на %s (Статус: %s)", serverURL, resp.Status)
	} else {
		log.Printf("⚠️ Сервер вернул ошибку: %s", resp.Status)
	}
}
