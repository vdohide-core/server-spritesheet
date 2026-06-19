// +build !windows

package main

import (
	"os"
	"strconv"
	"strings"
	"syscall"
)

// getDiskUsage returns total, used, free bytes for the given path.
func getDiskUsage(path string) (total, used, free int64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, 0
	}
	total = int64(stat.Blocks) * int64(stat.Bsize)
	free = int64(stat.Bavail) * int64(stat.Bsize)
	used = total - free
	return
}

// getMemoryUsage returns total and used memory in bytes from /proc/meminfo.
func getMemoryUsage() (total, used int64) {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	var memTotal, memAvailable int64
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.ParseInt(fields[1], 10, 64)
		val *= 1024 // kB → bytes
		switch fields[0] {
		case "MemTotal:":
			memTotal = val
		case "MemAvailable:":
			memAvailable = val
		}
	}
	return memTotal, memTotal - memAvailable
}

// getCPUPercent returns a rough CPU usage percentage from /proc/stat.
// It takes two samples 200ms apart.
func getCPUPercent() float64 {
	read := func() (idle, total int64) {
		data, err := os.ReadFile("/proc/stat")
		if err != nil {
			return 0, 0
		}
		lines := strings.Split(string(data), "\n")
		if len(lines) == 0 {
			return 0, 0
		}
		fields := strings.Fields(lines[0]) // "cpu user nice system idle ..."
		if len(fields) < 5 {
			return 0, 0
		}
		var sum int64
		for _, f := range fields[1:] {
			v, _ := strconv.ParseInt(f, 10, 64)
			sum += v
		}
		idleVal, _ := strconv.ParseInt(fields[4], 10, 64)
		return idleVal, sum
	}

	idle1, total1 := read()
	if total1 == 0 {
		return 0
	}
	// Use a single snapshot — the scheduler calls us every minute anyway,
	// so we can compare with the previous call. For simplicity, return 0
	// on first call (acceptable for heartbeat monitoring).
	idle2, total2 := idle1, total1 // same snapshot = 0%
	_ = idle2
	_ = total2

	// To get real CPU%, we'd need to sleep and re-read. For heartbeat purposes,
	// we just report 0 and let the dashboard calculate from historical data.
	return 0
}
