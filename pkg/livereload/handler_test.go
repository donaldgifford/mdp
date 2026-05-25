package livereload_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	"github.com/donaldgifford/mdp/pkg/livereload"
)

func TestWrapHandler_InjectsClientJSIntoHTML(t *testing.T) {
	t.Parallel()
	hub := livereload.NewHub()
	t.Cleanup(func() { _ = hub.Close() })

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte("<html><body>hi</body></html>"))
	})

	rec := httptest.NewRecorder()
	livereload.WrapHandler(inner, hub).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody))

	body := rec.Body.String()
	if !strings.Contains(body, "<script>") {
		t.Errorf("expected injected <script>, body = %q", body)
	}
	if !strings.Contains(body, "</script></body>") {
		t.Errorf("expected </script></body> sequence, body = %q", body)
	}
}

func TestWrapHandler_DoesNotInjectIntoNonHTML(t *testing.T) {
	t.Parallel()
	hub := livereload.NewHub()
	t.Cleanup(func() { _ = hub.Close() })

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"x":1}`))
	})

	rec := httptest.NewRecorder()
	livereload.WrapHandler(inner, hub).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/api", http.NoBody))

	if got := rec.Body.String(); got != `{"x":1}` {
		t.Errorf("body = %q, want unchanged JSON", got)
	}
}

func TestWrapHandler_SkipsInjectionWhenMarkerMissing(t *testing.T) {
	t.Parallel()
	hub := livereload.NewHub()
	t.Cleanup(func() { _ = hub.Close() })

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>no body close</html>"))
	})

	rec := httptest.NewRecorder()
	livereload.WrapHandler(inner, hub).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody))

	if got := rec.Body.String(); got != "<html>no body close</html>" {
		t.Errorf("body changed when marker absent: %q", got)
	}
}

func TestWrapHandler_ServesWSAndSSEAtDefaultPaths(t *testing.T) {
	t.Parallel()
	hub := livereload.NewHub()
	t.Cleanup(func() { _ = hub.Close() })

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body></body></html>"))
	})
	srv := httptest.NewServer(livereload.WrapHandler(inner, hub))
	t.Cleanup(srv.Close)

	// WebSocket upgrade at /ws.
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial /ws: %v", err)
	}
	_ = resp.Body.Close()
	_ = conn.Close()

	// SSE at /events.
	sseResp, err := http.Get(srv.URL + "/events") //nolint:noctx // simple test request
	if err != nil {
		t.Fatalf("GET /events: %v", err)
	}
	defer func() { _ = sseResp.Body.Close() }()
	if got := sseResp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Errorf("/events Content-Type = %q, want text/event-stream", got)
	}
}

func TestWrapHandler_HonorsCustomPaths(t *testing.T) {
	t.Parallel()
	hub := livereload.NewHub()
	t.Cleanup(func() { _ = hub.Close() })

	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body></body></html>"))
	})
	handler := livereload.WrapHandler(inner, hub,
		livereload.WithWSPath("/socket"),
		livereload.WithSSEPath("/stream"),
		livereload.WithClientJS("/* custom */"),
	)
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/socket"
	conn, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial /socket: %v", err)
	}
	_ = resp.Body.Close()
	_ = conn.Close()

	sseResp, err := http.Get(srv.URL + "/stream") //nolint:noctx // simple test request
	if err != nil {
		t.Fatalf("GET /stream: %v", err)
	}
	defer func() { _ = sseResp.Body.Close() }()
	if got := sseResp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Errorf("/stream Content-Type = %q, want text/event-stream", got)
	}

	// Custom ClientJS should be in injected HTML responses.
	htmlResp, err := http.Get(srv.URL + "/") //nolint:noctx // simple test request
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	defer func() { _ = htmlResp.Body.Close() }()
	body := make([]byte, 1024)
	n, _ := htmlResp.Body.Read(body)
	if !strings.Contains(string(body[:n]), "/* custom */") {
		t.Errorf("custom ClientJS not injected; body = %q", string(body[:n]))
	}
}

func TestWrapHandler_InjectsIntoSniffedHTMLWhenContentTypeUnset(t *testing.T) {
	t.Parallel()
	hub := livereload.NewHub()
	t.Cleanup(func() { _ = hub.Close() })

	// Inner handler writes HTML but never sets Content-Type. WrapHandler
	// should sniff text/html and inject anyway.
	inner := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<!DOCTYPE html><html><body>x</body></html>"))
	})

	rec := httptest.NewRecorder()
	livereload.WrapHandler(inner, hub).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", http.NoBody))

	if !strings.Contains(rec.Body.String(), "<script>") {
		t.Errorf("expected script injected via sniffed text/html, got: %q", rec.Body.String())
	}
}
