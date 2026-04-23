package agent

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"

	"linkmus-monitor/internal/collector"
)

// MetricPayload — полный набор метрик, отправляемый агентом на сервер
type MetricPayload struct {
	NodeName    string  `json:"node_name"`
	OS          string  `json:"os"`
	IP          string  `json:"ip"`
	Uptime      string  `json:"uptime"`
	BootTime    string  `json:"boot_time"`
	Timestamp   string  `json:"timestamp"`
	LoggedUsers int     `json:"logged_users"`

	// CPU
	CPUUsage     float64 `json:"cpu_usage"`
	CPUUser      float64 `json:"cpu_user"`
	CPUSystem    float64 `json:"cpu_system"`
	CPUIOwait    float64 `json:"cpu_iowait"`
	CPUSteal     float64 `json:"cpu_steal"`
	CPUTemp      float64 `json:"cpu_temp"`
	CPUModel     string  `json:"cpu_model"`
	CPUFreqMHz   float64 `json:"cpu_freq_mhz"`
	CPUCoresJSON string  `json:"cpu_cores_json"` // JSON []float64

	// Load average
	LoadAvg1  float64 `json:"load_avg_1"`
	LoadAvg5  float64 `json:"load_avg_5"`
	LoadAvg15 float64 `json:"load_avg_15"`

	// RAM
	RAMUsage   float64 `json:"ram_usage"`   // ГБ используется
	RAMTotal   float64 `json:"ram_total"`   // ГБ всего
	RAMCached  float64 `json:"ram_cached"`  // ГБ кэш
	RAMBuffers float64 `json:"ram_buffers"` // ГБ буферы
	SwapUsed   float64 `json:"swap_used"`   // ГБ
	SwapTotal  float64 `json:"swap_total"`  // ГБ

	// Диск
	DiskUsage    float64 `json:"disk_usage"`
	DiskReadSec  float64 `json:"disk_read_sec"`
	DiskWriteSec float64 `json:"disk_write_sec"`
	DiskQueue    float64 `json:"disk_queue"`
	DisksJSON    string  `json:"disks_json"`

	// FSRM (Windows only, srv-corp-01)
	FSRMJson string `json:"fsrm_json"`

	// Службы
	RDPRunning bool `json:"rdp_running"`
	SMBRunning bool `json:"smb_running"`

	// Сеть — основной интерфейс
	NetInterface string  `json:"net_interface"`
	NetBytesRecv float64 `json:"net_bytes_recv"` // байт/сек
	NetBytesSent float64 `json:"net_bytes_sent"` // байт/сек

	// Сеть — все интерфейсы
	AllIfacesJSON string `json:"all_ifaces_json"` // JSON []NetIfaceInfo

	// TCP-соединения
	TCPTotal       int `json:"tcp_total"`
	TCPEstablished int `json:"tcp_established"`
	TCPTimeWait    int `json:"tcp_timewait"`

	// Процессы
	ProcessCount  int    `json:"process_count"`
	ProcessesJSON string `json:"processes_json"`  // JSON топ-10 по CPU
	TopMemJSON    string `json:"top_mem_json"`    // JSON топ-10 по RAM
}

const maxLogSize = 5 * 1024 * 1024 // 5 МБ

