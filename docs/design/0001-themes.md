---
id: DESIGN-0001
title: "Themes"
status: Approved
author: Donald Gifford
created: 2026-03-05
updated: 2026-03-06
---

# DESIGN 0001: Themes

**Status:** Approved
**Author:** Donald Gifford
**Created:** 2026-03-05
**Updated:** 2026-03-06

## Overview

Add a first-class theme system to mdp. Built-in themes are self-contained CSS
files that fully own prose styling, code syntax highlighting, and Mermaid
diagram colours — no dependency on vendored hljs theme sheets beyond the two
already present for the `github` family. The default theme is driven by
`vim.o.background` so the preview matches the editor without any configuration.

## Goals and Non-Goals

### Goals

- Named built-in themes selectable via CLI flag and Lua plugin option
- `auto` special value: reads `vim.o.background` at startup; falls back to
  `prefers-color-scheme` when background is unset
- Full theme set for v1: Tokyo Night (4 variants), Rosé Pine (3 variants),
  GitHub (dark, light, dimmed), Catppuccin (4 variants) — 12 named themes
- Built-in themes are **first class**: each owns its complete CSS including hljs
  token colours; no additional hljs vendoring required
- Mermaid diagrams coordinated per theme via Mermaid's `theme: 'base'` + CSS
  variables
- User-defined theme files on disk (`--theme=/path/to/theme.css`)
- `custom_css` remains as an escape hatch layered on top of any theme
- Zero breaking changes to the existing config surface

### Non-Goals

- Runtime theme switching without restarting the server
- A theme editor or preview UI
- Fonts — typography stays outside theme scope
- Vendoring additional hljs CSS sheets (first-class themes write their own
  token colours)
- Packaging or distributing third-party themes

## Background

`assets/preview.html` already has `{{.Theme}}` and `{{.CustomCSS}}` template
fields and a `data-theme` attribute on `<body>`. `assets/preview.css` uses CSS
custom properties for all colours with a `@media (prefers-color-scheme: dark)`
block for the automatic dark variant. Two hljs sheets are vendored
(`github.min.css`, `github-dark.min.css`) and switched via `media` attributes.

The wiring is in place. What's missing is the theme registry, the CLI flag, the
Lua plugin option, and the theme CSS files themselves.

## Decisions

Answers to the open questions raised in the initial draft:

| # | Question | Decision |
|---|----------|----------|
| 1 | `vim.o.background` as default? | **Yes** — plugin reads background at startup and maps to the appropriate named theme before passing to the binary |
| 2 | Which built-in themes? | **Tokyo Night** (night/moon/storm/day), **Rosé Pine** (pine/moon/dawn), **GitHub** (dark/light/dimmed), **Catppuccin** (latte/frappé/macchiato/mocha) |
| 3 | User-defined theme files? | **Yes** — `--theme=/abs/path/to/theme.css`; file is treated as a complete theme CSS |
| 4 | hljs pairing for custom file themes? | **Yes** — custom theme files should include hljs token CSS; an optional `--hljs-theme=<vendored-name>` flag lets users opt into a vendored hljs sheet instead. `--hljs-theme` with a built-in theme name returns a **hard error** |
| 5 | Vendor more hljs sheets? | **No** — first-class built-in themes write their own hljs token CSS using direct scoped rules (`[data-theme="X"] .hljs-keyword { color: ...; }`); no new vendored sheets added |
| 6 | Mermaid granularity? | **Full `theme: 'base'` + CSS variables**: each built-in theme defines `--mermaid-*` CSS custom properties; JS reads them via `getComputedStyle` and passes as `themeVariables`. `auto` falls back to `prefers-color-scheme` |

**Core philosophy:** Built-in themes are first class. They own their full visual
surface — prose, syntax highlighting, and diagrams — in a single embedded CSS
file. hljs is still used for tokenisation; we just supply our own colours for
the output classes (`.hljs-keyword`, `.hljs-string`, etc.) rather than relying
on vendored hljs stylesheets. The vendored github sheets are retained only for
the `github-light` and `github-dark` variants that already match them exactly.

