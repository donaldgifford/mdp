package livereload_test

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"

	"github.com/donaldgifford/mdp/pkg/livereload"
)

const testTimeout = 2 * time.Second

func newTestServer(t *testing.T) (*livereload.Hub, *httptest.Server) {
	t.Helper()
	hub := livereload.NewHub()
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWebSocket)
	mux.HandleFunc("/events", hub.HandleSSE)
	srv := httptest.NewServer(mux)
	t.Cleanup(func() {
		srv.Close()
		_ = hub.Close()
	})
	return hub, srv
}

func dialWS(t *testing.T, srv *httptest.Server) *websocket.Conn {
	t.Helper()
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial websocket: %v", err)
	}
	if resp != nil && resp.Body != nil {
		_ = resp.Body.Close()
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

// waitForCount polls hub.Count until it equals want or the timeout fires.
func waitForCount(t *testing.T, hub *livereload.Hub, want int) {
	t.Helper()
	deadline := time.Now().Add(testTimeout)
	for time.Now().Before(deadline) {
		if hub.Count() == want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("hub.Count = %d, want %d", hub.Count(), want)
}

func TestHub_BroadcastDeliversToWSClient(t *testing.T) {
	t.Parallel()
	hub, srv := newTestServer(t)

	conn := dialWS(t, srv)
	waitForCount(t, hub, 1)

	hub.Broadcast([]byte("hello"))

	if err := conn.SetReadDeadline(time.Now().Add(testTimeout)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read message: %v", err)
	}
	if string(msg) != "hello" {
		t.Errorf("got %q, want %q", string(msg), "hello")
	}
}

func TestHub_BroadcastDeliversToSSEClient(t *testing.T) {
	t.Parallel()
	hub, srv := newTestServer(t)

	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/events", http.NoBody)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("get events: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", got)
	}

	waitForCount(t, hub, 1)
	hub.Broadcast([]byte("sse-payload"))

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			got := strings.TrimPrefix(line, "data: ")
			if got != "sse-payload" {
				t.Errorf("SSE data = %q, want %q", got, "sse-payload")
			}
			return
		}
	}
	t.Fatal("SSE stream ended before data was received")
}

func TestHub_CountReflectsConnections(t *testing.T) {
	t.Parallel()
	hub, srv := newTestServer(t)

	wsConns := make([]*websocket.Conn, 3)
	for i := range wsConns {
		wsConns[i] = dialWS(t, srv)
	}
	waitForCount(t, hub, 3)

	// Add two SSE clients.
	for range 2 {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/events", http.NoBody)
		if err != nil {
			t.Fatalf("new request: %v", err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("get events: %v", err)
		}
		t.Cleanup(func() { _ = resp.Body.Close() })
	}
	waitForCount(t, hub, 5)
}

func TestHub_CloseDropsAllClients(t *testing.T) {
	t.Parallel()
	hub := livereload.NewHub()
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWebSocket)
	mux.HandleFunc("/events", hub.HandleSSE)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	conn := dialWS(t, srv)
	waitForCount(t, hub, 1)

	if err := hub.Close(); err != nil {
		t.Fatalf("close hub: %v", err)
	}
	if hub.Count() != 0 {
		t.Errorf("after Close, Count = %d, want 0", hub.Count())
	}

	// WS reads should now fail (connection closed by hub).
	if err := conn.SetReadDeadline(time.Now().Add(testTimeout)); err != nil {
		t.Fatalf("set read deadline: %v", err)
	}
	if _, _, err := conn.ReadMessage(); err == nil {
		t.Error("expected read error after hub close, got nil")
	}

	// Close is idempotent.
	if err := hub.Close(); err != nil {
		t.Errorf("second Close returned %v, want nil", err)
	}
}

// TestHub_HandleAfterClose verifies that HandleWebSocket and
// HandleSSE refuse to register clients on a closed Hub instead of
// crashing — exercises the addWS/addSSE closed-hub branch.
func TestHub_HandleAfterClose(t *testing.T) {
	t.Parallel()
	hub := livereload.NewHub()
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", hub.HandleWebSocket)
	mux.HandleFunc("/events", hub.HandleSSE)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	if err := hub.Close(); err != nil {
		t.Fatalf("close hub: %v", err)
	}

	// SSE GET against a closed hub should return without registering.
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/events", http.NoBody)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	_ = resp.Body.Close()

	// WS dial against a closed hub: upgrade succeeds but the conn is
	// closed immediately. The read fails.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, dialResp, dialErr := websocket.DefaultDialer.Dial(wsURL, nil)
	if dialErr == nil {
		if dialResp != nil && dialResp.Body != nil {
			_ = dialResp.Body.Close()
		}
		_ = conn.SetReadDeadline(time.Now().Add(testTimeout))
		if _, _, readErr := conn.ReadMessage(); readErr == nil {
			t.Error("expected read error from closed-hub WS, got nil")
		}
		_ = conn.Close()
	}

	if got := hub.Count(); got != 0 {
		t.Errorf("hub.Count = %d after close, want 0", got)
	}
}

func TestHub_ConcurrentBroadcastNoRace(t *testing.T) {
	t.Parallel()
	hub, srv := newTestServer(t)

	// Spin up some clients first so Broadcast has work to do.
	for range 3 {
		dialWS(t, srv)
	}
	waitForCount(t, hub, 3)

	const goroutines = 20
	const broadcasts = 50

	var wg sync.WaitGroup
	for range goroutines {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range broadcasts {
				hub.Broadcast([]byte("race-check"))
			}
		}()
	}

	// Concurrent connect/disconnect activity.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range 10 {
			c := dialWS(t, srv)
			time.Sleep(time.Millisecond)
			_ = c.Close()
		}
	}()

	wg.Wait()
}

func TestHub_SlowClientDoesNotBlockBroadcast(t *testing.T) {
	t.Parallel()
	hub, srv := newTestServer(t)

	// A WS client that we never read from. Its send channel will fill;
	// the assertion is that Broadcast remains non-blocking regardless.
	slowConn := dialWS(t, srv)
	defer func() { _ = slowConn.Close() }()
	waitForCount(t, hub, 1)

	// 1000 broadcasts against a jammed client. With non-blocking sends,
	// this should complete in well under the test timeout.
	const broadcasts = 1000
	start := time.Now()
	for range broadcasts {
		hub.Broadcast([]byte("flood"))
	}
	elapsed := time.Since(start)

	if elapsed > testTimeout {
		t.Fatalf("broadcast loop took %v, expected non-blocking under %v", elapsed, testTimeout)
	}
}
