package main

import (
	"context"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"server-spritesheet/internal/db/models"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ─── Heartbeat ────────────────────────────────────────────────

// startHeartbeat sends a heartbeat to the workers collection every 1 minute.
// It upserts by workerId (type_hostname@n) and reports status, active jobs, and system metrics.
func startHeartbeat(wID string) {
	log.Printf("💓 Starting heartbeat (every 1 min, workerId=%s)", wID)

	workerType, hostname, pid := parseWorkerID(wID)
	ip := getOutboundIP()

	doHeartbeat := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Count active jobs for this worker
		activeJobs, _ := models.VideoProcessModel.Col().CountDocuments(ctx, bson.M{
			"workerId":    wID,
			"status":      models.ProcessStatusProcessing,
			"processType": "spritesheet",
		})

		status := "idle"
		if activeJobs > 0 {
			status = "busy"
		}

		// Gather system metrics
		sys := gatherSystemInfo()

		now := time.Now()
		filter := bson.M{"workerId": wID}
		update := bson.M{
			"$set": bson.M{
				"hostname":    hostname,
				"ip":          ip,
				"pid":         pid,
				"type":        workerType,
				"status":      status,
				"activeJobs":  activeJobs,
				"maxJobs":     1, // each worker handles 1 job at a time
				"system":      sys,
				"heartbeatAt": now,
				"updatedAt":   now,
			},
			"$setOnInsert": bson.M{
				"_id":       uuid.New().String(),
				"enable":    true,
				"createdAt": now,
			},
		}

		opts := options.Update().SetUpsert(true)
		if _, err := models.WorkerModel.Col().UpdateOne(ctx, filter, update, opts); err != nil {
			log.Printf("⚠️ Heartbeat failed: %v", err)
		}
	}

	// First heartbeat immediately
	doHeartbeat()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		doHeartbeat()
	}
}

// parseWorkerID splits "type_hostname@n" into worker type, hostname, and pid.
// Supports legacy "hostname@n" (type defaults to "spritesheet").
func parseWorkerID(wID string) (workerType, hostname string, pid int) {
	parts := strings.SplitN(wID, "@", 2)
	prefix := parts[0]

	workerType = "spritesheet"
	if idx := strings.Index(prefix, "_"); idx >= 0 {
		workerType = prefix[:idx]
		hostname = prefix[idx+1:]
	} else {
		hostname = prefix
	}

	pid = os.Getpid()
	return
}

// getOutboundIP returns the preferred outbound IP of this machine.
func getOutboundIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		// Fallback: scan interfaces
		addrs, err := net.InterfaceAddrs()
		if err != nil {
			return "unknown"
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
		return "unknown"
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

// ─── System Metrics ───────────────────────────────────────────

type systemInfo struct {
	DiskTotal  int64   `bson:"diskTotal,omitempty"`
	DiskUsed   int64   `bson:"diskUsed,omitempty"`
	DiskFree   int64   `bson:"diskFree,omitempty"`
	MemTotal   int64   `bson:"memTotal,omitempty"`
	MemUsed    int64   `bson:"memUsed,omitempty"`
	CPUPercent float64 `bson:"cpuPercent,omitempty"`
}

// gatherSystemInfo collects disk, memory, and CPU metrics.
// Uses /proc on Linux; returns zero values on unsupported platforms.
func gatherSystemInfo() *systemInfo {
	info := &systemInfo{}

	// Disk: use syscall.Statfs on the storage path
	info.DiskTotal, info.DiskUsed, info.DiskFree = getDiskUsage("/")

	// Memory: parse /proc/meminfo
	info.MemTotal, info.MemUsed = getMemoryUsage()

	// CPU: parse /proc/stat (simplified — single snapshot)
	info.CPUPercent = getCPUPercent()

	return info
}