## Detailed Design

### Theme definition

A built-in theme is a single embedded CSS file containing:

1. **Prose variables** — `:root` overrides for `--color-*` custom properties
   defined in `preview.css`
2. **hljs token colours** — `.hljs-*` rules scoped to `[data-theme="<name>"]`
   so they only apply when that theme is active
3. **Mermaid variables** — `--mermaid-*` CSS variables consumed by
   `mermaid.initialize({ theme: 'base', themeVariables: { ... } })`

```css
/* Example: assets/themes/tokyo-night.css */

[data-theme="tokyo-night"] {
  --color-fg-default:    #c0caf5;
  --color-canvas-default:#1a1b26;
  --color-canvas-subtle: #16161e;
  --color-border-default:#29293d;
  --color-accent-fg:     #7aa2f7;
  /* ... */

  /* hljs tokens */
  --hljs-keyword:  #bb9af7;
  --hljs-string:   #9ece6a;
  --hljs-comment:  #565f89;
  /* ... */

  /* Mermaid (theme: 'base') */
  --mermaid-primaryColor:      #7aa2f7;
  --mermaid-primaryTextColor:  #c0caf5;
  --mermaid-lineColor:         #565f89;
  --mermaid-background:        #1a1b26;
  /* ... */
}
```

The `[data-theme]` scope ensures themes don't bleed if the attribute is missing.

### Built-in theme registry

| Theme name           | Family       | Dark? |
|----------------------|--------------|-------|
| `auto`               | —            | auto  |
| `tokyo-night`        | Tokyo Night  | dark  |
| `tokyo-night-moon`   | Tokyo Night  | dark  |
| `tokyo-night-storm`  | Tokyo Night  | dark  |
| `tokyo-night-day`    | Tokyo Night  | light |
| `rose-pine`          | Rosé Pine    | dark  |
| `rose-pine-moon`     | Rosé Pine    | dark  |
| `rose-pine-dawn`     | Rosé Pine    | light |
| `github-dark`        | GitHub       | dark  |
| `github-light`       | GitHub       | light |
| `github-dimmed`      | GitHub       | dark  |
| `catppuccin-latte`   | Catppuccin   | light |
| `catppuccin-frappe`  | Catppuccin   | dark  |
| `catppuccin-macchiato`| Catppuccin  | dark  |
| `catppuccin-mocha`   | Catppuccin   | dark  |

`github-light` and `github-dark` re-use the vendored hljs sheets (no custom
token rules needed). `github-dimmed` writes its own token rules (no matching
vendored sheet). All three GitHub variants share one `assets/themes/github.css`
file with separate `[data-theme]` selector blocks. All other themes write their
own hljs token colours using direct scoped rules.

### `auto` theme resolution

`auto` is a special value, not a real theme CSS file. Resolution order:

1. Lua plugin reads `vim.o.background` at server start time
2. If `"dark"` → passes `--theme=github-dark` (or user-configured dark default)
3. If `"light"` → passes `--theme=github-light`
4. If unset or `""` → passes `--theme=auto`; the browser's
   `prefers-color-scheme` media query takes over (existing behaviour)

This means the binary never sees `auto` unless the Lua plugin explicitly passes
it. The binary's default is still `auto` for direct CLI use.

### Asset layout

