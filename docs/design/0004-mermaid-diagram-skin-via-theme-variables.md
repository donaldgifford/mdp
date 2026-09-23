---
id: DESIGN-0004
title: "Mermaid diagram skin via theme variables"
status: Implemented
author: Donald Gifford
created: 2026-09-22
---

<!-- markdownlint-disable-file MD025 MD041 -->

# DESIGN-0004: Mermaid diagram skin via theme variables

**Status:** Implemented
**Author:** Donald Gifford
**Date:** 2026-09-22

<!--toc:start-->
- [Overview](#overview)
- [Goals and Non-Goals](#goals-and-non-goals)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Background](#background)
  - [What renders today](#what-renders-today)
  - [The target](#the-target)
  - [Precedents in this repo](#precedents-in-this-repo)
- [Detailed Design](#detailed-design)
  - [Diagram palette: seven custom properties per theme](#diagram-palette-seven-custom-properties-per-theme)
  - [Seed values](#seed-values)
  - [Skin configuration in preview.js](#skin-configuration-in-previewjs)
  - [Palette expansion](#palette-expansion)
  - [themeCSS](#themecss)
  - [Fonts](#fonts)
  - [Auto and custom themes](#auto-and-custom-themes)
- [API / Interface Changes](#api--interface-changes)
- [Data Model](#data-model)
- [Testing Strategy](#testing-strategy)
- [Migration / Rollout Plan](#migration--rollout-plan)
- [Open Questions](#open-questions)
- [Resolved Decisions](#resolved-decisions)
  - [Amendment (2026-09-23): series palette](#amendment-2026-09-23-series-palette)
- [References](#references)
<!--toc:end-->

## Overview

Restyle the preview's Mermaid diagrams to match the look of
<https://agents.craft.do/mermaid> using mermaid.js's own theme variables,
config, and `themeCSS`, rather than adopting a second renderer. Each
built-in theme gains a seven-color diagram palette modelled on
beautiful-mermaid's `bg / fg / line / accent / muted / surface / border`,
seeded verbatim from its palettes where the two projects share a theme.
Everything that is not a color — look, fonts, stroke, radius, spacing —
is set once in `preview.js`. Inter and JetBrains Mono are vendored and
embedded so the result is identical offline.

This implements the decisions recorded in
[INV-0004](../investigation/0004-evaluate-beautiful-mermaid-for-diagram-rendering-and-ascii.md).

## Goals and Non-Goals

### Goals

- Diagrams read as the Craft look: flat 1 px strokes, rounded corners,
  no gradient, no drop shadow, Inter labels, JetBrains Mono class
  members, muted edge labels, accent-colored arrowheads, tighter
  spacing.
- tokyo-night, tokyo-night-storm, tokyo-night-day, github-light,
  github-dark, catppuccin-latte, and catppuccin-mocha use
  beautiful-mermaid's palette values; the other eight built-ins get a
  coherent palette derived the same way upstream derives its own.
- All twelve diagram types in `docs/examples` keep rendering, including
  `architecture-beta` with its Iconify icons.
- Fully offline: fonts are embedded in the binary like the KaTeX fonts.
- `--theme=<file>` custom themes and `--theme=auto` keep working.

### Non-Goals

- ASCII / box-drawing output — tracked in
  [#89](https://github.com/donaldgifford/mdp/issues/89).
- Replacing or adding to mermaid.js. `pkg/parser` is untouched.
- Changing `--dagre` (#82) or the Iconify packs (#83).
- Per-theme geometry or typography. Themes supply colors only
  (INV-0004 decision 2a).
- Changing the prose or code fonts of the page. Inter and JetBrains
  Mono are used inside diagrams only (see Open Question 2).
- Pixel-identical output to beautiful-mermaid. Layout is still ELK via
  mermaid.js; the goal is the same visual language.

## Background

### What renders today

`assets/preview.js` calls `mermaid.initialize({ theme: "base",
themeVariables: {…} })` with twelve color variables read from
`--mermaid-*` custom properties, and nothing else. Mermaid 12 (vendored
since #82) resolves `look` and `theme` per diagram type in the order
front matter, `initialize()`, the diagram type's default, then the
global default. mdp sets `theme` but never `look`, so flowchart,
sequence, class, state, ER, requirement, use case, agentflow, and
swimlane diagrams all render in v12's `neo` look on top of the base
theme. That combination brings:

- gradient node strokes (`useGradient: true` in base, only disabled by
  setting `nodeBorder` or `useGradient: false`);
- a drop shadow (`dropShadow` in base);
- `"trebuchet ms", verdana, arial` at 16 px, with sequence actors in
  `"Open Sans"` at 14 px, because none of the twelve variables is a
  font;
- 28 × 24 px rectangle padding, which `neo` pins regardless of
  `flowchart.padding`.

INV-0004 Observation 9 has the full trait-by-trait comparison.

### The target

beautiful-mermaid's `src/styles.ts` fixes the numbers behind the Craft
look: Inter at 13 px (weight 500) for node labels, 11 px for edge
labels, 12 px (weight 600) for group headers; JetBrains Mono for class
members; 1 px outer strokes, 0.75 px inner, 1 px connectors; 8 × 5 px
arrowheads; node padding 20 × 10; `nodeSpacing` 24, `layerSpacing` 40.
Its color model is two required slots (`bg`, `fg`) plus five optional
ones, each of which falls back to a `color-mix()` of `fg` into `bg` at a
fixed weight: line 50 %, arrow 85 %, muted text 40 %, node fill 3 %,
node stroke 20 %.

### Precedents in this repo

- `assets/vendor/katex/fonts/` holds 20 woff2 files (296 KB) embedded
  through `//go:embed … vendor …` in `assets/assets.go` and served by
  `http.FileServer` under `/vendor/` (`internal/server/server.go:233`).
  `mime.TypeByExtension(".woff2")` resolves to `font/woff2`.
- `assets/footnotecss_test.go` derives the custom properties a CSS rule
  consumes and asserts every built-in theme defines them. The palette
  test below follows that pattern.
- The Theme CSS Format in `CLAUDE.md` already requires a `--mermaid-*`
  block per theme; this design changes its contents, not its existence.

## Detailed Design

### Diagram palette: seven custom properties per theme

The twelve `--mermaid-<mermaidVariableName>` properties are replaced by
seven slot properties named after beautiful-mermaid's model. Values are
plain hex so `preview.js` can hand them to mermaid, which does its own
color arithmetic on them and cannot consume `var()` or `color-mix()`.

```css
[data-theme="tokyo-night"] {
  /* … prose custom properties … */

  /* Diagram palette (beautiful-mermaid model) — required */
  --mermaid-bg:      #1a1b26;  /* canvas behind the diagram; edge-label backing */
  --mermaid-fg:      #a9b1d6;  /* node, actor, title text */
  --mermaid-line:    #3d59a1;  /* edges, lifelines, relations, transitions */
  --mermaid-accent:  #7aa2f7;  /* arrowheads, activations, special states */
  --mermaid-muted:   #565f89;  /* edge labels, secondary text */
  --mermaid-surface: #1e202b;  /* node, actor, note, cluster fill */
  --mermaid-border:  #373949;  /* node, actor, cluster stroke */
}
```

Seven values per theme is the whole per-theme contract. The mapping
from slots to mermaid's variables lives in one place in `preview.js`
(next sections), so a theme author never has to know mermaid's variable
names.

### Seed values

`upstream` rows copy beautiful-mermaid's `THEMES` entry for the five
slots it defines (`tokyo-night-day` takes `tokyo-night-light` with mdp's
page background). `derived` rows take `bg`, `fg`, `muted`, and `accent`
from the theme's own `--color-canvas-default`, `--color-fg-default`,
`--color-fg-muted`, and `--color-accent-fg`, and `line` from a 50 % mix.
`surface` and `border` are the 3 % and 20 % mixes for every theme, which
is exactly what upstream computes when those slots are unset.

| Theme | Source | bg | fg | line | accent | muted | surface | border |
| ----- | ------ | -- | -- | ---- | ------ | ----- | ------- | ------ |
| github-light | upstream | `#ffffff` | `#1f2328` | `#d1d9e0` | `#0969da` | `#59636e` | `#f8f8f9` | `#d2d3d4` |
| github-dark | upstream | `#0d1117` | `#e6edf3` | `#3d444d` | `#4493f8` | `#9198a1` | `#14181e` | `#383d43` |
| github-dimmed | derived | `#22272e` | `#adbac7` | `#68707a` | `#539bf5` | `#768390` | `#262b33` | `#3e444d` |
| tokyo-night | upstream | `#1a1b26` | `#a9b1d6` | `#3d59a1` | `#7aa2f7` | `#565f89` | `#1e202b` | `#373949` |
| tokyo-night-moon | derived | `#222436` | `#c8d3f5` | `#757c96` | `#82aaff` | `#828bb8` | `#27293c` | `#43475c` |
| tokyo-night-storm | upstream | `#24283b` | `#a9b1d6` | `#3d59a1` | `#7aa2f7` | `#565f89` | `#282c40` | `#3f435a` |
| tokyo-night-day | upstream | `#e1e2e7` | `#343b58` | `#34548a` | `#34548a` | `#9699a3` | `#dcdde3` | `#bec1ca` |
| rose-pine | derived | `#191724` | `#e0def4` | `#7c7a8c` | `#c4a7e7` | `#908caa` | `#1f1d2a` | `#413f4e` |
| rose-pine-moon | derived | `#232136` | `#e0def4` | `#828095` | `#c4a7e7` | `#908caa` | `#29273c` | `#49475c` |
| rose-pine-dawn | derived | `#faf4ed` | `#575279` | `#a8a3b3` | `#907aa9` | `#9893a5` | `#f5efea` | `#d9d4d6` |
| donald | derived | `#16161e` | `#b4b4b4` | `#656569` | `#7aa2f7` | `#6b6b6b` | `#1b1b22` | `#36363c` |
| catppuccin-latte | upstream | `#eff1f5` | `#4c4f69` | `#9ca0b0` | `#8839ef` | `#9ca0b0` | `#eaecf1` | `#ced1d9` |
| catppuccin-frappe | derived | `#303446` | `#c6d0f5` | `#7b829e` | `#8caaee` | `#737994` | `#34394b` | `#4e5369` |
| catppuccin-macchiato | derived | `#24273a` | `#cad3f5` | `#777d98` | `#8aadf4` | `#6e738d` | `#292c40` | `#45495f` |
| catppuccin-mocha | upstream | `#1e1e2e` | `#cdd6f4` | `#585b70` | `#cba6f7` | `#6c7086` | `#232434` | `#414356` |

Shipped unchanged in IMPL-0007 Phase 3. These are starting values. The
screenshot pass in the rollout plan may
swap `surface` / `border` for a theme's `--color-canvas-subtle` /
`--color-border-default` where the mix reads wrong against that page.

### Skin configuration in preview.js

Everything that is not a color is a constant in `preview.js`, applied
identically for every theme:

```js
var SKIN = {
  font: '"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
  mono: '"JetBrains Mono", ui-monospace, SFMono-Regular, Menlo, monospace',
};

function buildMermaidInit(palette, layout) {
  return {
    startOnLoad: false,
    look: "neo",                       // INV-0004 decision 1b
    layout: layout || undefined,       // "dagre" escape hatch from #82, else v12 ELK default
    fontFamily: SKIN.font,
    theme: "base",
    themeVariables: expandPalette(palette),
    themeCSS: buildThemeCSS(palette),
    flowchart: { nodeSpacing: 24, rankSpacing: 40, diagramPadding: 8 },
    sequence: {
      actorFontFamily: SKIN.font, messageFontFamily: SKIN.font, noteFontFamily: SKIN.font,
      actorFontSize: 13, messageFontSize: 12, noteFontSize: 12,
    },
  };
}
```

`readPalette()` replaces the current twelve `getPropertyValue` calls: it
reads the seven `--mermaid-*` slots from `getComputedStyle(document.body)`
and, when `--mermaid-bg` is empty, derives the palette from the required
`--color-*` properties with the same mixes as the seed table (see
[Auto and custom themes](#auto-and-custom-themes)).

Consequence of `look: "neo"` worth stating up front: neo pins rectangle
padding at 28 × 24 px, wider than the 20 × 10 px target, and
`flowchart.padding` does not apply to it. If the screenshot pass finds
nodes too roomy, the fallback is `look: "classic"` with
`flowchart.padding: 10`, which is INV-0004's option 1a.

### Palette expansion

`expandPalette(p)` turns seven slots into the mermaid variables that
matter, plus the fixed geometry. Grouped by slot:

| Slot | mermaid variables |
| ---- | ----------------- |
| `bg` | `background`, `edgeLabelBackground`, `labelBackgroundColor` |
| `fg` | `primaryTextColor`, `textColor`, `nodeTextColor`, `titleColor`, `actorTextColor`, `signalTextColor`, `labelTextColor`, `loopTextColor`, `noteTextColor`, `classText`, `stateLabelColor`, `transitionLabelColor` |
| `line` | `lineColor`, `defaultLinkColor`, `signalColor`, `actorLineColor`, `transitionColor`, `relationColor`, `archEdgeColor` |
| `accent` | `arrowheadColor`, `archEdgeArrowColor`, `activationBorderColor`, `specialStateColor` |
| `muted` | `secondaryTextColor`, `tertiaryTextColor`, `sequenceNumberColor` (edge-label text goes through `themeCSS`) |
| `surface` | `primaryColor`, `secondaryColor`, `tertiaryColor`, `nodeBkg`, `mainBkg`, `actorBkg`, `noteBkgColor`, `labelBoxBkgColor`, `activationBkgColor`, `clusterBkg`, `stateBkg`, `compositeBackground`, `altBackground`, `attributeBackgroundColorOdd`, `attributeBackgroundColorEven`, `requirementBackground` |
| `border` | `primaryBorderColor`, `secondaryBorderColor`, `tertiaryBorderColor`, `nodeBorder`, `clusterBorder`, `actorBorder`, `noteBorderColor`, `labelBoxBorderColor`, `compositeBorder`, `requirementBorderColor`, `archGroupBorderColor` |
| fixed | `useGradient: false`, `dropShadow: "none"`, `strokeWidth: 1`, `radius: 6`, `fontFamily: SKIN.font`, `fontSize: "13px"` |

Setting `nodeBorder` already switches the gradient off in base;
`useGradient: false` is set as well so the intent is explicit. The
list shipped exactly as tabled in IMPL-0007 Phase 2: 56 colour
variables plus the 6 fixed entries, 62 in total. The screenshot pass may
still trim or extend it; update this table to match if it does.

### themeCSS

Three things have no variable and go through `config.themeCSS`, which
mermaid injects inside its own id-scoped `<style>` block so it
outranks the theme's stylesheet where page CSS would not:

As shipped (IMPL-0007 Phase 2; the class rules also restore `fill` /
`color`, because mermaid paints class text with `nodeBorder`):

```css
.edgeLabel, .edgeLabel span, .edgeLabel p { color: <muted>; font-size: 11px; }
.marker, .marker path { fill: <accent>; stroke: <accent>; }
.marker.cross { stroke: <accent>; }
g.classGroup text, .classLabel .label { fill: <fg>; font-family: <mono>; font-size: 12px; }
.classGroup .nodeLabel, .classGroup .label { color: <fg>; font-family: <mono>; font-size: 12px; }
.classTitle, .classTitleText { font-family: <font>; font-weight: 600; }
.cluster-label text, .cluster-label span { font-size: 12px; font-weight: 600; }
.branchLabelBkg { filter: none !important; } /* gitGraph inline shadow under neo */
```

`buildThemeCSS(palette)` interpolates the slot values. Selectors are
taken from mermaid's v12 flowchart and class stylesheets; confirming
each against the rendered DOM is part of the IMPL-0007 screenshot
gates. A wrong selector is a cosmetic miss, not a broken diagram.
Arrowhead *size* is not adjustable through CSS and stays at mermaid's
default.

### Fonts

Vendored from the fontsource variable packages (SIL OFL 1.1, LICENSE
shipped alongside) into `assets/vendor/fonts/`, which the existing
`//go:embed … vendor …` picks up:

| File | Size |
| ---- | ---- |
| `inter-latin-wght-normal.woff2` | 47.1 KB |
| `inter-latin-ext-wght-normal.woff2` | 83.1 KB |
| `jetbrains-mono-latin-wght-normal.woff2` | 39.5 KB |
| `jetbrains-mono-latin-ext-wght-normal.woff2` | 14.8 KB |
| `LICENSE-Inter`, `LICENSE-JetBrainsMono` | 4.4 KB each |

`preview.css` declares them with the same `unicode-range` split
fontsource uses, so a browser only decodes the subset a diagram needs:

```css
@font-face {
  font-family: "Inter";
  font-style: normal;
  font-weight: 100 900;
  font-display: swap;
  src: url("/vendor/fonts/inter-latin-wght-normal.woff2") format("woff2");
  unicode-range: U+0000-00FF, U+0131, U+0152-0153, U+02BB-02BC, U+02C6, U+02DA, U+02DC,
    U+0304, U+0308, U+0329, U+2000-206F, U+20AC, U+2122, U+2191, U+2193, U+2212, U+2215, U+FEFF, U+FFFD;
}
/* latin-ext, JetBrains Mono latin and latin-ext follow the same shape */
```

`make update-vendor` gains six `curl` lines against
`cdn.jsdelivr.net/npm/@fontsource-variable/inter@5/files/…` and
`…/jetbrains-mono@5/files/…` plus the two LICENSE files, so the fonts
refresh with the other vendored assets instead of by hand.

Mermaid renders labels in `foreignObject` HTML by default, so the
`@font-face` on the page applies to them directly; `fontFamily` in the
config covers the SVG `<text>` paths (sequence, class, ER) as well.

### Auto and custom themes

- **`--theme=auto`** has no theme stylesheet; `preview.css` defines the
  `--color-*` properties under a `prefers-color-scheme` media query.
  `readPalette()` finds no `--mermaid-bg` and derives the seven slots
  from `--color-*` with the seed-table mixes. Auto therefore gets the
  same skin, with colors that track the scheme. The current
  `theme: "dark"` / `"default"` branch for auto is removed;
  `data-mermaid-theme` on `<body>` keeps its values (`"base"` for named
  themes, `""` for auto) so `internal/server` and its tests do not
  change.
- **`--theme=<file>`** custom themes that follow the documented format
  define the nine `--color-*` properties, so they take the derived path
  too. A custom file that adds the seven `--mermaid-*` slots gets full
  control. The legacy twelve `--mermaid-<mermaidVariableName>` names are
  no longer read (Open Question 3).

## API / Interface Changes

- **No Go API or CLI change.** `pkg/parser`, `pkg/theme`, `pkg/livereload`,
  and the flag set are untouched.
- **Theme CSS Format** (`CLAUDE.md`, README theming section): the
  required `--mermaid-*` block becomes the seven slots above. All
  fifteen built-in theme files are updated.
- **New embedded assets:** `assets/vendor/fonts/` (≈ 195 KB) served at
  `/vendor/fonts/…`.
- **`assets/preview.js`:** `readPalette`, `expandPalette`,
  `buildThemeCSS`, `buildMermaidInit` replace the inline variable list
  and the auto-theme branch.
- **`assets/preview.css`:** four `@font-face` rules.
- **`Makefile`:** `update-vendor` fetches the fonts.

## Data Model

Not applicable. The only stored data is the seed table above, held as
CSS custom properties in the theme files.

## Testing Strategy

There is no JavaScript test harness (#77), so the Go side guards the
contracts the JS depends on and a manual screenshot matrix covers the
rendering.

**Go, `assets/`:**

- `TestDiagramPaletteDefinedByEveryTheme` — for every
  `[data-theme="…"]` block in `assets/themes/*.css` that defines
  `--color-fg-default`, all seven `--mermaid-*` slots are present with a
  six-digit hex value, and none of the legacy names remain. Same shape
  as `TestFootnoteCSSPropertiesDefinedByEveryTheme`.
- `TestPreviewCSSDeclaresVendoredFonts` — `preview.css` has an
  `@font-face` for `Inter` and for `JetBrains Mono`, and every
  `url(/vendor/fonts/…)` it references exists in `assets.FS`.

**Go, `internal/server`:**

- `TestVendorFontsServed` — `GET /vendor/fonts/inter-latin-wght-normal.woff2`
  returns 200 with `Content-Type: font/woff2`.
- Existing `data-mermaid-theme` / `data-mermaid-layout` assertions stay
  green unchanged.

**Manual matrix (screenshots attached to the PR):**
`docs/examples/all.md` under tokyo-night, github-light,
catppuccin-mocha, rose-pine-dawn, and auto in both schemes. Checks: no
gradient strokes, no shadow, Inter labels and JetBrains Mono class
members visibly applied, edge labels muted, arrowheads in accent, all
twelve diagram types render, `architecture-beta` icons still load,
`--dagre` still switches layout, a custom theme file without the seven
slots renders with a derived palette.

## Migration / Rollout Plan

1. **Fonts.** Vendor the four woff2 files and licenses, add the
   `@font-face` rules and the `update-vendor` lines, add the two Go
   tests for them. No visible change yet.
2. **Skin without touching themes.** Add `look`, fonts, gradient and
   shadow off, spacing, sequence config, and `themeCSS` to
   `preview.js`, still fed by the existing twelve variables. This is
   most of the visual gain and is reviewable on its own screenshots.
3. **Palette.** Migrate all fifteen theme files to the seven slots with
   the seed table, switch `preview.js` to `readPalette` /
   `expandPalette` with the `--color-*` fallback, remove the auto-theme
   branch, add `TestDiagramPaletteDefinedByEveryTheme`.
4. **Docs.** Update the Theme CSS Format in `CLAUDE.md`, the README
   theming section, and `CHANGELOG.md`. PR label `minor`.
5. **Screenshot pass** across the matrix; tune seeds and the variable
   list; update this document's tables to what shipped.

Rollback at any step is a revert; nothing persists outside the binary.

## Open Questions

Each lists **a** as my recommendation and **b…** as alternatives.
All four were decided on 2026-09-22 and stay for the record.

**1. Which font subsets ship?** — **Decided: a** (2026-09-22)

- **a.** Latin and latin-ext for both faces (≈ 185 KB of woff2). Covers
  Western European diacritics in labels; `unicode-range` means the
  extra subset is only decoded when used.
- **b.** Latin only (≈ 87 KB). Smaller binary; labels with `ø`, `ł`,
  `ș` fall back to the system font mid-diagram.
- Other:

**2. How far outside the diagrams do the vendored fonts reach?** — **Decided: a** (2026-09-22)

To be precise about what each option changes: today the page prose uses
the system stack and page `code` / `pre` blocks use the `ui-monospace`
stack (`preview.css:64`, `:202`). Inside diagrams, this design applies
Inter to labels and JetBrains Mono to class-diagram member text only.

- **a.** Diagrams only. Page prose and page `code` / `pre` blocks are
  unchanged; JetBrains Mono appears solely in class-diagram members.
  Smallest scope and exactly INV-0004 decision 4b.
- **b.** Diagrams plus page code: also set `code, pre { font-family:
  "JetBrains Mono", … }` so fenced code blocks, inline code, and class
  members share one mono face. One CSS rule; prose stays on the system
  stack. Changes the look of every code block on every theme.
- **c.** Full Craft: **b** plus `body { font-family: "Inter", … }` so
  prose matches the diagram labels too.
- Other:

**3. What happens to the legacy twelve `--mermaid-*` names in custom theme files?** — **Decided: a** (2026-09-22)

- **a.** Ignore them. A custom theme without the seven slots gets the
  derived palette from its `--color-*` properties, which every documented
  custom theme already defines. One code path, documented in the
  CHANGELOG as a behaviour change.
- **b.** Map the legacy names to the slots for one minor release with a
  `console.info` deprecation, then remove. Preserves hand-tuned custom
  palettes for a release at the cost of a second code path.
- Other:

**4. Starting corner radius?** — **Decided: a** (2026-09-22)

- **a.** `radius: 6`. Slightly rounder than base's 5; adjusted in the
  screenshot pass.
- **b.** Keep base's 5.
- Other:

## Resolved Decisions

Carried over from INV-0004 (2026-09-22):

1. `look: "neo"` with `useGradient: false` and `dropShadow: "none"` (1b).
2. Non-color settings live once in `preview.js` (2a).
3. Per-theme colors grow to the seven-slot model seeded from
   beautiful-mermaid's palettes (3a).
4. Inter and JetBrains Mono are vendored (4b).
5. ASCII output is deferred to #89.

Decided in this document (2026-09-22):

- Latin and latin-ext subsets ship for both faces (Open Question 1a).
- The vendored fonts are used inside diagrams only; page prose and
  code blocks keep their current stacks (Open Question 2a).
- Custom theme files without the seven slots get the derived palette;
  the legacy twelve names are ignored (Open Question 3a).
- Starting corner radius is 6 (Open Question 4a).

### Amendment (2026-09-23): series palette

The seven slots made every diagram monochrome. Diagrams whose meaning
depends on distinct hues (timeline, kanban, and mindmap sections,
gitGraph branches, pie slices, journey sections and actors, xychart
series) lost the colours mermaid used to derive from an accent-filled
`primaryColor`, and on dark themes mermaid's derivation from the new
near-black `primaryColor` gave black sections. Each theme now adds an
optional eight-hue series, `--mermaid-series-1` to `-8`, taken from the
theme's own palette with series-1 equal to the accent. `expandPalette`
maps it onto `cScale*` (18% tint over surface, full-strength
`cScaleInv*` rule), `git*`, `pie*`, `fillType*` (25% tint), and
`xyChart.plotColorPalette` (which must be passed complete: xychart merges
it over the stock light theme). Journey actor dots are set in
`buildThemeCSS` because mermaid's config merge appends
`journey.actorColours` instead of replacing it. Missing entries fall back
to accent, success, danger, their 50% mixes, muted, and fg. Nodes,
edges, and borders stay monochrome.

Semantic node classes (same date). `buildThemeCSS` styles nodes with the
class `danger`, `success`, `warning`, or `accent` (plain `class D danger`,
no `classDef`) with an 18% tint of the theme's status colour and a
full-strength stroke. Colours come from `--color-danger-fg`,
`--color-success-fg`, `--callout-warning-color`, and the accent slot.
Colouring edges by label text ("No", "needs work") was rejected as a
guess that is often wrong, and Mermaid v12 does not apply classes to
edges. Edges instead take the status names through `linkStyle`
(`linkStyle 2 stroke:danger`). Mermaid passes the unknown value through
to the inline style; `colourEdges` runs after each render, replaces the
name with the theme colour, and points every edge that has its own
stroke (named or hex) at a marker clone filled with that colour.

## References

- [INV-0004](../investigation/0004-evaluate-beautiful-mermaid-for-diagram-rendering-and-ascii.md),
  Observations 4, 5, 9 and the Decisions section
- [#89](https://github.com/donaldgifford/mdp/issues/89) — deferred ASCII mode
- PR #82 — Mermaid v12, ELK default, `--dagre`; PR #83 — Iconify packs,
  `docs/examples/`
- beautiful-mermaid `src/theme.ts` (`THEMES`, `MIX`) and `src/styles.ts`:
  <https://github.com/lukilabs/beautiful-mermaid>
- Mermaid 12.0.0 release notes (per-type `look` / `theme`, gradient note):
  <https://github.com/mermaid-js/mermaid/releases/tag/mermaid%4012.0.0>
- Mermaid `themes/theme-base.js` (224 variables), `mermaidAPI.ts`
  (`createUserStyles`, `themeCSS`), `schemas/config.schema.yaml`
- Fonts: <https://www.npmjs.com/package/@fontsource-variable/inter>,
  <https://www.npmjs.com/package/@fontsource-variable/jetbrains-mono>
  (5.3.0, SIL OFL 1.1)
- `assets/assets.go` (embed), `internal/server/server.go:233` (vendor
  route), `assets/footnotecss_test.go` (CSS test pattern)
