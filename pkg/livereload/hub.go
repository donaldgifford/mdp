package livereload

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

// sendBuffer is the per-connection outbound buffer. Matches the SSE
// buffer used historically in mdp; revisit with a WithSendBuffer option
// if profiling shows drops.
const sendBuffer = 8

// Hub fans out byte payloads to all connected clients (WebSocket and
// SSE). Safe for concurrent use.
//
// Each WebSocket connection has a dedicated writer goroutine, so
// Broadcast can be called from any goroutine without violating
// gorilla/websocket's single-writer requirement.
//
// Hub is intended for local-only use: HandleWebSocket accepts any
// origin. A future WithCheckOrigin option can be added if a non-local
// consumer materializes.
type Hub struct {
	mu         sync.RWMutex
	wsClients  map[*wsClient]struct{}
	sseClients map[chan []byte]struct{}
	closed     bool
	upgrader   websocket.Upgrader
}

// wsClient pairs a gorilla WebSocket connection with its outbound queue.
// A single writer goroutine pulls from send and calls WriteMessage,
// closing the gorilla concurrent-write gap.
type wsClient struct {
	conn *websocket.Conn
	send chan []byte
}

// NewHub returns an empty hub with no connected clients.
func NewHub() *Hub {
	return &Hub{
		wsClients:  make(map[*wsClient]struct{}),
		sseClients: make(map[chan []byte]struct{}),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}
}

// Broadcast sends msg to every connected WebSocket and SSE client.
// Sends are non-blocking — a client whose outbound queue is full is
// dropped so fast clients aren't held up by slow ones.
func (h *Hub) Broadcast(msg []byte) {
	h.mu.RLock()
	var dropWS []*wsClient
	for c := range h.wsClients {
		select {
		case c.send <- msg:
		default:
			dropWS = append(dropWS, c)
		}
	}
	var dropSSE []chan []byte
	for ch := range h.sseClients {
		select {
		case ch <- msg:
		default:
			dropSSE = append(dropSSE, ch)
		}
	}
	h.mu.RUnlock()

	for _, c := range dropWS {
		slog.Debug("livereload: dropping slow WS client")
		h.removeWS(c)
	}
	for _, ch := range dropSSE {
		// coverage: triggered only when the SSE consumer goroutine
		// stalls under load (an HTTP/2 client that stops reading on
		// the open response). The unit suite uses cooperative clients
		// so this branch is exercised in production only.
		slog.Debug("livereload: dropping slow SSE client")
		h.removeSSE(ch)
	}
}

// Count returns the number of currently connected clients across both
// transports.
func (h *Hub) Count() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.wsClients) + len(h.sseClients)
}

// Close drops all connections and prevents new ones from being added.
// Returns the first per-connection close error, if any. Subsequent
// errors are logged at debug.
func (h *Hub) Close() error {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return nil
	}
	h.closed = true

	wsClients := h.wsClients
	sseClients := h.sseClients
	h.wsClients = make(map[*wsClient]struct{})
	h.sseClients = make(map[chan []byte]struct{})
	h.mu.Unlock()

	var firstErr error
	for c := range wsClients {
		close(c.send)
		if err := c.conn.Close(); err != nil {
			// coverage: gorilla *Conn.Close only fails on already-closed
			// sockets, which the writer goroutine takes care of before
			// we get here in normal flows.
			if firstErr == nil {
				firstErr = fmt.Errorf("closing websocket: %w", err)
			} else {
				slog.Debug("livereload: closing websocket", "error", err)
			}
		}
	}
	for ch := range sseClients {
		close(ch)
	}

	return firstErr
}

// HandleWebSocket upgrades the HTTP request to a WebSocket and registers
// the connection with the hub. Wire this to an http.ServeMux at the
// consumer's chosen path.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// coverage: gorilla logs+writes its own 400 response before
		// returning the error, so the only thing left to do is log
		// at error level. Reaching this requires a malformed upgrade
		// request that the test harness doesn't produce.
		slog.Error("livereload: websocket upgrade failed", "error", err)
		return
	}

	client := &wsClient{
		conn: conn,
		send: make(chan []byte, sendBuffer),
	}

	if !h.addWS(client) {
		// Hub closed mid-upgrade. Drop the connection immediately.
		if err := conn.Close(); err != nil {
			slog.Debug("livereload: closing websocket after closed hub", "error", err)
		}
		return
	}

	slog.Debug("livereload: websocket client connected", "addr", conn.RemoteAddr())

	go h.wsWriter(client)
	h.wsReader(client)
}

// HandleSSE streams hub broadcasts over Server-Sent Events. Wire this to
// an http.ServeMux at the consumer's chosen path.
func (h *Hub) HandleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		// coverage: net/http's response writer always implements
		// http.Flusher in real servers; this branch only fires
		// against a hand-rolled non-flushing ResponseWriter.
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)
	flusher.Flush()

	ch := make(chan []byte, sendBuffer)
	if !h.addSSE(ch) {
		return
	}
	defer h.removeSSE(ch)

	//nolint:gosec // G706: r.RemoteAddr is a structured slog field, not a format string.
	slog.Debug("livereload: SSE client connected", "addr", r.RemoteAddr)

	for {
		select {
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", msg); err != nil {
				// coverage: SSE write failure means the client
				// disconnected mid-stream; the outer return cleans up.
				return
			}
			flusher.Flush()
		case <-r.Context().Done():
			return
		}
	}
}

// addWS registers a WS client. Returns false if the hub is closed.
func (h *Hub) addWS(c *wsClient) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return false
	}
	h.wsClients[c] = struct{}{}
	return true
}

// removeWS unregisters a WS client. Safe to call multiple times.
func (h *Hub) removeWS(c *wsClient) {
	h.mu.Lock()
	_, present := h.wsClients[c]
	if present {
		delete(h.wsClients, c)
		close(c.send)
	}
	h.mu.Unlock()
	if present {
		if err := c.conn.Close(); err != nil {
			// coverage: see Close — gorilla conn.Close only errors on
			// already-half-closed sockets.
			slog.Debug("livereload: closing websocket", "error", err)
		}
	}
}

// addSSE registers an SSE client. Returns false if the hub is closed.
func (h *Hub) addSSE(ch chan []byte) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return false
	}
	h.sseClients[ch] = struct{}{}
	return true
}

// removeSSE unregisters an SSE client. Safe to call multiple times.
func (h *Hub) removeSSE(ch chan []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.sseClients[ch]; !ok {
		// coverage: idempotent guard for the rare race between
		// Broadcast's drop path and HandleSSE's defer cleanup.
		return
	}
	delete(h.sseClients, ch)
	close(ch)
}

// wsWriter pulls messages off the client's send channel and writes them
// to the WebSocket. It exits when the channel is closed (which happens
// on remove or hub Close).
func (h *Hub) wsWriter(c *wsClient) {
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			slog.Debug("livereload: write to client failed", "error", err)
			h.removeWS(c)
			// Drain any pending messages so the goroutine exits cleanly
			// after removeWS closes c.send.
			for range c.send { //nolint:revive // empty body: drain channel.
			}
			return
		}
	}
}

// wsReader runs the WebSocket read loop, draining pings/close frames so
// the connection stays alive and disconnect is detected promptly.
func (h *Hub) wsReader(c *wsClient) {
	defer h.removeWS(c)
	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			var closeErr *websocket.CloseError
			if !errors.As(err, &closeErr) {
				slog.Debug("livereload: websocket read error", "error", err)
			}
			return
		}
	}
}
