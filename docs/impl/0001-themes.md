---
id: IMPL-0001
title: "Themes"
status: In Progress
author: Donald Gifford
created: 2026-03-06
updated: 2026-03-06
---

# IMPL 0001: Themes

**Status:** In Progress
**Author:** Donald Gifford
**Date:** 2026-03-06

**Implements:** [DESIGN-0001](../design/0001-themes.md)

## Objective

Implement a first-class theme system for the mdp preview page. Built-in themes
own prose styling, hljs syntax token colours, and Mermaid diagram settings in
a single embedded CSS file per theme. The Lua plugin defaults to the theme that
matches `vim.o.background`. 14 named themes ship in v1.

## Scope

### In Scope

- `internal/theme` package with `Resolve()` and `Names()`
- 14 built-in theme CSS files under `assets/themes/`
- Updated `assets.go` embed directive
- Updated `server.go` (`pageData`, `handleIndex`, `Server.theme` field)
- Updated `preview.html` template (new fields, conditional hljs injection)
- Updated `preview.js` Mermaid initialisation
- Updated `serve.go` (`--theme`, `--hljs-theme` flags)
- Updated `lua/mdp/init.lua` (`theme` option, `resolve_theme()`)
- Unit tests for `internal/theme`
- Server integration tests for themed page rendering

### Out of Scope

- Runtime theme switching
- A theme editor or preview UI
- Additional vendored hljs sheets beyond the two already present
- Font changes

---

## Phase 1: `internal/theme` Package and Asset Scaffold

Create the theme resolution package and the asset directory structure. Nothing
visual changes yet — this is the foundation everything else builds on.

### Tasks

- [x] Create `assets/themes/` directory
- [x] Add stub CSS files for all themes (empty files are fine; content comes
      in later phases). The three GitHub variants share one file:
  - [x] `github.css` (contains stubs for all three `[data-theme="github-*"]` blocks)
  - [x] `tokyo-night.css`
  - [x] `tokyo-night-moon.css`
  - [x] `tokyo-night-storm.css`
  - [x] `tokyo-night-day.css`
  - [x] `rose-pine.css`
  - [x] `rose-pine-moon.css`
  - [x] `rose-pine-dawn.css`
  - [x] `catppuccin-latte.css`
  - [x] `catppuccin-frappe.css`
  - [x] `catppuccin-macchiato.css`
  - [x] `catppuccin-mocha.css`
- [x] Update `assets/assets.go` embed directive to include `themes/`:
  ```go
  //go:embed preview.html preview.css preview.js vendor themes
  var FS embed.FS
  ```
- [x] Create `internal/theme/theme.go`:
  - [x] Define `Theme` struct with fields: `CSS string`, `HljsVendorCSS string`,
        `MermaidTheme string`, `IsAuto bool`
  - [x] Load theme CSS from `assets.FS` at package init via `mustReadThemeCSS`
        (reads `themes/<file>` from the shared embed FS)
  - [x] Define `builtinThemes` registry mapping name → `Theme` struct. All 14
        named built-in themes set `MermaidTheme: "base"` (JS reads CSS variables
        at runtime). `github-dark` / `github-light` set `HljsVendorCSS` to the
        corresponding vendor path. All three github variants load CSS from the
        shared `github.css` file. `auto` → `IsAuto: true`, `MermaidTheme: ""`
  - [x] Implement `Resolve(name string) (Theme, error)`:
    - If name is `""` or `"auto"` → return auto theme
    - If name is a known built-in → return from registry
    - If name starts with `/` or `./` → read file from disk, return as CSS
    - Otherwise → return error with list of valid names via `Names()`
  - [x] Implement `Names() []string` returning sorted built-in names
- [x] Create `internal/theme/theme_test.go`:
  - [x] `TestResolve_Auto` — empty string and `"auto"` both return `IsAuto: true`
  - [x] `TestResolve_BuiltinNames` — all 14 names resolve without error and
        return non-empty `CSS`
  - [x] `TestResolve_UnknownName` — returns error containing valid name list
  - [x] `TestResolve_FilePath` — temp file with CSS content resolves correctly
  - [x] `TestResolve_FileNotFound` — missing path returns error
  - [x] `TestNames` — returns slice of length 14, sorted, no `"auto"`
  - [x] `TestGithubVariantsHaveVendorCSS` — `github-light` and `github-dark`
        have non-empty `HljsVendorCSS`, all others do not

### Success Criteria

