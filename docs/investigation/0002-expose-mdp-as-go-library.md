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

The server and watcher are less obviously library-friendly: the server
embeds an opinionated WebSocket/SSE hub and assumes browser clients;
the watcher is so thin (fsnotify + debounce) that consumers are better
off writing their own. So the right move is probably **lift `parser`
and `theme` out of `internal/`** while leaving `server`, `watcher`, and
`cli` internal.

## Context

`mdp` is structured as a CLI binary with everything under `internal/`,
which Go enforces as non-importable from outside the module. Anyone
wanting to embed mdp's markdown rendering into their own Go app today
has no choice but to `os/exec` the binary and parse its output. That
costs a process boundary, a stdin/stdout protocol, and the ability to
compose with the consumer's own goldmark extensions.

The question came up while thinking about whether a separate Go app
could reuse mdp's renderer directly.

**Triggered by:** ad-hoc product question — no parent RFC/DESIGN yet.
This investigation is the input for one.

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
| `assets` | Already public | Already public | — | Keep as-is |
| `watcher` | Low — 79 lines wrapping fsnotify with a debounce; consumers can roll their own in 20 lines | Low | None | Keep internal |
| `server` | Medium — but the server bakes in a specific HTTP/WS contract, embeds `assets`, owns the lifecycle; library consumers usually want building blocks, not "start a server for me" | High — every flag, route, and JSON message becomes public contract | Imports `parser`, `theme`, `assets`, plus stdin/cursor protocol | Keep internal for v1; revisit if demand appears |
| `cli` | None | n/a | Cobra; CLI-only | Keep internal forever |

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

**Answer:** Yes — exposing the library is straightforward, but only a
subset of internal packages is worth lifting. Concretely:

- `parser` and `theme` should be promoted to public; their existing API
  shapes already pass library design review.
- `server`, `watcher`, and `cli` should stay internal.
- `assets` is already public and needs no change.
- **Option A** (`pkg/parser`, `pkg/theme`) or **Option B** (`./parser`,
  `./theme`) are both workable; Option B is more idiomatic but
  clutters the root.

## Recommendation

Promote this investigation into an **RFC** (`RFC-XXXX: Public mdp Go
library`) covering:

1. Pick **Option B** (root-level packages) unless there's a strong
   reason for `pkg/` — the directory is small and library packages
   read most cleanly as `mdp.Parser`-adjacent imports.
2. Lift `internal/parser` -> `parser`; `internal/theme` -> `theme`.
   Update internal imports in `cli`, `server`, etc. — small mechanical
   change (~15 files).
3. Before tagging v0.2.0:
   - Add `WithMermaidRenderMode(mode)` option to `parser` so library
     consumers aren't locked to client-side rendering.
   - Audit `theme.Theme` exported fields — consider whether `IsAuto`,
     `HljsVendorCSS`, `MermaidTheme` belong in the public surface or
     should be hidden behind accessors.
   - Write a doc.go for each public package with a usage example.
4. Add a `Library` section to `README.md` showing a 10-line example of
   another Go app rendering markdown via `mdp.Parser`.
5. **Defer** exposing `server` until a concrete consumer asks for it —
   exposing too much locks in API surface that's expensive to remove.

A follow-up **ADR** may be useful to record the decision to keep
`server`, `watcher`, and `cli` internal, so the rationale survives
future "why isn't X exported?" questions.

## References

- `internal/parser/parser.go` — current parser package; already
  library-shaped
- `internal/theme/theme.go` — theme registry; already library-shaped
- `internal/server/server.go` — current server; tightly coupled to
  WS/SSE hub and CLI lifecycle
- `internal/watcher/watcher.go` — fsnotify wrapper; too thin to
  justify exposing
- `assets/assets.go` — already public `embed.FS`
- Go spec on `internal/` packages — https://go.dev/doc/go1.4#internalpackages
- Common debate on `pkg/` vs root packages — Go community
  discussions; both are valid choices
