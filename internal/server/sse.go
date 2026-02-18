package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"sync"
)

// sseHub manages Server-Sent Events connections as a WebSocket fallback.
type sseHub struct {
	mu      sync.RWMutex
	clients map[chan []byte]struct{}
}

func newSSEHub() *sseHub {
	return &sseHub{
		clients: make(map[chan []byte]struct{}),
	}
}

func (h *sseHub) add(ch chan []byte) {
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
}

func (h *sseHub) remove(ch chan []byte) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
}

func (h *sseHub) broadcast(msg []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for ch := range h.clients {
		select {
		case ch <- msg:
		default:
			// Skip slow clients.
			slog.Debug("dropping SSE message for slow client")
		}
	}
}

func (h *sseHub) closeAll() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for ch := range h.clients {
		close(ch)
		delete(h.clients, ch)
	}
}

// handleSSE streams server-sent events to the client.
func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := make(chan []byte, 8)
	s.sse.add(ch)
	defer s.sse.remove(ch)

	slog.Debug("SSE client connected", "addr", r.RemoteAddr)

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", msg); err != nil {
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}