func Run() {
	// Пишем логи в файл рядом с exe — работает при запуске как служба Windows
	exe, _ := os.Executable()
	logPath := strings.TrimSuffix(exe, ".exe") + ".log"
	rotateLog(logPath)
	if f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644); err == nil {
		log.SetOutput(f)
	}

	cfg, err := LoadConfig(configPath())
	if err != nil {
		log.Fatalf("Ошибка загрузки конфига: %v", err)
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
	hostname = strings.TrimSuffix(hostname, ".localdomain")
	outboundIP := getOutboundIP()

	// Системная информация
	h, _ := host.Info()

	// Дата последней загрузки
	bootTime := ""
	if h != nil {
		bootTime = time.Unix(int64(h.BootTime), 0).Local().Format("02.01.2006 15:04")
	}

	// Залогиненные пользователи
	loggedUsers := 0
	if users, err := host.Users(); err == nil {
		loggedUsers = len(users)
	}

	// Память
	v, _ := mem.VirtualMemory()
	s, _ := mem.SwapMemory()

	// CPU: user%, system%, iowait%, steal%, total%
	cpuUser, cpuSystem, cpuIOwait, cpuSteal, cpuTotal := collector.CollectCPUBreakdown()

	// CPU по ядрам
	cpuCores := collector.CollectCPUPerCore()
	cpuCoresJSON, _ := json.Marshal(cpuCores)

	// Модель и частота CPU
	cpuModel, cpuFreq := collector.CollectCPUInfo()

	// Load average (0 на Windows)
	var la1, la5, la15 float64
	if avg, err := load.Avg(); err == nil && avg != nil {
		la1, la5, la15 = avg.Load1, avg.Load5, avg.Load15
	}

	// Диск — корневой раздел
	diskUsg := collector.CollectDisk()

	// Диск — все разделы
	allDisks := collector.CollectAllDisks()
	disksJSON, _ := json.Marshal(allDisks)

	// Диск — I/O
	diskReadSec, diskWriteSec, diskQueue := collector.CollectDiskIO()

	// FSRM (только Windows-агент, на Linux вернёт nil)
	fsrmList := collector.CollectFSRM()
	fsrmJSON, _ := json.Marshal(fsrmList)

	// Службы (RDP/SMB) — только Windows, на Linux возвращает false/false
	rdpStatus, smbStatus := collector.CollectServices()

	// Сеть — основной интерфейс
	netInfo := collector.CollectNetwork(outboundIP)

	// Сеть — все интерфейсы
	allIfaces := collector.CollectAllInterfaces()
	allIfacesJSON, _ := json.Marshal(allIfaces)

	// Температура CPU
	cpuTemp := collector.CollectCPUTemp()

	// TCP-соединения
	tcpTotal, tcpEstablished, tcpTimeWait := collector.CollectTCPConnections()

	// Процессы
	processCount := collector.CollectProcessCount()
	procs := collector.CollectProcesses()
	procsJSON, _ := json.Marshal(procs)
	topMem := collector.CollectTopMemProcesses()
	topMemJSON, _ := json.Marshal(topMem)

	uptime := "0 ч."
	if h != nil {
		uptime = fmt.Sprintf("%d ч.", h.Uptime/3600)
	}
	osStr := ""
	if h != nil {
		osStr = h.OS + " " + h.Platform
	}

	payload := MetricPayload{
		NodeName:     hostname,
		OS:           osStr,
		IP:           outboundIP,
		Uptime:       uptime,
		BootTime:     bootTime,
		Timestamp:    t.Format(time.RFC3339),
		LoggedUsers:  loggedUsers,
		CPUUsage:     cpuTotal,
		CPUUser:      cpuUser,
		CPUSystem:    cpuSystem,
		CPUIOwait:    cpuIOwait,
		CPUSteal:     cpuSteal,
		CPUTemp:      cpuTemp,
		CPUModel:     cpuModel,
		CPUFreqMHz:   cpuFreq,
		CPUCoresJSON: string(cpuCoresJSON),
		LoadAvg1:     la1,
		LoadAvg5:     la5,
		LoadAvg15:    la15,
		RAMUsage:     float64(v.Used) / 1024 / 1024 / 1024,
		RAMTotal:     float64(v.Total) / 1024 / 1024 / 1024,
		RAMCached:    float64(v.Cached) / 1024 / 1024 / 1024,
		RAMBuffers:   float64(v.Buffers) / 1024 / 1024 / 1024,
		SwapUsed:     float64(s.Used) / 1024 / 1024 / 1024,
		SwapTotal:    float64(s.Total) / 1024 / 1024 / 1024,
		DiskUsage:    diskUsg,
		DiskReadSec:  diskReadSec,
		DiskWriteSec: diskWriteSec,
		DiskQueue:    diskQueue,
		DisksJSON:    string(disksJSON),
		FSRMJson:     string(fsrmJSON),
		RDPRunning:   rdpStatus,
		SMBRunning:   smbStatus,
		NetInterface: netInfo.Interface,
		NetBytesRecv: netInfo.BytesRecvSec,
		NetBytesSent: netInfo.BytesSentSec,
		AllIfacesJSON: string(allIfacesJSON),
		TCPTotal:       tcpTotal,
		TCPEstablished: tcpEstablished,
		TCPTimeWait:    tcpTimeWait,
		ProcessCount:   processCount,
		ProcessesJSON:  string(procsJSON),
		TopMemJSON:     string(topMemJSON),
	}

	SendToServer(serverURL, payload)
}

// rotateLog переименовывает лог в .log.old если он превышает maxLogSize.
func rotateLog(path string) {
	info, err := os.Stat(path)
	if err != nil || info.Size() < maxLogSize {
		return
	}
	os.Rename(path, path+".old")
}

// getOutboundIP определяет IP-адрес, с которого идёт трафик к внешним хостам
func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "127.0.0.1"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}
