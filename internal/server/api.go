package server

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

// CpuPoint — одна точка истории CPU
type CpuPoint struct {
	Value int    `json:"value"`
	Time  string `json:"time"`
}

// RamPoint — одна точка истории RAM (процент)
type RamPoint struct {
	Value int    `json:"value"`
	Time  string `json:"time"`
}

// NetPoint — одна точка истории сети (байт/сек)
type NetPoint struct {
	Recv float64 `json:"recv"`
	Sent float64 `json:"sent"`
	Time string  `json:"time"`
}

// ProcessInfo — информация об одном процессе
type ProcessInfo struct {
	PID  int32   `json:"pid"`
	Name string  `json:"name"`
	CPU  float64 `json:"cpu"`
	RAM  float64 `json:"ram"` // МБ
	User string  `json:"user"`
}

// FSRMInfo — квота FSRM (Windows, srv-corp-01)
type FSRMInfo struct {
	QuotaPath         string  `json:"quotaPath"`
	QuotaLimitBytes   int64   `json:"quotaLimitBytes"`
	QuotaUsedBytes    int64   `json:"quotaUsedBytes"`
	QuotaUsedPercent  float64 `json:"quotaUsedPercent"`
	Violations24h     int     `json:"violations24h"`
	LastViolationTime string  `json:"lastViolationTime"`
	LastViolationType string  `json:"lastViolationType"`
}

// DiskInfo — дисковый раздел
type DiskInfo struct {
	Mount       string  `json:"mount"`
	FSType      string  `json:"fstype"`
	TotalGB     float64 `json:"totalGB"`
	UsedGB      float64 `json:"usedGB"`
	UsedPercent float64 `json:"usedPercent"`
}

// NetIfaceInfo — сетевой интерфейс
type NetIfaceInfo struct {
	Name         string  `json:"name"`
	BytesRecvSec float64 `json:"bytesRecvSec"`
	BytesSentSec float64 `json:"bytesSentSec"`
}

// NodeSummary — полный набор данных по узлу, отдаваемый фронтенду
type NodeSummary struct {
	Name     string `json:"name"`
	OS       string `json:"os"`
	IP       string `json:"ip"`
	Online   bool   `json:"online"`
	LastSeen string `json:"lastSeen"`
	Uptime   string `json:"uptime"`
	BootTime string `json:"bootTime"`


	// CPU
	CPU        int       `json:"cpu"`
	CPUUser    float64   `json:"cpuUser"`
	CPUSystem  float64   `json:"cpuSystem"`
	CPUIOwait  float64   `json:"cpuIowait"`
	CPUSteal   float64   `json:"cpuSteal"`
	CPUTemp    float64   `json:"cpuTemp"`
	CPUModel   string    `json:"cpuModel"`
	CPUFreqMHz float64   `json:"cpuFreqMHz"`
	CPUCores   []float64 `json:"cpuCores"`
	LoadAvg1   float64   `json:"loadAvg1"`
	LoadAvg5   float64   `json:"loadAvg5"`
	LoadAvg15  float64   `json:"loadAvg15"`

	// RAM
	RAMUsed    float64 `json:"ramUsed"`    // ГБ
	RAMTotal   float64 `json:"ramTotal"`   // ГБ
	RAMCached  float64 `json:"ramCached"`  // ГБ
	RAMBuffers float64 `json:"ramBuffers"` // ГБ
	SwapUsed   float64 `json:"swapUsed"`   // ГБ
	SwapTotal  float64 `json:"swapTotal"`  // ГБ

	// Диск
	DiskUsage    float64    `json:"diskUsage"`
	DiskReadSec  float64    `json:"diskReadSec"`
	DiskWriteSec float64    `json:"diskWriteSec"`
	DiskQueue    float64    `json:"diskQueue"`
	Disks        []DiskInfo `json:"disks"`

	// Службы
	RDPRunning bool `json:"rdpRunning"`
	SMBRunning bool `json:"smbRunning"`

	// Сеть
	NetInterface string         `json:"netInterface"`
	NetRecvSec   float64        `json:"netRecvSec"`
	NetSentSec   float64        `json:"netSentSec"`
	AllIfaces    []NetIfaceInfo `json:"allIfaces"`

	// TCP-соединения
	TCPTotal       int `json:"tcpTotal"`
	TCPEstablished int `json:"tcpEstablished"`
	TCPTimeWait    int `json:"tcpTimeWait"`

	// Процессы
	ProcessCount    int           `json:"processCount"`
	LoggedUsers     int           `json:"loggedUsers"`
	Processes       []ProcessInfo `json:"processes"`
	TopMemProcesses []ProcessInfo `json:"topMemProcesses"`

	// Сервисные пробы (server-side)
	SSHReachable   bool    `json:"sshReachable"`
	RDPReachable   bool    `json:"rdpReachable"`
	SMBReachable   bool    `json:"smbReachable"`
	HTTPReachable  bool    `json:"httpReachable"`
	WinRMReachable bool    `json:"winrmReachable"`
	DNSReachable   bool    `json:"dnsReachable"`
	SSHMs          float64 `json:"sshMs"`
	RDPMs          float64 `json:"rdpMs"`
	SMBMs          float64 `json:"smbMs"`
	HTTPMs         float64 `json:"httpMs"`
	WinRMMs        float64 `json:"winrmMs"`
	DNSMs          float64 `json:"dnsMs"`

	// SNMP (server-side poller)
	SNMPCollected  bool   `json:"snmpCollected"`
	SNMPSysUpTime  uint32 `json:"snmpSysUpTime"`
	SNMPSysName    string `json:"snmpSysName"`
	SNMPCPULoad    int    `json:"snmpCpuLoad"`
	SNMPIfCount    int    `json:"snmpIfCount"`

	// FSRM (agent-side, Windows only)
	FSRM []FSRMInfo `json:"fsrm"`

	// История (для графиков)
	CPUHistory []CpuPoint `json:"cpuHistory"`
	RAMHistory []RamPoint `json:"ramHistory"`
	NetHistory []NetPoint `json:"netHistory"`
}

// HandleNodeDelete — DELETE /api/nodes/{name}
func HandleNodeDelete(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodDelete {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	name := strings.TrimPrefix(r.URL.Path, "/api/nodes/")
	if name == "" {
		http.Error(w, `{"error":"node name required"}`, http.StatusBadRequest)
		return
	}
	if _, err := dbConn.Exec(`DELETE FROM metrics WHERE node_name = ?`, name); err != nil {
		http.Error(w, `{"error":"delete failed"}`, http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// HandleNodes — GET /api/nodes
func HandleNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if dbConn == nil {
		http.Error(w, `{"error":"нет подключения к БД"}`, http.StatusInternalServerError)
		return
	}

	nodes, err := GetLatestNodes(dbConn)
	if err != nil {
		log.Printf("Ошибка GetLatestNodes: %v", err)
		http.Error(w, `{"error":"ошибка получения данных"}`, http.StatusInternalServerError)
		return
	}

	if nodes == nil {
		nodes = []NodeSummary{}
	}

	json.NewEncoder(w).Encode(nodes)
}