- `make test` passes
- `make build` passes (binary embeds theme files)
- `go vet ./internal/theme/...` clean
- `make lint` clean

---

## Phase 2: Server, Template, and JS Plumbing

Wire the theme resolution into the server. The page will render the correct
`data-theme` and `data-mermaid-theme` attributes, inject theme CSS, and
conditionally use vendored or in-theme hljs styling. All themes look the same
visually (stub CSS) but the infrastructure is complete.

### Tasks

**`internal/server/server.go`**

- [x] Add `theme theme.Theme` field to `Server` struct (resolved at startup,
      not per-request)
- [x] In `Server.New()`: call `theme.Resolve(cfg.Theme)` and store result;
      return error if resolution fails so invalid theme names surface immediately
      at startup
- [x] Rename `pageData.CSS` → `pageData.BaseCSS` (template field rename, purely
      internal)
- [x] Add new fields to `pageData`:
  - `ThemeCSS`, `HljsVendorCSS`, `IsAuto`, `MermaidTheme`
- [x] Update `handleIndex` to populate new fields from `s.theme`
- [x] Remove `cssData` / `jsData` read-per-request; read once in `New()` and
      store in `Server` struct (`baseCSS`, `js` fields)
- [x] `--hljs-theme` validation: hard error if used with non-file-path theme

**`assets/preview.html`**

- [x] Replace `<style>{{.CSS}}</style>` with `<style>{{.BaseCSS}}</style>`
- [x] Add conditional theme CSS injection after base CSS
- [x] Replace hardcoded hljs `<link>` tags with conditional blocks
      (HljsVendorCSS for named themes, IsAuto for media-query links)
- [x] Add `data-mermaid-theme="{{.MermaidTheme}}"` to `<body>`

**`assets/preview.js`**

- [x] Replace the existing Mermaid init block with the full `theme: 'base'` +
      CSS variable approach (reads `--mermaid-*` via `getComputedStyle`)
- [x] Keep `prefersDark` — still needed for the `auto` Mermaid fallback

**`internal/server` tests**

- [x] Update existing tests: `server.Config{}` now resolves `""` as auto — no
      test breakage
- [x] Add `TestServer_ThemeAttribute`
- [x] Add `TestServer_MermaidThemeAttribute`
- [x] Add `TestServer_ThemeCSS_Injection`
- [x] Add `TestServer_HljsVendorCSS_Injection`
- [x] Add `TestServer_CustomCSS_AfterTheme`
- [x] Add `TestServer_InvalidTheme`
- [x] Add `TestServer_HljsTheme_WithBuiltinTheme_Errors`

### Success Criteria

- `make test` passes
- `make lint` passes
- GET `/` with `--theme=auto` renders identical to current behaviour (no
  regression for existing users)
- GET `/` with `--theme=github-dark` renders `data-theme="github-dark"` and
  `data-mermaid-theme="dark"` in the response HTML
- `server.New(Config{Theme: "bad-theme"})` returns a non-nil error

---

## Phase 3: CLI and Lua Plugin Integration

Expose `--theme` and `--hljs-theme` flags in the binary and add `theme` option
and `resolve_theme()` to the Lua plugin. End-to-end flow works: running
`:MdpPreview` picks up `vim.o.background` and passes the right theme name.

### Tasks

**`internal/cli/serve.go`**

- [x] Update `--theme` flag description (descriptive example-based help text)
- [x] Add `--hljs-theme` flag wired to `server.Config.HljsTheme`
- [x] Pass `HljsTheme: hljsTheme` to `server.Config`
- [x] Add `"theme", theme` to `slog.Info` startup log

**`lua/mdp/init.lua`**

- [x] Change `defaults.theme` to `""` (empty = resolve from `vim.o.background`)
- [x] Add `resolve_theme()` local function
- [x] In `M.start()`, call `resolve_theme()` and pass `--theme=<result>` to cmd

### Success Criteria

- `mdp serve --help` shows `--theme` and `--hljs-theme` flags
- `mdp serve --theme=nonexistent README.md` exits non-zero with a clear error
      listing valid theme names
- `mdp serve --theme=tokyo-night README.md` starts successfully
- In Neovim with `background=dark`: `:MdpPreview` passes `--theme=github-dark`
  to the binary (verify in logs)
- In Neovim with `background=light`: `:MdpPreview` passes `--theme=github-light`
- `theme = "tokyo-night"` in plugin opts overrides `vim.o.background`

---

## Phase 4: GitHub Theme Family

