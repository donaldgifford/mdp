# mdp

A fast markdown preview server for Neovim with live reload, scroll sync, and
client-side rendering of Mermaid diagrams, KaTeX math, syntax highlighting, and
GitHub-style callouts. All assets are embedded in the binary -- no CDN requests
at runtime.

## Quick Start

```bash
# Install
go install github.com/donaldgifford/mdp/cmd/mdp@latest

# Preview a markdown file
mdp serve README.md
```

The browser opens automatically. Edit the file and save -- the preview updates
instantly.

## Neovim Plugin

### lazy.nvim / LazyVim

Minimal — defaults are provided by the plugin's `lazy.lua`:

```lua
{ "donaldgifford/mdp" }
```

With custom options:

```lua
{
  "donaldgifford/mdp",
  opts = {
    port = 0,               -- 0 = auto-assign
    browser = true,         -- Open browser on start
    theme = "",             -- "" = auto-detect from vim.o.background, or any built-in name
    scroll_sync = true,     -- Sync preview scroll with cursor
    dagre = false,          -- Pin Mermaid diagrams to dagre (false = Mermaid v12 ELK default)
    idle_timeout_secs = 30, -- Shut down after N seconds with no open tab (0 = disabled)
    log_file = vim.fn.stdpath("log") .. "/mdp.log", -- "" to disable
  },
}
```

When `theme` is empty (the default), the plugin resolves the theme from
`vim.o.background`: `dark` → `github-dark`, `light` → `github-light`, unset →
`auto` (browser `prefers-color-scheme`). Set `theme` to any built-in name (e.g.
`"tokyo-night"`) to pin a specific theme regardless of background setting.

On install/update, `build.lua` downloads a pre-built binary from GitHub
releases. If no release is available (e.g., testing a branch), it falls back to
building from source with `go build`.

### Mermaid Layout

Mermaid v12 lays out flowchart, state, class, ER, requirement, and use-case
diagrams with ELK by default, which renders differently than the dagre layout
of Mermaid v11. If you prefer the old layout, set `dagre = true` in `opts`
(or pass `--dagre` to `mdp serve`).

### Mermaid Icon Packs

