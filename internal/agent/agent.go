package agent

import (
	"log"
	"os"
	"time"

	"linkmus-monitor/internal/collector"
)

// MetricPayload для агента (теперь совпадает с сервером)
type MetricPayload struct {
	NodeName   string  `json:"node_name"`
	Timestamp  string  `json:"timestamp"`
	CPUUsage   float64 `json:"cpu_usage"`
	RAMUsage   float64 `json:"ram_usage"`
	DiskUsage  float64 `json:"disk_usage"`
	RDPRunning bool    `json:"rdp_running"`
	SMBRunning bool    `json:"smb_running"`
}

func Run() {
	cfg, err := LoadConfig("configs/agent-config.yaml")
	if err != nil {
		log.Fatalf("❌ Ошибка загрузки конфига: %v", err)
	}

	log.Printf("Агент запущен. Сервер: %s, Интервал: %s", cfg.Server.URL, cfg.Server.Interval)

	ticker := time.NewTicker(cfg.Server.Interval)
	defer ticker.Stop()

	for t := range ticker.C {
		collectAndSend(t, cfg.Server.URL)
	}
}

func collectAndSend(t time.Time, serverURL string) {
	hostname, _ := os.Hostname()

	// 1. Вызываем отдельные коллекторы!
	cpuUsg := collector.CollectCPU()
	ramUsg := collector.CollectMemory()
	diskUsg := collector.CollectDisk()
	rdpStatus, smbStatus := collector.CollectServices() // Эту функцию мы поправим на следующем микрошаге

	// 2. Формируем структуру
	payload := MetricPayload{
		NodeName:   hostname,
		Timestamp:  t.Format(time.RFC3339),
		CPUUsage:   cpuUsg,
		RAMUsage:   ramUsg,
		DiskUsage:  diskUsg,
		RDPRunning: rdpStatus,
		SMBRunning: smbStatus,
	}

	// 3. Отдаем работу сендеру!
	SendToServer(serverURL, payload)
}
