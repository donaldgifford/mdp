# mdp — Implementation Plan

## Phase 1: Project Scaffolding & Core Parser

**Goal:** Establish the Go project structure, get goldmark parsing markdown to
HTML, and serve it over HTTP with embedded assets.

### Tasks

- [x] Initialize Go module (`github.com/<owner>/mdp`)
- [x] Set up project directory structure:
  ```
  mdp/
  ├── cmd/mdp/          # CLI entrypoint
  ├── internal/
  │   ├── server/       # HTTP server + WebSocket
  │   ├── parser/       # Goldmark pipeline
  │   ├── watcher/      # File watching
  │   └── plugin/       # Neovim communication protocol
  ├── assets/           # Embedded client-side assets
  │   ├── preview.html
  │   ├── preview.css
  │   ├── preview.js
  │   └── vendor/       # Mermaid, KaTeX, highlight.js
  ├── nvim/             # Lua plugin
  └── go.mod
  ```
- [x] Add goldmark with GFM extensions (tables, strikethrough, task lists,
      autolinks)
- [x] Add goldmark-highlighting for syntax class annotation on code blocks
- [x] Build the HTML template (`preview.html`) with GitHub-like markdown CSS
- [x] Implement `//go:embed` for the `assets/` directory
- [x] Wire up a basic `cobra` CLI: `mdp serve <file>`
- [x] Implement HTTP server that serves the rendered HTML on `localhost:<port>`
- [x] Add `--port` flag (default: auto-assign from ephemeral range)
- [x] Add `--browser` / `--no-browser` flag to control auto-open behavior
- [x] Implement browser auto-open using `os/exec` (respects `$BROWSER`, falls
      back to `xdg-open` / `open` / `start`)
- [x] Write integration test: parse a markdown fixture, assert expected HTML
      output
- [x] Write integration test: start server, HTTP GET `/`, assert 200 with
      rendered content

### Success Criteria

A user can run `mdp serve README.md`, a browser tab opens, and the fully
rendered markdown is displayed with syntax-highlighted code blocks and styled
tables. No live reload yet — just static render and serve.

---

## Phase 2: Live Reload via WebSocket

**Goal:** File changes are automatically pushed to the browser without a full
page refresh. The preview updates in real time.

### Tasks

- [x] Add `gorilla/websocket` dependency
- [x] Implement WebSocket hub (connection registry, broadcast to all clients)
- [x] Add `/ws` endpoint to HTTP server
- [x] Implement client-side WebSocket connection in `preview.js`:
  - Connect on page load
  - Reconnect with exponential backoff on disconnect
  - On message: replace `#content` innerHTML with received HTML
- [x] Add `fsnotify` file watcher
- [x] Wire watcher → goldmark parse → WebSocket broadcast pipeline
- [x] Debounce rapid file changes (50ms window) to avoid excessive re-renders
- [x] Implement graceful shutdown: close WebSocket connections, stop watcher,
      release port
- [x] Add SSE (`/events`) as fallback transport for environments where WebSocket
      is blocked
- [x] Client-side: prefer WebSocket, fall back to SSE automatically
- [x] Add visual indicator in browser when connection is lost (subtle banner)
- [x] Write test: modify watched file, assert WebSocket receives new content
      within 200ms

### Success Criteria

A user runs `mdp serve README.md`, edits the file in any editor, saves, and the
browser preview updates within ~100ms. Connection drops show a visual indicator
and auto-reconnect.

---

## Phase 3: Mermaid, KaTeX & highlight.js Client-Side Rendering

**Goal:** Mermaid diagrams, math expressions, and syntax-highlighted code render
correctly in the browser using embedded JS libraries.

### Tasks

- [x] Download and vendor latest `mermaid.min.js` into `assets/vendor/`
- [x] Download and vendor latest `katex.min.js`, `katex.min.css`, and KaTeX font
      files into `assets/vendor/`
- [x] Download and vendor `highlight.min.js` + commonly used language packs into
      `assets/vendor/`
- [x] Configure goldmark-mermaid extension in client-side render mode (emits
      `<div class="mermaid">` blocks)
