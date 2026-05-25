package server_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/donaldgifford/mdp/internal/server"
)

// TestWireFormat_WSContentMatchesPriorBaseline locks the exact bytes
// the server broadcasts for a content update. If the parser, JSON
// marshaling, or message shape ever changes the on-wire layout, this
// test catches it.
//
// To intentionally update the baseline, delete testdata/ws_content.bin
// (test will regenerate and pass on next run; verify the new bytes are
// the intended change, then commit).
func TestWireFormat_WSContentMatchesPriorBaseline(t *testing.T) {
	t.Parallel()
	bytesOut := captureBroadcast(t, func(s *server.Server) error {
		return s.Broadcast([]byte("hi"))
	})
	assertGolden(t, "testdata/ws_content.bin", bytesOut)
}

// TestWireFormat_WSCursorMatchesPriorBaseline locks the exact bytes
// for a cursor update.
func TestWireFormat_WSCursorMatchesPriorBaseline(t *testing.T) {
	t.Parallel()
	bytesOut := captureBroadcast(t, func(s *server.Server) error {
		return s.SendCursor(42)
	})
	assertGolden(t, "testdata/ws_cursor.bin", bytesOut)
}

// captureBroadcast runs a single broadcast through a fully wired
// Server + livereload.Hub and returns the bytes received over WS.
func captureBroadcast(t *testing.T, broadcast func(*server.Server) error) []byte {
	t.Helper()

	dir := t.TempDir()
	mdFile := filepath.Join(dir, "test.md")
	if err := os.WriteFile(mdFile, []byte("hi"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	srv, err := server.New(server.Config{
		File:        mdFile,
		Port:        0,
		OpenBrowser: false,
		ScrollSync:  true,
	})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}
	t.Cleanup(srv.Close)

	go func() { _ = srv.ListenAndServe() }()
	waitForServer(t, "http://"+srv.Addr())

	wsURL := "ws://" + srv.Addr() + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial ws: %v", err)
	}
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	defer func() { _ = conn.Close() }()

	if err := broadcast(srv); err != nil {
		t.Fatalf("broadcast: %v", err)
	}

	if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	return msg
}

// assertGolden compares got to the file at path. If the file is
// missing, it writes got as the new baseline and fails the test with a
// clear message — re-running passes once the new baseline is staged.
func assertGolden(t *testing.T, path string, got []byte) {
	t.Helper()
	want, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			t.Fatalf("read golden %s: %v", path, err)
		}
		if writeErr := os.WriteFile(path, got, 0o644); writeErr != nil {
			t.Fatalf("write new golden %s: %v", path, writeErr)
		}
		t.Fatalf("golden %s did not exist; wrote new baseline (%d bytes). "+
			"Inspect, then re-run to verify.", path, len(got))
	}
	if !bytes.Equal(got, want) {
		t.Errorf("wire format drift at %s:\n  got:  %q\n  want: %q", path, got, want)
	}
}
