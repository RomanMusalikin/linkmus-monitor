package agent

import (
	"fmt"
	"log"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
)

func Run() {
	log.Println("Агент запущен и готов к работе...")

	ticker := time.NewTicker(10 * time.Second) // 10 секунд — золотой стандарт
	defer ticker.Stop()

	for t := range ticker.C {
		fmt.Printf("\n--- [Срез метрик: %s] ---\n", t.Format("15:04:05"))
		collectAndSend()
	}
}

func collectAndSend() {
	// 1. CPU
	cpuPercent, _ := cpu.Percent(time.Second, false)

	// 2. RAM (Оперативка)
	vMem, _ := mem.VirtualMemory()

	// 3. Disk (Место на диске C: или /)
	// В Windows используем "C:", в Linux обычно "/"
	// Библиотека на Windows умная, поймет и "/"
	dUsage, _ := disk.Usage("/")

	// Выводим результат красиво
	if len(cpuPercent) > 0 {
		fmt.Printf("💻 CPU: %.2f%%\n", cpuPercent[0])
	}
	fmt.Printf("🧠 RAM: %.2f%% (Использовано: %v MB / Всего: %v MB)\n",
		vMem.UsedPercent, vMem.Used/1024/1024, vMem.Total/1024/1024)

	fmt.Printf("💾 Disk: %.2f%% (Свободно: %v GB)\n",
		dUsage.UsedPercent, dUsage.Free/1024/1024/1024)

	log.Println("\n[!] Подготовка JSON для отправки на сервер...")
}
