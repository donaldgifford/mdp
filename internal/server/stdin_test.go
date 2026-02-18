package server_test

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/donaldgifford/mdp/internal/server"
)

func TestReadStdin_ContentMessage(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Initial"), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	srv, err := server.New(server.Config{
		File:        mdFile,
		Port:        0,
		OpenBrowser: false,
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

	// Create a pipe to simulate stdin.
	pr, pw := io.Pipe()
	go srv.ReadStdin(pr)

	// Send a content message via the pipe.
	msg := map[string]string{
		"type": "content",
		"data": "# Updated via stdin\n\nNew content here.\n",
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	msgBytes = append(msgBytes, '\n')
	if _, err := pw.Write(msgBytes); err != nil {
		t.Fatalf("writing to pipe: %v", err)
	}

	// Read the broadcast from WebSocket.
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("setting read deadline: %v", err)
	}
	_, wsMsg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("reading websocket: %v", err)
	}

	var envelope struct {
		Type string `json:"type"`
		HTML string `json:"html"`
	}
	if err := json.Unmarshal(wsMsg, &envelope); err != nil {
		t.Fatalf("parsing ws message: %v", err)
	}

	if envelope.Type != "content" {
		t.Errorf("expected type 'content', got %q", envelope.Type)
	}
	if !strings.Contains(envelope.HTML, "Updated via stdin") {
		t.Errorf("expected 'Updated via stdin' in HTML, got: %s", envelope.HTML)
	}

	pw.Close()
}

func TestReadStdin_CursorMessage(t *testing.T) {
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

	// Create a pipe to simulate stdin.
	pr, pw := io.Pipe()
	go srv.ReadStdin(pr)

	// Send a cursor message.
	msg := map[string]any{
		"type": "cursor",
		"line": 10,
	}
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshalling: %v", err)
	}
	msgBytes = append(msgBytes, '\n')
	if _, err := pw.Write(msgBytes); err != nil {
		t.Fatalf("writing to pipe: %v", err)
	}

	// Read the cursor message from WebSocket.
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("setting read deadline: %v", err)
	}
	_, wsMsg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("reading websocket: %v", err)
	}

	var envelope struct {
		Type string `json:"type"`
		Line int    `json:"line"`
	}
	if err := json.Unmarshal(wsMsg, &envelope); err != nil {
		t.Fatalf("parsing ws message: %v", err)
	}

	if envelope.Type != "cursor" {
		t.Errorf("expected type 'cursor', got %q", envelope.Type)
	}
	if envelope.Line != 10 {
		t.Errorf("expected line 10, got %d", envelope.Line)
	}

	pw.Close()
}
