# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**mdp** is a Go-based markdown preview server for Neovim with live reload and scroll synchronization. A single Go binary serves rendered markdown to a browser via WebSocket/SSE with Mermaid, KaTeX, and highlight.js rendered client-side. All assets are embedded in the binary.

## Build & Development Commands

Tool versions are managed with `mise` (see `mise.toml`). Run `mise install` to set up the environment.

```bash
make build          # Build binary with version info via ldflags
make lint           # golangci-lint v2 (Uber Go Style Guide)
make fmt            # Format code
make test           # Run all tests
make test-coverage  # Run tests with coverage output
make update-vendor  # Update vendored JS libraries from CDN

# Run a single test
go test ./internal/parser/ -run TestSpecificName -v

# Run benchmarks
go test -bench=. ./internal/parser/
```

## Architecture

```
cmd/mdp/         -> CLI entrypoint (cobra with --version, --verbose)
internal/
  cli/           -> Root and serve commands with flag handling
  parser/        -> Goldmark pipeline: GFM, highlighting, mermaid, math, line annotations
  server/        -> HTTP server, WebSocket/SSE hubs, stdin reader, auth token
  watcher/       -> fsnotify file watcher with 50ms debounce
assets/          -> Embedded: preview.html/css/js + vendored Mermaid, KaTeX, highlight.js
lua/mdp/         -> Neovim plugin: setup(), MdpStart/Stop/Toggle/Open commands
```

**Data flow:** Neovim buffer -> Lua plugin -> stdin JSON -> Go binary -> goldmark parse -> WebSocket/SSE hub -> browser. Browser handles Mermaid, KaTeX, highlight.js client-side.

**Dual input modes:** Editor plugin pipes buffer via stdin (`--stdin` flag); standalone CLI watches file on disk via fsnotify.

**Scroll sync:** Block-level HTML elements get `data-source-line="N"` attributes via goldmark AST transformer. Browser finds nearest annotated element for cursor position and smooth-scrolls to it. Cursor updates flow via `POST /cursor` endpoint or stdin JSON.

**JSON protocol (WS/SSE):** `{"type":"content","html":"..."}` for content updates, `{"type":"cursor","line":N}` for scroll sync.

**Stdin protocol (Neovim -> binary):** Newline-delimited JSON: `{"type":"content","data":"..."}` and `{"type":"cursor","line":N}`.

## Code Style

- **Uber Go Style Guide** enforced via `.golangci.yml` with 30+ linters
- Complexity limits: cyclomatic 15, cognitive 30, function length 100 lines
- Import order: stdlib -> third-party -> `github.com/donaldgifford/mdp`
- `nolint` directives require explanation and specific linter name
- Test files have relaxed linting (errcheck, funlen, gocyclo, gosec excluded)
- Coverage target: 60% (minimum 40%)

## Key Dependencies

- `github.com/yuin/goldmark` + extensions (GFM, mermaid, highlighting, mathjax)
- `github.com/gorilla/websocket` -- WebSocket server
- `github.com/fsnotify/fsnotify` -- file watching
- `github.com/spf13/cobra` -- CLI framework

## CI/CD

GitHub Actions: lint -> test -> build on push/PR. Releases use GoReleaser with GPG signing and semver (PR labels: `major`, `minor`, `patch`, `dont-release`).