```
assets/
  preview.css               # base prose CSS — layout, typography, custom props
  preview.html              # template
  preview.js
  themes/
    tokyo-night.css
    tokyo-night-moon.css
    tokyo-night-storm.css
    tokyo-night-day.css
    rose-pine.css
    rose-pine-moon.css
    rose-pine-dawn.css
    github.css               # all three GitHub variants in one DRY file:
                             #   [data-theme="github-light"] — prose vars; hljs via vendored sheet
                             #   [data-theme="github-dark"]  — prose vars; hljs via vendored sheet
                             #   [data-theme="github-dimmed"] — prose vars + own hljs token rules
    catppuccin-latte.css
    catppuccin-frappe.css
    catppuccin-macchiato.css
    catppuccin-mocha.css
  vendor/
    hljs/
      github.min.css         # used by github-light and auto-light
      github-dark.min.css    # used by github-dark and auto-dark
      highlight.min.js
    katex/
    mermaid.min.js
```

### Go: `internal/theme` package

New package `internal/theme` owns theme resolution and keeps it out of the
server package.

```go
package theme

// Theme holds everything the server needs to render a page with the
// correct styling.
type Theme struct {
    // CSS is the complete theme stylesheet (prose + hljs tokens + mermaid vars).
    // Empty for "auto" — the base preview.css handles auto via media query.
    CSS string

    // HljsVendorCSS is the path to a vendored hljs sheet to inject via <link>.
    // Only set for github-light / github-dark. Empty for all other themes.
    HljsVendorCSS string

    // MermaidTheme is the string passed to mermaid.initialize().
    // "base" for named themes (uses CSS vars), "default"/"dark" for auto.
    MermaidTheme string

    // IsAuto skips server-side CSS injection and lets the browser's
    // prefers-color-scheme media query drive appearance.
    IsAuto bool
}

// Resolve returns the Theme for the given name.
// name may be a built-in name or an absolute path to a CSS file.
func Resolve(name string) (Theme, error)

// Names returns all valid built-in theme names.
func Names() []string
```

Themes are loaded once at binary init via `go:embed assets/themes/*.css`.

### Go: server changes

`server.Config` gains:

```go
Theme string  // resolved theme name or file path; default "auto"
```

The handler resolves the theme once at startup and stores the `theme.Theme`
value. Each page render injects `ThemeCSS` and `HljsVendorCSS` into the
template data.

### HTML template changes

Replace the current hardcoded hljs `<link>` tags with injected fields:

```html
<head>
  <style>{{.BaseCSS}}</style>
  {{if .ThemeCSS}}<style>{{.ThemeCSS}}</style>{{end}}
  {{if .HljsVendorCSS}}<link rel="stylesheet" href="{{.HljsVendorCSS}}">{{end}}
  {{if .IsAuto}}
  <link rel="stylesheet" href="/vendor/hljs/github.min.css"
        media="(prefers-color-scheme: light)">
  <link rel="stylesheet" href="/vendor/hljs/github-dark.min.css"
        media="(prefers-color-scheme: dark)">
  {{end}}
  {{if .CustomCSS}}<style>{{.CustomCSS}}</style>{{end}}
</head>
<body data-theme="{{.Theme}}" data-mermaid-theme="{{.MermaidTheme}}">
```

Injection order (last wins): base prose CSS → theme CSS → vendored hljs (if
any) → custom CSS. This lets `custom_css` override everything.

### JS: Mermaid initialisation

`preview.js` reads `data-mermaid-theme` from `<body>` and initialises Mermaid:

```js
const mermaidTheme = document.body.dataset.mermaidTheme || 'default';
mermaid.initialize({
  startOnLoad: false,
  theme: mermaidTheme,  // 'base' for named themes, 'default'/'dark' for auto
});
```

For `theme: 'base'`, Mermaid reads `--mermaid-*` CSS variables from the theme
stylesheet. For `auto`, the browser's colour scheme picks `default` or `dark`.

### User-defined theme files

`--theme=/absolute/path/to/theme.css` is detected by a leading `/` (or `./`).
The file is read at startup, validated as non-empty, and treated identically to
a built-in theme CSS string. `HljsVendorCSS` is empty (user owns hljs token
colours in their CSS); `MermaidTheme` defaults to `"base"`.

An optional companion flag handles users who want a vendored hljs sheet instead
of writing their own tokens:

