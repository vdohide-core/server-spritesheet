package handlers

import (
	"os"
	"path/filepath"
	"strings"
)

// FileInfo is a lightweight log-file descriptor sent over WebSocket.
type FileInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

// Handler holds dependencies for HTTP handlers
type Handler struct {
	LogDir string // directory containing .log files
}

// NewHandler creates a new Handler instance
func NewHandler(h Handler) *Handler {
	return &h
}

// ─── Shared Helpers ───────────────────────────────────────────────────

// readLogTail reads the last n lines of a log file.
// Returns lines in newest-first order, total line count, and any error.
func readLogTail(path string, n int) ([]string, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, err
	}
	allLines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(allLines) > 0 && allLines[len(allLines)-1] == "" {
		allLines = allLines[:len(allLines)-1]
	}
	total := len(allLines)
	lines := allLines
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	// Reverse: newest first
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	return lines, total, nil
}

// listLogFiles returns all .log files in dir and its process/ subdirectory.
// Process logs are returned with "process/" prefix in their name.
func listLogFiles(dir string) ([]FileInfo, error) {
	var files []FileInfo

	// Root level .log files
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		info, _ := e.Info()
		size := int64(0)
		if info != nil {
			size = info.Size()
		}
		files = append(files, FileInfo{Name: e.Name(), Size: size})
	}

	// Process subdirectory .log files
	processDir := filepath.Join(dir, "process")
	if pEntries, err := os.ReadDir(processDir); err == nil {
		for _, e := range pEntries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
				continue
			}
			info, _ := e.Info()
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			files = append(files, FileInfo{
				Name: "process/" + e.Name(),
				Size: size,
			})
		}
	}

	if len(files) == 0 {
		return []FileInfo{}, nil
	}
	return files, nil
}
