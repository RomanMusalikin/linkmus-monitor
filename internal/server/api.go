package server

import (
	"encoding/json"
	"log"
	"net/http"
)

// CpuPoint описывает одну точку на мини-графике
type CpuPoint struct {
	Value int `json:"value"`
}

// NodeSummary описывает данные для карточки на главном Дашборде
type NodeSummary struct {
	Name       string     `json:"name"`
	OS         string     `json:"os"`
	IP         string     `json:"ip"`
	Online     bool       `json:"online"`
	CPU        int        `json:"cpu"`
	RAMUsed    int        `json:"ramUsed"`
	RAMTotal   int        `json:"ramTotal"`
	Uptime     string     `json:"uptime"`
	Ping       int        `json:"ping"`
	CPUHistory []CpuPoint `json:"cpuHistory"`
}

// HandleNodes — HTTP-обработчик для пути /api/nodes
func HandleNodes(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// ВАЖНО: Мы обращаемся к глобальной переменной dbConn из файла server.go
	if dbConn == nil {
		http.Error(w, `{"error": "Нет подключения к БД"}`, http.StatusInternalServerError)
		return
	}

	// Получаем реальные данные из БД!
	nodes, err := GetLatestNodes(dbConn)
	if err != nil {
		log.Printf("Ошибка формирования списка узлов: %v", err)
		http.Error(w, `{"error": "Ошибка получения данных"}`, http.StatusInternalServerError)
		return
	}

	// Если база пустая, отправляем пустой массив, а не null
	if nodes == nil {
		nodes = []NodeSummary{}
	}

	json.NewEncoder(w).Encode(nodes)
}
