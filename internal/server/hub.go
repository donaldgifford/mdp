package server

import (
	"log/slog"
	"sync"

	"github.com/gorilla/websocket"
)

// hub manages WebSocket connections and broadcasts content to all clients.
type hub struct {
	mu      sync.RWMutex
	clients map[*websocket.Conn]struct{}
}

func newHub() *hub {
	return &hub{
		clients: make(map[*websocket.Conn]struct{}),
	}
}

// add registers a new WebSocket connection.
func (h *hub) add(conn *websocket.Conn) {
	h.mu.Lock()
	h.clients[conn] = struct{}{}
	h.mu.Unlock()
}

// remove unregisters a WebSocket connection.
func (h *hub) remove(conn *websocket.Conn) {
	h.mu.Lock()
	delete(h.clients, conn)
	h.mu.Unlock()
}

// broadcast sends a message to all connected clients.
func (h *hub) broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for conn := range h.clients {
		if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			slog.Debug("write to client failed", "error", err)
		}
	}
}

// closeAll closes all WebSocket connections.
func (h *hub) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for conn := range h.clients {
		if err := conn.Close(); err != nil {
			slog.Debug("closing websocket client", "error", err)
		}
		delete(h.clients, conn)
	}
}
