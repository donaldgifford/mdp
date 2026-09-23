---
id: IMPL-0007
title: "Mermaid diagram skin per DESIGN-0004"
status: In Progress
author: Donald Gifford
created: 2026-09-22
---

<!-- markdownlint-disable-file MD025 MD041 MD024 -->

# IMPL-0007: Mermaid diagram skin per DESIGN-0004

**Status:** In Progress
**Author:** Donald Gifford
**Date:** 2026-09-22

<!--toc:start-->
- [Objective](#objective)
  - [Design re-verified against the current toolchain](#design-re-verified-against-the-current-toolchain)
  - [New findings not in DESIGN-0004](#new-findings-not-in-design-0004)
- [Scope](#scope)
  - [In Scope](#in-scope)
  - [Out of Scope](#out-of-scope)
- [Implementation Phases](#implementation-phases)
  - [Phase 1: Vendor the fonts](#phase-1-vendor-the-fonts)
    - [Tasks](#tasks)
    - [Success Criteria](#success-criteria)
  - [Phase 2: Skin configuration in preview.js](#phase-2-skin-configuration-in-previewjs)
    - [Tasks](#tasks-1)
    - [Success Criteria](#success-criteria-1)
  - [Phase 3: Palette migration across the fifteen themes](#phase-3-palette-migration-across-the-fifteen-themes)
    - [Tasks](#tasks-2)
    - [Success Criteria](#success-criteria-2)
  - [Phase 4: Documentation](#phase-4-documentation)
    - [Tasks](#tasks-3)
    - [Success Criteria](#success-criteria-3)
  - [Phase 5: Screenshot pass, tuning, and release](#phase-5-screenshot-pass-tuning-and-release)
    - [Tasks](#tasks-4)
    - [Success Criteria](#success-criteria-4)
- [File Changes](#file-changes)
- [Testing Plan](#testing-plan)
- [Performance](#performance)
- [Dependencies](#dependencies)
- [Open Questions](#open-questions)
- [Resolved Decisions](#resolved-decisions)
- [References](#references)
<!--toc:end-->

## Objective

Make the preview's Mermaid diagrams look like
<https://agents.craft.do/mermaid> using mermaid.js's own theme
variables, config, and `themeCSS`: the `neo` look with gradient and
shadow off, Inter labels and JetBrains Mono class members from
vendored fonts, a seven-slot diagram palette per built-in theme seeded
from beautiful-mermaid, and one skin configuration in
`assets/preview.js`.

**Implements:** DESIGN-0004 (all four design questions decided; INV-0004
decisions 1b, 2a, 3a, 4b carried over).

### Design re-verified against the current toolchain

Checked on 2026-09-22 against the vendored Mermaid 12.0.0 bundle and
the upstream sources at `mermaid@12.0.0` / `develop`:

- Every variable in the DESIGN-0004 palette-expansion table exists in
  `themes/theme-base.js` (61 names checked, none missing).
- The `neo` look's rules in `packages/mermaid/src/styles.ts` gate the
  gradient on `useGradient` and the shadow on `dropShadow`:
  `stroke: useGradient ? url(#…-gradient) : nodeBorder` and
  `filter: dropShadow ? … : 'none'`. Setting `useGradient: false` and
  `dropShadow: "none"` in `themeVariables` therefore removes both
  without any `themeCSS` override. `insertLookDefs.ts` still inserts
  the (unused) filter definitions, which is harmless.
- Text inherits from `& svg { font-family: fontFamily; font-size:
  fontSize }` in the global stylesheet, so the two theme variables are
  the font control for every diagram type; sequence diagrams
  additionally read their own `actorFontFamily` / `messageFontFamily` /
  `noteFontFamily` config keys (present in `config.schema.yaml`).
- Front matter inside a fence reaches the browser verbatim through
  `pkg/parser` (INV-0004 Observation 9), which is how individual knobs
  can be tried without a rebuild.
- `mime.TypeByExtension(".woff2")` returns `font/woff2` on this
  machine, and `assets/assets.go` embeds the whole `vendor` tree, so a
  new `vendor/fonts/` directory needs no Go change to be served at
  `/vendor/fonts/…` (`internal/server/server.go:233`).
- The fontsource files are named `inter-latin-wght-normal.woff2` etc.
  and their own CSS declares `font-family: 'Inter Variable'` with
  `font-weight: 100 900` (JetBrains Mono: `100 800`). mdp declares
  them as `"Inter"` and `"JetBrains Mono"`; the family name is ours to
  choose.
- Google Chrome is installed at
  `/Applications/Google Chrome.app/Contents/MacOS/Google Chrome`, so
  `--headless=new --screenshot` and `--dump-dom` would be available;
  Decision 2 chose author-captured screenshots instead.

### New findings not in DESIGN-0004

Three stylesheet details found while re-verifying change what
`themeCSS` must contain. They are folded into Phase 2.

1. **Arrowheads take `lineColor`, not `arrowheadColor`.** The global
   stylesheet has `.marker { fill: lineColor; stroke: lineColor }`.
   DESIGN-0004 already routes arrowhead color through `themeCSS`; this
   confirms it is required, not optional.
2. **Class-diagram text is painted with the border color.** The class
   stylesheet emits `g.classGroup text { fill: nodeBorder || classText
   }` and `.classLabel .label { fill: nodeBorder }`. With the design's
   `border` slot (a 20 % mix, deliberately faint) as `nodeBorder`,
   class members and titles would render nearly invisible. `themeCSS`
   must restore `fill: <fg>` / `color: <fg>` on those selectors, and
   this is where the JetBrains Mono rule goes as well.
3. **Edge labels take the node text color.** Flowchart
   `.label text,span { fill/color: nodeTextColor || textColor }`
   applies to edge labels too, so the muted edge-label color also
   needs `themeCSS` (`.edgeLabel, .edgeLabel span, .edgeLabel p`).

A fourth, cosmetic: under `neo`, gitGraph branch labels set
`filter: url(#…-drop-shadow)` as an **inline** style
(`branchLabelBkg`), which `themeVariables` cannot reach. See Open
Question 6.

## Scope

### In Scope

- Four vendored woff2 files plus two OFL license files under
  `assets/vendor/fonts/`, four `@font-face` rules in
  `assets/preview.css`, and `make update-vendor` lines to refresh them.
- `assets/preview.js`: `SKIN` constants, `readPalette`, `expandPalette`,
  `buildThemeCSS`, `buildMermaidInit`; removal of the twelve inline
  `getPropertyValue` calls and the auto-theme `dark` / `default`
  branch.
- Seven `--mermaid-*` slots in all fifteen built-in themes (thirteen
  CSS files), replacing the legacy twelve, with values from the
  DESIGN-0004 seed table.
- Go tests: `TestPreviewCSSDeclaresVendoredFonts`,
  `TestServer_VendorFontsServed`,
  `TestDiagramPaletteDefinedByEveryTheme`.
- Documentation: Theme CSS Format in `CLAUDE.md`, README Themes
  section, comments in `pkg/theme/theme.go`, DESIGN-0004 tables updated
  to what shipped.
- A screenshot matrix over `docs/examples/all.md` and the built-in
  themes, attached to the PR.

### Out of Scope

- ASCII / box-drawing output — [#89](https://github.com/donaldgifford/mdp/issues/89).
- Any change to `pkg/parser`, `pkg/theme`'s exported API, `pkg/livereload`,
  the CLI flags, or the Neovim plugin.
- Page prose and page code fonts (DESIGN-0004 decision 2a: fonts are
  used inside diagrams only).
- `--dagre` behaviour and the Iconify icon packs (#82, #83) — they must
  keep working, but are not modified.
- Honouring the legacy twelve `--mermaid-<mermaidVariableName>` names in
  custom theme files (DESIGN-0004 decision 3a: ignored; the derived
  palette applies).
- A JavaScript test harness (#77). The skin functions are kept pure and
  table-driven so a harness can cover them later.
- `CHANGELOG.md` edits — generated by `.github/workflows/release.yml`.

## Implementation Phases

Each phase builds on the previous one. A phase is complete when all
its tasks are checked off and its success criteria are met. Every
phase ends with `make fmt`, `make lint`, and `make test`; the lint gate
is "no findings beyond the pre-existing baseline tracked in #80".

Phase 2 is the visible change and can be reviewed on its own
screenshots before the theme files move in Phase 3.

---

### Phase 1: Vendor the fonts

Get Inter and JetBrains Mono into the binary and served, with no
visible change yet. Establishes the `/vendor/fonts/` path the skin
depends on.

#### Tasks

- [ ] 1. Create `assets/vendor/fonts/` and download the four subsets
  from jsDelivr with `curl -sL -o` (pinned major, like the other
  vendored assets):
  `https://cdn.jsdelivr.net/npm/@fontsource-variable/inter@5/files/inter-latin-wght-normal.woff2`,
  `…/inter@5/files/inter-latin-ext-wght-normal.woff2`,
  `…/jetbrains-mono@5/files/jetbrains-mono-latin-wght-normal.woff2`,
  `…/jetbrains-mono@5/files/jetbrains-mono-latin-ext-wght-normal.woff2`.
  Expected sizes at 5.3.0: 47.1 KB, 83.1 KB, 39.5 KB, 14.8 KB.
- [ ] 2. Download `…/inter@5/LICENSE` as
  `assets/vendor/fonts/LICENSE-Inter` and `…/jetbrains-mono@5/LICENSE`
  as `assets/vendor/fonts/LICENSE-JetBrainsMono` (both SIL OFL 1.1).
- [ ] 3. Add the same six `curl` lines to the `update-vendor` target in
  `Makefile` (after the hljs lines, before the `✓` echo) so the fonts
  refresh with everything else. Replace the trailing "update KaTeX
  fonts manually" note with one that covers both font sets.
- [ ] 4. Add four `@font-face` rules at the top of `assets/preview.css`
  (before `:root`), one per file, declaring `font-family: "Inter"` /
  `"JetBrains Mono"`, `font-style: normal`, `font-weight: 100 900`
  (Inter) / `100 800` (JetBrains Mono), `font-display: swap`,
  `src: url("/vendor/fonts/<file>") format("woff2")`, and the
  `unicode-range` copied verbatim from the fontsource `index.css` for
  that subset (latin:
  `U+0000-00FF,U+0131,U+0152-0153,U+02BB-02BC,U+02C6,U+02DA,U+02DC,U+0304,U+0308,U+0329,U+2000-206F,U+20AC,U+2122,U+2191,U+2193,U+2212,U+2215,U+FEFF,U+FFFD`;
  latin-ext:
  `U+0100-02BA,U+02BD-02C5,U+02C7-02CC,U+02CE-02D7,U+02DD-02FF,U+0304,U+0308,U+0329,U+1D00-1DBF,U+1E00-1E9F,U+1EF2-1EFF,U+2020,U+20A0-20AB,U+20AD-20C0,U+2113,U+2C60-2C7F,U+A720-A7FF`).
- [ ] 5. Add `assets/fontcss_test.go` with
  `TestPreviewCSSDeclaresVendoredFonts`: parse every `@font-face` block
  in `preview.css`, assert one family named `Inter` and one named
  `JetBrains Mono` exist, and assert every `url(/vendor/fonts/…)` it
  references is present in `assets.FS`. Same package and style as
  `footnotecss_test.go`.
- [ ] 6. Add `TestServer_VendorFontsServed` to
  `internal/server/server_test.go`: start a server with `fetchBody`'s
  config pattern, `GET /vendor/fonts/inter-latin-wght-normal.woff2`,
  assert status 200, `Content-Type` starting with `font/woff2`, and a
  body longer than 1 KB.
- [ ] 7. `make build`; record the binary size before and after in this
  document (baseline from #82: 26 MB; expected delta ≈ +0.2 MB).
- [ ] 8. `make fmt && make lint && make test`.

#### Success Criteria

- `assets/vendor/fonts/` contains four `.woff2` files and two license
  files, all reachable at `/vendor/fonts/…` from a running server.
- `TestPreviewCSSDeclaresVendoredFonts` and
  `TestServer_VendorFontsServed` pass.
- `make update-vendor` re-downloads the fonts byte-identically (run it,
  `git status` shows no change).
- No visible change in the preview (fonts are declared but nothing
  references them yet).
- Binary size delta recorded and under 0.3 MB.

---

### Phase 2: Skin configuration in preview.js

Replace the inline Mermaid initialisation with the skin: `neo` look
without gradient or shadow, vendored fonts, spacing, sequence config,
`themeCSS`, and a palette reader with the `--color-*` fallback. This
phase delivers most of the visual change and is reviewable on its own
screenshots. Theme files are untouched; every theme takes the derived
palette until Phase 3 (Open Question 1).

#### Tasks

- [ ] 1. Add a `SKIN` constant near the top of the IIFE in
  `assets/preview.js`:
  `font: '"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif'`,
  `mono: '"JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, monospace'`.
- [ ] 2. Add `mixHex(fg, bg, pct)` — per-channel sRGB mix returning a
  six-digit lowercase hex, the same arithmetic as beautiful-mermaid's
  `color-mix(in srgb, fg pct%, bg)` and as the DESIGN-0004 seed table.
- [ ] 3. Add `readPalette(style)`: read `--mermaid-bg`, `-fg`, `-line`,
  `-accent`, `-muted`, `-surface`, `-border` from
  `getComputedStyle(document.body)`. If `--mermaid-bg` is empty, derive
  from the required prose properties: `bg` ← `--color-canvas-default`,
  `fg` ← `--color-fg-default`, `muted` ← `--color-fg-muted`, `accent` ←
  `--color-accent-fg`, `line` ← `mixHex(fg, bg, 50)`, `surface` ←
  `mixHex(fg, bg, 3)`, `border` ← `mixHex(fg, bg, 20)`. Any individual
  slot that is empty after reading is filled by the same derivation, so
  a custom theme may set only some slots.
- [ ] 4. Add `expandPalette(p)` returning the `themeVariables` object:
  the slot-to-variable table from DESIGN-0004 "Palette expansion"
  (61 variables) plus the fixed entries `useGradient: false`,
  `dropShadow: "none"`, `strokeWidth: 1`, `radius: 6`,
  `fontFamily: SKIN.font`, `fontSize: "13px"`. Keep it as a literal
  object so the mapping is readable in one screen.
- [ ] 5. Add `buildThemeCSS(p)` returning:

  ```css
  .edgeLabel, .edgeLabel span, .edgeLabel p { color: <muted>; font-size: 11px; }
  .marker, .marker path { fill: <accent>; stroke: <accent>; }
  .marker.cross { stroke: <accent>; }
  g.classGroup text, .classLabel .label { fill: <fg>; font-family: <mono>; font-size: 12px; }
  .classGroup .nodeLabel, .classGroup .label { color: <fg>; font-family: <mono>; font-size: 12px; }
  .classTitle, .classTitleText { font-family: <font>; font-weight: 600; }
  .cluster-label text, .cluster-label span { font-size: 12px; font-weight: 600; }
  .branchLabelBkg { filter: none !important; } /* gitGraph sets this inline under neo; Decision 6 */
  ```

  The class rules exist because of
  [new finding 2](#new-findings-not-in-design-0004); the selector list
  is confirmed against the rendered DOM in task 9 and trimmed to what
  actually matches.
- [ ] 6. Add `buildMermaidInit(palette, layout)` returning the object in
  DESIGN-0004 "Skin configuration": `startOnLoad: false`,
  `look: "neo"`, `layout` only when set (preserves the `--dagre`
  escape hatch), `fontFamily: SKIN.font`, `theme: "base"`,
  `themeVariables: expandPalette(palette)`,
  `themeCSS: buildThemeCSS(palette)`,
  `flowchart: { nodeSpacing: 24, rankSpacing: 40, diagramPadding: 8 }`,
  `sequence: { actorFontFamily, messageFontFamily, noteFontFamily:
  SKIN.font, actorFontSize: 13, messageFontSize: 12, noteFontSize: 12 }`.
- [ ] 7. Replace `assets/preview.js:39-72` (from `var prefersDark` to
  the closing `}` of the `else` branch) with
  `mermaid.initialize(buildMermaidInit(readPalette(getComputedStyle(document.body)), document.body.dataset.mermaidLayout));`.
  Delete `prefersDark` and `mermaidTheme`; `data-mermaid-theme` stays
  on `<body>` for `internal/server` and its tests but is no longer read
  here. Leave the Iconify registration (lines 16-38) and
  `renderClientSide` untouched.
- [ ] 8. Update the comment block above the initialisation to describe
  the skin in three lines: neo look with gradient and shadow off,
  palette from the seven `--mermaid-*` slots or derived from
  `--color-*`, everything else constant.
- [ ] 9. `make build`, serve `docs/examples/all.md` with
  `--theme=tokyo-night` and `--theme=github-light`, and pause for the
  author's screenshots (Decision 2); use the browser's element
  inspector for the selector check. Confirm: no
  gradient strokes, no shadow, Inter visible on labels, JetBrains Mono
  on class members, class text readable, edge labels muted, arrowheads
  in accent, all twelve diagram types rendered, `architecture-beta`
  icons present. Trim the task-5 selectors to those that match.
- [ ] 10. Serve with `--dagre` and confirm the layout still switches.
- [ ] 11. Serve with `--theme=auto` and confirm both schemes render with
  the derived palette (toggle the OS appearance or use Chrome's
  `--force-dark-mode`).
- [ ] 12. `make fmt && make lint && make test` — no Go changes expected
  in this phase, so this is a regression check.

#### Success Criteria

- `preview.js` contains no `--mermaid-<mermaidVariableName>` reads and no
  `theme: "dark"` / `"default"` branch; all Mermaid configuration flows
  through `buildMermaidInit`.
- Screenshots for tokyo-night and github-light show every check in
  task 9 met, on all twelve example diagram types.
- `--dagre` and `--theme=auto` behave as before (layout switch; scheme
  tracking).
- `readPalette`, `expandPalette`, `buildThemeCSS`, `buildMermaidInit`,
  and `mixHex` are pure functions of their arguments (no DOM access
  inside), so #77 can test them later.
- Existing Go tests pass unchanged, including
  `TestServer_MermaidThemeAttribute` and
  `TestServer_MermaidLayoutAttribute`.

---

### Phase 3: Palette migration across the fifteen themes

Give every built-in theme its seven slots with the DESIGN-0004 seed
values, remove the legacy twelve, and guard the contract with a test.

#### Tasks

- [ ] 1. In each of the thirteen files under `assets/themes/`, replace
  the `/* Mermaid theme variables … */` block (twelve
  `--mermaid-<name>` lines; `github.css` has three such blocks) with the
  seven-slot block, values copied from the DESIGN-0004 "Seed values"
  table for that theme. Use the tokyo-night block in DESIGN-0004
  "Diagram palette" as the template, including its per-slot comments.
  Files: `catppuccin-frappe`, `catppuccin-latte`, `catppuccin-macchiato`,
  `catppuccin-mocha`, `donald`, `github` (light, dark, dimmed),
  `rose-pine-dawn`, `rose-pine-moon`, `rose-pine`, `tokyo-night-day`,
  `tokyo-night-moon`, `tokyo-night-storm`, `tokyo-night`.
- [ ] 2. Update each file's header comment where it references Mermaid
  variables so it names the seven slots.
- [ ] 3. Add `assets/diagramcss_test.go` with
  `TestDiagramPaletteDefinedByEveryTheme`: for every
  `[data-theme="…"] {…}` block in `assets/themes/*.css` that defines
  `--color-fg-default`, assert the seven slots are present, each value
  matches `^#[0-9a-f]{6}$`, and no `--mermaid-` name outside the seven
  appears anywhere in the file. Derive the theme list from `assets.FS`
  so a new theme is covered automatically (same approach as
  `footnotecss_test.go`).
- [ ] 4. Update the comments in `pkg/theme/theme.go` (line 15 "mermaid
  vars", lines 43-45 `mermaidBase`) to describe the seven slots and
  that `preview.js` derives a palette when they are absent. No exported
  API change; `Theme.MermaidTheme` keeps its values.
- [ ] 5. Screenshots of `docs/examples/all.md` for the seven seeded
  themes (tokyo-night, tokyo-night-storm, tokyo-night-day,
  github-light, github-dark, catppuccin-latte, catppuccin-mocha) —
  confirm the palette changed from the Phase 2 derived one to the
  upstream values (tokyo-night edges are now `#3d59a1`).
- [ ] 6. Write a throwaway custom theme file containing only the nine
  `--color-*` properties, serve with `--theme=/path/to/it.css`, and
  confirm diagrams render with a derived palette (decision 3a).
- [ ] 7. `make fmt && make lint && make test`.

#### Success Criteria

- `grep -r -- '--mermaid-' assets/themes` lists only the seven slot
  names, 105 occurrences (15 themes × 7).
- `TestDiagramPaletteDefinedByEveryTheme` passes and fails when any one
  slot is deleted from any theme (verify once by hand).
- Seeded-theme screenshots match the upstream palette values; derived
  themes are unchanged from Phase 2.
- The custom theme file without slots renders diagrams with a derived
  palette and no console errors.
- `pkg/theme` tests and `internal/server` tests pass unchanged.

---

### Phase 4: Documentation

Bring the written contract in line with the code.

#### Tasks

- [ ] 1. `CLAUDE.md` "Theme CSS Format": replace the twelve-line
  `--mermaid-*` example with the seven slots and a one-line note that
  values must be six-digit hex (mermaid does color arithmetic on them;
  `var()` and `color-mix()` are not accepted). Keep the hljs rules and
  the keyword/operator warning unchanged. Add a bullet pointing at
  `TestDiagramPaletteDefinedByEveryTheme` next to the existing "Update
  theme count assertions" bullet.
- [ ] 2. `README.md` "Themes": amend the closing sentence ("Each built-in
  theme provides … Mermaid diagram theming …") and add a short
  "Custom theme files" subsection: the nine `--color-*` properties are
  required, the seven `--mermaid-*` slots are optional and are derived
  from the prose colors when absent, and diagrams use vendored Inter and
  JetBrains Mono.
- [ ] 3. `CLAUDE.md` architecture notes: add a paragraph under the
  existing Mermaid notes describing `buildMermaidInit` as the single
  place Mermaid is configured, the `neo` look with gradient and shadow
  off, and that `themeCSS` exists because of the three stylesheet
  findings above (so nobody removes it as redundant).
- [ ] 4. `docs/examples/README.md`: one line noting the examples are the
  screenshot corpus for the diagram skin.
- [ ] 5. Run `markdownlint-cli2` on every edited markdown file.

#### Success Criteria

- `CLAUDE.md` and `README.md` describe exactly the seven slots the test
  enforces; no mention of the legacy twelve remains anywhere in the
  repo (`grep -rn "primaryBorderColor" --include=*.md --include=*.css
  .` returns nothing outside `docs/`).
- `markdownlint-cli2` reports zero issues on the edited files.

---

### Phase 5: Screenshot pass, tuning, and release

Verify the whole matrix, tune what looks wrong, and ship.

#### Tasks

- [ ] 1. Author captures `docs/examples/all.md` under tokyo-night,
  github-light, catppuccin-mocha, rose-pine-dawn, and auto in light and
  dark (Decision 2). Attach the images to the PR.
- [ ] 2. Check every item of the DESIGN-0004 manual matrix on each
  image: no gradient, no shadow, Inter labels, JetBrains Mono class
  members, muted edge labels, accent arrowheads, twelve types rendered,
  `architecture-beta` icons loaded, `--dagre` switch, custom theme
  fallback.
- [ ] 3. Judge node padding against the Craft page (neo pins 28 × 24 px;
  Craft uses 20 × 10 px). Decision 3 pre-authorises the switch to
  `look: "classic"` with `flowchart.padding: 10` if nodes are
  noticeably roomier; record the outcome under
  [Resolved Decisions](#resolved-decisions) either way.
- [ ] 4. Tune seed values or the expansion table where a theme reads
  wrong (typical candidates: `surface` / `border` on the derived
  themes, `line` on dark themes). Keep the JS derivation and the CSS
  seeds consistent.
- [ ] 5. Update the DESIGN-0004 "Seed values" and "Palette expansion"
  tables and the `themeCSS` block to what shipped; set DESIGN-0004
  status to `Implemented`.
- [ ] 6. Record the final binary size and the count of theme variables
  set per theme in this document.
- [ ] 7. `make fmt && make lint && make test && make build`; run
  `go test -race ./...` once.
- [ ] 8. Open the PR against `main` with label `minor`, the screenshot
  matrix, and links to INV-0004, DESIGN-0004, and this document. After
  merge, set this document's status to `Completed`.

#### Success Criteria

- All matrix checks pass on all six captures, and the images are in
  the PR.
- Open Question 3 is resolved and recorded; if `classic` was chosen,
  `buildMermaidInit` and DESIGN-0004 both say so.
- DESIGN-0004 tables match the shipped code.
- CI (lint, test, build) is green on the PR; `go test -race ./...` is
  clean.

## File Changes

| File | Action | Description |
| ---- | ------ | ----------- |
| `assets/vendor/fonts/inter-latin-wght-normal.woff2` | Create | Inter variable, latin subset (47.1 KB) |
| `assets/vendor/fonts/inter-latin-ext-wght-normal.woff2` | Create | Inter variable, latin-ext subset (83.1 KB) |
| `assets/vendor/fonts/jetbrains-mono-latin-wght-normal.woff2` | Create | JetBrains Mono variable, latin (39.5 KB) |
| `assets/vendor/fonts/jetbrains-mono-latin-ext-wght-normal.woff2` | Create | JetBrains Mono variable, latin-ext (14.8 KB) |
| `assets/vendor/fonts/LICENSE-Inter`, `LICENSE-JetBrainsMono` | Create | SIL OFL 1.1 texts |
| `assets/preview.css` | Modify | Four `@font-face` rules before `:root` |
| `assets/preview.js` | Modify | `SKIN`, `mixHex`, `readPalette`, `expandPalette`, `buildThemeCSS`, `buildMermaidInit`; remove inline variable reads and auto branch |
| `assets/themes/*.css` (13 files) | Modify | Seven `--mermaid-*` slots replace the legacy twelve |
| `assets/fontcss_test.go` | Create | `TestPreviewCSSDeclaresVendoredFonts` |
| `assets/diagramcss_test.go` | Create | `TestDiagramPaletteDefinedByEveryTheme` |
| `internal/server/server_test.go` | Modify | `TestServer_VendorFontsServed` |
| `Makefile` | Modify | `update-vendor` fetches fonts and licenses |
| `pkg/theme/theme.go` | Modify | Comments only |
| `CLAUDE.md`, `README.md`, `docs/examples/README.md` | Modify | Theme CSS Format, Themes section, corpus note |
| `docs/design/0004-…md` | Modify | Tables updated to shipped values; status Implemented |

## Testing Plan

- **Go, `assets/`:** `TestPreviewCSSDeclaresVendoredFonts` (Phase 1),
  `TestDiagramPaletteDefinedByEveryTheme` (Phase 3). Both derive their
  expectations from the embedded files rather than hardcoded lists.
- **Go, `internal/server`:** `TestServer_VendorFontsServed` (Phase 1);
  existing `TestServer_MermaidThemeAttribute`,
  `TestServer_MermaidLayoutAttribute`, `TestServer_AllBuiltinThemes`
  must stay green without edits.
- **Go, `pkg/theme`:** no new tests; `TestAllBuiltins` and
  `TestEmbeddedThemeFilesExist` stay green.
- **Browser (author screenshots, Decision 2):** the DESIGN-0004
  matrix at the Phase 2, Phase 3, and Phase 5 gates, plus an element
  inspector check once in Phase 2 to confirm the `themeCSS` selectors
  match real classes.
- **Race:** `go test -race ./...` once in Phase 5 (no new goroutines
  are introduced; this is a regression check).

## Performance

No runtime cost is expected. `readPalette` and `expandPalette` run once
at page load; `mermaid.initialize` is already called once today. The
`@font-face` declarations use `font-display: swap` and `unicode-range`,
so a diagram with only ASCII labels decodes one 47 KB file. The binary
grows by the six vendored files (≈ 195 KB, under 1 % of the current
26 MB). Phase 1 task 7 and Phase 5 task 6 record the measured sizes.

## Dependencies

- No new Go modules.
- `@fontsource-variable/inter` and `@fontsource-variable/jetbrains-mono`
  5.x files (SIL OFL 1.1), fetched by `curl` in `make update-vendor`;
  not an npm dependency of the repo.

## Open Questions

All six were decided by the author on 2026-09-22 — see
[Resolved Decisions](#resolved-decisions). Each retains its options for
the record; the decision is marked on the question line.

**1. What feeds the palette in Phase 2, before the theme files migrate?** — **Decided: a** (2026-09-22)

- **a.** Implement `readPalette` with the `--color-*` fallback
  immediately. No theme defines the seven slots yet, so every theme
  takes the derived palette in Phase 2, and Phase 3 only adds CSS. The
  fallback is a permanent feature (auto and custom themes), so no
  throwaway code, and the Phase 2 screenshots show the exact code path
  that ships.
- **b.** Add a temporary adapter that maps the legacy twelve variables
  onto the seven slots, as DESIGN-0004's rollout text literally says,
  then delete it in Phase 3. Phase 2 screenshots keep each theme's
  current hand-tuned colors, at the cost of code that exists for one
  phase.
- Other:

**2. How are screenshots captured during implementation?** — **Decided: b** (2026-09-22)

- **a.** Add `scripts/screenshot.sh <theme> <file.md> <out.png>`: starts
  `mdp serve --browser=false --port <n> --theme <theme> <file>`, runs
  Chrome with `--headless=new --window-size=1400,6000
  --virtual-time-budget=8000 --screenshot=<out>` against
  `http://127.0.0.1:<n>/`, with a `--dump-dom` mode for selector
  checks, then stops the server. Dev-only, not embedded. Lets me verify
  each phase's visual criteria myself and makes the PR matrix
  reproducible.
- **b.** You capture screenshots by hand at the Phase 2, 3, and 5 gates,
  as with the footnote work. No script in the repo; each gate waits on
  you.
- Other:

**3. Who decides the neo-padding fallback in Phase 5?** — **Decided: a** (2026-09-22)

- **a.** Pre-authorise it: if the Phase 5 pass shows rectangles
  noticeably roomier than the Craft page, switch to `look: "classic"`
  with `flowchart.padding: 10` (INV-0004 option 1a), record the switch
  in DESIGN-0004 and here, and include before/after images in the PR.
- **b.** Stop at that point and ask, with side-by-side screenshots, before
  changing the look.
- Other:

**4. Branch and PR shape?** — **Decided: a** (2026-09-22)

- **a.** One branch `feat/mermaid-skin` from `main`, one PR, one commit
  per task with conventional-commit messages, all five phases — the
  shape #76 used. Reviewable as a unit; the screenshot matrix documents
  the whole change once.
- **b.** Two PRs: Phase 1 (fonts, no visible change) first, then
  Phases 2-5. Smaller diffs, but the first PR adds 195 KB with nothing
  to show for it until the second lands.
- Other:

**5. Where does the skin code live?** — **Decided: a** (2026-09-22)

- **a.** In `assets/preview.js`, next to the code it replaces. No new
  asset, no template change, no extra request; the functions are kept
  pure and grouped under one comment header so they are easy to lift
  into a module if #77 lands a harness.
- **b.** A new `assets/mermaid-skin.js` loaded by a second `<script>`
  tag in `preview.html` before `{{.JS}}`. Cleaner separation, at the
  cost of a new embedded asset, a template edit, and one more file for
  `assets_test` to know about.
- Other:

**6. gitGraph branch labels keep their neo shadow via an inline style. Fix or accept?** — **Decided: a** (2026-09-22)

- **a.** Add one `themeCSS` rule, `.branchLabelBkg { filter: none
  !important; }`. A stylesheet `!important` outranks an inline
  non-important style, so this is the only way to reach it from
  config; it is the single `!important` in the skin and is commented as
  such.
- **b.** Accept it as a known cosmetic gap. gitGraph is not one of the
  diagram types the Craft look covers, and the shadow only touches
  branch labels.
- Other:

## Resolved Decisions

Decided by the author on 2026-09-22. Decisions from INV-0004 (1b, 2a,
3a, 4b, 5 → #89) and DESIGN-0004 (1a, 2a, 3a, 4a) are inputs to this
document and are not repeated here.

1. **Phase 2 palette source (Q1 → a).** `readPalette` ships with the
   `--color-*` derived fallback from the start; every theme is on the
   final code path in Phase 2 and Phase 3 only adds CSS. No temporary
   adapter.
2. **Screenshots (Q2 → b).** Captured by hand by the author at the
   Phase 2, Phase 3, and Phase 5 gates. No `scripts/screenshot.sh`;
   the implementation pauses at those gates and requests captures.
3. **Neo-padding fallback (Q3 → a).** Pre-authorised: if the Phase 5
   pass shows rectangles noticeably roomier than the Craft page,
   switch to `look: "classic"` with `flowchart.padding: 10`, record
   it here and in DESIGN-0004, and include before/after images in the
   PR.
4. **Branch and PR (Q4 → a).** One branch `feat/mermaid-skin` from
   `main`, one PR, one conventional commit per task, all five phases.
5. **Code placement (Q5 → a).** All skin code lives in
   `assets/preview.js` under one comment header, as pure functions.
6. **gitGraph branch-label shadow (Q6 → a).** One `themeCSS` rule,
   `.branchLabelBkg { filter: none !important; }`, commented as the
   skin's only `!important`.

## References

- [DESIGN-0004](../design/0004-mermaid-diagram-skin-via-theme-variables.md)
  — seed table, palette expansion, `themeCSS`, fonts, rollout plan
- [INV-0004](../investigation/0004-evaluate-beautiful-mermaid-for-diagram-rendering-and-ascii.md)
  — Observations 4, 5, 9; Decisions
- [#89](https://github.com/donaldgifford/mdp/issues/89) — deferred
  ASCII mode; [#77](https://github.com/donaldgifford/mdp/issues/77) —
  JS test harness; [#80](https://github.com/donaldgifford/mdp/issues/80)
  — golangci-lint baseline
- [IMPL-0006](0006-footnote-support-per-design-0003.md) — the phase /
  task / success-criteria shape this document follows
- Mermaid sources (`packages/mermaid/src`): `styles.ts` (global rules:
  `.marker`, `& svg` font, `[data-look="neo"]` gradient and shadow),
  `diagrams/flowchart/styles.ts` (`.label`, `.edgeLabel`,
  `.cluster-label`), `diagrams/class/styles.js` (`g.classGroup text`,
  `.classLabel .label`), `rendering-util/insertLookDefs.ts`,
  `themes/theme-base.js`, `schemas/config.schema.yaml`
- fontsource: `@fontsource-variable/inter@5.3.0/index.css`,
  `@fontsource-variable/jetbrains-mono@5.3.0/index.css` (family names,
  weight ranges, `unicode-range` values)
- `assets/preview.js:11-72` (current init), `assets/preview.css:1-50`
  (`:root` and the dark media query), `assets/assets.go` (embed),
  `internal/server/server.go:228-233` (vendor route),
  `internal/server/server_test.go:79,110,149,182` (`fetchBody`,
  `tempMDFile`, Mermaid attribute tests), `Makefile:87-99`
  (`update-vendor`)
