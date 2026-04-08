// cmd/server/main.go
package main

import (
	"linkmus-monitor/internal/server"
	"log"
)

func main() {
	log.Println("Запуск Linkmus Monitor Server...")
	server.Run()
}
