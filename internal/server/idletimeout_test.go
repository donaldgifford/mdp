package server_test

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/donaldgifford/mdp/internal/server"
)

// TestIdleTimeout_ShutdownWithNoClients verifies that the server exits cleanly
// after IdleTimeout elapses with no connected clients.
func TestIdleTimeout_ShutdownWithNoClients(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Idle Test"), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	srv, err := server.New(server.Config{
		File:        mdFile,
		Port:        0,
		OpenBrowser: false,
		IdleTimeout: 300 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("creating server: %v", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- srv.ListenAndServe()
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected nil on idle shutdown, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not shut down within expected window")
	}
}

// TestIdleTimeout_ResetByActiveClient verifies that an active WebSocket
// connection prevents the idle timer from triggering a shutdown.
func TestIdleTimeout_ResetByActiveClient(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Idle Test"), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	srv, err := server.New(server.Config{
		File:        mdFile,
		Port:        0,
		OpenBrowser: false,
		IdleTimeout: 300 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("creating server: %v", err)
	}

	addr := srv.Addr()

	done := make(chan error, 1)
	go func() {
		done <- srv.ListenAndServe()
	}()

	waitForServer(t, "http://"+addr)

	// Connect a WebSocket client — this should hold the idle timer off.
	conn, resp, err := websocket.DefaultDialer.Dial("ws://"+addr+"/ws", nil)
	if err != nil {
		t.Fatalf("websocket dial: %v", err)
	}
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}

	// Wait longer than the idle timeout — server should still be running.
	time.Sleep(600 * time.Millisecond)

	select {
	case err := <-done:
		t.Fatalf("server shut down while client was connected: %v", err)
	default:
		// Expected: server is still running.
	}

	// Disconnect the client — now the idle timer should start.
	conn.Close()

	// Server should shut down after the timeout.
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected nil on idle shutdown, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not shut down after client disconnected")
	}
}

// TestIdleTimeout_DisabledWhenZero verifies that the server stays up
// indefinitely when IdleTimeout is 0.
func TestIdleTimeout_DisabledWhenZero(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Idle Test"), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	srv, err := server.New(server.Config{
		File:        mdFile,
		Port:        0,
		OpenBrowser: false,
		IdleTimeout: 0, // Disabled.
	})
	if err != nil {
		t.Fatalf("creating server: %v", err)
	}

	addr := srv.Addr()

	done := make(chan error, 1)
	go func() {
		done <- srv.ListenAndServe()
	}()

	waitForServer(t, "http://"+addr)

	// Wait longer than a typical timeout — server must not shut down.
	time.Sleep(200 * time.Millisecond)

	select {
	case <-done:
		t.Fatal("server shut down despite IdleTimeout=0")
	default:
		// Expected: still running.
	}

	// Clean up.
	srv.Close()

	select {
	case <-done:
		// Stopped cleanly.
	case <-time.After(2 * time.Second):
		t.Fatal("server did not stop after Close()")
	}
}

// TestIdleTimeout_CloseStopsServer verifies that Close() triggers a clean
// shutdown regardless of idle timeout setting.
func TestIdleTimeout_CloseStopsServer(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("# Close Test"), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	srv, err := server.New(server.Config{
		File:        mdFile,
		Port:        0,
		OpenBrowser: false,
		IdleTimeout: 30 * time.Second, // Long timeout — should not fire.
	})
	if err != nil {
		t.Fatalf("creating server: %v", err)
	}

	addr := srv.Addr()

	done := make(chan error, 1)
	go func() {
		done <- srv.ListenAndServe()
	}()

	waitForServer(t, "http://"+addr)

	// Connect a client so we know the server is up.
	resp, err := http.Get("http://" + addr) //nolint:noctx // Test code.
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	resp.Body.Close()

	// Shut down via Close().
	srv.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Errorf("expected nil after Close(), got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not stop after Close()")
	}
}
