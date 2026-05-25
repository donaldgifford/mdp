// Package livereload provides browser live-reload primitives: a
// transport-agnostic Hub that fans byte payloads out to connected
// WebSocket and SSE clients, plus a WrapHandler middleware that
// injects a reload <script> into HTML responses.
//
// # Transport contract
//
// Hub.Broadcast accepts opaque []byte payloads. The package makes no
// assumption about message shape — consumers may broadcast JSON, raw
// text, binary, or anything else, and the payload is delivered
// verbatim to every connected client.
//
//   - WebSocket clients receive each Broadcast call as a single
//     gorilla/websocket TextMessage frame.
//   - SSE clients receive each Broadcast call as a single SSE event
//     framed as `data: <payload>\n\n`. Payloads containing newlines
//     should be encoded by the caller (the package does not split
//     into multiple data: lines).
//
// The default ClientJS that ships with the package treats any
// incoming message as a reload trigger and calls location.reload().
// Consumers that want smarter behavior (in-place DOM swap, typed
// message handling) can supply their own script via WithClientJS.
//
// # Concurrency
//
// Hub is safe for concurrent use. Each WebSocket connection has its
// own writer goroutine pulling from a buffered channel, so Broadcast
// can be called from any goroutine without violating
// gorilla/websocket's single-writer requirement. Slow clients are
// dropped without blocking fast ones.
//
// # Security
//
// Hub is intended for local-only use. HandleWebSocket accepts any
// origin without an origin check. Do not expose a Hub directly to
// the public internet without wrapping it in your own authentication
// layer.
//
// # Composition example
//
// A minimal live-reload server: any GET to /reload triggers a
// broadcast, and the injected script reloads the page.
//
//	hub := livereload.NewHub()
//	defer hub.Close()
//
//	page := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
//	    w.Header().Set("Content-Type", "text/html")
//	    _, _ = w.Write([]byte("<html><body>hi</body></html>"))
//	})
//
//	mux := http.NewServeMux()
//	mux.Handle("/", livereload.WrapHandler(page, hub))
//	mux.HandleFunc("/reload", func(_ http.ResponseWriter, _ *http.Request) {
//	    hub.Broadcast([]byte("reload"))
//	})
//
//	_ = http.ListenAndServe(":8080", mux)
package livereload
