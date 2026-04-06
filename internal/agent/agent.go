package agent

import (
	"encoding/json" // Новый импорт для работы с JSON
	"fmt"
	"log"
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
	log.Println("Агент запущен. Начинаем упаковку данных...")

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
	hostname, _ := os.Hostname() // Получаем имя твоего ПК (DESKTOP-936KA0K)

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

	// 3. Превращаем структуру в JSON (маршалинг)
	// MarshalIndent делает JSON красивым для глаз (с отступами)
	jsonData, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		log.Printf("Ошибка упаковки JSON: %v", err)
		return
	}

	// Выводим результат
	fmt.Println("\n📦 Сформирован JSON-пакет:")
	fmt.Println(string(jsonData))
}