- [x] Add KaTeX support: configure goldmark to identify `$...$` (inline) and
      `$$...$$` (block) math delimiters and emit appropriate markup
- [x] Implement client-side rendering pipeline in `preview.js`:
  - After DOM update, call `mermaid.run()` on all `.mermaid` elements
  - After DOM update, call `renderMathInElement()` for KaTeX auto-render
  - After DOM update, call `hljs.highlightAll()` for any un-highlighted code
    blocks
- [x] Handle re-rendering correctly on live reload (Mermaid needs element
      cleanup before re-init)
- [x] Add `--theme` flag: `auto` (OS preference via `prefers-color-scheme`),
      `light`, `dark`
- [x] Implement theme switching in CSS and pass theme preference to Mermaid
      config
- [x] Create a test markdown fixture that exercises all rendering features:
  - GFM table, task list, strikethrough
  - Fenced code block (Go, Python, JS, Bash, YAML, Rust)
  - Mermaid diagram (flowchart, sequence, gantt)
  - Inline and block math
  - Nested blockquotes
  - Images (both URL and relative path)
- [x] Write test: render fixture, assert Mermaid/KaTeX/highlight.js scripts are
      present in served HTML
- [x] Write Makefile target or script to update vendored JS libraries from CDN

### Success Criteria

The test fixture renders with all diagrams, math, and code blocks correctly
displayed. Theme toggle works. Libraries are self-contained in the binary — no
CDN requests at runtime.

---

## Phase 4: Scroll Sync

**Goal:** The browser preview scrolls to match the cursor position in Neovim (or
any editor that provides line numbers).

### Tasks

- [x] Implement a custom goldmark renderer that annotates block-level HTML
      elements with `data-source-line="N"` attributes
  - Headings, paragraphs, list items, code blocks, blockquotes, tables,
    horizontal rules
  - Track source line from goldmark's AST node positions
- [x] Add a `/cursor` endpoint (or extend WebSocket protocol) that accepts
      `{ "line": N }` messages
- [x] Implement client-side scroll logic in `preview.js`:
  - On receiving cursor line, find nearest element where
    `data-source-line <= cursor_line`
  - Smooth scroll to that element
  - Highlight the target element briefly (subtle flash) for visual feedback
