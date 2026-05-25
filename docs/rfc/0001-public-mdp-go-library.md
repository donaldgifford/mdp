---
id: RFC-0001
title: "Public mdp Go library"
status: Accepted
author: Donald Gifford
created: 2026-05-23
---
<!-- markdownlint-disable-file MD025 MD041 -->

# RFC 0001: Public mdp Go library

**Status:** Accepted
**Author:** Donald Gifford
**Date:** 2026-05-23

<!--toc:start-->
- [Summary](#summary)
- [Problem Statement](#problem-statement)
- [Proposed Solution](#proposed-solution)
- [Design](#design)
  - [Package layout](#package-layout)
  - [pkg/parser](#pkgparser)
  - [pkg/theme](#pkgtheme)
  - [pkg/livereload](#pkglivereload)
  - [What stays internal](#what-stays-internal)
  - [Composition patterns](#composition-patterns)
- [Alternatives Considered](#alternatives-considered)
- [Implementation Phases](#implementation-phases)
  - [Phase 1: Lift parser and theme to pkg/](#phase-1-lift-parser-and-theme-to-pkg)
  - [Phase 2: Extract pkg/livereload from internal/server](#phase-2-extract-pkglivereload-from-internalserver)
  - [Phase 2.5: Validate against docz](#phase-25-validate-against-docz)
  - [Phase 3: Harden for v1](#phase-3-harden-for-v1)
  - [Phase 4: Tag v0.2.0](#phase-4-tag-v020)
- [Risks and Mitigations](#risks-and-mitigations)
- [Success Criteria](#success-criteria)
- [References](#references)
<!--toc:end-->

## Summary

Expose mdp's markdown rendering, theming, and live-reload mechanism as
three independent, composable Go packages (`pkg/parser`, `pkg/theme`,
`pkg/livereload`) so that other Go applications — starting with `docz`
— can embed mdp's preview pipeline directly instead of shelling out to
the binary. `internal/server`, `internal/watcher`, and `internal/cli`
remain unexposed.

## Problem Statement

mdp is structured as a CLI binary with all functional code under
`internal/`. The Go compiler enforces that `internal/` packages cannot
be imported from outside the module, so any consumer wanting to embed
mdp's markdown rendering must `os/exec` the binary. That costs a
process boundary, a stdin/stdout protocol, an inability to compose with
the consumer's own goldmark extensions, and complications around binary
distribution and version management.

The first concrete consumer is `docz`. `docz serve` will start a local
preview server with a navigation sidebar (doc types → per-type README →
individual rendered docs like ADRs). docz needs mdp's markdown
rendering and theming, but its own HTTP routing layer — a meaningfully
different server shape from mdp's single-file editor preview. The
current internal layout blocks this use case entirely.

INV-0002 documented the full analysis. This RFC codifies the proposal
that came out of it.

## Proposed Solution

Promote a curated subset of `internal/` packages to a new public `pkg/`
directory:

- `pkg/parser` — goldmark pipeline (markdown → HTML)
- `pkg/theme` — theme registry (built-in themes + resolver)
- `pkg/livereload` — **new** package extracted from `internal/server`,
  holding the reusable broadcast hub, an `http.Handler` middleware
  that injects auto-reload `<script>` + serves WS/SSE, and the
  client-side JS snippet

Each package is independently importable: a consumer who wants
markdown → HTML imports only `pkg/parser`; a consumer who wants mdp's
look adds `pkg/theme`; a consumer who wants browser auto-reload adds
`pkg/livereload`. None of the three depend on each other.

`internal/server`, `internal/watcher`, and `internal/cli` stay
internal. `internal/server` becomes the in-tree reference consumer of
`pkg/livereload`.

## Design

### Package layout

```text
cmd/mdp/                    -- CLI entrypoint (unchanged)
pkg/parser/                 -- NEW (lifted from internal/parser)
pkg/theme/                  -- NEW (lifted from internal/theme)
pkg/livereload/             -- NEW (extracted from internal/server)
internal/server/            -- Slimmed: orchestrator over pkg/livereload + watcher
internal/watcher/           -- Unchanged
internal/cli/               -- Unchanged
assets/                     -- Unchanged (already public)
```

The choice of `pkg/` follows from the existing use of `internal/`:
when a project already signals "non-public" with `internal/`, `pkg/`
is the natural counterpart for the public API. It also keeps the
module root tidy as the project grows.

### pkg/parser

Lifted as-is from `internal/parser`. Public surface:

```go
package parser

type Parser struct { /* ... */ }
type Option func(*config)

func New(opts ...Option) *Parser
func (p *Parser) Render(src []byte) ([]byte, error)

func WithGFM(enabled bool) Option
func WithSyntaxHighlighting(enabled bool) Option
func WithMermaid(enabled bool) Option
func WithMath(enabled bool) Option
func WithCallouts(enabled bool) Option

// NEW for v1 — unlocks server-side mermaid rendering
func WithMermaidRenderMode(mode mermaid.RenderMode) Option
```

Dependencies: goldmark + extensions only. No dependency on `pkg/theme`,
`pkg/livereload`, or `assets`.

### pkg/theme

Lifted as-is from `internal/theme`. Public surface:

```go
package theme

type Theme struct {
    CSS           string
    HljsVendorCSS string
    MermaidTheme  string
    IsAuto        bool
}

func Resolve(name string) (Theme, error)
func Names() []string
```

Dependencies: standard library + `github.com/donaldgifford/mdp/assets`
(already public).

`Theme`'s exported fields are part of the v1 contract. They're audited
in phase 3 — see [Implementation Phases](#implementation-phases).

### pkg/livereload

**New** package, extracted from `internal/server/{hub,sse}.go`. Public
surface:

```go
package livereload

// Hub broadcasts updates to all connected clients (WebSocket + SSE).
type Hub struct { /* ... */ }
func NewHub() *Hub
func (h *Hub) Broadcast(msg []byte)
func (h *Hub) Count() int
func (h *Hub) Close() error
func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request)
func (h *Hub) HandleSSE(w http.ResponseWriter, r *http.Request)

// WrapHandler wraps next, injecting the reload <script> into
// HTML responses and serving WS/SSE endpoints for the hub.
type HandlerOption func(*handlerConfig)

func WrapHandler(next http.Handler, hub *Hub, opts ...HandlerOption) http.Handler
func WithWSPath(path string) HandlerOption    // default: /ws
func WithSSEPath(path string) HandlerOption   // default: /events
func WithInjectionPoint(marker string) HandlerOption // default: </body>
func WithClientJS(script string) HandlerOption       // override when paths change

// ClientJS is the small client-side reload script, exported for
// consumers who want to inject it themselves rather than use
// WrapHandler.
var ClientJS string
```

Boundary types are stdlib (`net/http`, `[]byte`, `string`). No
dependency on `pkg/parser`, `pkg/theme`, or `assets`. External
dependency: `github.com/gorilla/websocket` only.

**Wire format ownership.** `pkg/livereload.Hub.Broadcast` takes raw
`[]byte` and ships them over WS or SSE. The bytes are opaque to the
package — *consumers* define the message shape. mdp's existing
JSON shape (`{"type":"content","html":"..."}`, `{"type":"cursor","line":N}`)
remains mdp's contract with `assets/preview.js`, not
`pkg/livereload`'s contract with library consumers. docz or any
other consumer can broadcast any bytes they want; the bundled
`ClientJS` just calls `location.reload()` on any incoming message
and treats the payload as opaque.

### What stays internal

| Package | Why it stays internal |
|---------|----------------------|
| `internal/server` | Opinionated single-file/stdin/cursor server. Exposing `server.Config` locks in flags and the CLI lifecycle as public API; consumers who want a server want building blocks (`pkg/livereload`), not "start a server for me." Becomes the in-tree reference consumer of `pkg/livereload`. |
| `internal/watcher` | 79 lines wrapping fsnotify with a debounce. No markdown content. Consumers can roll their own in ~20 lines; if it's worth a library, it's a separate library. |
| `internal/cli` | CLI-only by definition. |

This decision will be recorded in a follow-up ADR so the rationale
survives future "why isn't X exported?" questions.

### Composition patterns

| Consumer profile | Imports |
|------------------|---------|
| Static site generator: markdown → HTML, ship to disk | `pkg/parser` only |
| Same but with mdp's look | `pkg/parser` + `pkg/theme` (+ `assets` for CSS) |
| Lambda/API: render markdown on request | `pkg/parser` only |
| docz `serve`: own HTTP routing + nav, wants auto-reload | `pkg/parser` + `pkg/theme` + `assets` + `pkg/livereload` |
| Custom server with own WS protocol (htmx, Turbo, ...) | `pkg/parser` + `pkg/theme` — skip `pkg/livereload` |
| Render-once-to-disk batch tool | `pkg/parser` (+ `pkg/theme` if styled) |

There is no "use mdp" all-or-nothing import. Each package stands alone.

## Alternatives Considered

**Move to module root (`./parser`, `./theme`, `./livereload`).**
Discussed in INV-0002 as Option B. Shorter import paths
(`mdp/parser` vs `mdp/pkg/parser`) and arguably more idiomatic.
Rejected because the project already uses `internal/`; `pkg/` is its
natural counterpart, and the module root stays tidier as the project
grows.

**Expose the full `internal/server`.** Tempting because it's the
"packaged example" of a working server. Rejected because
`server.Config` hardcodes a single-file model (`File`, `Stdin`,
`Cursor`); fitting docz's multi-file/nav use case would require adding
several knobs that bloat the public API while still being the wrong
shape for the next consumer. Worse, *every* config field, route, and
JSON message would become a public contract. `pkg/livereload` captures
the reusable slice without the lifecycle baggage.

**Expose `internal/watcher`.** Rejected because it's a thin wrapper
around fsnotify with no markdown content. Exposing it would create a
permanent ongoing semver cost for negligible value. Consumers who need
"watch a file with debounce" can write it themselves; if there's
enough demand for a library, it belongs in a separate repo, not in
`mdp`.

**Carve a sub-module (`pkg/mdp/go.mod`).** Lets the library version
diverge from the CLI binary. Rejected — meaningful ongoing complexity
(two `go.mod`s, special release tagging like `pkg/mdp/v0.1.0`) without
proportionate benefit at the project's current scale.

**Separate repo (`mdp-go`/`gomdp`).** Cleanest API contract, fully
decoupled lifecycle. Rejected for now — duplication or vendoring,
asset-embedding has to be re-solved, two repos to maintain. Reconsider
if the library audience materially diverges from the CLI audience.

**Status quo (consumers `os/exec` the binary).** Rejected as the
explicit motivation for this RFC.

**Shared-types package (`pkg/mdptypes`, `pkg/core`).** Considered as a
way to share common types across `parser` / `theme` / `livereload`.
Rejected — shared-types packages quickly become dumping grounds and
recreate the coupling that the three-package split is supposed to
eliminate. Each package owns its own types; boundary types stay
stdlib (`io.Reader`, `http.Handler`, `[]byte`, `string`, `embed.FS`).

## Implementation Phases

Each phase is independently shippable and has a concrete acceptance
test.

### Phase 1: Lift parser and theme to pkg/

- `git mv internal/parser pkg/parser`
- `git mv internal/theme pkg/theme`
- Rewrite imports in `cmd/mdp`, `internal/cli`, `internal/server`
  (~15 files).
- Update `package` doc comments to reflect the new path.

**Acceptance criteria:**

- `make build && make test && make lint` clean.
- Manual smoke: `mdp --file <doc>` renders identically to before.
- Neovim plugin still works (Lua side has no Go imports — should be
  no-op).

After this phase, library consumers can already use `pkg/parser` and
`pkg/theme`; the rest of the RFC adds the reload story.

### Phase 2: Extract pkg/livereload from internal/server

- Create `pkg/livereload` with `Hub`, `WrapHandler`, `ClientJS`.
- Extract WS hub (`internal/server/hub.go`) and SSE hub
  (`internal/server/sse.go`) into `pkg/livereload`, unified behind a
  single `Hub` type (DESIGN-0002 specifies the file layout and
  concurrency model).
- Refactor `internal/server` so it constructs a `livereload.Hub`,
  wraps its HTTP handler with `livereload.WrapHandler`, and pushes
  reload broadcasts through `Hub.Broadcast` when the watcher fires.
- Move the auto-reload `<script>` from
  `assets/preview.html`/`preview.js` into `livereload.ClientJS`
  (keeping the existing behavior).

**Acceptance criteria:**

- WS/SSE wire protocol byte-for-byte unchanged (no changes needed in
  `assets/preview.js` or `lua/mdp/init.lua`).
- All existing tests pass, including `livereload_test.go`,
  `scrollsync_test.go`, `idletimeout_test.go`, `features_test.go`.
- `internal/server` still works end-to-end via `mdp --file`.

### Phase 2.5: Validate against docz

Before tagging v0.2.0, validate the public API against the first
external consumer.

- In a docz branch, add a `serve` command that uses `pkg/parser`,
  `pkg/theme`, `assets`, and `pkg/livereload` via a `replace` directive
  pointing at the local mdp checkout.
- Implement docz's nav sidebar + per-type-README + per-doc rendering
  using the public API only — no reach-arounds into `internal/`.

**Acceptance criteria:**

- docz `serve` compiles against the un-tagged mdp packages.
- Nav-and-render UX works end-to-end (browse types, click into ADR,
  see rendered output with mdp's theme).
- Any API friction discovered is fed back into mdp packages *before*
  v0.2.0 is tagged. This is the cheapest moment to break the API.

### Phase 3: Harden for v1

- **Audit:** grep each public package to confirm no exported function
  takes or returns a type from another mdp package. Boundary types
  must be standard library (`io.Reader`, `http.Handler`, `[]byte`,
  `string`, `embed.FS`).
- **`example_test.go` per package**, each importing *only* that
  package. Go's compile step enforces "no hidden cross-deps."
- **`doc.go` per package** with a short usage example.
- **API hardening:**
  - Add `parser.WithMermaidRenderMode(mode)` so library consumers
    aren't locked to client-side mermaid rendering.
  - Audit `theme.Theme` exported fields (`IsAuto`, `HljsVendorCSS`,
    `MermaidTheme`) — decide whether they belong in the public surface
    or behind accessors (e.g., `Theme.IsAuto()`).
  - Document the `pkg/livereload` transport contract explicitly in
    the package doc — the WS frame and SSE event framing are part of
    the public contract; the *payload bytes* are opaque to the
    package and defined by consumers.

### Phase 4: Tag v0.2.0

- Add a `Library` section to `README.md` with two worked examples:
  parser-only (10 lines, markdown → HTML), and parser + theme +
  livereload (the docz `serve` shape).
- Update `CHANGELOG.md` (`feat:` entries trigger v0.2.0 minor bump).
- Tag `v0.2.0`. From this point, the public packages follow semver.

## Risks and Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Wire-protocol drift between `pkg/livereload` and the existing `preview.js` | Phase 2 breaks the browser preview silently | Medium | Acceptance criterion: byte-for-byte unchanged wire; `livereload_test.go` and `scrollsync_test.go` exercise the integration |
| docz's needs reveal that the API surface is wrong (e.g., needs a higher-level "serve a directory" helper) | API churn post-v0.2.0 | Medium | Phase 2.5 validates docz against the un-tagged packages; revise before tagging |
| `theme.Theme` exported fields become a contract we regret (renaming `HljsVendorCSS` would be breaking) | Long-term API debt | Medium | Phase 3 audit; consider replacing fields with accessor methods before v1 |
| Consumers couple to mdp's specific JSON message shape, expecting it from `pkg/livereload` | Frozen protocol the package doesn't own | Low | DESIGN-0002 makes the payload opaque to `pkg/livereload`; consumers define their own message shape. Document this clearly in the package doc. |
| Phase 1 import-path rewrite breaks the Neovim plugin's `build.lua` source-build fallback | Plugin install fails on non-release branches | Low | `build.lua` runs `go build` against whatever's in the tree — passes or fails with the Go build, which phase 1's acceptance covers |
| Asset weight: consumers importing `pkg/theme` pull in `assets`' embedded vendor JS/CSS | Larger consumer binaries | Low | Document in `pkg/theme` doc.go; consumers who don't want the bloat can use `pkg/parser` alone |

## Success Criteria

The RFC is successful if:

1. **Phase 1–2 land without regressions.** Existing mdp users (Neovim
   plugin, standalone CLI) see no behavioral change.
2. **docz `serve` ships using only the public packages.** No
   reach-arounds into `internal/`, no `os/exec`, no duplicated
   goldmark setup.
3. **A third consumer (real or experimental) can be written against
   the public API in under a day.** Validates that the package shapes
   generalize beyond docz's specific needs.
4. **v0.2.0 ships with `doc.go` + `example_test.go` for each public
   package.** Library is documented and the isolation property is
   compile-time enforced.

## References

- INV-0002 — Expose mdp as a Go library for other apps
  (`docs/investigation/0002-expose-mdp-as-go-library.md`) — the
  analytical basis for this RFC
- DESIGN-0002 — Refactor mdp internals into public pkg packages
  (`docs/design/0002-refactor-mdp-internals-into-public-pkg-packages.md`)
  — detailed design for phases 1 and 2
- `internal/parser/parser.go` — package to be lifted in phase 1
- `internal/theme/theme.go` — package to be lifted in phase 1
- `internal/server/hub.go` and `internal/server/sse.go` — code to be
  extracted into `pkg/livereload` in phase 2
- `assets/assets.go` — already public; documented composition target
- Go spec on `internal/` packages —
  https://go.dev/doc/go1.4#internalpackages
