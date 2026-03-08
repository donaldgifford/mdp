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
cmd/mdp/           -> CLI entrypoint (cobra with --version, --verbose)
internal/
  cli/             -> Root and serve commands with flag handling
  parser/          -> Goldmark pipeline: GFM, highlighting, mermaid, math, line annotations
  server/          -> HTTP server, WebSocket/SSE hubs, stdin reader, auth token
  theme/           -> Built-in theme registry; Resolve() maps name/path/auto -> Theme struct
  watcher/         -> fsnotify file watcher with 50ms debounce
assets/
  themes/          -> One CSS file per built-in theme (embedded via go:embed)
  (other)          -> preview.html/css/js + vendored Mermaid, KaTeX, highlight.js
lua/mdp/init.lua   -> Neovim plugin: setup(), commands, buffer/cursor sync
lazy.lua           -> Default lazy.nvim plugin spec (main, ft, cmd, opts)
build.lua          -> Auto-run by lazy.nvim on install/update (binary download/build)
scripts/install.sh -> CLI alternative to build.lua (same logic in bash)
```

**Data flow:** Neovim buffer -> Lua plugin -> stdin JSON -> Go binary -> goldmark parse -> WebSocket/SSE hub -> browser. Browser handles Mermaid, KaTeX, highlight.js client-side.

**Dual input modes:** Editor plugin pipes buffer via stdin (`--stdin` flag); standalone CLI watches file on disk via fsnotify.

**Scroll sync:** Block-level HTML elements get `data-source-line="N"` attributes via goldmark AST transformer. Browser finds nearest annotated element for cursor position and smooth-scrolls to it. Cursor updates flow via `POST /cursor` endpoint or stdin JSON.

**JSON protocol (WS/SSE):** `{"type":"content","html":"..."}` for content updates, `{"type":"cursor","line":N}` for scroll sync.

**Stdin protocol (Neovim -> binary):** Newline-delimited JSON: `{"type":"content","data":"..."}` and `{"type":"cursor","line":N}`.

## Neovim Plugin

- `lazy.lua` provides default spec with `main = "mdp"` so lazy.nvim can auto-detect the module for `opts`-based setup
- `build.lua` runs as a coroutine in lazy.nvim's build runner: downloads release binary from GitHub, falls back to `go build` from source
- Binary is installed to `<plugin-dir>/bin/mdp`; `resolve_binary()` in `lua/mdp/init.lua` checks there before `$PATH`
- `scripts/install.sh` is the bash equivalent of `build.lua` for CLI use
- To test a dev branch: add `branch = "feat/your-branch"` to the lazy.nvim spec, then `:Lazy update mdp`
- `:MdpInstall` re-downloads the release binary; `:MdpInstall!` forces a source build

## Theme CSS Format

Each built-in theme lives in `assets/themes/<name>.css` and must follow this structure:

```css
/* Theme comment block with palette reference */

[data-theme="<name>"] {
  /* Prose custom properties — required */
  --color-fg-default:     #hex;
  --color-fg-muted:       #hex;
  --color-canvas-default: #hex;
  --color-canvas-subtle:  #hex;
  --color-border-default: #hex;
  --color-border-muted:   #hex;
  --color-accent-fg:      #hex;
  --color-danger-fg:      #hex;
  --color-success-fg:     #hex;

  /* Mermaid theme variables (theme: 'base') — required */
  --mermaid-primaryColor:        #hex;
  --mermaid-primaryTextColor:    #hex;
  --mermaid-primaryBorderColor:  #hex;
  --mermaid-lineColor:           #hex;
  --mermaid-secondaryColor:      #hex;
  --mermaid-tertiaryColor:       #hex;
  --mermaid-background:          #hex;
  /* ... other mermaid vars */
}

/* Direct scoped hljs token rules — NO CSS variable indirection */
[data-theme="<name>"] .hljs                { background: #hex; color: #hex; }
[data-theme="<name>"] .hljs-keyword        { color: #hex; }
[data-theme="<name>"] .hljs-operator       { color: #hex; } /* MUST differ from keyword */
/* ... other tokens ... */
```

- Use **direct hex values** in hljs rules, not `var(--some-var)`
- `.hljs-keyword` and `.hljs-operator` MUST use different colors — sharing them collapses syntax to a single hue
- Register the theme in `internal/theme/theme.go` `builtinThemes` map via `mustReadThemeCSS()`
- Update theme count assertions in `internal/theme/theme_test.go`

## Code Style

- **Uber Go Style Guide** enforced via `.golangci.yml` with 30+ linters
- Complexity limits: cyclomatic 15, cognitive 30, function length 100 lines
- Import order: stdlib -> third-party -> `github.com/donaldgifford/mdp`
- `nolint` directives require explanation and specific linter name
- Test files have relaxed linting (errcheck, funlen, gocyclo, gosec excluded)
- Coverage target: 60% (minimum 40%)

### Linting Gotchas

- **gocritic `sprintfQuotedString`**: Use `fmt.Sprintf("data-theme=%q", name)` not `fmt.Sprintf(\`data-theme="%s"\`, name)`
- **gocritic `hugeParam`**: Pass large structs by pointer in test helpers (e.g., `*server.Config` not `server.Config`)
- **gci import formatting**: Run `make fmt` after any import changes — gci enforces import group order and will fail lint if out of sync

## Key Dependencies

- `github.com/yuin/goldmark` + extensions (GFM, mermaid, highlighting, mathjax)
- `github.com/gorilla/websocket` -- WebSocket server
- `github.com/fsnotify/fsnotify` -- file watching
- `github.com/spf13/cobra` -- CLI framework

## Git Workflow

**Never commit or push directly to `main`.** Always:
1. Create a branch: `git checkout -b feat/<description>` (or `fix/`, `chore/`, `docs/`)
2. Commit to the branch
3. Push the branch: `git push origin <branch>`
4. Open a PR targeting `main`

`main` is branch-protected — force pushes are rejected by GitHub.

## CI/CD

GitHub Actions: lint -> test -> build on push/PR. License check uses `go-licenses` (goldmark-mathjax is ignored since it declares MIT but has no LICENSE file). Releases use GoReleaser with GPG signing and semver (PR labels: `major`, `minor`, `patch`, `dont-release`). Archive naming: `mdp_<os>_<arch>.tar.gz`.
