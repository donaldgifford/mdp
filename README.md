# mdp

A fast markdown preview server for Neovim with live reload, scroll sync, and
client-side rendering of Mermaid diagrams, KaTeX math, and syntax highlighting.
All assets are embedded in the binary -- no CDN requests at runtime.

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
    idle_timeout_secs = 30, -- Shut down after N seconds with no open tab (0 = disabled)
    log_file = vim.fn.stdpath("log") .. "/mdp.log", -- "" to disable
  },
}
```

When `theme` is empty (the default), the plugin resolves the theme from
`vim.o.background`: `dark` → `github-dark`, `light` → `github-light`,
unset → `auto` (browser `prefers-color-scheme`). Set `theme` to any
built-in name (e.g. `"tokyo-night"`) to pin a specific theme regardless
of background setting.

On install/update, `build.lua` downloads a pre-built binary from GitHub
releases. If no release is available (e.g., testing a branch), it falls
back to building from source with `go build`.

### Commands

| Command        | Description                                              |
| -------------- | -------------------------------------------------------- |
| `:MdpPreview`  | Show preview — starts if needed, otherwise syncs buffer  |
| `:MdpStop`     | Stop the preview server                                  |
| `:MdpStart`    | Start the preview server explicitly                      |
| `:MdpToggle`   | Toggle start/stop                                        |
| `:MdpOpen`     | Re-open the browser tab without restarting               |
| `:MdpInstall`  | Download latest release binary                           |
| `:MdpInstall!` | Build binary from source                                 |

The default keybinding is `<leader>mp` → `:MdpPreview`. This is the only
key you need for day-to-day use. `:MdpStop` is available if you want to
shut down explicitly rather than waiting for the idle timeout.

### Idle Timeout

By default the server shuts down automatically 30 seconds after the last
browser tab is closed. This prevents orphaned processes when switching
between tmux sessions or Neovim instances. Set `idle_timeout_secs = 0`
to disable.

### Logging

Server output is written to `~/.local/state/nvim/mdp.log` by default
(XDG-compliant, same directory as other Neovim logs). Each session is
delimited by start/end markers so multiple runs are easy to distinguish.

```bash
# Watch logs in real time
tail -f ~/.local/state/nvim/mdp.log
```

Set `log_file = ""` in `opts` to disable logging.

### How It Works

The plugin starts `mdp serve --stdin <file>` as a background job. Buffer
content is sent over stdin as newline-delimited JSON on every save and
during insert mode (debounced). Cursor position is sent on every cursor
movement (throttled) for scroll sync.

## CLI Reference

```
mdp serve [flags] <file>
```

### Flags

| Flag                | Default | Description                                              |
| ------------------- | ------- | -------------------------------------------------------- |
| `--port`            | `0`     | Port to listen on (0 = auto-assign)                      |
| `--browser`         | `true`  | Open browser automatically                               |
| `--theme`           | `auto`  | Built-in theme name, `auto`, or path to CSS file         |
| `--hljs-theme`      | `""`    | Path to custom hljs CSS (only with `--theme=<file>`)     |
| `--scroll-sync`     | `true`  | Enable scroll sync via cursor tracking                   |
| `--stdin`           | `false` | Read content/cursor updates from stdin                   |
| `--css`             | `""`    | Path to custom CSS file appended after theme CSS         |
| `--open-to-network` | `false` | Listen on `0.0.0.0` instead of `localhost`               |
| `--idle-timeout`    | `30s`   | Shut down after no clients for this duration (0=disabled)|
| `-v, --verbose`     | `false` | Enable debug logging                                     |
| `--version`         |         | Print version, commit, and build date                    |

## Supported Markdown Features

- **GitHub Flavored Markdown**: tables, task lists, strikethrough, autolinks
- **Syntax highlighting**: all languages supported by highlight.js with GitHub
  light/dark themes
- **Mermaid diagrams**: flowcharts, sequence diagrams, gantt charts, etc.
- **KaTeX math**: inline `$...$` and block `$$...$$` expressions
- **Relative images**: images referenced with relative paths are resolved from
  the markdown file's directory

## Themes

Pass `--theme=<name>` to pin a specific built-in theme, or `--theme=auto`
(the default) to follow the browser's `prefers-color-scheme` setting.

```bash
mdp serve --theme=tokyo-night README.md
mdp serve --theme=/path/to/my-theme.css README.md   # custom CSS file
```

### Built-in themes

| Name | Family | Style |
|------|--------|-------|
| `github-light` | GitHub | Light |
| `github-dark` | GitHub | Dark |
| `github-dimmed` | GitHub | Dark (dimmed) |
| `tokyo-night` | Tokyo Night | Dark |
| `tokyo-night-moon` | Tokyo Night | Dark (blue-tinted) |
| `tokyo-night-storm` | Tokyo Night | Dark (storm) |
| `tokyo-night-day` | Tokyo Night | Light |
| `rose-pine` | Rosé Pine | Dark |
| `rose-pine-moon` | Rosé Pine | Dark (moon) |
| `rose-pine-dawn` | Rosé Pine | Light |
| `catppuccin-latte` | Catppuccin | Light |
| `catppuccin-frappe` | Catppuccin | Dark |
| `catppuccin-macchiato` | Catppuccin | Dark |
| `catppuccin-mocha` | Catppuccin | Dark |

Each built-in theme provides prose styling, syntax-highlighting token
colours, and Mermaid diagram theming in a single embedded CSS file.

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

The lazy.nvim plugin spec handles everything — see [Neovim Plugin](#neovim-plugin)
above. `build.lua` downloads a pre-built binary on install/update.

### Standalone binary

```bash
# Via Go
go install github.com/donaldgifford/mdp/cmd/mdp@latest

# Via Homebrew (when tap is set up)
brew install donaldgifford/tap/mdp

# From GitHub releases
# Download the archive for your platform from the releases page
```

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

Then `:Lazy update mdp`. With no release for the branch, `build.lua` falls
back to building from source. See [CONTRIBUTING.md](CONTRIBUTING.md) for
more details.

## License

MIT
