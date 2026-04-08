// internal/server/api.go
package server

import (
	"encoding/json"
	"net/http"
)

// handleGetHistory отдает данные из БД по запросу от браузера/фронтенда
func handleGetHistory(w http.ResponseWriter, r *http.Request) {
	// Проверяем, что к нам стучатся именно методом GET
	if r.Method != http.MethodGet {
		http.Error(w, "Ожидается метод GET", http.StatusMethodNotAllowed)
		return
	}

	// Вытаскиваем последние 50 записей из нашей базы
	metrics, err := GetRecentMetrics(dbConn, 50)
	if err != nil {
		http.Error(w, "Ошибка чтения из БД", http.StatusInternalServerError)
		return
	}

	// Важнейшие заголовки!
	// CORS позволяет нашему будущему React-приложению запрашивать эти данные без ошибок безопасности
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Превращаем массив данных в JSON и сразу отправляем в ответ
	json.NewEncoder(w).Encode(metrics)
}
