package handlers

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// ─── WebSocket Hub ────────────────────────────────────────────────────

// WsMessage is the JSON envelope sent to clients.
type WsMessage struct {
	Type  string     `json:"type"`            // "log" | "files"
	Room  string     `json:"room,omitempty"`  // log file name (may include "process/" prefix)
	Lines []string   `json:"lines,omitempty"` // newest-first log lines
	Total int        `json:"total,omitempty"`
	Count int        `json:"count,omitempty"`
	Files []FileInfo `json:"files,omitempty"` // for "files" messages
}

// WsClient represents a connected WebSocket client.
type WsClient struct {
	Send chan []byte
	Room string // log file name the client is watching ("" = file list only)
}

// Hub manages all WebSocket clients and broadcasts updates.
type Hub struct {
	mu      sync.RWMutex
	clients map[*WsClient]bool
	rooms   map[string]map[*WsClient]bool // room → set of clients

	Register   chan *WsClient
	Unregister chan *WsClient
	Broadcast  chan *WsMessage
}

var GlobalHub = &Hub{
	clients:    make(map[*WsClient]bool),
	rooms:      make(map[string]map[*WsClient]bool),
	Register:   make(chan *WsClient, 64),
	Unregister: make(chan *WsClient, 64),
	Broadcast:  make(chan *WsMessage, 256),
}

// Run starts the hub event loop. Call as a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case c := <-h.Register:
			h.mu.Lock()
			h.clients[c] = true
			if c.Room != "" {
				if h.rooms[c.Room] == nil {
					h.rooms[c.Room] = make(map[*WsClient]bool)
				}
				h.rooms[c.Room][c] = true
			}
			h.mu.Unlock()

		case c := <-h.Unregister:
			h.mu.Lock()
			if h.clients[c] {
				delete(h.clients, c)
				if c.Room != "" {
					delete(h.rooms[c.Room], c)
				}
				close(c.Send)
			}
			h.mu.Unlock()

		case msg := <-h.Broadcast:
			data, err := json.Marshal(msg)
			if err != nil {
				continue
			}
			h.mu.RLock()
			if msg.Type == "files" {
				// Broadcast file list to ALL clients
				for c := range h.clients {
					select {
					case c.Send <- data:
					default:
						// slow client — drop
					}
				}
			} else if msg.Room != "" {
				// Broadcast log update to room subscribers only
				for c := range h.rooms[msg.Room] {
					select {
					case c.Send <- data:
					default:
					}
				}
			}
			h.mu.RUnlock()
		}
	}
}

// ─── File Watcher ─────────────────────────────────────────────────────

// WatchLogDir polls the log directory every second and broadcasts
// log updates to room subscribers and file-list updates to all clients.
// Watches both the root log dir and the process/ subdirectory.
func WatchLogDir(dir string) {
	type fileState struct {
		size    int64
		modTime time.Time
	}
	states := make(map[string]fileState)

	for {
		time.Sleep(1 * time.Second)

		var currentFiles []FileInfo

		// ── Scan root dir ──────────────────────────────────────────
		if entries, err := os.ReadDir(dir); err == nil {
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
					continue
				}
				info, err := e.Info()
				if err != nil {
					continue
				}
				currentFiles = append(currentFiles, FileInfo{
					Name: e.Name(),
					Size: info.Size(),
				})
			}
		}

		// ── Scan process/ subdirectory ─────────────────────────────
		processDir := filepath.Join(dir, "process")
		if entries, err := os.ReadDir(processDir); err == nil {
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
					continue
				}
				info, err := e.Info()
				if err != nil {
					continue
				}
				currentFiles = append(currentFiles, FileInfo{
					Name: "process/" + e.Name(),
					Size: info.Size(),
				})
			}
		}

		// ── Check for file list changes ──────────────────────────
		if len(currentFiles) != len(states) {
			// File added/removed — broadcast file list update
			GlobalHub.Broadcast <- &WsMessage{Type: "files", Files: currentFiles}
		}

		// ── Check for file content changes ───────────────────────
		for _, fi := range currentFiles {
			path := filepath.Join(dir, fi.Name) // fi.Name may include "process/" prefix
			info, err := os.Stat(path)
			if err != nil {
				continue
			}

			prev, seen := states[fi.Name]
			if !seen || info.Size() != prev.size || info.ModTime() != prev.modTime {
				states[fi.Name] = fileState{size: info.Size(), modTime: info.ModTime()}
				if !seen {
					continue // first scan — don't broadcast initial state
				}

				// Read tail and broadcast to room
				GlobalHub.mu.RLock()
				hasSubscribers := len(GlobalHub.rooms[fi.Name]) > 0
				GlobalHub.mu.RUnlock()

				if !hasSubscribers {
					continue
				}

				lines, total, err := readLogTail(path, 300)
				if err != nil {
					log.Printf("⚠️ WatchLogDir read %s: %v", fi.Name, err)
					continue
				}

				GlobalHub.Broadcast <- &WsMessage{
					Type:  "log",
					Room:  fi.Name,
					Lines: lines,
					Total: total,
					Count: len(lines),
				}
			}
		}
	}
}
