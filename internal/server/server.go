// Package server provides the HTTP preview server for mdp.
package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"github.com/donaldgifford/mdp/assets"
	"github.com/donaldgifford/mdp/internal/parser"
)

// Config holds the server configuration.
type Config struct {
	File          string
	Port          int
	OpenBrowser   bool
	Theme         string        // "auto", "light", or "dark".
	ScrollSync    bool          // Enable scroll sync via /cursor endpoint.
	CustomCSS     string        // Path to custom CSS file to inject after default styles.
	OpenToNetwork bool          // Listen on 0.0.0.0 instead of localhost.
	IdleTimeout   time.Duration // Shut down when no clients connected for this long (0 = disabled).
}

// Server is the HTTP preview server.
type Server struct {
	cfg      Config
	addr     string
	token    string // Auth token for network-exposed servers.
	parser   *parser.Parser
	tmpl     *template.Template
	hub      *hub
	sse      *sseHub
	upgrader websocket.Upgrader
	httpSrv  *http.Server
	httpMu   sync.Mutex
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

	var token string
	if cfg.OpenToNetwork {
		token, err = generateToken()
		if err != nil {
			return nil, fmt.Errorf("generating auth token: %w", err)
		}
	}

	return &Server{
		cfg:    cfg,
		addr:   addr,
		token:  token,
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

// generateToken creates a random 16-byte hex token.
func generateToken() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
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

	s.httpMu.Lock()
	srv := s.httpSrv
	s.httpMu.Unlock()

	if srv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			slog.Error("shutting down http server", "error", err)
		}
	}
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
// Returns nil on clean shutdown (e.g. idle timeout or Close()).
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

	var handler http.Handler = mux
	baseURL := "http://" + s.addr

	// When exposed to the network, require a token in the query string.
	if s.token != "" {
		baseURL += "?token=" + s.token
		handler = s.tokenMiddleware(mux)
	}

	//nolint:gosec // Bind address is intentionally configurable.
	httpSrv := &http.Server{Addr: s.addr, Handler: handler}

	s.httpMu.Lock()
	s.httpSrv = httpSrv
	s.httpMu.Unlock()

	slog.Info("serving", "addr", baseURL, "file", s.cfg.File)

	if s.cfg.IdleTimeout > 0 {
		go s.idleWatcher(httpSrv)
	}

	if s.cfg.OpenBrowser {
		go openBrowser(baseURL)
	}

	if err := httpSrv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// idleWatcher shuts down the HTTP server after IdleTimeout elapses with no
// active clients. The poll interval scales with the timeout so tests stay fast.
func (s *Server) idleWatcher(httpSrv *http.Server) {
	pollInterval := s.cfg.IdleTimeout / 6
	if pollInterval < 50*time.Millisecond {
		pollInterval = 50 * time.Millisecond
	}
	if pollInterval > 5*time.Second {
		pollInterval = 5 * time.Second
	}

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var idleSince time.Time

	for range ticker.C {
		if s.hub.count()+s.sse.count() > 0 {
			// Clients are connected — reset the idle clock.
			idleSince = time.Time{}
			continue
		}

		if idleSince.IsZero() {
			idleSince = time.Now()
			slog.Info("no clients connected, idle timer started", "timeout", s.cfg.IdleTimeout)
			continue
		}

		if time.Since(idleSince) >= s.cfg.IdleTimeout {
			slog.Info("idle timeout reached, shutting down")
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := httpSrv.Shutdown(ctx)
			cancel()
			if err != nil {
				slog.Error("idle shutdown", "error", err)
			}
			return
		}
	}
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

// tokenMiddleware checks for a valid token in the query string.
// Vendor assets are exempted so stylesheets and scripts load correctly.
func (s *Server) tokenMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow vendor assets without auth (they're public embedded files).
		if len(r.URL.Path) >= 8 && r.URL.Path[:8] == "/vendor/" {
			next.ServeHTTP(w, r)
			return
		}

		if r.URL.Query().Get("token") != s.token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}
