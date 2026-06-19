package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// clientMsg is a message sent FROM the client to the server.
type clientMsg struct {
	Type string `json:"type"` // "subscribe"
	Room string `json:"room"` // log file name (may include "process/" prefix)
}

// HandleWS upgrades an HTTP connection to WebSocket and manages the client lifecycle.
func (h *Handler) HandleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("⚠️ WS upgrade: %v", err)
		return
	}

	client := &WsClient{
		Send: make(chan []byte, 256),
	}
	GlobalHub.Register <- client

	// ── Write pump ───────────────────────────────────────────────────
	go func() {
		defer func() {
			GlobalHub.Unregister <- client
			conn.Close()
		}()
		for msg := range client.Send {
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		}
	}()

	// ── Read pump (handle subscribe messages) ─────────────────────────
	defer func() {
		GlobalHub.Unregister <- client
		conn.Close()
	}()

	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Send initial file list on connect
	go func() {
		files, _ := listLogFiles(h.LogDir)
		data, _ := json.Marshal(&WsMessage{Type: "files", Files: files})
		select {
		case client.Send <- data:
		default:
		}
	}()

	for {
		_, raw, err := conn.ReadMessage()
		if err != nil {
			break
		}
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))

		var msg clientMsg
		if err := json.Unmarshal(raw, &msg); err != nil {
			continue
		}

		if msg.Type == "subscribe" && msg.Room != "" {
			// Move client to new room
			GlobalHub.mu.Lock()
			if client.Room != "" {
				delete(GlobalHub.rooms[client.Room], client)
			}
			client.Room = msg.Room
			if GlobalHub.rooms[msg.Room] == nil {
				GlobalHub.rooms[msg.Room] = make(map[*WsClient]bool)
			}
			GlobalHub.rooms[msg.Room][client] = true
			GlobalHub.mu.Unlock()

			// Send current log content immediately
			path := filepath.Join(h.LogDir, msg.Room)
			lines, total, err := readLogTail(path, 300)
			if err == nil {
				data, _ := json.Marshal(&WsMessage{
					Type:  "log",
					Room:  msg.Room,
					Lines: lines,
					Total: total,
					Count: len(lines),
				})
				select {
				case client.Send <- data:
				default:
				}
			}
		}
	}
}