Implement the three GitHub themes. `github-dark` and `github-light` are the
simplest — they reuse vendored hljs CSS and need only prose variable overrides
to pin the colours without the `prefers-color-scheme` media query. `github-dimmed`
writes its own hljs token colours (no matching vendored sheet).

### Colour reference

**github-dark** (pin the dark vars from `preview.css`)

| Variable | Value |
|----------|-------|
| `--color-fg-default` | `#e6edf3` |
| `--color-canvas-default` | `#0d1117` |
| `--color-canvas-subtle` | `#161b22` |
| `--color-border-default` | `#30363d` |
| `--color-border-muted` | `#21262d` |
| `--color-accent-fg` | `#58a6ff` |
| `--color-danger-fg` | `#f85149` |
| `--color-success-fg` | `#3fb950` |

**github-light** (pin the light vars from `preview.css`)

| Variable | Value |
|----------|-------|
| `--color-fg-default` | `#1f2328` |
| `--color-canvas-default` | `#ffffff` |
| `--color-canvas-subtle` | `#f6f8fa` |
| `--color-border-default` | `#d0d7de` |
| `--color-border-muted` | `#d8dee4` |
| `--color-accent-fg` | `#0969da` |
| `--color-danger-fg` | `#d1242f` |
| `--color-success-fg` | `#1a7f37` |

