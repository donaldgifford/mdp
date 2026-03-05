---
id: DESIGN-0001
title: "Themes"
status: Draft
author: Donald Gifford
created: 2026-03-05
---

# DESIGN 0001: Themes

**Status:** Draft
**Author:** Donald Gifford
**Date:** 2026-03-05

## Overview

Add a theme system to mdp so users can control the visual appearance of the
preview page beyond the current implicit GitHub light/dark default. A theme
bundles prose CSS, code syntax highlighting CSS, and Mermaid theme settings
into a named, selectable unit. The `auto` default preserves existing behavior
(follows `prefers-color-scheme`).

## Goals and Non-Goals

### Goals

- Named built-in themes selectable via CLI flag and Lua plugin option
- `auto` default that respects system `prefers-color-scheme` (no regression)
- Force light or dark regardless of system preference (`light`, `dark`)
- Code syntax highlighting (hljs) theme coordinated with prose theme
- Mermaid diagram theme coordinated with prose theme
- `custom_css` remains as an escape hatch layered on top of any theme
- Zero breaking changes to the existing config surface

### Non-Goals

- Runtime theme switching without restarting the server (can revisit later)
- User-defined theme files on disk (custom_css covers most of this need)
- A theme editor or preview UI
- Fonts — typography is intentionally outside theme scope for now
- Packaging and distributing third-party themes

## Background

The preview page (`assets/preview.html`) already has `{{.Theme}}` and
`{{.CustomCSS}}` template fields wired in and a `data-theme` attribute on
`<body>`. The prose CSS (`assets/preview.css`) uses CSS custom properties
(`--color-fg-default`, `--color-canvas-default`, etc.) for all colours, with a
`@media (prefers-color-scheme: dark)` block for the automatic dark variant.
Two hljs CSS files are already vendored (`github.min.css`,
`github-dark.min.css`) and switched via `media` attributes on their `<link>`
tags.

The infrastructure is mostly in place. What's missing is the theme selection
logic in the Go server, the CLI flag, and the Lua plugin option.

## Detailed Design

### Theme definition

A theme is a named set of four things:

| Component       | Mechanism                                    |
|-----------------|----------------------------------------------|
| Prose CSS       | CSS custom property overrides in `:root`     |
| Code CSS        | hljs stylesheet name (vendored)              |
| Mermaid theme   | String passed to `mermaid.initialize()`      |
| Dark flag       | Boolean — tells JS which hljs sheet to load  |

Built-in themes are embedded in the binary as small CSS snippets that override
the custom properties defined in `preview.css`. Only the overrides ship — the
base layout, typography, and structural rules stay in `preview.css` unchanged.

### Built-in theme set (initial)

| Name          | Prose base     | hljs theme       | Mermaid theme | Notes                        |
|---------------|----------------|------------------|---------------|------------------------------|
| `auto`        | system default | github / dark    | default/dark  | Current behaviour, default   |
| `light`       | github light   | github           | default       | Force light regardless of OS |
| `dark`        | github dark    | github-dark      | dark          | Force dark regardless of OS  |
| `gruvbox-dark`| gruvbox dark   | github-dark      | dark          | Warm dark palette            |

Starting with four keeps the binary size impact negligible. Additional themes
can be added later by adding a CSS file to `assets/themes/` and an entry to the
theme registry.

### Asset layout

```
assets/
  preview.css          # base prose CSS (custom properties, layout, structure)
  preview.html         # template — already has {{.Theme}}, {{.CustomCSS}}
  preview.js
  themes/
    auto.css           # empty — auto uses media query in preview.css
    light.css          # :root override to pin github light vars
    dark.css           # :root override to pin github dark vars
    gruvbox-dark.css   # :root override for gruvbox palette
  vendor/
    hljs/
      github.min.css
      github-dark.min.css
      highlight.min.js
    katex/
    mermaid.min.js
```

### Go server changes

**`internal/server/server.go`**

Add `Theme string` to `Config`. Default to `"auto"`.

**`assets/assets.go`** (or a new `internal/theme/theme.go`)

A small theme registry:

```go
type Theme struct {
    CSS          string // contents of assets/themes/<name>.css
    HljsLight    string // e.g. "github"
    HljsDark     string // e.g. "github-dark"
    MermaidTheme string // e.g. "default"
    ForceDark    bool   // true = skip media query, always use dark hljs
    ForceLight   bool   // true = skip media query, always use light hljs
}

var builtinThemes = map[string]Theme{
    "auto":         {...},
    "light":        {...},
    "dark":         {...},
    "gruvbox-dark": {...},
}

func Resolve(name string) (Theme, error) { ... }
```

**`assets/preview.html`** template already supports injection — the resolved
theme's CSS populates `{{.Theme}}` (currently a bare string, reuse as CSS) or
we add a dedicated `{{.ThemeCSS}}` field to the template data struct.

**`internal/server/handler.go`** (wherever the template is rendered)

