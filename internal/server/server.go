// internal/server/server.go
package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// Структура должна в точности совпадать с той, что в агенте
type MetricPayload struct {
	NodeName  string  `json:"node_name"`
	Timestamp string  `json:"timestamp"`
	CPUUsage  float64 `json:"cpu_usage"`
	RAMUsage  float64 `json:"ram_usage"`
	DiskUsage float64 `json:"disk_usage"`
}

func Run() {
	// Говорим серверу: если пришел запрос на /api/metrics, передай его в функцию handleMetrics
	http.HandleFunc("/api/metrics", handleMetrics)

	port := ":8080"
	log.Printf("🚀 Мастер-сервер запущен. Слушаю порт %s...", port)

	// Запускаем бесконечный цикл прослушивания порта
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

// Эта функция срабатывает каждый раз, когда кто-то стучится на /api/metrics
func handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Мы ждем только POST-запросы с данными
	if r.Method != http.MethodPost {
		http.Error(w, "Ожидается метод POST", http.StatusMethodNotAllowed)
		return
	}

	var payload MetricPayload
	// Декодируем JSON из тела запроса прямо в нашу структуру
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Ошибка чтения JSON", http.StatusBadRequest)
		return
	}

	// Красиво выводим то, что получили от агента
	fmt.Printf("\n📥 Получены метрики от узла [%s]\n", payload.NodeName)
	fmt.Printf("   Время: %s\n", payload.Timestamp)
	fmt.Printf("   CPU: %.2f%% | RAM: %.2f%% | Disk: %.2f%%\n", payload.CPUUsage, payload.RAMUsage, payload.DiskUsage)

	// Отправляем агенту HTTP-ответ 200 OK (Всё супер, данные принял!)
	w.WriteHeader(http.StatusOK)
}