```
--hljs-theme string   Vendored hljs theme to pair with a custom theme file
                      (github, github-dark). Only used with --theme=<file>.
```

### CLI changes

```
--theme string       Preview theme name or path to CSS file (default "auto")
--hljs-theme string  Vendored hljs stylesheet for custom theme files
                     (github, github-dark) (default "")
```

Valid built-in names are validated at startup with a clear error listing
available themes. File paths are validated for existence and readability.

### Lua plugin changes

```lua
local defaults = {
  -- ... existing fields ...
  theme = "",        -- empty = resolve from vim.o.background at start time
}

--- Resolve the effective theme name from vim.o.background when not set.
local function resolve_theme()
  if config.theme and config.theme ~= "" then
    return config.theme
  end
  local bg = vim.o.background
  if bg == "dark" then
    return "github-dark"
  elseif bg == "light" then
    return "github-light"
  end
  return "auto"
end

-- In M.start(), build cmd:
local theme = resolve_theme()
local cmd = { binary, "serve", "--stdin", "--theme=" .. theme, ... }
```

## API / Interface Changes

| Surface             | Change                                                              |
|---------------------|---------------------------------------------------------------------|
| `serve` CLI flag    | `--theme=<name\|path>` added, default `"auto"`                      |
| `serve` CLI flag    | `--hljs-theme=<name>` added, default `""`                           |
| Lua plugin option   | `theme = ""` added (empty = auto-resolve from `vim.o.background`)   |
| `server.Config`     | `Theme string` field added                                          |
| HTML template data  | `ThemeCSS`, `HljsVendorCSS`, `IsAuto`, `MermaidTheme` fields added  |
| `<body>` attributes | `data-mermaid-theme` added alongside existing `data-theme`          |
| New package         | `internal/theme` — `Resolve()`, `Names()`                           |
| New directory       | `assets/themes/` — 14 embedded CSS files                            |

No existing flags, options, or wire formats are removed or changed.

## Data Model

No persistent state. Theme is resolved once at server startup from the CLI flag
and held for the lifetime of the process.

## Testing Strategy

- Unit tests for `theme.Resolve()`:
  - Each built-in name returns the expected struct fields
  - Unknown name returns an error listing valid names
  - File path is read and returned as CSS string
  - Empty/unreadable file path returns error
- Server handler tests:
  - `data-theme` attribute on rendered page matches configured theme
  - `data-mermaid-theme` attribute correct per theme
  - `ThemeCSS` non-empty for named themes, empty for `auto`
  - `HljsVendorCSS` set only for `github-light` / `github-dark`
  - `custom_css` appears after theme CSS in rendered output
- Existing server tests continue to pass with default `auto`
- Manual visual check: fixture markdown file rendered with each theme

## Migration / Rollout Plan

- Default is `auto` (binary) / resolved from `vim.o.background` (plugin) — no
  change for existing users who haven't set a theme
- New flags are additive; existing configs and plugin specs are unaffected
- Themes are embedded in the binary; no external files or network access needed
- Release as a minor version bump

## Open Questions

All questions from the initial draft have been answered — see [Decisions](#decisions).

## References

- [Tokyo Night colour palette](https://github.com/enkia/tokyo-night-vscode-theme)
- [Rosé Pine colour palette](https://rosepinetheme.com/palette/)
- [GitHub Primer colour system](https://primer.style/foundations/color)
- [Catppuccin colour palette](https://github.com/catppuccin/catppuccin)
- [highlight.js token class reference](https://highlightjs.readthedocs.io/en/latest/css-classes-reference.html)
- [Mermaid theming — `theme: 'base'` and CSS variables](https://mermaid.js.org/config/theming.html#customizing-themes-with-themevariables)
- `assets/preview.css` — existing CSS custom property structure
- `assets/preview.html` — existing `{{.Theme}}`, `{{.CustomCSS}}`, `data-theme` hooks