Resolve the theme at startup, cache it, inject into every page render. No
per-request resolution needed.

### CLI changes

**`internal/cli/serve.go`**

```
--theme string   Preview theme (auto, light, dark, gruvbox-dark) (default "auto")
```

Valid theme names are validated at startup; unknown names return a clear error.

### Lua plugin changes

**`lua/mdp/init.lua`**

Add `theme` to defaults:

```lua
local defaults = {
  -- ...existing fields...
  theme = "auto",
}
```

Pass to the binary:

```lua
local cmd = { binary, "serve", "--stdin", "--theme=" .. config.theme, ... }
```

### Browser-side hljs coordination

Currently `preview.html` uses two `<link>` tags with `media` attributes to
switch hljs CSS automatically. For forced light/dark themes the JS needs to
swap to the correct sheet instead.

Two approaches:

**Option A** — Inject the correct hljs CSS as a `<style>` block alongside the
theme CSS at server render time. No JS needed. Simpler.

**Option B** — Keep the media-query links and have JS check `data-theme` on
`<body>` at init to override them. More dynamic but adds JS complexity.

Prefer **Option A** for the initial implementation. The server resolves
everything; the browser just renders.

### Mermaid coordination

`preview.js` calls `mermaid.initialize()`. The resolved theme name (or a
derived Mermaid theme string) needs to reach the browser. Two options:

- Embed it as a `data-mermaid-theme` attribute on `<body>` (server-rendered).
- Include it in the injected theme CSS as a custom property that JS reads.

Prefer the `data-mermaid-theme` attribute — explicit and no JS parsing needed.

## API / Interface Changes

| Surface              | Change                                                     |
|----------------------|------------------------------------------------------------|
| `serve` CLI flag     | `--theme=<name>` added, default `auto`                     |
| Lua plugin option    | `theme = "auto"` added to defaults                         |
| `server.Config`      | `Theme string` field added                                 |
| HTML template data   | `ThemeCSS string` and `HljsCSS string` fields added        |
| `<body>` attributes  | `data-mermaid-theme` added alongside existing `data-theme` |

No existing flags, options, or wire formats are removed or changed.

## Data Model

No persistent state. Theme is resolved once at server startup from the CLI
flag and held in `server.Config` for the lifetime of the process.

## Testing Strategy

- Unit tests for `theme.Resolve()` — valid names, unknown name error, case
  sensitivity behaviour
- Unit test that `server.Config` with each built-in theme name renders a page
  where `data-theme` matches the expected value
- Existing server tests continue to pass with `theme = "auto"` default
- Manual visual check of each theme against a fixture markdown file

## Migration / Rollout Plan

- Default is `auto` — zero change for existing users
- New `--theme` flag is additive; existing configs and plugin specs are
  unaffected
- Themes are embedded in the binary; no external files or network access
  required
- Release as a minor version bump

## Open Questions

1. **Should `vim.o.background` drive the default?** If Neovim is in dark mode
   (`vim.o.background == "dark"`), should mdp default to `dark` instead of
   `auto`? This would be a better out-of-the-box experience for terminal users
   whose system `prefers-color-scheme` may be light while their editor is dark.

2. **How many built-in themes for v1?** Four (auto/light/dark/gruvbox-dark) is
   a minimal set. Is there a specific theme (Nord, Catppuccin, Tokyo Night,
   Dracula) worth including before the first release, or should we ship minimal
   and add by request?

3. **Custom theme files on disk?** Should `--theme=/path/to/theme.css` be
   supported as an alternative to a named built-in? This would satisfy power
   users without bloating the binary. `custom_css` already exists as an overlay
   but a full theme file would replace the base prose colours entirely.

4. **hljs theme for custom themes?** If a user provides a custom theme CSS
   (question 3), how do we know which hljs stylesheet to pair with it?
   Options: always `auto`, require a flag like `--hljs-theme=monokai`, or infer
   dark/light from a CSS property.

5. **Should we vendor more hljs themes?** Currently only github light/dark are
   vendored. Themes like gruvbox-dark would ideally pair with a matching hljs
   theme (e.g. `gruvbox-dark` from hljs). We could vendor a handful of extras
   or generate them from hljs at build time via `update-vendor`.

6. **Mermaid theme granularity?** Mermaid themes are coarse (`default`, `dark`,
   `forest`, `neutral`, `base`). Is `default`/`dark` enough, or should each
   built-in prose theme map to a specific Mermaid theme?

## References

- [highlight.js theme gallery](https://highlightjs.org/demo)
- [Mermaid themes](https://mermaid.js.org/config/theming.html)
- [CSS custom properties — MDN](https://developer.mozilla.org/en-US/docs/Web/CSS/Using_CSS_custom_properties)
- `assets/preview.css` — existing CSS custom property structure
- `assets/preview.html` — existing `{{.Theme}}`, `{{.CustomCSS}}`, `data-theme` hooks