Architecture diagrams support `pack:icon` references (e.g.
`service api(logos:aws-ecs)[API]`). mdp registers the Iconify `logos`,
`devicon`, and `k8s` (official Kubernetes icons) packs with CDN loaders by
default — packs download lazily in the browser on first use, so icon diagrams
need network at view time and fall back to generic glyphs offline. Browse
available icons at [icones.js.org](https://icones.js.org/). See
`docs/examples/` for working samples, including AWS/EKS topologies.

### Commands

| Command        | Description                                             |
| -------------- | ------------------------------------------------------- |
| `:MdpPreview`  | Show preview — starts if needed, otherwise syncs buffer |
| `:MdpStop`     | Stop the preview server                                 |
| `:MdpStart`    | Start the preview server explicitly                     |
| `:MdpToggle`   | Toggle start/stop                                       |
| `:MdpOpen`     | Re-open the browser tab without restarting              |
| `:MdpInstall`  | Download latest release binary                          |
| `:MdpInstall!` | Build binary from source                                |

The default keybinding is `<leader>mp` → `:MdpPreview`. This is the only key you
need for day-to-day use. `:MdpStop` is available if you want to shut down
explicitly rather than waiting for the idle timeout.

### Idle Timeout

By default the server shuts down automatically 30 seconds after the last browser
tab is closed. This prevents orphaned processes when switching between tmux
sessions or Neovim instances. Set `idle_timeout_secs = 0` to disable.

### Logging

Server output is written to `~/.local/state/nvim/mdp.log` by default
(XDG-compliant, same directory as other Neovim logs). Each session is delimited
by start/end markers so multiple runs are easy to distinguish.

```bash
# Watch logs in real time
tail -f ~/.local/state/nvim/mdp.log
```

Set `log_file = ""` in `opts` to disable logging.

### How It Works

The plugin starts `mdp serve --stdin <file>` as a background job. Buffer content
is sent over stdin as newline-delimited JSON on every save and during insert
mode (debounced). Cursor position is sent on every cursor movement (throttled)
for scroll sync.

## CLI Reference

```
mdp serve [flags] <file>
```

### Flags

| Flag                | Default | Description                                               |
| ------------------- | ------- | --------------------------------------------------------- |
| `--port`            | `0`     | Port to listen on (0 = auto-assign)                       |
| `--browser`         | `true`  | Open browser automatically                                |
| `--theme`           | `auto`  | Built-in theme name, `auto`, or path to CSS file          |
| `--hljs-theme`      | `""`    | Path to custom hljs CSS (only with `--theme=<file>`)      |
| `--scroll-sync`     | `true`  | Enable scroll sync via cursor tracking                    |
| `--dagre`           | `false` | Pin Mermaid diagrams to dagre (default is Mermaid v12 ELK) |
| `--stdin`           | `false` | Read content/cursor updates from stdin                    |
| `--css`             | `""`    | Path to custom CSS file appended after theme CSS          |
| `--open-to-network` | `false` | Listen on `0.0.0.0` instead of `localhost`                |
| `--idle-timeout`    | `30s`   | Shut down after no clients for this duration (0=disabled) |
| `-v, --verbose`     | `false` | Enable debug logging                                      |
| `--version`         |         | Print version, commit, and build date                     |

## Supported Markdown Features

- **GitHub Flavored Markdown**: tables, task lists, strikethrough, autolinks
- **Syntax highlighting**: all languages supported by highlight.js with GitHub
  light/dark themes
- **Mermaid diagrams**: flowcharts, sequence diagrams, gantt charts, etc.
- **KaTeX math**: inline `$...$` and block `$$...$$` expressions
- **GitHub-style callouts**: `> [!NOTE]`, `> [!TIP]`, `> [!IMPORTANT]`,
  `> [!WARNING]`, `> [!CAUTION]` with themed icons and colors
- **Footnotes**: `[^1]` references with `[^1]: ...` definitions, collected
  into a linked endnote list with back-references
- **Relative images**: images referenced with relative paths are resolved from
  the markdown file's directory

## Themes

Pass `--theme=<name>` to pin a specific built-in theme, or `--theme=auto` (the
default) to follow the browser's `prefers-color-scheme` setting.

```bash
mdp serve --theme=tokyo-night README.md
mdp serve --theme=/path/to/my-theme.css README.md   # custom CSS file
```

### Built-in themes

| Name                   | Family      | Style              |
| ---------------------- | ----------- | ------------------ |
| `github-light`         | GitHub      | Light              |
| `github-dark`          | GitHub      | Dark               |
| `github-dimmed`        | GitHub      | Dark (dimmed)      |
| `tokyo-night`          | Tokyo Night | Dark               |
| `tokyo-night-moon`     | Tokyo Night | Dark (blue-tinted) |
| `tokyo-night-storm`    | Tokyo Night | Dark (storm)       |
| `tokyo-night-day`      | Tokyo Night | Light              |
| `rose-pine`            | Rosé Pine   | Dark               |
| `rose-pine-moon`       | Rosé Pine   | Dark (moon)        |
| `rose-pine-dawn`       | Rosé Pine   | Light              |
| `catppuccin-latte`     | Catppuccin  | Light              |
| `catppuccin-frappe`    | Catppuccin  | Dark               |
| `catppuccin-macchiato` | Catppuccin  | Dark               |
| `catppuccin-mocha`     | Catppuccin  | Dark               |

Each built-in theme provides prose styling, syntax-highlighting token colours,
a seven-colour Mermaid diagram palette, and an eight-colour series for
multi-hue diagrams in a single embedded CSS file.
Diagrams share one skin across every theme: flat strokes with no gradient or
shadow, rounded corners, and labels in vendored Inter and JetBrains Mono
(embedded, so they work offline).

### Custom theme files

A file passed with `--theme=/path/to/theme.css` must define the nine prose
properties on `[data-theme]` or `:root`: `--color-fg-default`,
`--color-fg-muted`, `--color-canvas-default`, `--color-canvas-subtle`,
`--color-border-default`, `--color-border-muted`, `--color-accent-fg`,
`--color-danger-fg`, and `--color-success-fg`.

The seven diagram slots are optional. Any slot you leave out is derived from
the prose colours:

| Slot                | Used for                              | Derived from when absent                      |
| ------------------- | ------------------------------------- | --------------------------------------------- |
| `--mermaid-bg`      | Diagram canvas, edge-label backing    | `--color-canvas-default`                      |
| `--mermaid-fg`      | Node, actor, and title text           | `--color-fg-default`                          |
| `--mermaid-line`    | Edges, lifelines, relations           | 50% mix of fg into bg                         |
| `--mermaid-accent`  | Arrowheads, activations               | `--color-accent-fg`                           |
| `--mermaid-muted`   | Edge labels, secondary text           | `--color-fg-muted`                            |
| `--mermaid-surface` | Node, actor, note, and cluster fill   | 3% mix of fg into bg                          |
| `--mermaid-border`  | Node, actor, and cluster stroke       | 20% mix of fg into bg                         |

Diagrams that need several distinct colours — timeline, kanban, and
mindmap sections, gitGraph branches, pie slices, journey sections and
actors, xychart series — read an optional series,
`--mermaid-series-1` through `--mermaid-series-8`. Section fills are faint
tints of these; branches, slices, and rules use them at full strength. A
missing entry falls back to, in order: accent, success, danger, the three
50% mixes of those, muted, and fg.

Flowchart and state nodes can carry a semantic class — `danger`,
`success`, `warning`, or `accent` — with plain Mermaid syntax and no
`classDef`:

```
flowchart TB
    B -->|No| D[Request changes]
    class D danger
```

The node gets a faint tint of the theme's `--color-danger-fg`,
`--color-success-fg`, `--callout-warning-color`, or `--mermaid-accent`
with a full-strength border. Other Mermaid renderers ignore the class.
Edges are not coloured: Mermaid v12 does not apply classes to edges.

Slot values must be six-digit hex (`#1a1b26`); `var()` and `color-mix()` are
not accepted. The Mermaid-specific `--mermaid-primaryColor`-style variables
used before this palette are no longer read.

## Architecture

```
Editor (Neovim)                    Browser
     |                                ^
     | stdin JSON                     | WebSocket/SSE
     | {"type":"content","data":"..."}| {"type":"content","html":"..."}
     | {"type":"cursor","line":N}     | {"type":"cursor","line":N}
     v                                |
   +-----------------------------------+
   |           mdp server              |
   |  +---------+  +---------------+   |
   |  | goldmark |  |  WebSocket + |   |
   |  | parser   |  |  SSE hub     |   |
   |  +---------+  +---------------+   |
   |  +---------+  +---------------+   |
   |  |  file   |  |  /vendor/     |   |
   |  |  watcher |  |  (embedded)  |   |
   |  +---------+  +---------------+   |
   +-----------------------------------+
```

## Install

### Neovim plugin (recommended)

The lazy.nvim plugin spec handles everything — see
[Neovim Plugin](#neovim-plugin) above. `build.lua` downloads a pre-built binary
on install/update.

### Standalone binary

```bash
# Via Go
go install github.com/donaldgifford/mdp/cmd/mdp@latest

# Via Homebrew (when tap is set up)
brew install donaldgifford/tap/mdp

# From GitHub releases
# Download the archive for your platform from the releases page
```

## Library

mdp's markdown parser, theme registry, and live-reload primitive are
also importable as Go packages — useful if you're building your own
preview-style app and want mdp's rendering or live-reload behavior
without shelling out to the CLI. See the package docs for usage:

- [`pkg/parser`](https://pkg.go.dev/github.com/donaldgifford/mdp/pkg/parser)
  — goldmark pipeline (GFM, syntax highlighting, Mermaid, math,
  callouts, footnotes) with `data-source-line` annotations for scroll
  sync.
- [`pkg/theme`](https://pkg.go.dev/github.com/donaldgifford/mdp/pkg/theme)
  — theme registry; resolves built-in names, `auto`, or a CSS file
  path to a Theme struct.
- [`pkg/livereload`](https://pkg.go.dev/github.com/donaldgifford/mdp/pkg/livereload)
  — transport-agnostic `Hub` (WebSocket + SSE) plus a `WrapHandler`
  middleware that injects a reload `<script>` into HTML responses.

| Use case | Import |
|---|---|
| Render markdown → HTML in your own app | `pkg/parser` |
| Same, with mdp's look | add `pkg/theme` |
| Same, with browser auto-reload | add `pkg/livereload` |

[docz](https://github.com/donaldgifford/docz) uses these packages
in its `serve` command — a real-world consumer to look at for
integration patterns.

## Development

```bash
make build          # Build binary with version info
make test           # Run tests
make test-coverage  # Run tests with coverage
make lint           # Run golangci-lint
make fmt            # Format code
make update-vendor  # Update vendored JS libraries from CDN
```

To test a development branch in Neovim, add `branch` to your spec:

```lua
{ "donaldgifford/mdp", branch = "feat/your-branch" }
```

Then `:Lazy update mdp`. With no release for the branch, `build.lua` falls back
to building from source. See [CONTRIBUTING.md](CONTRIBUTING.md) for more
details.

## License

MIT
