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

With custom options and keybindings:

```lua
{
  "donaldgifford/mdp",
  keys = {
    { "<leader>mp", "<cmd>MdpToggle<cr>", desc = "Toggle markdown preview" },
    { "<leader>mo", "<cmd>MdpOpen<cr>", desc = "Open preview in browser" },
  },
  opts = {
    port = 0,               -- 0 = auto-assign
    browser = true,         -- Open browser on start
    theme = "auto",         -- "auto", "light", or "dark"
    scroll_sync = true,     -- Sync preview scroll with cursor
    idle_timeout_secs = 30, -- Shut down after N seconds with no open tab (0 = disabled)
    log_file = vim.fn.stdpath("log") .. "/mdp.log", -- "" to disable
  },
}
```

On install/update, `build.lua` downloads a pre-built binary from GitHub
releases. If no release is available (e.g., testing a branch), it falls
back to building from source with `go build`.

### Commands

| Command         | Description                              |
| --------------- | ---------------------------------------- |
| `:MdpStart`     | Start the preview server                 |
| `:MdpStop`      | Stop the preview server                  |
| `:MdpToggle`    | Toggle the preview server                |
| `:MdpOpen`      | Re-open the browser (without restart)    |
| `:MdpInstall`   | Download release binary                  |
| `:MdpInstall!`  | Build binary from source                 |

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

| Flag                | Default | Description                            |
| ------------------- | ------- | -------------------------------------- |
| `--port`            | `0`     | Port to listen on (0 = auto-assign)    |
| `--browser`         | `true`  | Open browser automatically             |
| `--theme`           | `auto`  | Theme: `auto`, `light`, or `dark`      |
| `--scroll-sync`     | `true`  | Enable scroll sync via cursor tracking |
| `--stdin`           | `false` | Read content/cursor updates from stdin |
| `--css`             | `""`    | Path to custom CSS file                |
| `--open-to-network` | `false` | Listen on 0.0.0.0 instead of localhost           |
| `--idle-timeout`    | `30s`   | Shut down after no clients connected for this long (0 = disabled) |
| `-v, --verbose`     | `false` | Enable debug logging                             |
| `--version`         |         | Print version, commit, and build date            |

## Supported Markdown Features

- **GitHub Flavored Markdown**: tables, task lists, strikethrough, autolinks
- **Syntax highlighting**: all languages supported by highlight.js with GitHub
  light/dark themes
- **Mermaid diagrams**: flowcharts, sequence diagrams, gantt charts, etc.
- **KaTeX math**: inline `$...$` and block `$$...$$` expressions
- **Relative images**: images referenced with relative paths are resolved from
  the markdown file's directory

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
