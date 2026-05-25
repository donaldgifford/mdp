package livereload

import (
	"bytes"
	"log/slog"
	"net/http"
	"strings"
)

const (
	defaultWSPath          = "/ws"
	defaultSSEPath         = "/events"
	defaultInjectionMarker = "</body>"
)

type handlerConfig struct {
	wsPath          string
	ssePath         string
	injectionPoint  string
	clientJS        string
	clientJSCustom  bool
	pathsCustomized bool
}

// HandlerOption configures WrapHandler.
type HandlerOption func(*handlerConfig)

// WithWSPath overrides the WebSocket route path. Default: "/ws".
// When set, callers should pair this with WithClientJS so the injected
// script targets the new path; the bundled ClientJS hardcodes defaults.
func WithWSPath(path string) HandlerOption {
	return func(c *handlerConfig) {
		c.wsPath = path
		c.pathsCustomized = true
	}
}

// WithSSEPath overrides the Server-Sent Events route path. Default:
// "/events". When set, callers should pair this with WithClientJS so
// the injected script targets the new path.
func WithSSEPath(path string) HandlerOption {
	return func(c *handlerConfig) {
		c.ssePath = path
		c.pathsCustomized = true
	}
}

// WithInjectionPoint sets the marker before which ClientJS is inserted
// in text/html responses. Default: "</body>".
func WithInjectionPoint(marker string) HandlerOption {
	return func(c *handlerConfig) {
		c.injectionPoint = marker
	}
}

// WithClientJS overrides the script that gets injected into HTML
// responses. Required when WithWSPath or WithSSEPath are used, since
// the bundled ClientJS hardcodes the default paths.
func WithClientJS(script string) HandlerOption {
	return func(c *handlerConfig) {
		c.clientJS = script
		c.clientJSCustom = true
	}
}

// WrapHandler returns an http.Handler that delegates to next, with two
// additions:
//
//  1. Requests matching the WebSocket or SSE path are routed to hub.
//  2. Responses with Content-Type text/html have the reload <script>
//     injected before the configured injection point.
//
// Useful for consumers that render arbitrary HTML responses (e.g. a
// per-document HTTP handler). Consumers that own a single template can
// inject ClientJS directly and skip this wrapper.
func WrapHandler(next http.Handler, hub *Hub, opts ...HandlerOption) http.Handler {
	cfg := handlerConfig{
		wsPath:         defaultWSPath,
		ssePath:        defaultSSEPath,
		injectionPoint: defaultInjectionMarker,
		clientJS:       ClientJS,
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	if cfg.pathsCustomized && !cfg.clientJSCustom {
		slog.Warn("livereload: custom WS/SSE path set without WithClientJS — injected script will target default paths",
			"wsPath", cfg.wsPath, "ssePath", cfg.ssePath)
	}

	script := "<script>" + cfg.clientJS + "</script>"

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case cfg.wsPath:
			hub.HandleWebSocket(w, r)
			return
		case cfg.ssePath:
			hub.HandleSSE(w, r)
			return
		}

		rec := newInjectingRecorder(w, cfg.injectionPoint, script, r.URL.Path)
		next.ServeHTTP(rec, r)
		rec.Flush()
	})
}

// injectingRecorder buffers a response and rewrites the body to insert
// the reload script before injectionPoint when the Content-Type is HTML.
type injectingRecorder struct {
	w              http.ResponseWriter
	buf            bytes.Buffer
	status         int
	headerWritten  bool
	injectionPoint string
	script         string
	path           string
}

func newInjectingRecorder(w http.ResponseWriter, marker, script, path string) *injectingRecorder {
	return &injectingRecorder{
		w:              w,
		status:         http.StatusOK,
		injectionPoint: marker,
		script:         script,
		path:           path,
	}
}

func (r *injectingRecorder) Header() http.Header {
	return r.w.Header()
}

func (r *injectingRecorder) WriteHeader(status int) {
	r.status = status
	r.headerWritten = true
}

func (r *injectingRecorder) Write(p []byte) (int, error) {
	return r.buf.Write(p)
}

// Flush writes the buffered response, injecting the script when the
// response is text/html and the marker is present.
func (r *injectingRecorder) Flush() {
	body := r.buf.Bytes()
	ct := r.w.Header().Get("Content-Type")

	if ct == "" {
		ct = http.DetectContentType(body)
	}

	if isHTML(ct) {
		idx := bytes.Index(body, []byte(r.injectionPoint))
		if idx == -1 {
			//nolint:gosec // G706: marker and path are structured slog fields, not interpolated into a format string.
			slog.Warn("livereload: injection point not found, response unchanged",
				"marker", r.injectionPoint, "path", r.path)
		} else {
			injected := make([]byte, 0, len(body)+len(r.script))
			injected = append(injected, body[:idx]...)
			injected = append(injected, []byte(r.script)...)
			injected = append(injected, body[idx:]...)
			body = injected
			// Buffered Content-Length is now stale.
			r.w.Header().Del("Content-Length")
		}
	}

	if r.headerWritten {
		r.w.WriteHeader(r.status)
	}
	//nolint:gosec // G705: pass-through middleware — the wrapped handler is responsible for output safety.
	if _, err := r.w.Write(body); err != nil {
		// coverage: response write failure means the client
		// disconnected before we finished sending; nothing to do
		// except log.
		slog.Debug("livereload: write response failed", "error", err)
	}
}

// isHTML returns true when the Content-Type denotes HTML.
func isHTML(contentType string) bool {
	return strings.HasPrefix(strings.ToLower(contentType), "text/html")
}
