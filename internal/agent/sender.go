package agent

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
)

// SendToServer пакует данные в JSON и делает POST-запрос
func SendToServer(serverURL string, payload MetricPayload) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("❌ Ошибка упаковки JSON: %v", err)
		return
	}

	resp, err := http.Post(serverURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("❌ Ошибка связи с сервером: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.Printf("✅ Отправлено: CPU:%.1f%% | RAM:%.1f%% | RDP:%v | SMB:%v",
			payload.CPUUsage, payload.RAMUsage, payload.RDPRunning, payload.SMBRunning)
	} else {
		log.Printf("⚠️ Сервер вернул ошибку: %s", resp.Status)
	}
}
