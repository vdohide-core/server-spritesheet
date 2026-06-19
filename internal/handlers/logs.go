package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type logFileInfo struct {
	Name       string    `json:"name"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modifiedAt"`
}

// HandleLogList handles GET /logs
// Returns a list of all .log files in the log directory (newest first).
// Includes both root-level logs and process/ subdirectory logs.
func (h *Handler) HandleLogList(w http.ResponseWriter, r *http.Request) {
	files := make([]logFileInfo, 0)

	// Scan root log dir
	if entries, err := os.ReadDir(h.LogDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			files = append(files, logFileInfo{
				Name:       e.Name(),
				Size:       info.Size(),
				ModifiedAt: info.ModTime().UTC(),
			})
		}
	}

	// Scan process/ subdirectory
	processDir := filepath.Join(h.LogDir, "process")
	if entries, err := os.ReadDir(processDir); err == nil {
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
				continue
			}
			info, err := e.Info()
			if err != nil {
				continue
			}
			files = append(files, logFileInfo{
				Name:       "process/" + e.Name(),
				Size:       info.Size(),
				ModifiedAt: info.ModTime().UTC(),
			})
		}
	}

	// Sort: newest first
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModifiedAt.After(files[j].ModifiedAt)
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"files": files,
		"total": len(files),
	})
}

// HandleLogFile handles GET /logs/{path}
// Supports both root logs (e.g., /logs/server-spritesheet.log)
// and process logs (e.g., /logs/process/abc123.log).
// Query params:
//   - tail=N   → return last N lines (default 200, max 5000)
//   - offset=N → skip first N lines (for forward pagination)
func (h *Handler) HandleLogFile(w http.ResponseWriter, r *http.Request) {
	// Extract relative path from URL
	relPath := strings.TrimPrefix(r.URL.Path, "/logs/")
	relPath = filepath.Clean(relPath)

	if relPath == "" || relPath == "." {
		http.Error(w, `{"error":"filename required"}`, http.StatusBadRequest)
		return
	}
	if strings.Contains(relPath, "..") {
		http.Error(w, `{"error":"access denied"}`, http.StatusForbidden)
		return
	}
	if !strings.HasSuffix(relPath, ".log") {
		http.Error(w, `{"error":"only .log files are accessible"}`, http.StatusForbidden)
		return
	}

	fullPath := filepath.Join(h.LogDir, relPath)

	// Security: ensure resolved path is within logDir
	absLog, _ := filepath.Abs(h.LogDir)
	absFile, _ := filepath.Abs(fullPath)
	if !strings.HasPrefix(absFile, absLog+string(os.PathSeparator)) && absFile != absLog {
		http.Error(w, `{"error":"access denied"}`, http.StatusForbidden)
		return
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, `{"error":"file not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error":"cannot read file"}`, http.StatusInternalServerError)
		}
		return
	}

	// Split into lines
	allLines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	// Remove trailing empty line
	if len(allLines) > 0 && allLines[len(allLines)-1] == "" {
		allLines = allLines[:len(allLines)-1]
	}

	// Parse query params
	tailN := 200
	if t := r.URL.Query().Get("tail"); t != "" {
		if n, err := strconv.Atoi(t); err == nil && n > 0 {
			if n > 5000 {
				n = 5000
			}
			tailN = n
		}
	}
	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if n, err := strconv.Atoi(o); err == nil && n >= 0 {
			offset = n
		}
	}

	total := len(allLines)

	// Apply offset first
	lines := allLines
	if offset > 0 && offset < total {
		lines = lines[offset:]
	} else if offset >= total {
		lines = []string{}
	}

	// Apply tail (last N lines)
	if len(lines) > tailN {
		lines = lines[len(lines)-tailN:]
	}

	// Reverse: newest line first
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"filename": relPath,
		"total":    total,
		"count":    len(lines),
		"offset":   offset,
		"tail":     tailN,
		"lines":    lines,
	})
}
