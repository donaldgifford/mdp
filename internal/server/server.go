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
	cfg    Config
	parser *parser.Parser
	tmpl   *template.Template
}

// New creates a new Server from the given config.
func New(cfg Config) (*Server, error) {
	tmplData, err := assets.FS.ReadFile("preview.html")
	if err != nil {
		return nil, fmt.Errorf("reading template: %w", err)
	}

	tmpl, err := template.New("preview").Parse(string(tmplData))
	if err != nil {
		return nil, fmt.Errorf("parsing template: %w", err)
	}

	return &Server{
		cfg:    cfg,
		parser: parser.New(),
		tmpl:   tmpl,
	}, nil
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

	addr, err := s.resolve()
	if err != nil {
		return err
	}

	slog.Info("serving", "addr", "http://"+addr, "file", s.cfg.File)

	if s.cfg.OpenBrowser {
		go openBrowser("http://" + addr)
	}

	//nolint:gosec // Bind address is intentionally configurable.
	return http.ListenAndServe(addr, mux)
}

// resolve returns the listen address, auto-assigning a port if needed.
func (s *Server) resolve() (string, error) {
	if s.cfg.Port != 0 {
		return fmt.Sprintf("localhost:%d", s.cfg.Port), nil
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
