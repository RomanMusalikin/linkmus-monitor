package server

import (
	"encoding/json"
	"log"
	"net/http"
)

// CpuPoint — одна точка истории CPU
type CpuPoint struct {
	Value int `json:"value"`
}

// RamPoint — одна точка истории RAM (процент)
type RamPoint struct {
	Value int `json:"value"`
}

// NetPoint — одна точка истории сети (байт/сек)
type NetPoint struct {
	Recv float64 `json:"recv"`
	Sent float64 `json:"sent"`
}

// ProcessInfo — информация об одном процессе
type ProcessInfo struct {
	PID  int32   `json:"pid"`
	Name string  `json:"name"`
	CPU  float64 `json:"cpu"`
	RAM  float64 `json:"ram"` // МБ
	User string  `json:"user"`
}

// NodeSummary — полный набор данных по узлу, отдаваемый фронтенду
type NodeSummary struct {
	Name         string        `json:"name"`
	OS           string        `json:"os"`
	IP           string        `json:"ip"`
	Online       bool          `json:"online"`
	CPU          int           `json:"cpu"`
	CPUUser      float64       `json:"cpuUser"`
	CPUSystem    float64       `json:"cpuSystem"`
	LoadAvg1     float64       `json:"loadAvg1"`
	LoadAvg5     float64       `json:"loadAvg5"`
	LoadAvg15    float64       `json:"loadAvg15"`
	RAMUsed      float64       `json:"ramUsed"`    // ГБ
	RAMTotal     float64       `json:"ramTotal"`   // ГБ
	RAMCached    float64       `json:"ramCached"`  // ГБ
	RAMBuffers   float64       `json:"ramBuffers"` // ГБ
	SwapUsed     float64       `json:"swapUsed"`   // ГБ
	SwapTotal    float64       `json:"swapTotal"`  // ГБ
	DiskUsage    float64       `json:"diskUsage"`  // %
	RDPRunning   bool          `json:"rdpRunning"`
	SMBRunning   bool          `json:"smbRunning"`
	Uptime       string        `json:"uptime"`
	Ping         int           `json:"ping"`
	NetInterface string        `json:"netInterface"`
	NetRecvSec   float64       `json:"netRecvSec"` // байт/сек
	NetSentSec   float64       `json:"netSentSec"` // байт/сек
	CPUHistory   []CpuPoint    `json:"cpuHistory"`
	RAMHistory   []RamPoint    `json:"ramHistory"`
	NetHistory   []NetPoint    `json:"netHistory"`
	Processes    []ProcessInfo `json:"processes"`
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
