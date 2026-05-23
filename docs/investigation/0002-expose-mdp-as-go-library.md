---
id: INV-0002
title: "Expose mdp as a Go library for other apps"
status: Open
author: Donald Gifford
created: 2026-05-23
---
<!-- markdownlint-disable-file MD025 MD041 -->

# INV 0002: Expose mdp as a Go library for other apps

**Status:** Open
**Author:** Donald Gifford
**Date:** 2026-05-23

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
- [Approach](#approach)
- [Environment](#environment)
- [Findings](#findings)
  - [Current package layout](#current-package-layout)
  - [Existing public API surface (per package)](#existing-public-api-surface-per-package)
  - [Why internal/ blocks import today](#why-internal-blocks-import-today)
  - [Reorganization options](#reorganization-options)
  - [Per-package "should it be public?" scorecard](#per-package-should-it-be-public-scorecard)
  - [Composition rule of thumb](#composition-rule-of-thumb)
  - [Stability and asset-bloat considerations](#stability-and-asset-bloat-considerations)
- [Conclusion](#conclusion)
- [Recommendation](#recommendation)
- [References](#references)
<!--toc:end-->

## Question

What would it take to let another Go application `import
"github.com/donaldgifford/mdp/..."` and reuse mdp's building blocks
(markdown parser, theme registry, preview server, file watcher) without
shelling out to the `mdp` binary — and which subset of the current
internal packages is actually worth exposing?

## Hypothesis

The parser is the highest-value, lowest-friction package to expose: it
already has a clean functional-options API (`parser.New(opts ...Option)`),
no global state, and a single-method surface (`Render([]byte)`). The
theme package is a close second — `Resolve(name) (Theme, error)` and
`Names()` are already self-contained.

The server is *not* a good library surface as-is: `server.Config`
hardcodes a single-file model (`File`, `Stdin`, `Cursor`) and any real
consumer (docz, others) will need a different shape. But the *live
reload mechanism inside the server* — the WS/SSE hub, the auto-reload
`<script>` injection — is exactly the kind of generic, reusable
primitive that warrants its own package. The right move is to **lift
`parser` and `theme`, extract a new `livereload` package from
`server`**, and keep the full `server`, `watcher`, and `cli` packages
internal. Consumers compose only what they need.

## Context

`mdp` is structured as a CLI binary with everything under `internal/`,
which Go enforces as non-importable from outside the module. Anyone
wanting to embed mdp's markdown rendering into their own Go app today
has no choice but to `os/exec` the binary and parse its output. That
costs a process boundary, a stdin/stdout protocol, and the ability to
compose with the consumer's own goldmark extensions.

**First concrete consumer: `docz`.** `docz serve` would start a local
preview server with a navigation sidebar (doc types -> per-type README
-> individual rendered docs like ADRs). This is a meaningfully
*different* server shape from mdp's single-file editor preview:
multi-file routing, nav injection, no stdin/cursor protocol. docz will
want mdp's markdown rendering and theming, but its own HTTP/routing
layer. That use case is what drives the recommendation below to split
the current `server` package into reusable primitives rather than
expose it whole.

**Triggered by:** docz's planned `serve` command. This investigation
is the input for an RFC.

## Approach

1. Inventory each `internal/<pkg>`: what's exported, what depends on
   what, and how tightly it's tied to the CLI loop.
2. Identify Go's mechanical constraints (`internal/` visibility, module
   boundary, embedded assets travel with consumers).
3. Enumerate reorganization options: lift to `pkg/`, lift to module
   root, carve a sub-module, mirror to a separate repo.
4. Score each candidate package on reuse value vs. cost of exposing
   (API stability burden, asset weight, coupling to CLI).
5. Recommend a minimal v0 surface.

## Environment

| Component | Value |
|-----------|-------|
| Module path | `github.com/donaldgifford/mdp` |
| Go version | 1.26.x (`go.mod`) |
| Internal packages | `cli`, `parser`, `server`, `theme`, `watcher` |
| Already public | `assets` (sits at module root, `//go:embed`) |
| Module shape | Single module, no `pkg/`, no sub-modules |

## Findings

### Current package layout

```text
cmd/mdp/                    -- main, importable but useless to consumers
internal/cli/               -- cobra commands (cli.Execute)
internal/parser/            -- goldmark pipeline (Parser, options, Render)
internal/server/            -- HTTP + WS/SSE + stdin reader
internal/theme/             -- built-in theme registry, Resolve, Names
internal/watcher/           -- fsnotify + 50ms debounce
assets/                     -- already public; embed.FS of HTML/CSS/JS/themes/vendor
```

### Existing public API surface (per package)

| Package | Exported types | Exported funcs |
|---------|---------------|----------------|
| `internal/parser` | `Parser`, `Option` | `New`, `WithGFM`, `WithSyntaxHighlighting`, `WithMermaid`, `WithMath`, `WithCallouts`, `(*Parser).Render` |
| `internal/theme` | `Theme` | `Resolve`, `Names` |
| `internal/server` | `Config`, `Server` | `New`, plus `(*Server).Run`/related |
| `internal/watcher` | `Watcher` | `New` (`file`, `debounce`, `onChange`) |
| `assets` | — | `FS` (`embed.FS`) |

All four packages already follow idiomatic Go (functional options on
`parser`, value-returning constructors, no init-time globals). Promoting
them to `pkg/` is largely a `git mv` plus import-path bump.

### Why `internal/` blocks import today

Go enforces that any package under an `internal/` directory can only be
imported by packages within the *same parent of `internal/`*. Because
`internal/` sits at the module root (`github.com/donaldgifford/mdp/internal`),
only code inside `github.com/donaldgifford/mdp/...` can import it. A
consumer's `go build` will refuse with:

```text
use of internal package github.com/donaldgifford/mdp/internal/parser not allowed
```

There is **no flag, build tag, or `replace` directive** that bypasses
this — it's enforced by the compiler. Exposure requires moving the
package out of `internal/`.

### Reorganization options

**Option A — Move selected packages to `pkg/`.**
- `git mv internal/parser pkg/parser`, same for `theme`.
- Pros: explicit "public API" signal; widely understood convention;
  small diff (mostly import rewrites).
- Cons: `pkg/` itself is mildly controversial — some Go guidelines argue
  against it. But the project already uses `internal/` so `pkg/` is the
  natural counterpart.

**Option B — Move to module root.**
- `git mv internal/parser parser`, importable as
  `github.com/donaldgifford/mdp/parser`.
- Pros: shortest, most idiomatic import paths; no extra directory layer.
- Cons: clutters the root directory; mixes "exposed library" packages
  with `cmd/`, `assets/`, `lua/`, `scripts/`, etc.

**Option C — Carve a sub-module.**
- `pkg/mdp/go.mod` declares a separate module so its versioning can
  diverge from the CLI binary.
- Pros: library can move at a different semver pace from the CLI.
- Cons: significant ongoing complexity (two go.mods, careful
  release tagging like `pkg/mdp/v0.1.0`), and the project isn't big
  enough to justify it yet.

**Option D — Separate repo (`mdp-go`/`gomdp`/etc.).**
- Pros: cleanest API contract; library lifecycle fully decoupled.
- Cons: code duplication or vendoring, two repos to maintain, asset
  embedding has to be re-solved.

**Option E — Status quo.**
- Consumers `os/exec` the binary.
- Pros: zero work; current state.
- Cons: process boundary, stdin protocol, no composition with
  consumer's own goldmark extensions, can't drop the binary into a
  library context (servers, lambda handlers, etc.).

### Per-package "should it be public?" scorecard

| Package | Reuse value | API stability cost | Coupling | Verdict |
|---------|-------------|--------------------|----------|---------|
| `parser` | High — any Go app wanting GFM + mermaid + katex + callouts as a single rendering pipeline | Low — already option-based, single `Render` method | None (just goldmark + extensions) | **Expose v1** |
| `theme` | Medium-High — pairs naturally with `parser`; gives consumers the same look as mdp | Low — small surface (`Theme`, `Resolve`, `Names`) | Imports `assets` (already public) | **Expose v1** |
| `livereload` *(new, extracted from `server`)* | High — the WS/SSE hub + auto-reload `<script>` injection is the actually-reusable bit of the server. Consumers (docz, others) want to plug it into their *own* HTTP routing, not adopt mdp's server lifecycle. | Low if scoped to `Hub` + `Handler` + `ClientJS` and nothing else | `net/http` + `gorilla/websocket` | **Expose v1 (as a new package)** |
| `assets` | Already public | Already public | — | Keep as-is |
| `watcher` | Low — 79 lines wrapping fsnotify with a debounce; consumers can roll their own in 20 lines, and "watch a file with debounce" has no markdown content — if it's worth a library, it's a *different* library | Low (signature is small) but a permanent ongoing cost for negligible value | None | **Keep internal** |
| `server` (full, opinionated) | Low for libraries — `server.Config` is single-file-oriented (`File`, `Stdin`, `Cursor`); fitting docz's multi-file/nav use case would require adding several knobs, bloating the public API while still being the wrong shape for the next consumer | High — every config field, route, and JSON message becomes public contract | Imports `parser`, `theme`, `assets`, watcher, stdin protocol | **Keep internal** — splitting `livereload` out captures the reusable part |
| `cli` | None | n/a | Cobra; CLI-only | Keep internal forever |

**Composition rule of thumb.** A consumer picks only what they need:

| Consumer profile | Imports |
|------------------|---------|
| Static site generator: markdown -> HTML, ship to disk | `parser` only |
| Same but with mdp's look | `parser` + `theme` (+ `assets` for serving CSS) |
| Lambda/API: render markdown on request | `parser` only |
| docz `serve`: own HTTP routing + nav, wants auto-reload | `parser` + `theme` + `assets` + `livereload` |
| Custom server with own WS protocol (htmx, Turbo, LiveView, ...) | `parser` + `theme` — skip `livereload` |
| Render-once-to-disk batch tool | `parser` (+ `theme` if styled) — skip `livereload` |

`parser` has no dependencies on the others; `theme` depends only on
`assets`; `livereload` is standalone. There's no "use mdp" all-or-nothing
import — each package stands alone.

### Stability and asset-bloat considerations

- **Asset weight.** `theme` and (indirectly) `parser`'s mermaid renderer
  rely on the vendored JS/CSS in `assets/vendor`. A consumer importing
  just `parser` gets none of that bloat — `parser` produces HTML
  fragments only, no asset embedding. A consumer importing `theme` or
  `assets` *does* pull the embedded files into their binary. Worth
  documenting.
- **Semver promise.** Once a package is out of `internal/`, every
  field rename, signature change, or option removal becomes a
  breaking change under semver. The current Option-based shape of
  `parser` is well-suited to additive evolution; `theme.Theme`'s
  exported fields are slightly riskier (renaming a field is breaking).
- **Mermaid render mode.** `parser.New()` hardcodes
  `mermaid.RenderModeClient` (`internal/parser/parser.go:97`). A
  library consumer might want server-side rendering. Worth adding a
  `WithMermaidRenderMode` option before v1.

## Conclusion

**Answer:** Yes — exposing the library is straightforward, but the
right v1 surface is **three composable packages** (`parser`, `theme`,
`livereload`), not "lift the existing internal layout as-is."
Concretely:

- `parser` and `theme`: promote to public; their existing API shapes
  already pass library design review.
- `livereload`: **new** package extracted from `server`. Holds the
  reusable `Hub`, an `http.Handler`-style `Handler` that wraps a
  consumer's existing handler with auto-reload script injection +
  WS/SSE endpoints, and the small client `ClientJS` snippet.
- `server` (full), `watcher`, `cli`: stay internal. The full server is
  too opinionated for general reuse; `watcher` is too thin and too
  generic to belong in `mdp`'s surface; `cli` is CLI-only by
  definition.
- `assets`: already public, no change.
- **Option B** (root-level packages) is preferred: imports like
  `mdp/parser`, `mdp/theme`, `mdp/livereload` read cleanly.

## Recommendation

Promote this investigation into an **RFC** (`RFC-XXXX: Public mdp Go
library`) covering:

1. Pick **Option B** (root-level packages) unless there's a strong
   reason for `pkg/`.
2. Lift `internal/parser` -> `parser`; `internal/theme` -> `theme`.
   Mechanical refactor (~15 files).
3. Extract `livereload` from `internal/server`:
   - `livereload.Hub` — broadcast hub (currently `internal/server/hub.go`)
   - `livereload.Handler` — `http.Handler` middleware: serves WS at
     `/ws`, SSE at `/sse` (configurable paths), injects the auto-reload
     `<script>` into HTML responses
   - `livereload.ClientJS` — the JS snippet (as `string` or `[]byte`)
     for consumers who want to inject it themselves
   - Keep mdp's existing `internal/server` as the *consumer* of
     `livereload` — demonstrates the package on real code
4. Before tagging v0.2.0:
   - Add `WithMermaidRenderMode(mode)` option to `parser` so library
     consumers aren't locked to client-side rendering.
   - Audit `theme.Theme` exported fields — consider whether `IsAuto`,
     `HljsVendorCSS`, `MermaidTheme` belong in the public surface or
     should be hidden behind accessors.
   - Write a doc.go for each public package with a usage example.
5. Add a `Library` section to `README.md` showing two examples:
   - parser-only (10 lines, markdown -> HTML)
   - parser + theme + livereload (the docz `serve` shape)
6. **Defer** exposing the full `server`, `watcher`, and a higher-level
   "serve a directory" helper until a second consumer asks for them.
   docz will be the forcing function for what the v1 API actually
   needs to look like.

A follow-up **ADR** should record the decisions to (a) keep `server`
internal in favor of `livereload`, and (b) keep `watcher` internal, so
the rationale survives future "why isn't X exported?" questions.

## References

- `internal/parser/parser.go` — current parser package; already
  library-shaped
- `internal/theme/theme.go` — theme registry; already library-shaped
- `internal/server/server.go` — current server; the slice worth
  extracting into a `livereload` package
- `internal/server/hub.go` — broadcast hub; the load-bearing part of
  the live-reload story
- `internal/server/sse.go` — SSE wire; companion to the WS path in `hub.go`
- `internal/watcher/watcher.go` — fsnotify wrapper; too thin and too
  generic to justify exposing
- `assets/assets.go` — already public `embed.FS`
- `cmd/mdp/main.go` — CLI entrypoint; the example for "consumer of the
  internal server"
- Go spec on `internal/` packages — https://go.dev/doc/go1.4#internalpackages
- Common debate on `pkg/` vs root packages — Go community
  discussions; both are valid choices
