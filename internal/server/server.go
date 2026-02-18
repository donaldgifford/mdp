// Package server provides the HTTP preview server for mdp.
package server

import (
	"context"
	"fmt"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"os"

	"github.com/gorilla/websocket"

	"github.com/donaldgifford/mdp/assets"
	"github.com/donaldgifford/mdp/internal/parser"
)

// Config holds the server configuration.
type Config struct {
	File        string
	Port        int
	OpenBrowser bool
}

// Server is the HTTP preview server.
type Server struct {
	cfg      Config
	addr     string
	parser   *parser.Parser
	tmpl     *template.Template
	hub      *hub
	upgrader websocket.Upgrader
}

// New creates a new Server from the given config. The listen address is
// resolved eagerly so callers can read it via Addr() before calling
// ListenAndServe.
func New(cfg Config) (*Server, error) {
	tmplData, err := assets.FS.ReadFile("preview.html")
	if err != nil {
		return nil, fmt.Errorf("reading template: %w", err)
	}

	tmpl, err := template.New("preview").Parse(string(tmplData))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	addr, err := resolveAddr(cfg.Port)
	if err != nil {
		return nil, err
	}

	return &Server{
		cfg:    cfg,
		addr:   addr,
		parser: parser.New(),
		tmpl:   tmpl,
		hub:    newHub(),
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

// Broadcast parses the given markdown and sends the rendered HTML to all
// connected WebSocket clients.
func (s *Server) Broadcast(md []byte) error {
	html, err := s.parser.Render(md)
	if err != nil {
		return fmt.Errorf("rendering markdown: %w", err)
	}
	s.hub.broadcast(html)
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

// Close shuts down the server, closing all WebSocket connections.
func (s *Server) Close() {
	s.hub.closeAll()
}

// pageData is the template data for preview.html.
type pageData struct {
	Title string
	CSS   template.CSS
	JS    template.JS
	Body  template.HTML
}

// ListenAndServe starts the HTTP server and blocks until it exits.
func (s *Server) ListenAndServe() error {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.HandleFunc("GET /ws", s.handleWebSocket)

	slog.Info("serving", "addr", "http://"+s.addr, "file", s.cfg.File)

	if s.cfg.OpenBrowser {
		go openBrowser("http://" + s.addr)
	}

	//nolint:gosec // Bind address is intentionally configurable.
	return http.ListenAndServe(s.addr, mux)
}

// resolveAddr returns the listen address, auto-assigning a port if needed.
func resolveAddr(port int) (string, error) {
	if port != 0 {
		return fmt.Sprintf("localhost:%d", port), nil
	}

	// Bind to :0 to get an ephemeral port, then close and reuse it.
	var lc net.ListenConfig
	ln, err := lc.Listen(context.Background(), "tcp", "localhost:0")
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

	//nolint:gosec // All values are from our own embedded assets and renderer.
	data := pageData{
		Title: s.cfg.File,
		CSS:   template.CSS(cssData),
		JS:    template.JS(jsData),
		Body:  template.HTML(html),
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
