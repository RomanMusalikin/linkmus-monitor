package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

var httpClient = &http.Client{Timeout: 10 * time.Second}

// SendToServer пакует данные в JSON и делает POST-запрос.
// Таймаут 10с — если сеть зависла, не блокируем следующий интервал.
func SendToServer(serverURL string, payload MetricPayload) {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Ошибка упаковки JSON: %v", err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, serverURL, bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("Ошибка создания запроса: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Ошибка связи с сервером: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Сервер вернул ошибку: %s", resp.Status)
	}
}
