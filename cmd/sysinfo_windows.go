// +build windows

package main

// getDiskUsage returns total, used, free bytes. Stub for Windows dev.
func getDiskUsage(path string) (total, used, free int64) {
	return 0, 0, 0
}

// getMemoryUsage returns total and used memory in bytes. Stub for Windows dev.
func getMemoryUsage() (total, used int64) {
	return 0, 0
}

// getCPUPercent returns CPU usage percentage. Stub for Windows dev.
func getCPUPercent() float64 {
	return 0
}
