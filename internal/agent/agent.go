package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"

	"linkmus-monitor/internal/collector"
)

// MetricPayload — полный набор метрик, отправляемый агентом на сервер
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
	RAMUsage      float64 `json:"ram_usage"`   // ГБ используется
	RAMTotal      float64 `json:"ram_total"`   // ГБ всего
	RAMCached     float64 `json:"ram_cached"`  // ГБ кэш
	RAMBuffers    float64 `json:"ram_buffers"` // ГБ буферы
	SwapUsed      float64 `json:"swap_used"`   // ГБ
	SwapTotal     float64 `json:"swap_total"`  // ГБ
	DiskUsage     float64 `json:"disk_usage"`  // % заполнения C:\ или /
	RDPRunning    bool    `json:"rdp_running"`
	SMBRunning    bool    `json:"smb_running"`
	NetInterface  string  `json:"net_interface"`
	NetBytesRecv  float64 `json:"net_bytes_recv"` // байт/сек
	NetBytesSent  float64 `json:"net_bytes_sent"` // байт/сек
	ProcessesJSON string  `json:"processes_json"` // JSON-массив топ-10 процессов
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
	outboundIP := getOutboundIP()

	// Системная информация
	h, _ := host.Info()

	// Память
	v, _ := mem.VirtualMemory()
	s, _ := mem.SwapMemory()

	// CPU: user%, system%, total% через дельту между вызовами
	cpuUser, cpuSystem, cpuTotal := collector.CollectCPUBreakdown()

	// Load average (0/0/0 на Windows, реальные значения на Linux)
	var la1, la5, la15 float64
	if avg, err := load.Avg(); err == nil && avg != nil {
		la1, la5, la15 = avg.Load1, avg.Load5, avg.Load15
	}

	// Диск
	diskUsg := collector.CollectDisk()

	// Службы (RDP/SMB) — только Windows
	rdpStatus, smbStatus := collector.CollectServices()

	// Сеть
	netInfo := collector.CollectNetwork(outboundIP)

	// Процессы (топ-10 по CPU)
	procs := collector.CollectProcesses()
	procsJSON, _ := json.Marshal(procs)

	payload := MetricPayload{
		NodeName:      hostname,
		OS:            h.OS + " " + h.Platform,
		IP:            outboundIP,
		Uptime:        fmt.Sprintf("%d ч.", h.Uptime/3600),
		Timestamp:     t.Format(time.RFC3339),
		CPUUsage:      cpuTotal,
		CPUUser:       cpuUser,
		CPUSystem:     cpuSystem,
		LoadAvg1:      la1,
		LoadAvg5:      la5,
		LoadAvg15:     la15,
		RAMUsage:      float64(v.Used) / 1024 / 1024 / 1024,
		RAMTotal:      float64(v.Total) / 1024 / 1024 / 1024,
		RAMCached:     float64(v.Cached) / 1024 / 1024 / 1024,
		RAMBuffers:    float64(v.Buffers) / 1024 / 1024 / 1024,
		SwapUsed:      float64(s.Used) / 1024 / 1024 / 1024,
		SwapTotal:     float64(s.Total) / 1024 / 1024 / 1024,
		DiskUsage:     diskUsg,
		RDPRunning:    rdpStatus,
		SMBRunning:    smbStatus,
		NetInterface:  netInfo.Interface,
		NetBytesRecv:  netInfo.BytesRecvSec,
		NetBytesSent:  netInfo.BytesSentSec,
		ProcessesJSON: string(procsJSON),
	}

	SendToServer(serverURL, payload)
}

// getOutboundIP определяет IP-адрес, с которого идёт трафик к внешним хостам
func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}
