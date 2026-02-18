package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/donaldgifford/mdp/internal/server"
)

func TestScrollSync_CursorEndpointBroadcasts(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	content := []byte("# Heading\n\nParagraph one.\n\n## Subheading\n\nParagraph two.\n")
	if err := os.WriteFile(mdFile, content, 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	srv, err := server.New(server.Config{
		File:        mdFile,
		Port:        0,
		OpenBrowser: false,
		ScrollSync:  true,
	})
	if err != nil {
		t.Fatalf("creating server: %v", err)
	}
	defer srv.Close()

	addr := srv.Addr()
	go func() {
		_ = srv.ListenAndServe()
	}()

	waitForServer(t, "http://"+addr)

	// Connect WebSocket client.
	wsURL := "ws://" + addr + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("connecting websocket: %v", err)
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	defer conn.Close()

	// POST a cursor position.
	body, err := json.Marshal(map[string]int{"line": 5})
	if err != nil {
		t.Fatalf("marshalling cursor: %v", err)
	}

	cursorResp, err := http.Post( //nolint:noctx // Test code.
		"http://"+addr+"/cursor",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("POST /cursor: %v", err)
	}
	cursorResp.Body.Close()

	if cursorResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", cursorResp.StatusCode)
	}

	// Read the cursor message from WebSocket.
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("setting read deadline: %v", err)
	}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("reading websocket message: %v", err)
	}

	var envelope struct {
		Type string `json:"type"`
		Line int    `json:"line"`
	}
	if err := json.Unmarshal(msg, &envelope); err != nil {
		t.Fatalf("parsing JSON message: %v", err)
	}

	if envelope.Type != "cursor" {
		t.Errorf("expected type 'cursor', got %q", envelope.Type)
	}
	if envelope.Line != 5 {
		t.Errorf("expected line 5, got %d", envelope.Line)
	}
}

func TestScrollSync_DisabledReturns404(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Test"), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	srv, err := server.New(server.Config{
		File:        mdFile,
		Port:        0,
		OpenBrowser: false,
		ScrollSync:  false,
	})
	if err != nil {
		t.Fatalf("creating server: %v", err)
	}
	defer srv.Close()

	addr := srv.Addr()
	go func() {
		_ = srv.ListenAndServe()
	}()

	waitForServer(t, "http://"+addr)

	body, err := json.Marshal(map[string]int{"line": 1})
	if err != nil {
		t.Fatalf("marshalling cursor: %v", err)
	}

	resp, err := http.Post( //nolint:noctx // Test code.
		"http://"+addr+"/cursor",
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		t.Fatalf("POST /cursor: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 when scroll sync disabled, got %d", resp.StatusCode)
	}
}

func TestScrollSync_SendCursorMethod(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Test"), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	srv, err := server.New(server.Config{
		File:        mdFile,
		Port:        0,
		OpenBrowser: false,
		ScrollSync:  true,
	})
	if err != nil {
		t.Fatalf("creating server: %v", err)
	}
	defer srv.Close()

	addr := srv.Addr()
	go func() {
		_ = srv.ListenAndServe()
	}()

	waitForServer(t, "http://"+addr)

	// Connect WebSocket client.
	wsURL := "ws://" + addr + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("connecting websocket: %v", err)
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}
	defer conn.Close()

	// Use the SendCursor method directly.
	if err := srv.SendCursor(42); err != nil {
		t.Fatalf("SendCursor: %v", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("setting read deadline: %v", err)
	}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("reading websocket message: %v", err)
	}

	var envelope struct {
		Type string `json:"type"`
		Line int    `json:"line"`
	}
	if err := json.Unmarshal(msg, &envelope); err != nil {
		t.Fatalf("parsing JSON message: %v", err)
	}

	if envelope.Type != "cursor" {
		t.Errorf("expected type 'cursor', got %q", envelope.Type)
	}
	if envelope.Line != 42 {
		t.Errorf("expected line 42, got %d", envelope.Line)
	}
}
