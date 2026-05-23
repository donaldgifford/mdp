---
id: DESIGN-0002
title: "Refactor mdp internals into public pkg packages"
status: Draft
author: Donald Gifford
created: 2026-05-23
---
<!-- markdownlint-disable-file MD025 MD041 -->

# DESIGN 0002: Refactor mdp internals into public pkg packages

**Status:** Draft
**Author:** Donald Gifford
**Date:** 2026-05-23

<!--toc:start-->
- [Overview](#overview)
- [Goals and Non-Goals](#goals-and-non-goals)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Background](#background)
- [Detailed Design](#detailed-design)
  - [Target package layout](#target-package-layout)
  - [pkg/parser (phase 1)](#pkgparser-phase-1)
  - [pkg/theme (phase 1)](#pkgtheme-phase-1)
  - [pkg/livereload (phase 2)](#pkglivereload-phase-2)
    - [Hub](#hub)
    - [WrapHandler (injection middleware)](#wraphandler-injection-middleware)
    - [ClientJS](#clientjs)
  - [internal/server after refactor](#internalserver-after-refactor)
  - [Wire format ownership](#wire-format-ownership)
  - [Reload script (ClientJS)](#reload-script-clientjs)
- [API / Interface Changes](#api--interface-changes)
- [Testing Strategy](#testing-strategy)
- [Migration / Rollout Plan](#migration--rollout-plan)
  - [Phase 1 PR — lift parser and theme](#phase-1-pr--lift-parser-and-theme)
  - [Phase 2 PR — extract livereload](#phase-2-pr--extract-livereload)
- [Open Questions](#open-questions)
- [References](#references)
<!--toc:end-->

## Overview

Detailed design for **phases 1 and 2** of RFC-0001. Covers the
mechanical refactor (lift `internal/parser` and `internal/theme` to
`pkg/`) and the extraction of `pkg/livereload` from
`internal/server`. After this design lands, `internal/server` becomes
a thin orchestrator that wires mdp-specific concerns (file watcher,
stdin protocol, single-file model) on top of generic primitives.

Out-of-scope phases (2.5 docz validation, 3 hardening, 4 release) are
covered by the RFC and don't require detailed design here.

## Goals and Non-Goals

### Goals

- Make `parser`, `theme`, and `livereload` importable by external Go
  apps (starting with docz `serve`).
- Preserve byte-for-byte WS/SSE wire-protocol compatibility with the
  existing `assets/preview.js` and the Neovim plugin.
- Keep `internal/server` working end-to-end after each phase — `mdp
  --file <doc>` and the Neovim plugin must behave identically.
- Define a generic-bytes `Hub` interface so consumers aren't locked
  into mdp's specific JSON message shape.

### Non-Goals

- Change the mdp CLI surface, flags, or default behavior.
- Change mdp's WS/SSE message shape (`{"type":"content","html":"..."}`).
- Expose `internal/watcher`, `internal/server`, or `internal/cli`.
- Build a "serve a directory of markdown files" higher-level helper
  (left to docz to implement and graduate later if useful).
- Pick a Mermaid render-mode default (deferred to phase 3 alongside
  the `WithMermaidRenderMode` option).

## Background

RFC-0001 picks `pkg/parser`, `pkg/theme`, `pkg/livereload` as the v1
surface. INV-0002 documents the analytical basis. This design fills in
the actual file moves, function signatures, and tests.

Important findings from reading the current code that shape this
design:

1. **Two parallel hubs today.** `internal/server/hub.go` (WebSocket)
   and `internal/server/sse.go` (SSE) are separate types
   (`hub`, `sseHub`) with parallel `add`/`remove`/`broadcast`/`count`/
   `closeAll` methods. `server.go:169-170` calls `s.hub.broadcast(msg)`
   and `s.sse.broadcast(msg)` back-to-back — the fan-out is hardcoded
   at the call site, not in the hub. The new `pkg/livereload.Hub`
   unifies them behind a single `Broadcast([]byte)` that delivers to
   both transports internally.

   **Latent concurrency bug in the current code:** `hub.broadcast`
   (`internal/server/hub.go:37-46`) holds an `RLock` while calling
   `conn.WriteMessage`. gorilla/websocket documents that concurrent
   writes to the same connection are unsafe. Today's code is safe
   only because mdp serializes broadcasts (single watcher goroutine →
   single `BroadcastFile` call). Once `Hub` is public, library
   consumers can call `Broadcast` from any goroutine — the new
   implementation must close this gap. See [Hub concurrency](#hub-concurrency).
2. **No `<script>` injection today.** Reload works because
   `assets/preview.html:25` renders `{{.JS}}` (the bundled
   `preview.js`) into a single page that connects WS+SSE. This is
   fine for mdp (single page, template-rendered) but doesn't fit
   docz (many pages, dynamically rendered). `livereload.WrapHandler`
   provides `<script>`-injection middleware for the docz-style case;
   mdp can keep its template approach.
3. **`Server.BroadcastFile()` / `Server.SendCursor(line)`** are
   mdp-specific JSON marshalers that wrap raw `Hub.broadcast`. These
   stay in `internal/server`. The wire format is mdp's contract with
   `preview.js` — *not* `pkg/livereload`'s contract with consumers.
4. **`assets/preview.js` is ~300 lines** and includes UI logic
   (cursor sync, theme switching) beyond reload. The reload-only
   slice is small (~30–50 lines); only that slice becomes
   `livereload.ClientJS`.

## Detailed Design

### Target package layout

End-state after phase 2 (entries marked `[phase 3]` are listed for
reference but are not created in this design's scope):

```text
cmd/mdp/                                -- unchanged
pkg/
  parser/                               -- moved from internal/parser
    parser.go
    lineannotator.go
    parser_test.go
    lineannotator_test.go
    bench_test.go
    doc.go                              -- [phase 3]
    example_test.go                     -- [phase 3]
  theme/                                -- moved from internal/theme
    theme.go
    theme_test.go
    doc.go                              -- [phase 3]
    example_test.go                     -- [phase 3]
  livereload/                           -- NEW
    hub.go                              -- unified WS+SSE broadcast
    handler.go                          -- WrapHandler + injection
    clientjs.go                         -- embedded reload script (string)
    client.js                           -- source for ClientJS (//go:embed)
    hub_test.go
    handler_test.go
    doc.go                              -- [phase 3]
    example_test.go                     -- [phase 3]
internal/server/                        -- slimmed
  server.go                             -- orchestrator: builds Hub, owns mdp flow
  browser.go                            -- unchanged
  stdin.go                              -- unchanged
  (hub.go, sse.go REMOVED — logic moved)
internal/watcher/                       -- unchanged
internal/cli/                           -- unchanged (import paths updated)
assets/                                 -- unchanged
```

### pkg/parser (phase 1)

Pure mechanical lift. No API change in this design. Files moved:

| From | To |
|------|----|
| `internal/parser/parser.go` | `pkg/parser/parser.go` |
| `internal/parser/lineannotator.go` | `pkg/parser/lineannotator.go` |
| `internal/parser/parser_test.go` | `pkg/parser/parser_test.go` |
| `internal/parser/lineannotator_test.go` | `pkg/parser/lineannotator_test.go` |
| `internal/parser/bench_test.go` | `pkg/parser/bench_test.go` |

Import-path updates needed in:

- `internal/server/server.go` (multiple call sites)
- `internal/cli/serve.go` (if it imports parser directly)
- any test files that import parser

Run `goimports -w ./...` after the move.

Benchmark baselines (`bench_test.go`) are not part of phase 1
acceptance — the lift is mechanical and goldmark performance is
unaffected. Re-baselining is a phase 3 hygiene task if desired.

### pkg/theme (phase 1)

Pure mechanical lift. Files moved:

| From | To |
|------|----|
| `internal/theme/theme.go` | `pkg/theme/theme.go` |
| `internal/theme/theme_test.go` | `pkg/theme/theme_test.go` |

The `import "github.com/donaldgifford/mdp/assets"` in `theme.go` is
unchanged — `assets` is already at the module root.

Import-path updates in `internal/server/server.go` and
`internal/cli/serve.go`.

### pkg/livereload (phase 2)

This is the load-bearing phase. The intent is a *generic* live-reload
package that does one thing well: deliver bytes from a server to
connected browser clients over WS or SSE, with optional auto-reload
`<script>` injection for HTML responses.

#### Hub

Unifies the current `hub` (WS) and `sseHub` (SSE):

```go
package livereload

// Hub fans out byte payloads to all connected clients (WebSocket
// and SSE). Safe for concurrent use.
type Hub struct {
    // unexported fields
}

// NewHub creates an empty hub with no connected clients.
func NewHub() *Hub

// Broadcast sends msg to every connected WS and SSE client.
// Failed sends are dropped; the connection is closed and removed.
func (h *Hub) Broadcast(msg []byte)

// Count returns the number of currently connected clients
// (WS + SSE combined). Useful for idle-timeout logic.
func (h *Hub) Count() int

// Close drops all connections and prevents new ones from being
// added.
func (h *Hub) Close() error

// HandleWebSocket upgrades the HTTP request to a WebSocket and
// registers the connection with the hub. Intended to be wired to
// an http.ServeMux at the consumer's chosen path.
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request)

// HandleSSE streams hub broadcasts over Server-Sent Events.
// Intended to be wired to an http.ServeMux at the consumer's
// chosen path.
func (h *Hub) HandleSSE(w http.ResponseWriter, r *http.Request)
```

Internally, `Hub` keeps two collections (WS conns, SSE channels) but
the public API is one method per concern. Consumers don't need to
care which transport a given client uses.

#### Hub concurrency

Closes the gorilla/websocket concurrent-write gap called out in
[Background](#background). The new design uses **per-connection
sender goroutines with a buffered channel**, which is the upstream
gorilla pattern:

- On `HandleWebSocket`, after the upgrade, `Hub` starts a goroutine
  per connection that reads from a per-client `chan []byte` (buffer
  size 8) and calls `conn.WriteMessage`. All writes to a given
  connection therefore happen from a single goroutine.
- `Broadcast` takes `RLock`, ranges over clients, and does a
  non-blocking send (`select` with `default`) onto each client's
  channel. If the channel is full (slow client), the message for
  *that client* is dropped and the connection is scheduled for
  removal — fast clients aren't blocked by slow ones.
- `remove` closes the per-client channel; the sender goroutine
  exits when the channel is drained and closed.
- SSE clients already use `chan []byte` today (`sse.go:11-15`), so
  the SSE path is structurally similar; the unification is mostly
  about presenting a single `Hub` API.

This must be implemented as part of phase 2 — lifting the existing
`hub.go` verbatim would carry the race forward. The new test
`pkg/livereload/hub_test.go` should include a `t.Parallel()` test
firing `Broadcast` from N goroutines concurrently with
`Connect`/`remove` to assert no race under `-race`.

#### WrapHandler (injection middleware)

For consumers that render arbitrary HTML responses (docz's per-doc
pages):

```go
// WrapHandler wraps next, injecting the reload <script> into any
// "text/html" response and exposing the WS and SSE endpoints at
// configurable paths. Useful for consumers that don't render HTML
// through a single template (mdp does, docz doesn't).
//
// Default paths: /ws and /events.
func WrapHandler(next http.Handler, hub *Hub, opts ...HandlerOption) http.Handler

type HandlerOption func(*handlerConfig)

func WithWSPath(path string) HandlerOption
func WithSSEPath(path string) HandlerOption

// WithInjectionPoint sets the marker before which ClientJS is
// inserted in HTML responses. Default: "</body>".
func WithInjectionPoint(marker string) HandlerOption

// WithClientJS overrides the script that gets injected. Required
// when WithWSPath or WithSSEPath are used, since the bundled
// ClientJS hardcodes the default paths. If unset, the default
// ClientJS is injected.
func WithClientJS(script string) HandlerOption
```

Returning `http.Handler` (interface) instead of a concrete `*Handler`
type keeps the surface small and composes naturally with stdlib
middleware patterns. **This supersedes RFC-0001's `*Handler`
sketch** — the RFC has been updated to match. Adding methods later
is not currently planned; if future needs require methods, a
concrete return type can be introduced as additive API
(`*Handler` already implements `http.Handler`, so existing call
sites would continue compiling).

#### ClientJS

The reload-only JS slice extracted from `assets/preview.js`. Lives in
`pkg/livereload/client.js` and embedded via `//go:embed`:

```go
package livereload

import _ "embed"

//go:embed client.js
var ClientJS string
```

The script:
1. Opens a WebSocket to `/ws` (or the configured path).
2. Falls back to EventSource at `/events` if WS fails.
3. On any incoming message, reloads the page (or, if the consumer's
   message shape includes `type: "content"` with `html`, swaps the
   body — the script can be dumb and just `location.reload()` for
   v1 simplicity).

For v1, ClientJS does a simple `location.reload()` on incoming
message. The fancier "swap body without full reload" behavior stays
in `assets/preview.js` and remains mdp-specific. This keeps
`pkg/livereload` minimal and broadly applicable.

### internal/server after refactor

`internal/server/server.go` shrinks substantially. Responsibilities
that stay:

- HTTP routing — current routes per `server.go:230-244`:
  - `GET /` → `handleIndex`
  - `GET /ws` → was `handleWebSocket`, becomes `hub.HandleWebSocket`
  - `GET /events` → was `handleSSE`, becomes `hub.HandleSSE`
  - `POST /cursor` → `handleCursor`
  - `GET /vendor/` → `http.FileServer` over embedded vendor assets
  - `GET /local/` → `http.FileServer` over the markdown file's
    directory (relative asset loading)
- Token-based auth middleware
- Idle-watcher (close when no clients for N seconds)
- `pageData` rendering of `assets/preview.html`
- mdp-specific message shapes:
  - `wsMessage{Type: "content", HTML: ...}`
  - `wsMessage{Type: "cursor", Line: N}`
- `BroadcastFile()`, `SendCursor()` convenience methods that JSON-marshal
  and call `hub.Broadcast`
- stdin reader + browser opener (no change)

Responsibilities that move:

- WebSocket upgrade / connection lifecycle → `pkg/livereload`
- SSE streaming → `pkg/livereload`
- Hub fan-out logic → `pkg/livereload`

Concretely: `server.go` constructs a `*livereload.Hub`, wires
`mux.HandleFunc("GET /ws", hub.HandleWebSocket)` and
`mux.HandleFunc("GET /events", hub.HandleSSE)`, and replaces internal
calls to `s.hub.broadcast(...)` with `s.hub.Broadcast(...)` (capital B —
the public method on the new package).

`internal/server` does **not** use `livereload.WrapHandler` — mdp's
HTML is rendered through a template that already includes
`assets/preview.js` (which has the WS/SSE client baked in). The
injection middleware is for docz and other consumers that render
arbitrary HTML responses.

### Wire format ownership

| Wire detail | Owner | Why |
|-------------|-------|-----|
| `{"type":"content","html":"..."}` JSON shape | `internal/server` + `assets/preview.js` | mdp-specific. Consumers can broadcast any bytes. |
| `{"type":"cursor","line":N}` JSON shape | `internal/server` + `assets/preview.js` | mdp-specific (scroll sync). |
| WS frame transport, SSE event framing | `pkg/livereload` | Generic; same across all consumers. |
| Reconnect-on-disconnect behavior in browser | `pkg/livereload.ClientJS` | Generic. |

This split means `pkg/livereload.Hub.Broadcast` takes `[]byte` and
consumers marshal whatever shape they want. mdp keeps marshaling its
existing `wsMessage` JSON; docz can do something completely different
(or nothing — just call `Broadcast([]byte("reload"))` and let
`ClientJS` do `location.reload()`).

### Reload script (ClientJS)

The minimal v1 ClientJS (~30 lines, illustrative):

```javascript
(function () {
  var ws, sse, retry = 0;
  function reload() { location.reload(); }
  function connectWS() {
    try {
      ws = new WebSocket(
        (location.protocol === "https:" ? "wss:" : "ws:") +
        "//" + location.host + "/ws"
      );
      ws.onmessage = reload;
      ws.onclose = function () {
        ws = null;
        if (++retry < 5) setTimeout(connectWS, 500 * retry);
        else connectSSE();
      };
    } catch (e) { connectSSE(); }
  }
  function connectSSE() {
    sse = new EventSource("/events");
    sse.onmessage = reload;
  }
  connectWS();
})();
```

The snippet above is **illustrative**, not a literal copy — the
actual `client.js` will use whatever exact JS the implementer
chooses, as long as it connects to `/ws` and `/events` and reloads
on any incoming message. The default paths are hardcoded; consumers
who override paths via `WithWSPath` / `WithSSEPath` must pass a
matching `WithClientJS(script)` so the injected script targets the
right endpoints. `WrapHandler` will log a warning at construction if
custom paths are set without a custom `ClientJS`.

## API / Interface Changes

| Package | New public surface |
|---------|--------------------|
| `pkg/parser` | Same as `internal/parser` today: `Parser`, `Option`, `New`, `Render`, `WithGFM`, `WithSyntaxHighlighting`, `WithMermaid`, `WithMath`, `WithCallouts`. (`WithMermaidRenderMode` deferred to phase 3.) |
| `pkg/theme` | Same as `internal/theme` today: `Theme` struct (`CSS`, `HljsVendorCSS`, `MermaidTheme`, `IsAuto`), `Resolve`, `Names`. |
| `pkg/livereload` | `Hub`, `NewHub`, `(*Hub).Broadcast`, `(*Hub).Count`, `(*Hub).Close`, `(*Hub).HandleWebSocket`, `(*Hub).HandleSSE`, `WrapHandler`, `HandlerOption`, `WithWSPath`, `WithSSEPath`, `WithInjectionPoint`, `ClientJS`. |

No CLI changes. No config changes. No `go.mod` module-path change.

## Testing Strategy

**Existing tests, in place after refactor:**

| Test | Asserts | Moves to |
|------|---------|----------|
| `internal/parser/*_test.go` | parser behavior, line annotator, bench | `pkg/parser/*_test.go` |
| `internal/theme/theme_test.go` | theme registry, resolution | `pkg/theme/theme_test.go` |
| `internal/server/livereload_test.go` | end-to-end reload over WS/SSE | stays in `internal/server`, validates the integration |
| `internal/server/scrollsync_test.go` | cursor wire format end-to-end | stays in `internal/server` |
| `internal/server/idletimeout_test.go` | idle closure based on hub count | stays in `internal/server` (uses `hub.Count()` now) |

**New tests required (phase 2):**

| Test | Asserts |
|------|---------|
| `pkg/livereload/hub_test.go` | `Broadcast` delivers to WS and SSE clients; `Count` is accurate; `Close` shuts both transports; concurrent broadcast/connect safety |
| `pkg/livereload/handler_test.go` | `WrapHandler` injects `ClientJS` before `</body>` in `text/html` responses; doesn't touch non-HTML; serves `/ws` and `/events` at default and custom paths |
| `pkg/livereload/wire_test.go` | **Regression test**: server in `internal/server` configuration broadcasts a `wsMessage`; assert the bytes on the WS frame match the existing protocol byte-for-byte. Backstops phase 2's acceptance criterion. |

**Deferred to phase 3:** `example_test.go` per public package
(parser, theme, livereload) importing *only* that package. Used for
compile-time isolation proof + GoDoc examples.

**Manual smoke test after each phase:**

1. `make build && make test && make lint`
2. `./bin/mdp --file README.md` — render in browser, edit README.md,
   confirm live reload works
3. In Neovim with the plugin loaded: `:MdpPreview` on a `.md` file,
   edit, confirm reload + scroll sync work
4. `make test-coverage` — confirm no coverage regression

## Migration / Rollout Plan

Two PRs, one per phase. Each lands independently and can be reverted
without affecting the other.

### Phase 1 PR — lift parser and theme

**Branch:** `feat/lift-parser-theme-to-pkg`

**Diff shape:**
- `git mv internal/parser pkg/parser`
- `git mv internal/theme pkg/theme`
- Rewrite imports in `cmd/mdp/main.go`, `internal/cli/*.go`,
  `internal/server/*.go`, plus any test files
- Run `make fmt` (gci will reorganize import blocks)

**PR labels:** `minor` (RFC-0001 phase 1 is a v0.2.0 milestone but
phase 1 alone doesn't ship anything new to library consumers — they
can already see `pkg/parser` exists but it's pre-release).

**Acceptance:** as defined in RFC-0001 phase 1.

### Phase 2 PR — extract livereload

**Branch:** `feat/extract-livereload-package`

**Diff shape:**
- Create `pkg/livereload/{hub,handler,clientjs}.go` and `client.js`
- Move logic from `internal/server/hub.go` and `internal/server/sse.go`
  into `pkg/livereload/hub.go` (unified)
- Delete `internal/server/hub.go` and `internal/server/sse.go`
- Refactor `internal/server/server.go`:
  - Replace `s.hub = newHub()` and `s.sse = newSSEHub()` (per
    `server.go:125-126`, field is `s.sse`) with
    `s.hub = livereload.NewHub()`
  - Replace `mux.HandleFunc("GET /ws", s.handleWebSocket)` with
    `mux.HandleFunc("GET /ws", s.hub.HandleWebSocket)`
  - Same for SSE
  - Remove `handleWebSocket` and `handleSSE` methods from `Server`
  - Update `BroadcastFile()` / `SendCursor()` to call
    `s.hub.Broadcast` (public)
- Add the new tests listed in [Testing Strategy](#testing-strategy)

**Acceptance:** as defined in RFC-0001 phase 2.

## Open Questions

1. **Should `Hub` expose `HandleWebSocket` / `HandleSSE` as methods,
   or as standalone functions taking a `*Hub`?**
   Methods read more naturally at call sites
   (`mux.HandleFunc("GET /ws", hub.HandleWebSocket)`), but standalone
   funcs (`livereload.WebSocketHandler(hub)`) are easier to compose
   with stdlib middleware. Current design picks methods; reconsider
   if it bites in phase 2.5.

2. **Should the SSE channel be a `chan []byte` or `chan string`?**
   Today's `sseHub` uses `chan []byte`. The SSE spec requires
   string-encoded `data:` lines, so we'll end up converting either
   way. Sticking with `[]byte` for parity with the WS path.

3. **What happens if a consumer's response is `Content-Type:
   text/html; charset=utf-8` but doesn't contain `</body>`?**
   Inject before `</html>`, before EOF, or skip? Current design:
   skip (don't inject) and document the requirement in `WrapHandler`'s
   GoDoc. The "no `</body>` in HTML" case is rare and silently
   failing is better than corrupting markup.

4. **Should mdp's `internal/server` switch to using
   `livereload.WrapHandler` for consistency, or keep its template-
   based injection?** Current design: keep the template approach
   (less churn, mdp's HTML is template-rendered anyway). Worth
   revisiting later for code-path consolidation.

5. **Mermaid render mode default — does promoting `parser` to public
   force a decision now?** No — current behavior
   (`mermaid.RenderModeClient` hardcoded) is preserved in phase 1.
   `WithMermaidRenderMode` is added in phase 3 as additive API; the
   default stays client-side. Pre-v0.2.0 hardening can revisit.

### Resolved during review

- **`WrapHandler` return type.** Was open; resolved to `http.Handler`
  in the [WrapHandler section](#wraphandler-injection-middleware).
  RFC-0001 has been updated to match.
- **Custom-path `ClientJS` gap.** Was punted; resolved by adding
  `WithClientJS(script)` to the handler options.
- **`hub.broadcast` concurrent-write race.** Was unaddressed in the
  first draft; resolved by the [Hub concurrency](#hub-concurrency)
  design (per-connection sender goroutine + buffered channel).

## References

- [RFC-0001 — Public mdp Go library](../rfc/0001-public-mdp-go-library.md)
  — the proposal this design implements
- [INV-0002 — Expose mdp as a Go library for other apps](../investigation/0002-expose-mdp-as-go-library.md)
  — the analytical basis
- `internal/server/hub.go` — current WS hub, source for
  `pkg/livereload/hub.go`
- `internal/server/sse.go` — current SSE hub, merged into
  `pkg/livereload/hub.go`
- `internal/server/server.go:160-194` — `Broadcast`, `SendCursor`,
  `BroadcastFile` (stay in `internal/server`)
- `internal/server/server.go:389-414` — `handleWebSocket` (moves to
  `pkg/livereload`)
- `internal/server/sse.go:66` — `handleSSE` (moves to `pkg/livereload`)
- `assets/preview.html:25` — `{{.JS}}` template injection point
  (template-based path, kept)
- `assets/preview.js` — source of the reload-only slice that becomes
  `livereload.ClientJS`
