//go:build windows

package collector

// CollectCPUTemp на Windows возвращает 0 — датчики в VM обычно недоступны.
func CollectCPUTemp() float64 {
	return 0
}