- [x] Handle edge cases:
  - Cursor in frontmatter (don't scroll)
  - Cursor past end of document (scroll to bottom)
  - Cursor on an empty line between blocks (snap to nearest block above)
- [x] Add `--scroll-sync` flag (default: `true`) to disable if unwanted
- [x] Write test: send cursor position, assert correct element receives scroll

### Success Criteria

When the cursor moves in the editor, the browser scrolls to the corresponding
section within 50ms. The scroll is smooth and doesn't jump erratically on minor
cursor movements.

---

## Phase 5: Neovim Lua Plugin

**Goal:** A LazyVim-compatible Neovim plugin that provides `:MdpStart`,
`:MdpStop`, `:MdpToggle` commands with automatic buffer sync and cursor
tracking.

### Tasks

- [x] Create `nvim/lua/mdp/init.lua` — plugin entry point with setup function
- [x] Implement process management:
  - Start `mdp` binary as a background job via `vim.fn.jobstart()`
  - Track PID, manage lifecycle
  - Kill on `:MdpStop` or when Neovim exits (`VimLeavePre` autocmd)
- [x] Implement buffer content sync:
  - On `BufWritePost` and `TextChangedI` (debounced), send full buffer content
    to mdp via stdin
  - Protocol: newline-delimited JSON
    `{"type":"content","data":"...","file":"..."}`
- [x] Implement cursor position sync:
  - On `CursorMoved` and `CursorMovedI` (throttled to ~60fps), send cursor line
  - Protocol: `{"type":"cursor","line":N}`
- [x] Define the stdin protocol between plugin and binary:
  - Length-prefixed JSON messages OR newline-delimited JSON
  - Support `content`, `cursor`, and `config` message types
- [x] Implement `:MdpStart` command
  - Resolve `mdp` binary path (`vim.fn.exepath()` or configurable)
  - Start with appropriate flags based on user opts
  - Open browser (or let the binary handle it)
- [x] Implement `:MdpStop` command — graceful shutdown
- [x] Implement `:MdpToggle` command
- [x] Implement `:MdpOpen` command — re-open browser without restarting
- [x] Add LazyVim plugin spec in README with `build` step for `go install`
- [x] Add support for `opts` table: `port`, `browser`, `theme`, `scroll_sync`
- [x] Handle edge cases:
  - Multiple markdown buffers open (switch preview when buffer changes)
  - Binary not found (clear error message with install instructions)
  - Port conflict (retry with different port)
- [ ] Test manually with LazyVim on macOS and Linux

### Success Criteria

A LazyVim user adds the plugin spec, runs `:MdpToggle` on a markdown buffer, and
gets a live-updating browser preview with scroll sync. Switching to a different
markdown buffer updates the preview. Closing Neovim kills the server.

---

## Phase 6: Polish, Distribution & Documentation

**Goal:** Production-ready release with proper error handling, CI, release
binaries, and documentation.

### Tasks

- [x] Add structured logging (`slog` from stdlib) with `--verbose` flag
- [x] Add `--version` flag that prints version, commit SHA, and build date
- [x] Set up GoReleaser for cross-platform binary releases (macOS arm64/amd64,
      Linux arm64/amd64, Windows amd64)
- [x] Create GitHub Actions CI workflow:
  - `go test ./...` on push
  - `golangci-lint` for linting
  - GoReleaser on tag push
- [x] Write comprehensive README.md:
  - Quick start (install + use)
  - Neovim plugin setup for LazyVim, lazy.nvim, packer
  - CLI reference
  - Configuration
  - Supported markdown features
  - Screenshots / GIFs
- [ ] Add Homebrew formula (or tap)
- [x] Add `go install` instructions
- [x] Handle relative image paths: resolve them relative to the markdown file
      and serve via the HTTP server
- [x] Add custom CSS support: `--css style.css` injects user CSS after default
      styles
- [ ] Add frontmatter display option: render YAML frontmatter as a styled table
      at the top
- [x] Performance: benchmark goldmark parse time + WebSocket push for large
      files (target: <50ms for 10K line files)
- [x] Add `--open-to-network` flag for remote dev scenarios (listen on
      `0.0.0.0`, print URL with hostname)
- [ ] Security: when listening on non-localhost, add a random token to the URL
      to prevent unauthorized access
- [ ] Write CONTRIBUTING.md with instructions for updating vendored JS libraries

### Success Criteria

`go install` and `brew install` both work. CI is green. README has a GIF showing
the full workflow. Binary size is under 15MB. Preview of a 5K line markdown file
updates in under 100ms.

---

## Phase 7: Post-MVP Enhancements

**Goal:** Features that make mdp best-in-class but aren't required for initial
release.

### Tasks

- [ ] Table of contents sidebar (generated from heading structure, clickable,
      highlights current section)
- [ ] PlantUML rendering (via public server
      `https://www.plantuml.com/plantuml/svg/` or configurable local endpoint)
- [ ] Export to PDF button in preview (triggers `window.print()` with
      print-optimized CSS)
- [ ] Multiple file tabs (preview tracks the active markdown buffer, tabs show
      recent files)
- [ ] Vim modeline support: parse `<!-- mdp: theme=dark -->` from the markdown
      file
- [ ] Config file support (`~/.config/mdp/config.yaml` or `.mdp.yaml` in project
      root)
- [ ] Neovim floating window preview option (embedded webview via Neovim's
      terminal)
- [ ] Integration with other editors: VS Code extension, Helix plugin, Zed
      extension
- [ ] Emoji rendering (`:emoji_name:` → unicode)
- [ ] Footnote support via goldmark extension

### Success Criteria

Each feature ships independently behind a flag or as a non-breaking addition. No
regressions in core preview performance.

---

## Open Questions

### Architecture

- [ ] **Binary name:** `mdp` is clean and short, but is it taken? Need to check
      `brew`, `apt`, common `$PATH` conflicts. Alternatives: `mdpv` (markdown
      preview), `markd`, `mprev`, `peek`. **Action:** Check name availability
      across package managers before committing.

- [ ] **stdin protocol:** Should the Neovim → mdp communication use
      newline-delimited JSON (simpler, works with `vim.fn.chansend()`) or
      length-prefixed binary frames (more robust for large buffers with embedded
      newlines)? Leaning toward newline-delimited JSON with base64-encoded
      content field to avoid escaping issues.

- [ ] **WebSocket vs SSE as primary transport:** WebSocket is bidirectional
      (which we don't need for browser → server) but more widely supported in
      practice. SSE is simpler but has a 6-connection limit per domain in some
      browsers. Current plan: WebSocket primary, SSE fallback. Is the fallback
      worth the complexity?

### Goldmark / Rendering

- [ ] **Source line annotation accuracy:** Goldmark's AST tracks byte offsets,
      not line numbers directly. Need to verify that converting byte offset →
      line number is reliable across all node types, especially for multiline
      constructs like fenced code blocks and tables. **Action:** Prototype the
      line annotator and test with complex documents.

- [ ] **KaTeX delimiter detection:** Goldmark doesn't have a built-in extension
      for `$...$` math. Options: `goldmark-mathjax` extension (does it work for
      KaTeX too since it just identifies delimiters?), or a custom inline
      parser. **Action:** Evaluate `goldmark-mathjax` to see if it emits generic
      enough markup for KaTeX client-side rendering.

- [ ] **Mermaid re-initialization:** When the DOM is replaced on live reload,
      Mermaid needs its previous SVG output cleaned up before re-rendering. Does
      `mermaid.run()` handle this, or do we need to explicitly remove old SVGs
      before calling it? **Action:** Test Mermaid's behavior when `run()` is
      called on already-rendered elements.

### Neovim Plugin

- [ ] **Buffer sync granularity:** Should the plugin send the entire buffer on
      every change, or use incremental diffs? Full buffer is simpler and
      goldmark parses fast enough that it shouldn't matter for typical files
      (<10K lines). But for very large files, incremental updates would reduce
      IPC overhead. **Action:** Start with full buffer, benchmark with large
      files, add incremental if needed.

- [ ] **`TextChangedI` debounce interval:** Too short and we spam the parser;
      too long and the preview feels laggy. Need to find the sweet spot.
      markdown-preview.nvim uses ~300ms for insert mode. **Action:** Start with
      300ms, make configurable, tune based on feel.

- [ ] **Binary distribution for the plugin:** Should `build` in the lazy.nvim
      spec use `go install` (requires Go toolchain) or download a pre-built
      binary from GitHub releases? Pre-built is better UX but more complex
      plugin code. Consider supporting both: try pre-built first, fall back to
      `go install`. **Action:** Look at how other Go-based Neovim plugins handle
      this (e.g., `telescope-fzf-native` uses cmake, `nvim-treesitter` downloads
      pre-built parsers).

### Distribution

- [ ] **Minimum Go version:** Go 1.22+ for `go:embed` improvements and
      `net/http` routing? Or Go 1.21 for broader compatibility? **Action:**
      Check which goldmark version requires what Go version.

- [ ] **Asset update automation:** Should there be a GitHub Action that
      periodically checks for new Mermaid/KaTeX/highlight.js releases and opens
      a PR to update vendored assets? This would prevent the same staleness
      problem we're solving. **Action:** Design the update script and decide on
      cadence (weekly? monthly?).

- [ ] **Binary size budget:** Embedding Mermaid (~2MB min), KaTeX (~1.5MB min
      with fonts), and highlight.js (~500KB with common languages) puts the base
      binary around 10-15MB. Is this acceptable? Could strip unused highlight.js
      languages to save space. **Action:** Measure actual embedded size after
      Phase 3, decide if trimming is needed.

### Remote Development

- [ ] **SSH forwarding story:** When running Neovim over SSH (as with the
      LazyVim-over-SSH setup on the homelab), the preview server runs on the
      remote machine. Options: SSH port forwarding (`-L`), or have the plugin
      automatically set up the tunnel. This is a common enough scenario that it
      should work well out of the box. **Action:** Document the SSH forwarding
      workflow, consider auto-detection of SSH sessions in the Lua plugin.
