package main

import (
	"linkmus-monitor/internal/agent"
	"log"
)

func main() {
	log.Println("Запуск Linkmus Monitor Agent...")
	// Вызываем функцию Run из твоего файла internal/agent/agent.go
	agent.Run()
}
