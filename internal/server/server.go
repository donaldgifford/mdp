// Package server provides the HTTP preview server for mdp.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/websocket"

	"github.com/donaldgifford/mdp/assets"
	"github.com/donaldgifford/mdp/internal/parser"
)

// Config holds the server configuration.
type Config struct {
	File          string
	Port          int
	OpenBrowser   bool
	Theme         string // "auto", "light", or "dark".
	ScrollSync    bool   // Enable scroll sync via /cursor endpoint.
	CustomCSS     string // Path to custom CSS file to inject after default styles.
	OpenToNetwork bool   // Listen on 0.0.0.0 instead of localhost.
}

// Server is the HTTP preview server.
type Server struct {
	cfg      Config
	addr     string
	parser   *parser.Parser
	tmpl     *template.Template
	hub      *hub
	sse      *sseHub
	upgrader websocket.Upgrader
}

// New creates a new Server from the given config. The listen address is
// resolved eagerly so callers can read it via Addr() before calling
// ListenAndServe.
func New(cfg Config) (*Server, error) { //nolint:gocritic // Config is intentionally passed by value.
	tmplData, err := assets.FS.ReadFile("preview.html")
	if err != nil {
		return nil, fmt.Errorf("reading template: %w", err)
	}

	tmpl, err := template.New("preview").Parse(string(tmplData))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	addr, err := resolveAddr(cfg.Port, cfg.OpenToNetwork)
	if err != nil {
		return nil, err
	}

	return &Server{
		cfg:    cfg,
		addr:   addr,
		parser: parser.New(),
		tmpl:   tmpl,
		hub:    newHub(),
		sse:    newSSEHub(),
		upgrader: websocket.Upgrader{
			// Allow connections from any origin — this is a local dev tool.
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}, nil
}

// Addr returns the resolved listen address (host:port).
func (s *Server) Addr() string {
	return s.addr
}

// wsMessage is the JSON envelope for WebSocket/SSE messages.
type wsMessage struct {
	Type string `json:"type"`
	HTML string `json:"html,omitempty"`
	Line int    `json:"line,omitempty"`
}

// Broadcast parses the given markdown and sends the rendered HTML to all
// connected WebSocket and SSE clients.
func (s *Server) Broadcast(md []byte) error {
	html, err := s.parser.Render(md)
	if err != nil {
		return fmt.Errorf("rendering markdown: %w", err)
	}
	msg, err := json.Marshal(wsMessage{Type: "content", HTML: string(html)})
	if err != nil {
		return fmt.Errorf("marshalling message: %w", err)
	}
	s.hub.broadcast(msg)
	s.sse.broadcast(msg)
	return nil
}

// SendCursor broadcasts a cursor position update to all connected clients.
func (s *Server) SendCursor(line int) error {
	msg, err := json.Marshal(wsMessage{Type: "cursor", Line: line})
	if err != nil {
		return fmt.Errorf("marshalling cursor message: %w", err)
	}
	s.hub.broadcast(msg)
	s.sse.broadcast(msg)
	return nil
}

// BroadcastFile reads the configured file and broadcasts its rendered content.
func (s *Server) BroadcastFile() error {
	md, err := os.ReadFile(s.cfg.File)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}
	return s.Broadcast(md)
}

// Close shuts down the server, closing all WebSocket and SSE connections.
func (s *Server) Close() {
	s.hub.closeAll()
	s.sse.closeAll()
}

// pageData is the template data for preview.html.
type pageData struct {
	Title     string
	Theme     string
	CSS       template.CSS
	CustomCSS template.CSS
	JS        template.JS
	Body      template.HTML
}

