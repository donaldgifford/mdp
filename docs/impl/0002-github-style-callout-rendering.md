---
id: IMPL-0002
title: "GitHub-style callout rendering"
status: Draft
author: Donald Gifford
created: 2026-04-03
---
<!-- markdownlint-disable-file MD025 MD041 -->

# IMPL 0002: GitHub-style callout rendering

**Status:** Draft
**Author:** Donald Gifford
**Date:** 2026-04-03

<!--toc:start-->
- [Objective](#objective)
- [Scope](#scope)
  - [In Scope](#in-scope)
  - [Out of Scope](#out-of-scope)
- [Implementation Phases](#implementation-phases)
  - [Phase 1: Dependency and parser integration](#phase-1-dependency-and-parser-integration)
  - [Phase 2: Base callout CSS](#phase-2-base-callout-css)
  - [Phase 3: Theme callout colors](#phase-3-theme-callout-colors)
  - [Phase 4: Tests and documentation](#phase-4-tests-and-documentation)
- [File Changes](#file-changes)
- [Testing Plan](#testing-plan)
- [Dependencies](#dependencies)
- [Open Questions](#open-questions)
- [References](#references)
<!--toc:end-->

## Objective

Add rendering support for GitHub-style callouts/alerts (`> [!NOTE]`,
`> [!TIP]`, `> [!IMPORTANT]`, `> [!WARNING]`, `> [!CAUTION]`) so that
markdown files using this widely-adopted syntax render with proper styling
in mdp's browser preview, matching GitHub's visual treatment.

**Implements:** INV-0001

## Scope

### In Scope

- All 5 GitHub alert types: NOTE, TIP, IMPORTANT, WARNING, CAUTION
- SVG icons matching GitHub's visual style (provided by the extension)
- Per-type accent colors that harmonize with each of the 13 built-in theme CSS files
- Base callout layout CSS (padding, border, border-radius, spacing)
- Parser integration via the `gm-alert-callouts` goldmark extension
- Unit tests for callout parsing and output structure
- `data-source-line` annotation compatibility (callouts should not break scroll sync)

### Out of Scope

- Obsidian-style callouts (custom type names beyond the 5 GitHub types)
- Collapsible/foldable callouts (`+`/`-` suffix syntax)
- Custom icon overrides via CLI flags
- Callout nesting (GitHub does not support this either)
- New theme creation — only adding callout colors to existing themes

## Implementation Phases

Each phase builds on the previous one. A phase is complete when all its tasks
are checked off and its success criteria are met.

---

### Phase 1: Dependency and parser integration

Add the `gm-alert-callouts` goldmark extension and wire it into the parser
pipeline. After this phase, callout markdown produces styled HTML output
instead of raw `[!NOTE]` text in blockquotes.

#### Tasks

- [x] 1. Run `go get github.com/zmtcreative/gm-alert-callouts` to add the dependency
- [x] 2. Run `make update-vendor` or `go mod tidy` to update go.sum
- [x] 3. Add `WithCallouts` option to `internal/parser/parser.go` config struct (default: `true`)
- [x] 4. Wire `alertcallouts.NewAlertCallouts(alertcallouts.UseGFMStrictIcons())` into the goldmark extension slice when callouts are enabled
- [x] 5. Verify with `go build ./...` that the extension compiles and integrates
- [x] 6. Run `make lint && make fmt` to verify code style

#### Success Criteria

- `go build ./...` succeeds
- `make lint` passes
- Rendering `> [!NOTE]\n> Test` produces HTML containing `class="callout callout-note"` instead of literal `[!NOTE]` text
- Rendering a plain blockquote (`> quote`) still produces a normal `<blockquote>`

---

### Phase 2: Base callout CSS

Add layout and structural CSS for callout containers to `assets/preview.css`.
This phase uses CSS custom properties for colors so that themes can override
them. The default (light/dark) values come from GitHub's own color scheme.

#### Tasks

- [x] 1. Add default callout CSS custom properties to `:root` in `assets/preview.css` (light mode values)
- [x] 2. Add dark mode overrides in the existing `@media (prefers-color-scheme: dark)` block
- [x] 3. Add base callout layout rules targeting the extension's CSS classes:
  - `.callout` — container: `border-left: 3px solid var(--callout-color)`, `padding`, `margin-bottom: 16px`, `border-radius: 6px`, `background: var(--callout-bg)`
  - `.callout-title` — flex row, `align-items: center`, `gap: 8px`, `font-weight: 600`, `margin-bottom: 8px`
  - `.callout-title svg` — width/height `16px`, flex-shrink
  - `.callout-title-text` — inherits color from callout type
  - `.callout-body` — paragraph spacing, inherits `--color-fg-default`
  - `.callout-body > :last-child` — `margin-bottom: 0` to remove trailing space
- [x] 4. Add per-type color rules using CSS custom properties:
  - `.callout-note` — `--callout-color: var(--callout-note-color); --callout-bg: var(--callout-note-bg)` (blue)
  - `.callout-tip` — `--callout-color: var(--callout-tip-color); --callout-bg: var(--callout-tip-bg)` (green)
  - `.callout-important` — `--callout-color: var(--callout-important-color); --callout-bg: var(--callout-important-bg)` (purple)
  - `.callout-warning` — `--callout-color: var(--callout-warning-color); --callout-bg: var(--callout-warning-bg)` (yellow)
  - `.callout-caution` — `--callout-color: var(--callout-caution-color); --callout-bg: var(--callout-caution-bg)` (red)
- [x] 5. Run `make build` and visually verify callouts render correctly with the default theme

#### Success Criteria

- Callouts display with colored left border, background tint, icon, and bold title
- Each of the 5 types has a distinct color
- Regular blockquotes are unaffected
- Layout works at various viewport widths (no overflow or clipping)
- Code blocks and lists inside callouts render correctly

---

### Phase 3: Theme callout colors

Add callout color CSS custom properties to each of the 13 built-in theme CSS
files. Colors should be drawn from each theme's existing palette to maintain
visual harmony.

#### Tasks

- [ ] 1. Add callout color and background variables to `assets/themes/github.css` (light+dark auto theme):
  - `--callout-note-color`, `--callout-note-bg`, `--callout-tip-color`, `--callout-tip-bg`, `--callout-important-color`, `--callout-important-bg`, `--callout-warning-color`, `--callout-warning-bg`, `--callout-caution-color`, `--callout-caution-bg`
- [ ] 2. Add callout color variables to Tokyo Night family (4 files):
  - `tokyo-night.css` — night palette: blue `#7aa2f7`, green `#9ece6a`, purple `#bb9af7`, yellow `#e0af68`, red `#f7768e`
  - `tokyo-night-storm.css` — same palette, different canvas
  - `tokyo-night-moon.css` — moon palette: blue `#82aaff`, green `#c3e88d`, purple `#c099ff`, yellow `#ffc777`, red `#ff757f`
  - `tokyo-night-day.css` — day palette: blue `#2e7de9`, green `#587539`, purple `#9854f1`, yellow `#8c6c3e`, red `#f52a65`
- [ ] 3. Add callout color variables to Rose Pine family (3 files):
  - `rose-pine.css` — blue foam `#9ccfd8`, green pine `#31748f`, purple iris `#c4a7e7`, yellow gold `#f6c177`, red love `#eb6f92`
  - `rose-pine-moon.css` — moon variants of same roles
  - `rose-pine-dawn.css` — dawn (light) variants
- [ ] 4. Add callout color variables to Catppuccin family (4 files):
  - `catppuccin-mocha.css` — blue `#89b4fa`, green `#a6e3a1`, purple `#cba6f7`, yellow `#f9e2af`, red `#f38ba8`
  - `catppuccin-macchiato.css` — macchiato palette
  - `catppuccin-frappe.css` — frappe palette
  - `catppuccin-latte.css` — latte (light) palette
- [ ] 5. Add callout color variables to `assets/themes/donald.css`:
  - blue `#7aa2f7`, green `#9ece6a`, purple `#9d7cd8`, yellow `#e0af68`, red `#f7768e`
- [ ] 6. Visually verify at least one theme from each family renders callouts with correct colors

#### Success Criteria

- All 13 theme CSS files contain the 10 callout custom properties (5 `--callout-*-color` + 5 `--callout-*-bg`)
- Each callout type has a visually distinct color per theme
- Light themes (tokyo-night-day, rose-pine-dawn, catppuccin-latte) use appropriately saturated colors that contrast well on light backgrounds
- Dark themes use appropriately bright colors that contrast well on dark backgrounds

---

### Phase 4: Tests and documentation

Add unit tests for callout parsing, integration tests for server-rendered
callout output, and update documentation.

#### Tasks

- [ ] 1. Add `TestRender_GitHubCallout` to `internal/parser/parser_test.go`:
  - Table-driven test covering all 5 types (NOTE, TIP, IMPORTANT, WARNING, CAUTION)
  - Each case: verify `class="callout callout-{type}"` in output
  - Each case: verify `callout-title-text` contains the type name
  - Each case: verify `callout-content` contains the body text
- [ ] 2. Add `TestRender_CalloutPreservesBlockquote` to verify plain `> quote` is unaffected
- [ ] 3. Add `TestRender_CalloutWithNestedContent` to verify code blocks and lists inside callouts render correctly
- [ ] 4. Add `TestRender_CalloutDisabled` to verify `WithCallouts(false)` produces raw blockquote output
- [ ] 5. Verify `data-source-line` annotations work on elements inside callouts (scroll sync compatibility)
- [ ] 6. Run `make test` — all tests pass
- [ ] 7. Run `make lint` — passes with zero warnings
- [ ] 8. Update `CLAUDE.md` if any new patterns or conventions are established
- [ ] 9. Update `README.md` features list to mention GitHub-style callout support

#### Success Criteria

- `make test` passes with zero failures
- `make lint` passes with zero warnings
- Test coverage for `internal/parser` does not decrease
- All 5 callout types tested with correct HTML output assertions
- `WithCallouts(false)` correctly disables the extension
- Scroll sync (`data-source-line`) annotations verified on callout content
- README reflects the new feature

---

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `go.mod` | Modify | Add `github.com/zmtcreative/gm-alert-callouts` dependency |
| `go.sum` | Modify | Updated checksums |
| `internal/parser/parser.go` | Modify | Add `WithCallouts` option; wire extension into goldmark pipeline |
| `internal/parser/parser_test.go` | Modify | Add callout rendering tests (5 types, disabled, nested content) |
| `assets/preview.css` | Modify | Add default callout CSS variables and base layout rules |
| `assets/themes/github.css` | Modify | Add callout color variables |
| `assets/themes/tokyo-night.css` | Modify | Add callout color variables |
| `assets/themes/tokyo-night-storm.css` | Modify | Add callout color variables |
| `assets/themes/tokyo-night-moon.css` | Modify | Add callout color variables |
| `assets/themes/tokyo-night-day.css` | Modify | Add callout color variables |
| `assets/themes/rose-pine.css` | Modify | Add callout color variables |
| `assets/themes/rose-pine-moon.css` | Modify | Add callout color variables |
| `assets/themes/rose-pine-dawn.css` | Modify | Add callout color variables |
| `assets/themes/catppuccin-mocha.css` | Modify | Add callout color variables |
| `assets/themes/catppuccin-macchiato.css` | Modify | Add callout color variables |
| `assets/themes/catppuccin-frappe.css` | Modify | Add callout color variables |
| `assets/themes/catppuccin-latte.css` | Modify | Add callout color variables |
| `assets/themes/donald.css` | Modify | Add callout color variables |
| `README.md` | Modify | Add callout support to features list |

## Testing Plan

- [ ] Unit tests: table-driven for all 5 callout types with HTML output assertions
- [ ] Unit test: plain blockquote unaffected by extension
- [ ] Unit test: nested markdown content (code, lists) inside callouts
- [ ] Unit test: `WithCallouts(false)` disables the extension
- [ ] Integration: `data-source-line` scroll sync annotation compatibility
- [ ] Manual: visual check of each callout type in at least one theme per family

## Dependencies

| Dependency | Version | License | Purpose |
|-----------|---------|---------|---------|
| `github.com/zmtcreative/gm-alert-callouts` | v0.8.0 | MIT | Goldmark AST transformer for GitHub-style alerts |

**Prerequisites:**

- INV-0001 concluded (recommended `gm-alert-callouts`)
- All 13 theme CSS files exist with the current variable structure (completed in IMPL-0001)

## Resolved Questions

1. **Icon rendering:** Try inline SVGs first (extension default). We use `html.WithUnsafe()` so it should work. If issues arise, address them then.

2. **Background tint:** Use per-theme `--callout-{type}-bg` variables (10 vars per theme: 5 color + 5 bg) so backgrounds match each theme's palette precisely rather than relying on computed opacity. This follows the same direct-value pattern established for hljs token rules in IMPL-0001.

3. **License check CI:** Add to allowlist only if the license check fails after adding the dependency.

4. **Extension version pinning:** Pin to exact version (v0.8.0) in `go.mod`. Pre-1.0 API — fork if it breaks.

## References

- [INV-0001: GitHub-style callout and alert rendering](../investigation/0001-github-style-callout-and-alert-rendering.md)
- [GitHub Docs: Alerts syntax](https://docs.github.com/en/get-started/writing-on-github/getting-started-with-writing-and-formatting-on-github/basic-writing-and-formatting-syntax#alerts)
- [gm-alert-callouts](https://github.com/zmtcreative/gm-alert-callouts)
- [IMPL-0001: Theme system](./0001-themes.md) — established theme CSS variable pattern
