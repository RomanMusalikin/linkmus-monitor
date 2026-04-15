//go:build linux

package collector

// CollectServices на Linux всегда возвращает false/false —
// RDP и SMB являются Windows-сервисами.
func CollectServices() (bool, bool) {
	return false, false
}