// ListenAndServe starts the HTTP server and blocks until it exits.
func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /ws", s.handleWebSocket)
	mux.HandleFunc("GET /events", s.handleSSE)
	mux.HandleFunc("POST /cursor", s.handleCursor)

	// Serve embedded vendor assets (Mermaid, KaTeX, highlight.js).
	vendorFS, err := fs.Sub(assets.FS, "vendor")
	if err != nil {
		return fmt.Errorf("creating vendor sub-filesystem: %w", err)
	}
	mux.Handle("GET /vendor/", http.StripPrefix("/vendor/", http.FileServer(http.FS(vendorFS))))

	// Serve local files (images, etc.) relative to the markdown file's directory.
	mdDir := filepath.Dir(s.cfg.File)
	mux.Handle("GET /local/", http.StripPrefix("/local/", http.FileServer(http.Dir(mdDir))))

	slog.Info("serving", "addr", "http://"+s.addr, "file", s.cfg.File)

	if s.cfg.OpenBrowser {
		go openBrowser("http://" + s.addr)
	}

	//nolint:gosec // Bind address is intentionally configurable.
	return http.ListenAndServe(s.addr, mux)
}

// resolveAddr returns the listen address, auto-assigning a port if needed.
func resolveAddr(port int, openToNetwork bool) (string, error) {
	host := "localhost"
	if openToNetwork {
		host = "0.0.0.0"
	}

	if port != 0 {
		return fmt.Sprintf("%s:%d", host, port), nil
	}

	// Bind to :0 to get an ephemeral port, then close and reuse it.
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "tcp", host+":0")
	if err != nil {
		return "", fmt.Errorf("finding free port: %w", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		return "", fmt.Errorf("releasing ephemeral port: %w", err)
	}

	return addr, nil
}

func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	md, err := os.ReadFile(s.cfg.File)
	if err != nil {
		http.Error(w, fmt.Sprintf("reading file: %v", err), http.StatusInternalServerError)
		return
	}

	html, err := s.parser.Render(md)
	if err != nil {
		http.Error(w, fmt.Sprintf("rendering markdown: %v", err), http.StatusInternalServerError)
		return
	}

	cssData, err := assets.FS.ReadFile("preview.css")
	if err != nil {
		http.Error(w, fmt.Sprintf("reading css: %v", err), http.StatusInternalServerError)
		return
	}

	jsData, err := assets.FS.ReadFile("preview.js")
	if err != nil {
		http.Error(w, fmt.Sprintf("reading js: %v", err), http.StatusInternalServerError)
		return
	}

	theme := s.cfg.Theme
	if theme == "" {
		theme = "auto"
	}

	var customCSS template.CSS
	if s.cfg.CustomCSS != "" {
		customData, cssErr := os.ReadFile(s.cfg.CustomCSS)
		if cssErr != nil {
			slog.Error("reading custom CSS", "error", cssErr)
		} else {
			customCSS = template.CSS(customData) //nolint:gosec // User-provided CSS file.
		}
	}

	//nolint:gosec // All values are from our own embedded assets and renderer.
	data := pageData{
		Title:     s.cfg.File,
		Theme:     theme,
		CSS:       template.CSS(cssData),
		CustomCSS: customCSS,
		JS:        template.JS(jsData),
		Body:      template.HTML(html),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.tmpl.Execute(w, data); err != nil {
		slog.Error("executing template", "error", err)
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("websocket upgrade failed", "error", err)
		return
	}
	s.hub.add(conn)
	slog.Debug("websocket client connected", "addr", conn.RemoteAddr())

	// Keep the connection open; remove on close.
	defer func() {
		s.hub.remove(conn)
		if closeErr := conn.Close(); closeErr != nil {
			slog.Debug("closing websocket client", "error", closeErr)
		}
	}()

	// Read loop — we don't expect messages from the client, but we need
	// to drain reads so the connection stays alive and close is detected.
	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}

// cursorRequest is the JSON body for POST /cursor.
type cursorRequest struct {
	Line int `json:"line"`
}

func (s *Server) handleCursor(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.ScrollSync {
		http.Error(w, "scroll sync disabled", http.StatusNotFound)
		return
	}

	var req cursorRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if err := s.SendCursor(req.Line); err != nil {
		http.Error(w, "broadcast failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
