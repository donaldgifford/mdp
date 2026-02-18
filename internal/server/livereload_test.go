package server_test

import (
	"bufio"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/donaldgifford/mdp/internal/server"
)

func TestLiveReload_WebSocketReceivesUpdates(t *testing.T) {
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

	// Broadcast updated content.
	if err := srv.Broadcast([]byte("# Updated Content")); err != nil {
		t.Fatalf("broadcast: %v", err)
	}

	// Read the message from WebSocket.
	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("setting read deadline: %v", err)
	}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("reading websocket message: %v", err)
	}

	got := string(msg)
	if !strings.Contains(got, "Updated Content") {
		t.Errorf("expected 'Updated Content' in message, got: %s", got)
	}
	if !strings.Contains(got, "<h1") {
		t.Errorf("expected rendered HTML in message, got: %s", got)
	}
}

func TestLiveReload_SSEEndpoint(t *testing.T) {
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

	// Connect SSE client with a short timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://"+addr+"/events", http.NoBody)
	if err != nil {
		t.Fatalf("creating request: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("connecting SSE: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/event-stream") {
		t.Errorf("expected text/event-stream, got %q", ct)
	}

	// Broadcast content and read from the SSE stream.
	if err := srv.Broadcast([]byte("# SSE Test")); err != nil {
		t.Fatalf("broadcast: %v", err)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if strings.Contains(data, "SSE Test") {
				return // Success.
			}
		}
	}
	t.Error("did not receive expected SSE data")
}
