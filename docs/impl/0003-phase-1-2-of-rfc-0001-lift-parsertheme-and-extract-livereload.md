---
id: IMPL-0003
title: "Phase 1-2 of RFC-0001 — lift parser/theme and extract livereload"
status: Draft
author: Donald Gifford
created: 2026-05-24
---
<!-- markdownlint-disable-file MD025 MD041 -->

# IMPL 0003: Phase 1-2 of RFC-0001 — lift parser/theme and extract livereload

**Status:** Draft
**Author:** Donald Gifford
**Date:** 2026-05-24

<!--toc:start-->
- [Objective](#objective)
- [Scope](#scope)
  - [In Scope](#in-scope)
  - [Out of Scope](#out-of-scope)
- [Implementation Phases](#implementation-phases)
  - [Phase 0: Add -race to CI test step (precursor)](#phase-0-add--race-to-ci-test-step-precursor)
    - [Tasks](#tasks)
    - [Success Criteria](#success-criteria)
  - [Phase 1: Lift parser and theme to pkg/](#phase-1-lift-parser-and-theme-to-pkg)
    - [Tasks](#tasks-1)
    - [Success Criteria](#success-criteria-1)
  - [Phase 2: Extract pkg/livereload from internal/server](#phase-2-extract-pkglivereload-from-internalserver)
    - [Tasks](#tasks-2)
    - [Success Criteria](#success-criteria-2)
- [Cross-Phase Concerns](#cross-phase-concerns)
- [Open Questions](#open-questions)
  - [Resolved](#resolved)
- [References](#references)
<!--toc:end-->

## Objective

Execute phases 1 and 2 of RFC-0001 per the detailed design in
DESIGN-0002. Phase 1 lifts `internal/parser` and `internal/theme` to
`pkg/`. Phase 2 extracts a new `pkg/livereload` package from
`internal/server` (consolidating the parallel WS/SSE hubs, closing the
gorilla/websocket concurrent-write race) and refactors
`internal/server` to be its first in-tree consumer. After both phases,
`make build && make test && make lint` is clean, `mdp --file <doc>`
behaves identically to before, and library consumers can `import
"github.com/donaldgifford/mdp/pkg/{parser,theme,livereload}"`.

**Implements:** [DESIGN-0002](../design/0002-refactor-mdp-internals-into-public-pkg-packages.md)
(detailed design) and [RFC-0001](../rfc/0001-public-mdp-go-library.md)
phases 1–2.

## Scope

### In Scope

- Move `internal/parser` → `pkg/parser` (mechanical lift)
- Move `internal/theme` → `pkg/theme` (mechanical lift)
- Create `pkg/livereload` with `Hub`, `WrapHandler`, `ClientJS`
- Implement per-connection sender-goroutine concurrency in `Hub` (closes
  the latent WS write race)
- Refactor `internal/server` to consume `pkg/livereload`; delete
  `internal/server/hub.go` and `internal/server/sse.go`
- Preserve byte-for-byte wire compatibility with `assets/preview.js`
  and the Neovim plugin
- Add new tests under `pkg/livereload/` (hub, handler, wire regression)

### Out of Scope

- `pkg/livereload.WithMermaidRenderMode` and other phase-3 API
  hardening (deferred to a separate IMPL)
- `doc.go` / `example_test.go` files for the public packages (phase 3)
- docz `serve` integration (phase 2.5, handled in the docz repo)
- README "Library" section and v0.2.0 tag (phase 4)
- Mermaid render-mode option additions
- Changes to `internal/watcher`, `internal/cli`, the CLI flags, or any
  user-visible behavior

## Implementation Phases

Each phase ships as its own PR. Each phase is complete when **all
tasks are checked off and all success criteria are met**.

---

### Phase 0: Add `test-race` make target for local dev (optional precursor)

**Branch:** `chore/add-race-make-target`
**Estimated diff:** 1 file (Makefile).
**Status:** *Optional.* CI already runs race detection — `make
test-coverage` invokes `go test -race -coverprofile=...`
(`Makefile:52-54`), and the GitHub Actions `Test Go` job runs
`make test-coverage`. So phase 2's `Hub` concurrency test is already
defended in CI without any precursor.

This phase only adds a convenience target for local dev so contributors
can run race detection without producing a coverage file. Skip the
phase entirely if you prefer to keep using `make test-coverage`
locally.

#### Tasks

- [ ] Add a `test-race` target to `Makefile`:
      `$(GO) test -race ./...` (no coverage output)
- [ ] PR title: `chore: add test-race convenience make target`
- [ ] PR label: `patch`

#### Success Criteria

- `make test-race` is a defined target and runs cleanly on `main`
- No CI changes (race detection is already exercised by
  `make test-coverage`)

---

### Phase 1: Lift parser and theme to pkg/

**Branch:** `feat/lift-parser-theme-to-pkg`
**Estimated diff:** ~15 files touched, mostly import-path rewrites.
This is a mechanical refactor — no behavior change.

#### Tasks

**Move parser package**

- [x] `git mv internal/parser pkg/parser` (preserves history)
- [x] Verify `pkg/parser/parser.go` package declaration is still
      `package parser` (no edit needed — package name doesn't change)
- [x] Verify `pkg/parser/lineannotator.go`, `parser_test.go`,
      `lineannotator_test.go`, `bench_test.go` all moved cleanly

**Move theme package**

- [x] `git mv internal/theme pkg/theme`
- [x] Verify `pkg/theme/theme.go` package declaration is still
      `package theme`
- [x] Confirm `pkg/theme/theme.go` still imports
      `github.com/donaldgifford/mdp/assets` (unchanged — `assets` is
      at module root)

**Rewrite imports**

- [x] Update `internal/server/server.go` imports:
      `github.com/donaldgifford/mdp/internal/parser` →
      `github.com/donaldgifford/mdp/pkg/parser`
- [x] Update `internal/server/server.go` imports:
      `github.com/donaldgifford/mdp/internal/theme` →
      `github.com/donaldgifford/mdp/pkg/theme` (and dropped the
      `interntheme` alias — bare `theme` package qualifier is
      unambiguous against the `s.theme` field)
- [x] Run `grep -rn "internal/parser\|internal/theme" internal/cli/ cmd/`
      to identify call sites. Update any imports found
      (no matches — `internal/cli` and `cmd/mdp` don't import
      parser/theme directly; they go through `internal/server`)
- [x] Sweep with `goimports -w ./...` (or `make fmt`) to reorganize
      import blocks (gci enforces group order)
- [x] Verify with `grep -r "internal/parser\|internal/theme" .` that
      no stale paths remain (outside docs/)

**Verify and tidy**

- [x] `make fmt` clean
- [x] `make lint` clean (gci is the most likely failure surface)
- [x] `make test` green
- [x] `make build` produces a working binary
- [ ] Manual smoke: `./bin/mdp --file README.md` — open in browser,
      edit README.md, confirm live reload still works *(deferred —
      requires browser; the test suite covers the HTTP/WS paths)*
- [ ] Neovim smoke: `:MdpPreview` on a `.md` file, edit, confirm
      reload + scroll sync still work *(deferred — requires
      Neovim; covered by the binary build + test suite)*

**PR**

- [ ] Open PR with `patch` label (phase 1 is a mechanical refactor
      with no user-visible change; the actual v0.2.0 minor tag is
      reserved for the release IMPL)
- [ ] PR title: `feat: lift parser and theme into pkg/`
- [ ] PR body references DESIGN-0002 § "pkg/parser (phase 1)" and
      "pkg/theme (phase 1)"

#### Success Criteria

- `make build && make test && make lint` all green on the branch
- `find . -path ./docs -prune -o -name '*.go' -print | xargs grep -l "internal/parser\|internal/theme"` returns nothing (all
  internal references rewritten)
- `git log --follow pkg/parser/parser.go` shows pre-move history
  (lift preserves provenance)
- Manual smoke: live reload and Neovim scroll sync behave identically
  to `main`
- Code-coverage report (`make test-coverage`) within 1% of `main`
  baseline (no test deletions, no untested new code). Closing any
  pre-existing coverage gaps on `pkg/parser` and `pkg/theme` to the
  100%-on-exported standard is deferred to IMPL-0004 phase 3

---

### Phase 2: Extract pkg/livereload from internal/server

**Branch:** `feat/extract-livereload-package`
**Estimated diff:** ~10 files (new package + refactor + tests).
**Depends on:** Phase 1 merged to `main`.

This is the load-bearing phase. The byte-for-byte wire compatibility
requirement means tests are the critical artifact.

#### Tasks

**Create pkg/livereload skeleton**

- [ ] `mkdir pkg/livereload`
- [ ] Create `pkg/livereload/hub.go` with the public `Hub` type and
      method stubs (`NewHub`, `Broadcast`, `Count`, `Close`,
      `HandleWebSocket`, `HandleSSE`) per DESIGN-0002 § Hub
- [ ] Create `pkg/livereload/handler.go` with `WrapHandler`,
      `HandlerOption`, `WithWSPath`, `WithSSEPath`,
      `WithInjectionPoint`, `WithClientJS` per DESIGN-0002 § WrapHandler
- [ ] Create `pkg/livereload/client.js` with the reload-only script
      (extract from `assets/preview.js` lines 165-213, simplified to
      pure reload — no DOM swap, no cursor sync; consumers wanting
      smarter behavior can replace via `WithClientJS`)
- [ ] Create `pkg/livereload/clientjs.go` with `//go:embed client.js`
      to expose `var ClientJS string`

**Implement Hub (with concurrency fix)**

- [ ] Define internal `wsClient` struct holding `*websocket.Conn` and
      a `send chan []byte` (buffer size 8 — matches the existing SSE
      buffer in `sse.go:79`; revisit with `WithSendBuffer(n)` if
      profiling later shows drops)
- [ ] `Hub` stores `wsClients map[*wsClient]struct{}` and
      `sseClients map[chan []byte]struct{}` under a single `sync.RWMutex`
- [ ] `HandleWebSocket`: upgrade, register a `*wsClient`, spawn one
      writer goroutine that reads from `send` and calls
      `conn.WriteMessage(websocket.TextMessage, ...)` (closes the
      gorilla concurrent-write race documented in DESIGN-0002 §
      "Hub concurrency"). Reader goroutine reads pings/close and
      triggers cleanup on disconnect
- [ ] `HandleSSE`: lift from `internal/server/sse.go:66-99` verbatim
      (it already uses per-client `chan []byte`, no race) — just
      adjust the `Hub` field references
- [ ] `Broadcast`: `RLock`, range over both client maps, non-blocking
      `select { case c.send <- msg: default: removeQueue = append(...) }`
      for WS; same shape for SSE. Drain `removeQueue` after `RUnlock`
- [ ] `Count`: return `len(wsClients) + len(sseClients)` under `RLock`
- [ ] `Close`: lock, close all `send` channels (WS) and SSE channels,
      clear both maps. Return `error`. Real error paths to surface:
      `websocket.Conn.Close()` failures (rare but real on already-
      half-closed sockets). If any per-connection close returns an
      error, capture the first one and return it; continue closing
      the rest (don't short-circuit on first error)
- [ ] Move the `websocket.Upgrader{}` definition from
      `internal/server/server.go:128-131` into `Hub` as a private
      field with hardcoded permissive `CheckOrigin` (`func(_ *http.Request) bool { return true }`).
      Document in `Hub`'s GoDoc that `livereload` is intended for
      local-only use; a future `WithCheckOrigin(func)` option can be
      added if a non-local consumer materializes

**Implement WrapHandler**

- [ ] `handlerConfig` struct with `wsPath`, `ssePath`,
      `injectionPoint`, `clientJS`, defaults `"/ws"`, `"/events"`,
      `"</body>"`, `livereload.ClientJS`
- [ ] `WrapHandler` returns an `http.Handler` (per DESIGN-0002,
      *not* `*Handler` — supersedes RFC sketch). Logs a `slog.Warn` at
      construction if `WithWSPath` or `WithSSEPath` are used without
      `WithClientJS`
- [ ] Routing: if `r.URL.Path == cfg.wsPath`, delegate to
      `hub.HandleWebSocket`; if `cfg.ssePath`, delegate to
      `hub.HandleSSE`; otherwise wrap response writer to detect
      `Content-Type: text/html` and inject `<script>{cfg.clientJS}</script>`
      before `cfg.injectionPoint`. If the marker isn't present in the
      body, skip injection and `slog.Warn("livereload: injection point not found", "path", r.URL.Path)`

**Refactor internal/server to consume pkg/livereload**

- [ ] In `internal/server/server.go`:
  - [ ] Remove `hub *hub` and `sse *sseHub` fields (lines ~49-50);
        replace with single `hub *livereload.Hub`
  - [ ] In `New()` (line ~125-126): replace
        `hub: newHub(), sse: newSSEHub()` with
        `hub: livereload.NewHub()`
  - [ ] Move `upgrader websocket.Upgrader` initialization out (now
        owned by `Hub`)
- [ ] In `Server.Broadcast` (line ~169-170): collapse
      `s.hub.broadcast(msg); s.sse.broadcast(msg)` to
      `s.hub.Broadcast(msg)`
- [ ] In `Server.SendCursor` (line ~180-181): same collapse
- [ ] In `Server.Close` (line ~196-197): replace
      `s.hub.closeAll(); s.sse.closeAll()` with `_ = s.hub.Close()`
- [ ] In `ListenAndServe` (line ~230-232): replace
      `mux.HandleFunc("GET /ws", s.handleWebSocket)` with
      `mux.HandleFunc("GET /ws", s.hub.HandleWebSocket)`. Same for
      `/events` → `s.hub.HandleSSE`
- [ ] Delete `Server.handleWebSocket` (server.go ~389-414) and
      `Server.handleSSE` (sse.go ~66-end) — superseded by `Hub` methods
- [ ] Update `idleWatcher` (server.go ~280) to call `s.hub.Count()`
      instead of `s.hub.count() + s.sse.count()`
- [ ] `rm internal/server/hub.go internal/server/sse.go`
- [ ] Update `internal/server/server.go` imports to add
      `github.com/donaldgifford/mdp/pkg/livereload` and remove
      `github.com/gorilla/websocket` (now transitive through
      `pkg/livereload`)

**New tests under pkg/livereload**

- [ ] `pkg/livereload/hub_test.go`:
  - [ ] `TestHub_BroadcastDeliversToWSClient`: dial via WS, broadcast
        a payload, assert received
  - [ ] `TestHub_BroadcastDeliversToSSEClient`: connect via SSE,
        broadcast, assert received
  - [ ] `TestHub_CountReflectsConnections`: connect N WS + M SSE,
        assert `Count() == N+M`
  - [ ] `TestHub_CloseDropsAllClients`: connect, close, assert WS
        receives close frame and SSE channel closes
  - [ ] `TestHub_ConcurrentBroadcastNoRace` (runs under `-race`):
        spawn N goroutines firing `Broadcast` while a concurrent
        goroutine connects/disconnects clients. Must pass under
        `go test -race`
  - [ ] `TestHub_SlowClientDropsMessage`: connect a WS client that
        doesn't read, broadcast 100 messages, assert the slow client
        is dropped without blocking other clients
- [ ] `pkg/livereload/handler_test.go`:
  - [ ] `TestWrapHandler_InjectsClientJSIntoHTML`: wrap a handler
        returning `<html><body>x</body></html>`, request, assert
        response contains `<script>...</script></body>`
  - [ ] `TestWrapHandler_DoesNotInjectIntoNonHTML`: wrap a handler
        returning JSON, assert response is unchanged
  - [ ] `TestWrapHandler_SkipsInjectionWhenMarkerMissing`: wrap a
        handler returning `<html>no body close</html>`, assert
        response unchanged and a warning is logged
  - [ ] `TestWrapHandler_ServesWSAndSSEAtDefaultPaths`: assert `/ws`
        and `/events` reach the hub
  - [ ] `TestWrapHandler_HonorsCustomPaths`: with
        `WithWSPath("/socket")` + `WithSSEPath("/stream")` +
        `WithClientJS("...")`, assert the routes work
- [ ] `internal/server/wire_test.go` (regression test — lives in
      `internal/server` because it tests the integration of mdp's
      `wsMessage` JSON marshaling with `pkg/livereload`'s transport;
      `pkg/livereload` itself has no opinion on the bytes):
  - [ ] `TestWireFormat_WSFrameMatchesPriorBaseline`: golden-file
        test that captures the exact bytes broadcast for a
        canonical `wsMessage{Type:"content",HTML:"<p>hi</p>"}` and
        asserts equality with a checked-in `testdata/ws_content.bin`
  - [ ] Same for the `wsMessage{Type:"cursor",Line:42}` shape →
        `testdata/ws_cursor.bin`

**Verify and tidy**

- [ ] All existing tests in `internal/server/` pass unchanged
      (livereload_test.go, scrollsync_test.go, idletimeout_test.go,
      features_test.go, server_test.go, stdin_test.go)
- [ ] All existing tests in `pkg/parser/`, `pkg/theme/` pass unchanged
- [ ] `make test-coverage` clean (CI's existing race-detected path
      covers the new `TestHub_ConcurrentBroadcastNoRace`)
- [ ] `make fmt`, `make lint`, `make build` all clean
- [ ] Manual smoke: `./bin/mdp --file README.md` — open in browser,
      edit, confirm live reload still works
- [ ] Manual smoke: switch from WS to SSE by blocking WS in browser
      DevTools, edit file, confirm SSE fallback still works
- [ ] Neovim smoke: `:MdpPreview`, edit, confirm reload + scroll sync
      still work

**PR**

- [ ] Open PR with `patch` label (see Phase 1 PR notes)
- [ ] PR title: `feat: extract pkg/livereload from internal/server`
- [ ] PR body references DESIGN-0002 § "pkg/livereload (phase 2)" and
      explicitly notes the gorilla concurrent-write race fix as a
      side-effect

#### Success Criteria

- `make build && make test && make lint` all green on the branch
- `go test -race ./...` clean (zero race-detector reports)
- All existing `internal/server` tests pass *without modification*
  (proves wire compatibility from inside the server)
- `internal/server/wire_test.go` golden-file test passes (proves wire
  compatibility from outside `pkg/livereload`; locks the bytes for
  future refactors)
- `internal/server/hub.go` and `internal/server/sse.go` no longer
  exist
- `internal/server/server.go` no longer imports
  `github.com/gorilla/websocket` directly (transitive only)
- Manual smoke (mdp CLI + Neovim) behaves identically to `main`
- **100% line coverage on exported symbols** in `pkg/livereload`
  (matches the standard for public packages set in IMPL-0004
  phase 3). Defensive gaps annotated with `// coverage: <reason>`.
  Project-wide coverage threshold (currently 60% per CLAUDE.md)
  stays unchanged — the bar is raised only for the public packages

---

## Cross-Phase Concerns

- **No CHANGELOG hand-edits.** `git-cliff` regenerates `CHANGELOG.md`
  in the release workflow; per-PR changelog entries are not required.
  The `feat:` commit prefix triggers a Features section automatically.
- **Phase PRs use `patch` labels** (resolved per Open Question 1).
  Each phase bumps the patch version; the actual v0.2.0 minor tag
  happens in IMPL-0004 phase 4 when the library is announced. If
  the running patch count gets uncomfortable mid-rollout, switch
  later PRs to `dont-release`.
- **Pre-existing `internal/server` test files (server_test.go,
  features_test.go) use `package server_test`** and import
  `github.com/donaldgifford/mdp/internal/server`. These remain
  unchanged across both phases.
- **Neovim plugin (lua/mdp + lazy.lua + build.lua) is untouched.**
  The Go binary path doesn't move (still `cmd/mdp/main.go`); only
  internal imports change.

## Open Questions

None remaining. All eight questions raised during drafting are
captured below under [Resolved](#resolved).

### Resolved

1. **PR label.** Use `patch` for Phase 0/1/2 PRs. v0.2.0 minor tag is
   reserved for IMPL-0004 phase 4. If patch count gets uncomfortable
   mid-rollout, fall back to `dont-release` on later PRs.
2. **Wire-regression test location.** `internal/server/wire_test.go`
   (not `pkg/livereload/`). Reflected in Phase 2 tasks above.
3. **`Hub.Close() error` return type.** Keep `error`. Real error
   paths exist (e.g., `websocket.Conn.Close()` on a half-closed
   socket). Capture the first per-connection error, continue closing
   the rest, return the captured error.
4. **`CheckOrigin` configurability.** Hardcoded permissive
   (`func(_ *http.Request) bool { return true }`) for v1. Document
   in `Hub`'s GoDoc that `livereload` is intended for local-only
   use. Add `WithCheckOrigin(func)` later if a real non-local
   consumer materializes.
5. **Channel buffer size.** Keep 8 (matches existing `sse.go:79`).
   Add `WithSendBuffer(n int)` option later if profiling shows drops.
6. **`-race` in CI.** Already covered — `make test-coverage`
   (Makefile:52-54) runs with `-race` and is what CI invokes for
   the `Test Go` job. The original Q6 answer assumed `-race` needed
   wiring; on closer inspection it's already there. Phase 0 in this
   IMPL is reduced to an optional `test-race` make target for local
   dev convenience.
7. **`internal/cli/` import sweep.** Made explicit as a `grep` task
   in Phase 1 above.
8. **Phase 2 PR splitting.** Keep Phase 2 as a single PR. The
   wire-regression test (Phase 2 §Tests) is the load-bearing
   guarantee, not PR size.

## References

- [DESIGN-0002 — Refactor mdp internals into public pkg packages](../design/0002-refactor-mdp-internals-into-public-pkg-packages.md)
  — detailed design driving this implementation
- [RFC-0001 — Public mdp Go library](../rfc/0001-public-mdp-go-library.md)
  — phases 1–4 overview
- [INV-0002 — Expose mdp as a Go library for other apps](../investigation/0002-expose-mdp-as-go-library.md)
  — analytical basis
- `internal/server/hub.go:37-46` — current WS `broadcast` with the
  latent concurrent-write race (phase 2 closes this)
- `internal/server/sse.go:66-99` — current SSE handler (lifted
  largely verbatim into `pkg/livereload`)
- `internal/server/server.go:125-126` — current hub construction
- `internal/server/server.go:169-184` — current `Broadcast` /
  `SendCursor` fan-out (collapses to one `Hub.Broadcast` call)
- `internal/server/server.go:230-232` — current route registration
- `assets/preview.js:165-213` — source of the reload-only slice that
  becomes `pkg/livereload/client.js`
- `.github/workflows/release.yml` — release workflow; informs Q1
  (label choice) and Q6 (CI step for -race)
- `CLAUDE.md` § Git Workflow + CI/CD — branch naming, PR labels,
  coverage targets
