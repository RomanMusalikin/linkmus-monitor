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
	log.Println("Агент запущен. Начинаем сбор и отправку данных...")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for t := range ticker.C {
		collectAndSend(t)
	}
}

func collectAndSend(t time.Time) {
	// 1. Собираем данные
	cpuPercent, _ := cpu.Percent(time.Second, false)
	vMem, _ := mem.VirtualMemory()
	dUsage, _ := disk.Usage("/")
	hostname, _ := os.Hostname()

	// 2. Заполняем наш "бланк" (структуру)
	payload := MetricPayload{
		NodeName:  hostname,
		Timestamp: t.Format(time.RFC3339), // Стандарт времени для API
		RAMUsage:  vMem.UsedPercent,
		DiskUsage: dUsage.UsedPercent,
	}

	if len(cpuPercent) > 0 {
		payload.CPUUsage = cpuPercent[0]
	}

	// 3. Превращаем структуру в компактный JSON (без лишних пробелов)
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("❌ Ошибка упаковки JSON: %v", err)
		return
	}

	// 4. Отправляем данные на Мастер-сервер
	serverURL := "http://127.0.0.1:8080/api/metrics"

	// Используем bytes.NewBuffer для передачи JSON в тело запроса
	resp, err := http.Post(serverURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("❌ Ошибка связи с сервером: %v", err)
		return
	}
	// Обязательно закрываем тело ответа, чтобы не было утечек памяти
	defer resp.Body.Close()

	// 5. Выводим статус отправки
	if resp.StatusCode == http.StatusOK {
		log.Printf("✅ Метрики успешно отправлены (Статус: %s)", resp.Status)
	} else {
		log.Printf("⚠️ Сервер вернул ошибку: %s", resp.Status)
	}
}
