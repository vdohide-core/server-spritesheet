package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const defaultMaxSize = 25 * 1024 * 1024 // 25 MB

// GlobalWriter is the current log destination.
// ProcessLogger uses this to build a MultiWriter (rotating file + per-process file).
var GlobalWriter io.Writer = os.Stdout

// RotatingWriter is an io.Writer that rotates log files at a given size.
type RotatingWriter struct {
	mu      sync.Mutex
	file    *os.File
	path    string
	size    int64
	maxSize int64
}

// NewRotatingWriter opens (or creates) the log file at path.
// When the file reaches maxSizeBytes it is renamed with a timestamp suffix
// and a fresh file is opened.
func NewRotatingWriter(path string, maxSizeBytes int64) (*RotatingWriter, error) {
	rw := &RotatingWriter{
		path:    path,
		maxSize: maxSizeBytes,
	}
	if err := rw.openOrCreate(); err != nil {
		return nil, err
	}
	return rw, nil
}

func (rw *RotatingWriter) openOrCreate() error {
	f, err := os.OpenFile(rw.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	rw.file = f
	rw.size = info.Size()
	return nil
}

func (rw *RotatingWriter) rotate() error {
	if rw.file != nil {
		rw.file.Close()
		rw.file = nil
	}
	// Rename with timestamp: server-spritesheet_20060102_150405.log
	ts := time.Now().Format("20060102_150405")
	ext := filepath.Ext(rw.path)
	base := rw.path[:len(rw.path)-len(ext)]
	newPath := fmt.Sprintf("%s_%s%s", base, ts, ext)
	_ = os.Rename(rw.path, newPath)
	return rw.openOrCreate()
}

// Write implements io.Writer with automatic size-based rotation.
func (rw *RotatingWriter) Write(p []byte) (int, error) {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.size+int64(len(p)) >= rw.maxSize {
		if err := rw.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := rw.file.Write(p)
	rw.size += int64(n)
	return n, err
}

// Close closes the underlying file.
func (rw *RotatingWriter) Close() error {
	rw.mu.Lock()
	defer rw.mu.Unlock()
	if rw.file != nil {
		return rw.file.Close()
	}
	return nil
}

// rotateOnStartup renames an existing non-empty log file before the first open.
// Each server startup will begin with a fresh, empty log file.
func rotateOnStartup(path string) {
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		return // file doesn't exist or is already empty — nothing to rotate
	}
	ts := time.Now().Format("20060102_150405")
	ext := filepath.Ext(path)
	base := path[:len(path)-len(ext)]
	newPath := fmt.Sprintf("%s_%s%s", base, ts, ext)
	_ = os.Rename(path, newPath)
}

// Init configures the global logger to write to a rotating log file.
// Returns a Closer that should be deferred in main().
//
// Startup rotation: if the log file already exists and is non-empty,
// it is renamed (e.g. server-spritesheet_20060102_150405.log) before a
// fresh file is created for the new run.
//
// Runtime rotation: file is rotated automatically when it exceeds 25 MB.
//
// logPath defaults to "logs/server-spritesheet.log" if empty.
func Init(logPath string) (io.Closer, error) {
	if logPath == "" {
		logPath = "logs/server-spritesheet.log"
	}

	// Ensure log directory exists
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, fmt.Errorf("create log dir: %w", err)
	}

	// Rotate previous run's log file before opening a new one
	rotateOnStartup(logPath)

	rw, err := NewRotatingWriter(logPath, defaultMaxSize)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	// Set global writer so ProcessLogger can use it for MultiWriter
	GlobalWriter = rw

	// Write to file only (no stdout)
	log.SetOutput(rw)
	log.SetFlags(log.LstdFlags)

	return rw, nil
}
