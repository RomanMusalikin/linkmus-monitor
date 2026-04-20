//go:build linux

package collector

import (
	"strings"

	"github.com/shirou/gopsutil/v3/host"
)

// CollectCPUTemp возвращает температуру CPU в градусах Цельсия (0 если недоступно).
func CollectCPUTemp() float64 {
	temps, err := host.SensorsTemperatures()
	if err != nil {
		return 0
	}
	for _, t := range temps {
		key := strings.ToLower(t.SensorKey)
		if t.Temperature <= 0 || t.Temperature > 150 {
			continue
		}
		if strings.Contains(key, "cpu") || strings.Contains(key, "core") ||
			strings.Contains(key, "package") || strings.Contains(key, "thermal_zone") {
			return t.Temperature
		}
	}
	return 0
}
