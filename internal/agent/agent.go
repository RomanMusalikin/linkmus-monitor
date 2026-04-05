// internal/agent/agent.go
package agent

import (
	"fmt"
	"log"
	"time"
)

// Run — главная функция агента. Она блокирует выполнение и крутится в цикле.
func Run() {
	log.Println("Агент инициализирован. Начинаем работу...")

	// Создаем тикер. Пока зашьем 5 секунд для удобства тестирования.
	// Позже вынесем это в config.yaml
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Бесконечный цикл, который ждет "тиков" от таймера
	for t := range ticker.C {
		fmt.Printf("\n[%s] Время собирать метрики!\n", t.Format("15:04:05"))
		collectAndSend()
	}
}

// collectAndSend пока просто имитирует бурную деятельность
func collectAndSend() {
	// В будущем здесь мы вызовем сборщики CPU, RAM и т.д.
	log.Println(" -> [Сбор] Сбор метрик ОС (заглушка)")

	// А затем упакуем их в JSON и отправим POST-запросом
	log.Println(" -> [Отправка] POST http://10.10.10.10:8080/api/metrics (заглушка)")
}