**github-dimmed** (GitHub's dimmed dark variant)

| Variable | Value |
|----------|-------|
| `--color-fg-default` | `#adbac7` |
| `--color-canvas-default` | `#22272e` |
| `--color-canvas-subtle` | `#2d333b` |
| `--color-border-default` | `#444c56` |
| `--color-accent-fg` | `#539bf5` |
| hljs keyword | `#f47067` |
| hljs string | `#96d0ff` |
| hljs comment | `#636e7b` |
| hljs number | `#6cb6ff` |
| hljs function | `#dcbdfb` |
| hljs type | `#6cb6ff` |
| hljs variable | `#adbac7` |

### Tasks

All three GitHub variants live in a single file (`assets/themes/github.css`) with
three separate `[data-theme]` selector blocks. The Go registry maps each of the
three names to the same CSS string but with different `HljsVendorCSS` and
`MermaidTheme` values.

- [x] Write `assets/themes/github.css` — single file, three selector blocks:
  - [x] `[data-theme="github-light"]` — prose vars, mermaid vars (vendored hljs)
  - [x] `[data-theme="github-dark"]` — prose vars, mermaid vars (vendored hljs)
  - [x] `[data-theme="github-dimmed"]` — prose vars, mermaid vars, direct scoped
        hljs token rules (17 token classes covered)
- [x] Go registry correctly maps all three names (verified by tests)

### Note on CSS variable scope

`preview.css` defines variables on `:root` (`<html>`) with a media query override.
Theme CSS sets the same variables on `[data-theme]` which matches `<body>`. Since
`body` is a descendant of `html`, the `body`-scoped variables take precedence
for all elements inside body regardless of the `:root` media query. No changes to
`preview.css` are required.

### Success Criteria

- `mdp serve --theme=github-light README.md` → page looks identical to current
  light-mode appearance
- `mdp serve --theme=github-dark README.md` → page looks identical to current
  dark-mode appearance (even when system is in light mode)
- `mdp serve --theme=github-dimmed README.md` → prose and code blocks render in
  GitHub Dimmed palette
- No visual regression on `--theme=auto` (system preference still drives
  appearance)

---

## Phase 5: Tokyo Night Theme Family

Implement all four Tokyo Night variants. These write their own hljs token
colours in the theme CSS.

### Colour reference

Canonical source: [enkia/tokyo-night-vscode-theme](https://github.com/enkia/tokyo-night-vscode-theme)

| Variable | Night | Moon | Storm | Day |
|----------|-------|------|-------|-----|
| bg (`--color-canvas-default`) | `#1a1b26` | `#222436` | `#24283b` | `#e1e2e7` |
| fg (`--color-fg-default`) | `#c0caf5` | `#c8d3f5` | `#c0caf5` | `#3760bf` |
| subtle bg | `#16161e` | `#1e2030` | `#1f2335` | `#d5d6db` |
| border | `#29293d` | `#2f334d` | `#292e42` | `#b4b5c9` |
| accent | `#7aa2f7` | `#82aaff` | `#7aa2f7` | `#2e7de9` |
| keyword | `#bb9af7` | `#c099ff` | `#bb9af7` | `#9854f1` |
| string | `#9ece6a` | `#c3e88d` | `#9ece6a` | `#587539` |
| number | `#ff9e64` | `#ff966c` | `#ff9e64` | `#b15c00` |
| function | `#7aa2f7` | `#82aaff` | `#7aa2f7` | `#2e7de9` |
| comment | `#565f89` | `#636da6` | `#565f89` | `#848cb8` |
| operator | `#89ddff` | `#89ddff` | `#89ddff` | `#006c86` |
| type | `#2ac3de` | `#4fd6be` | `#2ac3de` | `#007197` |

### Tasks

Each file contains a single `[data-theme="X"]` block with:
1. Prose CSS custom property overrides
2. Direct scoped hljs token rules: `[data-theme="X"] .hljs-keyword { color: ...; }`
3. `--mermaid-*` CSS custom properties (read by JS `theme: 'base'` at runtime)

- [x] Write `assets/themes/tokyo-night.css` (Night variant)
  - `[data-theme="tokyo-night"]` block — prose vars + hljs token rules + mermaid vars
- [x] Write `assets/themes/tokyo-night-moon.css`
  - `[data-theme="tokyo-night-moon"]` block — prose vars + hljs token rules + mermaid vars
- [x] Write `assets/themes/tokyo-night-storm.css`
  - `[data-theme="tokyo-night-storm"]` block (close to Night, slightly blue-tinted)
- [x] Write `assets/themes/tokyo-night-day.css`
  - `[data-theme="tokyo-night-day"]` block — light variant; mermaid vars use
    lighter palette colours
- [ ] Manual visual check of all four variants against a markdown file with
      code blocks, tables, and a Mermaid diagram

### Success Criteria

- All four variants render with correct prose background, foreground, and
  link colours
- Code blocks render with correct syntax token colours (at minimum: keywords,
  strings, comments, numbers)
- Mermaid diagrams use `theme: 'base'` with each variant's `--mermaid-*`
  CSS variables — diagram colours coordinate with the prose theme
- `make test` still passes (CSS content doesn't affect Go tests)

---

## Phase 6: Rosé Pine Theme Family

Implement all three Rosé Pine variants.

### Colour reference

Canonical source: [rose-pine/palette](https://rosepinetheme.com/palette/)

| Variable | Pine (main) | Moon | Dawn (light) |
|----------|-------------|------|--------------|
| bg | `#191724` | `#232136` | `#faf4ed` |
| fg | `#e0def4` | `#e0def4` | `#575279` |
| subtle bg | `#1f1d2e` | `#2a273f` | `#fffaf3` |
| border | `#403d52` | `#44415a` | `#dfdad9` |
| accent (iris) | `#c4a7e7` | `#c4a7e7` | `#907aa9` |
| keyword (iris) | `#c4a7e7` | `#c4a7e7` | `#907aa9` |
| string (foam) | `#9ccfd8` | `#9ccfd8` | `#56949f` |
| number (gold) | `#f6c177` | `#f6c177` | `#ea9d34` |
| function (rose) | `#ebbcba` | `#ea9a97` | `#d7827e` |
| comment (muted) | `#6e6a86` | `#6e6a86` | `#9893a5` |
| operator (subtle) | `#908caa` | `#908caa` | `#797593` |
| type (pine) | `#31748f` | `#3e8fb0` | `#286983` |

### Tasks

Each file contains a single `[data-theme="X"]` block with prose vars, direct
scoped hljs token rules, and `--mermaid-*` CSS custom properties.

- [x] Write `assets/themes/rose-pine.css` (Pine/main variant)
  - `[data-theme="rose-pine"]` block — prose vars + hljs token rules + mermaid vars
- [x] Write `assets/themes/rose-pine-moon.css`
  - `[data-theme="rose-pine-moon"]` block — prose vars + hljs token rules + mermaid vars
- [x] Write `assets/themes/rose-pine-dawn.css`
  - `[data-theme="rose-pine-dawn"]` block — light variant; mermaid vars use
    dawn palette colours
- [ ] Manual visual check of all three variants

### Success Criteria

- All three variants render correct prose and code colours
- Mermaid diagrams use `theme: 'base'` with per-variant `--mermaid-*` CSS
  variables — dark palette for pine/moon, lighter palette for dawn
- `make test` passes

---

## Phase 7: Catppuccin Theme Family

Implement all four Catppuccin variants.

### Colour reference

Canonical source: [catppuccin/catppuccin](https://github.com/catppuccin/catppuccin)

| Variable | Latte (light) | Frappé | Macchiato | Mocha |
|----------|---------------|--------|-----------|-------|
| bg | `#eff1f5` | `#303446` | `#24273a` | `#1e1e2e` |
| fg (text) | `#4c4f69` | `#c6d0f5` | `#cad3f5` | `#cdd6f4` |
| subtle bg (mantle) | `#e6e9ef` | `#292c3c` | `#1e2030` | `#181825` |
| border (surface0) | `#ccd0da` | `#414559` | `#363a4f` | `#313244` |
| accent (blue) | `#1e66f5` | `#8caaee` | `#8aadf4` | `#89b4fa` |
| keyword (mauve) | `#8839ef` | `#ca9ee6` | `#c6a0f6` | `#cba6f7` |
| string (green) | `#40a02b` | `#a6d189` | `#a6da95` | `#a6e3a1` |
| number (peach) | `#fe640b` | `#ef9f76` | `#f5a97f` | `#fab387` |
| function (blue) | `#1e66f5` | `#8caaee` | `#8aadf4` | `#89b4fa` |
| comment (overlay0) | `#9ca0b0` | `#737994` | `#6e738d` | `#6c7086` |
| operator (sky) | `#04a5e5` | `#99d1db` | `#91d7e3` | `#89dceb` |
| type (teal) | `#179299` | `#81c8be` | `#8bd5ca` | `#94e2d5` |

### Tasks

Each file contains a single `[data-theme="X"]` block with prose vars, direct
scoped hljs token rules, and `--mermaid-*` CSS custom properties.

- [x] Write `assets/themes/catppuccin-latte.css` — light variant; mermaid vars
      use latte palette colours
- [x] Write `assets/themes/catppuccin-frappe.css` — mermaid vars use frappé palette
- [x] Write `assets/themes/catppuccin-macchiato.css` — mermaid vars use macchiato palette
- [x] Write `assets/themes/catppuccin-mocha.css` — mermaid vars use mocha palette
- [ ] Manual visual check of all four variants

### Success Criteria

- All four variants render correct prose and code colours
- Mermaid diagrams use `theme: 'base'` with per-variant `--mermaid-*` CSS
  variables; latte uses a light mermaid palette, others use dark palettes
- `make test` passes

---

## Phase 8: Integration Testing and Documentation

Cross-cutting validation, test coverage check, and user-facing docs update.

### Tasks

**Testing**

- [x] `make test-coverage` — confirm `internal/theme` coverage ≥ 80% (93.5%)
- [x] Add `TestResolve_AllBuiltins` — table-driven test looping over `Names()`
      and asserting each resolves to a non-zero `Theme` struct
- [x] Add `TestServer_AllBuiltinThemes` — table-driven test starting a server
      for each built-in theme name and asserting the rendered page contains the
      expected `data-theme` attribute
- [ ] Manually test the full flow end-to-end:
  - [ ] Set `theme = "tokyo-night"` in lazy.nvim opts, `:MdpPreview`, verify
        browser shows Tokyo Night colours
  - [ ] Remove theme opt (empty), confirm `vim.o.background=dark` picks
        `github-dark`, `background=light` picks `github-light`
  - [ ] Test `--theme=/path/to/custom.css` with a hand-written CSS file
  - [ ] Test `--theme=/path/to/custom.css --hljs-theme=github-dark`
  - [ ] Confirm `custom_css` overrides theme (colours from custom CSS win)

**Documentation**

- [x] Update `README.md`:
  - Add `--theme` to the flags table
  - Add `Themes` section listing all built-in names with a one-liner each
  - Note that `vim.o.background` drives the default
- [x] Update `lua/mdp/init.lua` docstring / comment on the `theme` option to
      describe the `vim.o.background` fallback behaviour
- [x] Update `docs/LOGGING.md` if the startup log line now includes `theme=`
      field (it should, if Phase 3 adds it to the `slog.Info` call)

**Release prep**

- [x] `make lint` clean on all new files
- [x] `make build` produces a binary where `--theme=nonexistent` exits non-zero
- [x] Update `CLAUDE.md` with any new patterns or conventions introduced

### Success Criteria

- `make ci` (lint + test + build + license-check) passes clean
- Coverage target met (`internal/theme` ≥ 80%, overall ≥ 60%)
- All 14 themes render visually correct prose and code colours against
  `assets/vendor/hljs/testdata` fixture or a hand-crafted markdown file
  containing: headings, paragraphs, code blocks (Go, JSON, bash), a table,
  a blockquote, and a Mermaid flowchart
- README `Themes` section is accurate and complete

---

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `internal/theme/theme.go` | Create | Theme struct, registry, Resolve(), Names() |
| `internal/theme/theme_test.go` | Create | Unit tests for theme resolution |
| `assets/themes/github.css` | Create | All three GitHub variants (light/dark/dimmed) in one DRY file |
| `assets/themes/tokyo-night*.css` | Create | 4 Tokyo Night variant CSS files |
| `assets/themes/rose-pine*.css` | Create | 3 Rosé Pine variant CSS files |
| `assets/themes/catppuccin-*.css` | Create | 4 Catppuccin variant CSS files |
| `assets/assets.go` | Modify | Add `themes` to embed directive |
| `internal/server/server.go` | Modify | `Server.theme` field, updated pageData, handleIndex |
| `internal/server/server_test.go` | Modify | New theme-related test cases |
| `assets/preview.html` | Modify | New template fields, conditional hljs injection |
| `assets/preview.js` | Modify | Mermaid init reads `data-mermaid-theme` |
| `internal/cli/serve.go` | Modify | Updated `--theme` flag, new `--hljs-theme` flag |
| `internal/server/server.go` | Modify | Add `HljsTheme string` to Config |
| `lua/mdp/init.lua` | Modify | `theme` option, resolve_theme(), pass flag |
| `README.md` | Modify | Themes section, flags table |
| `docs/LOGGING.md` | Modify | Startup log entry update if theme added |
| `CLAUDE.md` | Modify | Any new patterns |

---

## Testing Plan

- [ ] Unit: `internal/theme` — Resolve(), Names(), file path handling, error cases
- [ ] Integration: server renders correct attributes and CSS injection per theme
- [ ] Integration: `custom_css` always appears after theme CSS
- [ ] Integration: `server.New()` with invalid theme returns error
- [ ] Integration: `server.New()` with file-path theme reads and embeds CSS
- [ ] CLI: `--theme` and `--hljs-theme` flags wired to Config correctly
- [ ] Manual: visual check of all 14 themes against fixture markdown
- [ ] Regression: `--theme=auto` renders identically to pre-implementation baseline

---

## Rollback Plan

All changes are additive. The default is `auto` which preserves existing
behaviour. To roll back:

1. Revert `internal/theme/`, `assets/themes/`, and the template/server changes
2. Restore `assets/assets.go` embed directive
3. Restore `pageData` struct
4. Restore the hardcoded hljs `<link>` tags in `preview.html`
5. Remove `theme` option from `lua/mdp/init.lua`

No database migrations, no persistent state — rollback is a straight code revert.

---

## Dependencies

- Go 1.22+ (already in use) — `embed.FS` glob patterns for `themes/*.css`
- No new Go module dependencies
- No new vendored JS/CSS libraries

---

## Decisions

All open questions from the initial draft have been answered.

| # | Question | Decision |
|---|----------|----------|
| 1 | hljs token colour approach | **Direct scoped rules**: `[data-theme="X"] .hljs-keyword { color: #...; }`. No resolver layer in `preview.css`. |
| 2 | Mermaid theming | **Full `theme: 'base'` + CSS variable approach**: all 14 themes define `--mermaid-*` CSS custom properties; JS reads them via `getComputedStyle` at runtime and passes as `themeVariables` to `mermaid.initialize()`. `auto` falls back to `prefersDark ? "dark" : "default"`. |
| 3 | GitHub theme DRY | **Single `assets/themes/github.css`** file containing all three `[data-theme="github-*"]` selector blocks. Go registry maps each name to the same CSS string with different `HljsVendorCSS` values per entry. |
| 4 | `--hljs-theme` with built-in | **Hard error**: binary returns an error if `--hljs-theme` is provided alongside a built-in theme name. |
| 5 | Default dark theme | **Keep `github-dark`**: `vim.o.background=dark` maps to `github-dark` — zero visual surprise for existing users. |

---

## References

- [DESIGN-0001: Themes](../design/0001-themes.md)
- [highlight.js CSS classes reference](https://highlightjs.readthedocs.io/en/latest/css-classes-reference.html)
- [Mermaid theming docs](https://mermaid.js.org/config/theming.html)
- [Tokyo Night palette](https://github.com/enkia/tokyo-night-vscode-theme)
- [Rosé Pine palette](https://rosepinetheme.com/palette/)
- [GitHub Primer colours](https://primer.style/foundations/color)
- [Catppuccin palette](https://github.com/catppuccin/catppuccin)
