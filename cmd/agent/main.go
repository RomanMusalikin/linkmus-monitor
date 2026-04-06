package main

import (
	"linkmus-monitor/internal/agent"
	"log"
)

func main() {
	log.Println("Запуск Linkmus Monitor Agent...")
	agent.Run()
}
