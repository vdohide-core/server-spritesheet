package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"server-spritesheet/internal/logger"
)

// GenerateWorkerID returns WORKER_ID env or hostname@1.
func GenerateWorkerID() string {
	if envWorkerID := os.Getenv("WORKER_ID"); envWorkerID != "" {
		return envWorkerID
	}
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return fmt.Sprintf("%s@1", hostname)
}

// RandomString generates a random alphanumeric string.
func RandomString(n int, special bool) string {
	if special {
		return RandomStringSpecial(n)
	}
	return RandomAlphaNum(n)
}

// ProcessLogger writes to a per-process log file during job execution.
type ProcessLogger struct {
	file *os.File
}

// NewProcessLogger redirects global log output to logs/process/{slug}.log.
func NewProcessLogger(slug string) *ProcessLogger {
	logDir := filepath.Join("logs", "process")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Failed to create log dir: %v", err)
		return &ProcessLogger{}
	}

	logPath := filepath.Join(logDir, fmt.Sprintf("%s.log", slug))
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open process log: %v", err)
		return &ProcessLogger{}
	}

	log.SetOutput(f)
	return &ProcessLogger{file: f}
}

// Close restores the global logger and closes the per-process file.
func (pl *ProcessLogger) Close() {
	log.SetOutput(logger.GlobalWriter)
	if pl.file != nil {
		pl.file.Close()
	}
}

// Printf writes to the current process log.
func (pl *ProcessLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// LogMain writes a milestone to both the process log and main rotating log.
func LogMain(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("%s", msg)
	fmt.Fprintf(logger.GlobalWriter, "%s %s\n", time.Now().Format("2006/01/02 15:04:05"), msg)
}

// CleanOldLogs removes process log files older than 7 days.
func CleanOldLogs() {
	logDir := filepath.Join("logs", "process")
	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		return
	}

	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			os.Remove(filepath.Join(logDir, entry.Name()))
			removed++
		}
	}

	if removed > 0 {
		log.Printf("Removed %d old log files", removed)
	}
}

// ProcessingLock serializes heavy operations by name.
type ProcessingLock struct {
	mu   *sync.Mutex
	name string
}

var (
	locksMu sync.Mutex
	locks   = map[string]*sync.Mutex{}
)

// AcquireProcessingLock acquires a named mutex lock.
func AcquireProcessingLock(name string) *ProcessingLock {
	locksMu.Lock()
	mu, ok := locks[name]
	if !ok {
		mu = &sync.Mutex{}
		locks[name] = mu
	}
	locksMu.Unlock()

	mu.Lock()
	return &ProcessingLock{mu: mu, name: name}
}

// Release releases the processing lock.
func (l *ProcessingLock) Release() {
	if l.mu != nil {
		l.mu.Unlock()
	}
}
