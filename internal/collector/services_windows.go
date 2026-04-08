//go:build windows

package collector

import (
	"log"

	"github.com/yusufpapurcu/wmi"
)

type Win32_Service struct {
	Name  string
	State string
}

// CollectServices опрашивает WMI и возвращает статусы RDP и SMB
func CollectServices() (bool, bool) {
	var dst []Win32_Service
	rdpRunning := false
	smbRunning := false

	// Запрос к WMI для поиска нужных служб
	query := "SELECT Name, State FROM Win32_Service WHERE Name='TermService' OR Name='LanmanServer'"

	err := wmi.Query(query, &dst)
	if err != nil {
		log.Printf("⚠️ Ошибка опроса WMI: %v", err)
		return false, false
	}

	// Анализируем ответ от Windows
	for _, svc := range dst {
		isRunning := svc.State == "Running"
		if svc.Name == "TermService" {
			rdpRunning = isRunning
		} else if svc.Name == "LanmanServer" {
			smbRunning = isRunning
		}
	}

	return rdpRunning, smbRunning
}
