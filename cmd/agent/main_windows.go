//go:build windows

package main

import (
	"linkmus-monitor/internal/agent"
	"log"

	"golang.org/x/sys/windows/svc"
)

type monService struct{}

func (m *monService) Execute(args []string, r <-chan svc.ChangeRequest, s chan<- svc.Status) (bool, uint32) {
	s <- svc.Status{State: svc.Running, Accepts: svc.AcceptStop | svc.AcceptShutdown}

	go agent.Run()

	for c := range r {
		switch c.Cmd {
		case svc.Stop, svc.Shutdown:
			s <- svc.Status{State: svc.StopPending}
			return false, 0
		}
	}
	return false, 0
}

func main() {
	isService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("Ошибка определения режима: %v", err)
	}

	if isService {
		if err := svc.Run("MonAgent", &monService{}); err != nil {
			log.Fatalf("Ошибка службы: %v", err)
		}
		return
	}

	// Запуск вручную из консоли
	log.Println("Запуск агента в консольном режиме...")
	agent.Run()
}
