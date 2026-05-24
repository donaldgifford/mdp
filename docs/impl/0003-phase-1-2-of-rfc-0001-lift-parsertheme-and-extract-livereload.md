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
  - [Phase 1: Lift parser and theme to pkg/](#phase-1-lift-parser-and-theme-to-pkg)
    - [Tasks](#tasks)
    - [Success Criteria](#success-criteria)
  - [Phase 2: Extract pkg/livereload from internal/server](#phase-2-extract-pkglivereload-from-internalserver)
    - [Tasks](#tasks-1)
    - [Success Criteria](#success-criteria-1)
- [Cross-Phase Concerns](#cross-phase-concerns)
- [Open Questions](#open-questions)
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

### Phase 1: Lift parser and theme to pkg/

**Branch:** `feat/lift-parser-theme-to-pkg`
**Estimated diff:** ~15 files touched, mostly import-path rewrites.
This is a mechanical refactor — no behavior change.

#### Tasks

**Move parser package**

- [ ] `git mv internal/parser pkg/parser` (preserves history)
- [ ] Verify `pkg/parser/parser.go` package declaration is still
      `package parser` (no edit needed — package name doesn't change)
- [ ] Verify `pkg/parser/lineannotator.go`, `parser_test.go`,
      `lineannotator_test.go`, `bench_test.go` all moved cleanly

**Move theme package**

- [ ] `git mv internal/theme pkg/theme`
- [ ] Verify `pkg/theme/theme.go` package declaration is still
      `package theme`
- [ ] Confirm `pkg/theme/theme.go` still imports
      `github.com/donaldgifford/mdp/assets` (unchanged — `assets` is
      at module root)

**Rewrite imports**

- [ ] Update `internal/server/server.go` imports:
      `github.com/donaldgifford/mdp/internal/parser` →
      `github.com/donaldgifford/mdp/pkg/parser`
- [ ] Update `internal/server/server.go` imports:
      `github.com/donaldgifford/mdp/internal/theme` →
      `github.com/donaldgifford/mdp/pkg/theme`
- [ ] Check `internal/cli/serve.go` and `internal/cli/root.go` for
      parser/theme imports; update if present
- [ ] Check `cmd/mdp/main.go` for parser/theme imports; update if
      present
- [ ] Sweep with `goimports -w ./...` (or `make fmt`) to reorganize
      import blocks (gci enforces group order)
- [ ] Verify with `grep -r "internal/parser\|internal/theme" .` that
      no stale paths remain (outside docs/)

**Verify and tidy**

- [ ] `make fmt` clean
- [ ] `make lint` clean (gci is the most likely failure surface)
- [ ] `make test` green
- [ ] `make build` produces a working binary
- [ ] Manual smoke: `./bin/mdp --file README.md` — open in browser,
      edit README.md, confirm live reload still works
- [ ] Neovim smoke: `:MdpPreview` on a `.md` file, edit, confirm
      reload + scroll sync still work

**PR**

- [ ] Open PR with `minor` label (per `.github/workflows/release.yml`'s
      `jefflinse/pr-semver-bump` — see Q1 in Open Questions for
      whether this is the right label)
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
  baseline (no test deletions, no untested new code)

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
      a `send chan []byte` (buffer size 8 — see Q5 in Open Questions)
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
      clear both maps. Return `error` — currently always nil, but see
      Q3 in Open Questions
- [ ] Move the `websocket.Upgrader{}` definition from
      `internal/server/server.go:128-131` into `Hub` as a private field
      (or expose `Hub.SetCheckOrigin(func(*http.Request) bool)` if
      consumers need to override — see Q4 in Open Questions)

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
- [ ] `pkg/livereload/wire_test.go` (regression test — see Q2 in
      Open Questions for whether this lives here or in `internal/server`):
  - [ ] `TestWireFormat_WSFrameMatchesPriorBaseline`: golden-file
        test that captures the exact bytes broadcast for a
        canonical `wsMessage{Type:"content",HTML:"<p>hi</p>"}` and
        asserts equality with a checked-in `testdata/ws_content.bin`

**Verify and tidy**

- [ ] All existing tests in `internal/server/` pass unchanged
      (livereload_test.go, scrollsync_test.go, idletimeout_test.go,
      features_test.go, server_test.go, stdin_test.go)
- [ ] All existing tests in `pkg/parser/`, `pkg/theme/` pass unchanged
- [ ] `go test -race ./...` clean (this is **new** — currently `make
      test` doesn't include `-race`. See Q6 in Open Questions for CI
      wiring)
- [ ] `make fmt`, `make lint`, `make build` all clean
- [ ] Manual smoke: `./bin/mdp --file README.md` — open in browser,
      edit, confirm live reload still works
- [ ] Manual smoke: switch from WS to SSE by blocking WS in browser
      DevTools, edit file, confirm SSE fallback still works
- [ ] Neovim smoke: `:MdpPreview`, edit, confirm reload + scroll sync
      still work

**PR**

- [ ] Open PR with `minor` label
- [ ] PR title: `feat: extract pkg/livereload from internal/server`
- [ ] PR body references DESIGN-0002 § "pkg/livereload (phase 2)" and
      explicitly notes the gorilla concurrent-write race fix as a
      side-effect

#### Success Criteria

- `make build && make test && make lint` all green on the branch
- `go test -race ./...` clean (zero race-detector reports)
- All existing `internal/server` tests pass *without modification*
  (proves wire compatibility from inside the server)
- `pkg/livereload/wire_test.go` golden-file test passes (proves wire
  compatibility from outside — locks the bytes for future refactors)
- `internal/server/hub.go` and `internal/server/sse.go` no longer
  exist
- `internal/server/server.go` no longer imports
  `github.com/gorilla/websocket` directly (transitive only)
- Manual smoke (mdp CLI + Neovim) behaves identically to `main`
- Coverage (`make test-coverage`) for the new `pkg/livereload`
  package ≥ 70% (the project's coverage target is 60% with 40%
  minimum per CLAUDE.md; a load-bearing new package warrants more)

---

## Cross-Phase Concerns

- **No CHANGELOG hand-edits.** `git-cliff` regenerates `CHANGELOG.md`
  in the release workflow; per-PR changelog entries are not required.
  The `feat:` commit prefix triggers a Features section automatically.
- **No release tags between phases.** Phase 1 and phase 2 land on
  `main` but v0.2.0 is not tagged until phase 4 (handled by a
  separate IMPL). Use the `dont-release` label on the phase 1/2 PRs
  *only if* the `jefflinse/pr-semver-bump` action would otherwise tag
  v0.2.0 early — see Q1.
- **Pre-existing `internal/server` test files (server_test.go,
  features_test.go) use `package server_test`** and import
  `github.com/donaldgifford/mdp/internal/server`. These remain
  unchanged across both phases.
- **Neovim plugin (lua/mdp + lazy.lua + build.lua) is untouched.**
  The Go binary path doesn't move (still `cmd/mdp/main.go`); only
  internal imports change.

## Open Questions

Implementation-specific questions to resolve before phase 1 starts.
Design-level questions are settled in DESIGN-0002 § Resolved.

1. **`minor` vs `dont-release` label for phase 1/2 PRs.** Per
   `.github/workflows/release.yml`, every push to `main` with a
   `minor` label bumps the version. Do we want phase 1 (mechanical
   lift, no user-facing change) to bump from `v0.1.x` to `v0.2.0`,
   or should both phase 1 and phase 2 carry `dont-release` and the
   actual `v0.2.0` tag happen in phase 4 alongside the README
   Library section? My read of RFC-0001 phase 4 is that v0.2.0
   tagging is reserved for then — so both PRs probably want
   `dont-release`. Confirm?

2. **Wire-regression test location.** DESIGN-0002 § Testing Strategy
   places `wire_test.go` in `pkg/livereload/`. But it specifically
   tests the integration of `internal/server`'s `wsMessage` JSON
   marshaling with `pkg/livereload`'s transport — that's an
   internal/server concern, not a pkg/livereload concern. Should it
   live in `internal/server/wire_test.go` instead? My recommendation:
   yes, move it to `internal/server/`, since `pkg/livereload` itself
   has no opinion on the bytes.

3. **`Hub.Close() error` return type.** Today's `closeAll()` returns
   no error. If `Hub.Close()` will always return `nil`, returning
   `error` is API-design noise that future evolution can't easily
   walk back (removing the return value is a breaking change). My
   recommendation: change DESIGN to `Close()` (no return). Wanted to
   flag because DESIGN currently specifies `error`.

4. **`websocket.Upgrader.CheckOrigin` configurability.** Today's
   server hardcodes `CheckOrigin: func(_ *http.Request) bool { return true }`
   (server.go:130) because it's a local dev tool. After moving the
   upgrader into `Hub`, consumers writing a non-localhost server
   (production preview, perhaps?) might need to restrict origins. Two
   options: (a) hardcode permissive origin in `Hub` and document that
   `livereload` is for local-only use; (b) add `WithCheckOrigin(func)`
   as a hub option. My recommendation: (a) for v1 — keeps API
   minimal; add (b) later if a real use case appears. Confirm.

5. **SSE / WS channel buffer size.** Today: `chan []byte` with
   buffer 8 (sse.go:79, `ch := make(chan []byte, 8)`). DESIGN-0002
   prescribes the same buffer 8 for the new WS sender. Is 8 still
   right after consolidation? It's been sufficient for mdp because
   broadcasts are infrequent (file save → one broadcast). Library
   consumers might broadcast more often. Reasonable defaults to
   consider: 8 (current), 32, 128. My recommendation: keep 8, add a
   future `WithSendBuffer(n int)` option if profiling shows drops.
   Confirm.

6. **Add `make test-race` target and CI step before phase 2 lands.**
   The new `Hub` concurrency design is specifically defended by a
   race-detector test (`TestHub_ConcurrentBroadcastNoRace`). The
   current `make test` target doesn't pass `-race`. Should we add a
   `test-race` make target and a CI step for it in a tiny precursor
   PR (`chore: add -race to CI test step`) so phase 2's regression
   guarantee actually fires in CI, not just locally? My
   recommendation: yes, precursor PR.

7. **Phase ordering of import-path rewrites in `internal/cli/`.**
   `internal/cli/serve.go` likely imports neither parser nor theme
   directly (it imports `internal/server`, which re-exports them
   via constructor). Worth a `grep` before phase 1 to know whether
   any cli changes are needed; the IMPL assumes they may not be.

8. **Branch protection on `feat/extract-livereload-package`.** The
   PR will be large (~10 files, ~500 LoC moved/added). Worth
   considering whether to split the phase 2 PR further: (a) one PR
   for the new `pkg/livereload` package + its tests (no
   `internal/server` changes); (b) a follow-up PR refactoring
   `internal/server` to use it. This would keep each PR smaller and
   make the wire-compat test gate a separate review. Trade-off:
   slower end-to-end, harder to verify in isolation that
   `pkg/livereload` actually works in context until the second PR
   lands. My recommendation: single PR; the wire-regression test
   is the load-bearing guarantee, not PR size. Confirm.

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
