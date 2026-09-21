---
id: INV-0004
title: "Evaluate beautiful-mermaid for diagram rendering and ASCII output"
status: In Progress
author: Donald Gifford
created: 2026-09-21
---

<!-- markdownlint-disable-file MD025 MD041 -->

# INV-0004: Evaluate beautiful-mermaid for diagram rendering and ASCII output

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
- [Approach](#approach)
- [Environment](#environment)
- [Findings](#findings)
  - [Observation 1: What beautiful-mermaid is](#observation-1-what-beautiful-mermaid-is)
  - [Observation 2: Diagram-type coverage against mdp's examples](#observation-2-diagram-type-coverage-against-mdps-examples)
  - [Observation 3: Distribution format and real client payload](#observation-3-distribution-format-and-real-client-payload)
  - [Observation 4: Theme model and how it maps onto mdp's themes](#observation-4-theme-model-and-how-it-maps-onto-mdps-themes)
  - [Observation 5: Text measurement and fonts](#observation-5-text-measurement-and-fonts)
  - [Observation 6: ASCII rendering, client-side and server-side](#observation-6-ascii-rendering-client-side-and-server-side)
  - [Observation 7: What a switch would orphan](#observation-7-what-a-switch-would-orphan)
  - [Observation 8: Where it plugs into mdp's pipeline](#observation-8-where-it-plugs-into-mdps-pipeline)
- [Conclusion](#conclusion)
- [Recommendation](#recommendation)
- [Open Questions](#open-questions)
- [References](#references)
<!--toc:end-->

## Question

Can mdp render Mermaid diagrams with [beautiful-mermaid](https://github.com/lukilabs/beautiful-mermaid)
(the renderer behind <https://agents.craft.do/mermaid>) so that diagrams
pick up the look of the built-in themes — Tokyo Night in particular — and
offer an ASCII / Unicode box-drawing render mode, **without** regressing
the diagram types, layout controls, and icon packs mdp ships today?

## Hypothesis

- **Aesthetics and theming: yes.** beautiful-mermaid ships a Tokyo Night
  palette and exposes colors as CSS custom properties, so matching mdp's
  themes should be a mapping exercise rather than new CSS.
- **Coverage: no, not fully.** beautiful-mermaid is an independent
  implementation, not a mermaid.js skin, so it will support a subset of
  diagram types. Some of mdp's `docs/examples` will not render with it.
- **ASCII: yes, client-side.** The package exposes an ASCII renderer
  directly, so the preview can call it without a Go-side dependency.
- **Payload: smaller.** Expected a meaningful reduction against the
  5.4 MB vendored `mermaid.min.js`.

## Context

Mermaid diagrams in the preview currently use stock mermaid.js v12 styled
through twelve `--mermaid-*` custom properties per theme (see the Theme
CSS Format in `CLAUDE.md`). The result is functional but visibly "stock
Mermaid". beautiful-mermaid produces the cleaner, monochrome-with-accent
look seen on the Craft demo page and also emits box-drawing ASCII, which
would be a distinctive feature for a terminal-editor preview tool.

Two recent changes constrain any move: #82 made ELK the default layout
with a `--dagre` escape hatch, and #83 registered Iconify packs for
`architecture-beta` diagrams and added `docs/examples/` as the diagram
regression corpus. Whatever is adopted must keep those examples rendering.

**Triggered by:** user request on branch `docs/ui-updates` after reviewing
<https://agents.craft.do/mermaid>. Related: PR #82, PR #83,
`pkg/parser.WithMermaidRenderMode`.

## Approach

1. Inventory beautiful-mermaid from the npm registry, jsDelivr, and the
   GitHub source: API surface, supported diagram types, theme model,
   published module format, and dependencies.
2. Inventory mdp's Mermaid pipeline: `pkg/parser` options,
   `assets/preview.js` / `preview.html`, `internal/server` template data,
   the `--mermaid-*` theme properties, and the `update-vendor` Makefile
   target.
3. Cross-check the diagram types used by `docs/examples/*.md` against
   beautiful-mermaid's parser to size the coverage gap.
4. Measure the real client payload: the published bundle plus its
   external dependencies, versus the vendored `mermaid.min.js`.
5. Evaluate ASCII options: the client-side `renderMermaidASCII` API, and
   the Go project it was ported from (`mermaid-ascii`) as a server-side
   alternative, including a throwaway-module probe of its dependency
   footprint and exported API.
6. Map beautiful-mermaid's color slots onto mdp's existing `--color-*`
   properties and compare against its built-in palettes for the themes
   both projects share.

## Environment

| Component | Version / Value |
| --------- | --------------- |
| mdp | v0.5.0 (`main` at `d45c7b3`), branch `docs/ui-updates` |
| Go | 1.26.8 (`go.mod`) |
| Vendored `mermaid.min.js` | 12.0.0, 5,448 KB (ELK inlined by Mermaid v12) |
| `go.abhg.dev/goldmark/mermaid` | v0.6.0 |
| beautiful-mermaid | 1.1.3 on npm (published 2026-02-26), MIT, ESM only |
| elkjs (beautiful-mermaid dependency) | 0.11.0 |
| entities (beautiful-mermaid dependency) | 7.0.1 |
| mermaid-ascii (Go) | tag `1.6.1` (2026-09-08); resolves as pseudo-version `v0.0.0-20260908213847-5f00e3d9ac9f` because tags lack the `v` prefix |
| Local JS toolchain (dev machine only) | node 24.14.0, bun 1.3.14, esbuild 0.28.1; none pinned in `mise.toml`, `update-vendor` is curl-only |

## Findings

### Observation 1: What beautiful-mermaid is

beautiful-mermaid is a from-scratch TypeScript implementation of a Mermaid
subset — its own parser, its own SVG renderer, and layout via ELK.js. It
does **not** wrap or depend on mermaid.js. Public API (from
`src/index.ts` and `dist/index.d.ts`):

| Export | Notes |
| ------ | ----- |
| `renderMermaidSVG(text, opts): string` | **Synchronous.** ELK's FakeWorker is patched so layout runs inline (`src/elk-instance.ts`). |
| `renderMermaidSVGAsync(text, opts)` | Same output, promise-wrapped. |
| `renderMermaidASCII(text, opts): string` | Box-drawing or plain-ASCII text output. |
| `parseMermaid(text): MermaidGraph` | Throws `Invalid mermaid header: …` on unsupported diagram types. |
| `THEMES`, `fromShikiTheme`, `DEFAULTS` | 15 named palettes; Shiki theme adapter. |

Render options (README table): `bg`, `fg` (required foundation), optional
`line`, `accent`, `muted`, `surface`, `border`, plus `font` (default
`Inter`), `transparent`, `padding`, `nodeSpacing`, `layerSpacing`,
`componentSpacing`, `thoroughness` (ELK crossing-minimisation trials),
`interactive`. Every color option accepts a hex value **or a CSS variable
string**; colors are emitted as custom properties on the `<svg>` element
(`--bg`, `--fg`, `--line`, `--accent`, `--muted`, `--surface`, `--border`)
so a theme change restyles an already-rendered diagram with no re-render.

### Observation 2: Diagram-type coverage against mdp's examples

beautiful-mermaid's parser accepts exactly six headers: `flowchart` /
`graph`, `stateDiagram-v2`, `sequenceDiagram`, `classDiagram`,
`erDiagram`, and `xychart-beta`. `docs/examples/` (added in #83, validated
with `mermaid.parse()`) exercises twelve types:

| Type used in `docs/examples` | beautiful-mermaid | Notes |
| ---------------------------- | ----------------- | ----- |
| flowchart / graph | supported | |
| sequenceDiagram | supported | |
| classDiagram | supported | |
| stateDiagram-v2 | supported | |
| erDiagram | supported | |
| architecture-beta | **unsupported** | 10 uses; the entire point of #83's Iconify packs |
| usecase-beta | **unsupported** | |
| agentflow-beta | **unsupported** | |
| timeline | **unsupported** | |
| kanban | **unsupported** | |
| gitGraph | **unsupported** | |
| requirementDiagram | **unsupported** | |

Five of twelve types render; seven do not, including the most-used one.
A wholesale replacement of mermaid.js is therefore off the table. The
failure mode is clean and deterministic — `parseMermaid` throws before
any DOM work — so a hybrid renderer can try beautiful-mermaid first and
hand the block to `mermaid.run()` on `catch`.

### Observation 3: Distribution format and real client payload

The npm tarball ships only `dist/index.js` (ESM, 327.7 KB unminified),
`dist/index.d.ts`, and a source map. `tsup.config.ts` marks `elkjs` and
`entities` as `external`, and the published file still contains bare
imports:

```js
import { decodeXML } from "entities";
import ELKBundled from "elkjs/lib/elk.bundled.js";
```

There is **no browser-global build in the package**. `src/browser.ts`
exists but is a demo-only entry that hangs functions off
`window.__mermaid` for the repo's `samples.html`; it is built ad hoc with
`Bun.build` and not published. mdp cannot vendor `dist/index.js` with a
plain `<script>` tag the way it vendors `mermaid.min.js`; a bundling step
(or the jsDelivr `+esm` variant, which rewrites the two imports to
`/npm/...` URLs) is required.

Real client payload, measured from jsDelivr:

| Asset | Size |
| ----- | ---- |
| `beautiful-mermaid@1.1.3/+esm` (minified, deps external) | 155 KB |
| `elkjs@0.11.0/lib/elk.bundled.js` | 1,589 KB |
| `entities@7.0.1` (ESM dist) | ~87 KB |
| **Total beautiful-mermaid path** | **~1.8 MB** |
| `assets/vendor/mermaid.min.js` (v12, ELK inlined) | 5,448 KB |

So the honest comparison is roughly **3× smaller**, not the 16× the bare
`dist/index.js` number suggests — ELK dominates both. In a hybrid design
the preview would load **both** engines (~7.3 MB of embedded JS instead
of 5.4 MB) unless mermaid.js is loaded lazily only when an unsupported
type is encountered.

The ASCII path is lighter: `src/ascii/index.ts` imports only the parser
and the ASCII modules, never ELK. A tree-shaken ASCII-only bundle would
be small, but that again requires a bundler; the single published entry
cannot be split by hand.

### Observation 4: Theme model and how it maps onto mdp's themes

beautiful-mermaid ships 15 palettes: `zinc-light`, `zinc-dark`,
`tokyo-night`, `tokyo-night-storm`, `tokyo-night-light`,
`catppuccin-mocha`, `catppuccin-latte`, `nord`, `nord-light`, `dracula`,
`github-light`, `github-dark`, `solarized-light`, `solarized-dark`,
`one-dark`. mdp registers 15 themes. Overlap:

| mdp theme | beautiful-mermaid palette |
| --------- | ------------------------- |
| tokyo-night, tokyo-night-storm | exact name match |
| tokyo-night-day | `tokyo-night-light` (same upstream palette, different name) |
| github-light, github-dark | exact name match |
| catppuccin-latte, catppuccin-mocha | exact name match |
| tokyo-night-moon, catppuccin-frappe, catppuccin-macchiato, github-dimmed, rose-pine, rose-pine-moon, rose-pine-dawn, donald | no counterpart |

Seven of fifteen have a ready-made palette; eight need a derived one.
mdp's existing prose properties map naturally onto the seven color
slots:

| beautiful-mermaid slot | mdp property |
| ---------------------- | ------------ |
| `bg` | `--color-canvas-default` |
| `fg` | `--color-fg-default` |
| `muted` | `--color-fg-muted` |
| `accent` | `--color-accent-fg` |
| `border` | `--color-border-default` |
| `surface` | `--color-canvas-subtle` |
| `line` | `--color-border-default` (no direct equivalent) |

Checking that derivation against upstream's hand-tuned values for the
shared themes shows where it drifts:

| Theme / slot | upstream palette | derived from mdp `--color-*` |
| ------------ | ---------------- | ---------------------------- |
| tokyo-night `bg` / `fg` | `#1a1b26` / `#a9b1d6` | `#1a1b26` / `#c0caf5` (mdp's `fg-muted` is `#a9b1d6`) |
| tokyo-night `accent` | `#7aa2f7` | `#7aa2f7` |
| tokyo-night `muted` | `#565f89` | `#a9b1d6` |
| tokyo-night `line` | `#3d59a1` | `#29293d` (upstream uses a distinctly brighter blue for edges) |
| github-dark `accent` | `#4493f8` | `#58a6ff` |
| github-dark `line` / `muted` | `#3d444d` / `#9198a1` | `#30363d` / `#8b949e` |
| github-light `accent` | `#0969da` | `#0969da` |
| catppuccin-mocha `accent` | `#cba6f7` (mauve) | `#89b4fa` (blue) |
| catppuccin-mocha `line` | `#585b70` | `#313244` |

`bg` and `fg` land exactly everywhere; `accent`, `line`, and `muted`
diverge on several themes. To get the look on the Craft page for Tokyo
Night specifically, the diagram palette needs its own values rather
than a pure derivation from prose colors. This mirrors the existing
convention where each theme file already carries dedicated
`--mermaid-*` properties.

### Observation 5: Text measurement and fonts

beautiful-mermaid runs without a DOM (Bun / Node / browser alike), so it
cannot measure text with `canvas.measureText`. `src/text-metrics.ts`
estimates widths from character-class buckets tuned for Inter, and the
SVG carries `font` (default `Inter`). mdp's preview body uses the
system-UI stack (`-apple-system, BlinkMacSystemFont, "Segoe UI", …`,
`assets/preview.css:64`). If mdp passes a different font, labels may be
slightly over- or under-sized relative to their boxes; mermaid.js does not
have this class of issue because it measures in the live DOM. This is a
visual-verification item, not a blocker.

### Observation 6: ASCII rendering, client-side and server-side

**Client-side.** `renderMermaidASCII(text, opts)` supports flowchart,
sequence, class, ER, and XY chart (not state). `AsciiRenderOptions`:

```ts
useAscii?: boolean          // true = + - | > ; false (default) = ┌ ─ │ ►
paddingX?: number           // default 5
paddingY?: number           // default 5
boxBorderPadding?: number   // default 1
colorMode?: 'none' | 'auto' | 'ansi16' | 'ansi256' | 'truecolor' | 'html'
theme?: Partial<AsciiTheme> // via diagramColorsToAsciiTheme(colors)
```

`colorMode: 'html'` emits `<span style="color:…">` runs, so the output
drops straight into a `<pre>` and can be themed with the same color slots
as the SVG path. No ELK involvement.

**Server-side.** The ASCII engine is a port of
[mermaid-ascii](https://github.com/AlexanderGrooff/mermaid-ascii) (Go,
MIT, actively released — 1.6.1 on 2026-09-08). Its README documents only
the CLI, but the packages under `pkg/` are importable. A throwaway-module
probe shows:

```go
// github.com/AlexanderGrooff/mermaid-ascii/pkg/render
func RenderDiagram(input string, config *diagram.Config) (string, error)
func RenderDiagramWithStatus(input string, config *diagram.Config) (string, WidthStatus, error)
func DiagramFactory(input string) (diagram.Diagram, error)

// github.com/AlexanderGrooff/mermaid-ascii/pkg/diagram
type Config struct {
    UseAscii, ShowCoords, Verbose bool
    BoxBorderPadding, PaddingBetweenX, PaddingBetweenY, MaxWidth int
    GraphDirection string // "LR" | "TD"
    StyleType      string // "cli" | "html"
}
```

Importing `pkg/render` pulls in `logrus`, `gookit/color`,
`mattn/go-runewidth`, `elliotchance/orderedmap/v2`, `clipperhouse/uax29`,
`xo/terminfo`, and `golang.org/x/sys` — no gin or cobra, which stay in
the CLI module. Costs: an undocumented library surface, a pseudo-version
dependency (tags are not `v`-prefixed), and coverage limited to
flowchart / sequence / ER (no class or XY chart).

The goldmark hook for a server-side path exists: `goldmark-mermaid`
exports `mermaid.Kind` and `mermaid.Block`, so a custom `NodeRenderer`
can emit `<pre class="mermaid-ascii">…</pre>` at parse time. The
`Compiler` interface (`CompileRequest{Source}` → `CompileResponse{SVG}`)
is SVG-only and is not the right seam for text output.

### Observation 7: What a switch would orphan

| Feature | Effect under beautiful-mermaid |
| ------- | ------------------------------ |
| `--dagre` (#82) | Meaningless. beautiful-mermaid is ELK-only; its knobs are `thoroughness` and the spacing options. Still valid for the mermaid.js fallback path. |
| Iconify packs (#83) | Unused. Only `architecture-beta` consumes them, and that type falls back to mermaid.js anyway. |
| `--mermaid-*` theme properties (12 per theme) | Unused by beautiful-mermaid; still required by the fallback path. |
| `data-mermaid-theme` / `data-mermaid-layout` body attributes | Fallback path only. |
| `pkg/parser` public API | Unaffected for a client-side design: the parser keeps emitting `<pre class="mermaid">`. A server-side ASCII path would add a `With…` option. |

### Observation 8: Where it plugs into mdp's pipeline

`renderClientSide()` in `assets/preview.js` clears `data-processed` on
`.mermaid` nodes and calls `mermaid.run({ nodes })`. A hybrid would walk
the same nodes first:

```js
for (const node of content.querySelectorAll(".mermaid")) {
  try {
    node.innerHTML = renderMermaidSVG(node.textContent, {
      bg: "var(--diagram-bg)", fg: "var(--diagram-fg)", /* … */
      transparent: true,
    });
    node.dataset.processed = "true";     // keep mermaid.run() off it
  } catch (_) { /* unsupported type: leave for mermaid.run() */ }
}
mermaid.run({ nodes: content.querySelectorAll(".mermaid:not([data-processed])") });
```

Because the SVG call is synchronous and colors are CSS variables, there
is no async re-init dance and theme switching is free. Because the
published module is ESM, `preview.html` would load it with
`<script type="module">` (or an IIFE produced by the bundling step),
and `data-source-line` annotations on the `<pre>` wrapper are untouched,
so scroll sync is unaffected.

## Conclusion

**Answer:** Yes, with limits.

- Adopting beautiful-mermaid as **the** renderer is not viable: it covers
  five of the twelve diagram types in `docs/examples` and none of the
  `architecture-beta` diagrams #83 was built for.
- Adopting it as the **first-choice renderer for the types it supports,
  with mermaid.js as the fallback**, is viable, low-risk to the Go side,
  and gives the Craft-style look for flowcharts, sequence, class, state,
  and ER diagrams. Theme matching is a small per-theme palette addition
  in the existing theme-CSS convention; seven themes have upstream
  palettes to copy.
- **ASCII output is viable client-side today** via `renderMermaidASCII`
  for five types, with HTML-colored output that fits a `<pre>`. A
  Go-native server-side path also exists (importing `mermaid-ascii`
  packages) but is undocumented, narrower, and adds seven dependencies.
- The payload win is real but modest (~1.8 MB vs 5.4 MB); in a hybrid
  both engines ship, so the binary grows unless the fallback loads
  lazily.
- The package needs a bundling step to vendor; the current curl-only
  `update-vendor` cannot produce a self-contained file.

## Recommendation

1. Resolve the open questions below, then write a DESIGN doc for a hybrid
   client-side renderer (beautiful-mermaid first, mermaid.js fallback)
   with an ASCII view mode.
2. Before the design is finalised, run a visual spike: bundle
   beautiful-mermaid with esbuild, render `docs/examples/flowchart.md`,
   `sequence.md`, and `class.md` side by side under tokyo-night,
   github-dark, catppuccin-mocha, and rose-pine, and confirm (a) the
   fallback catches every unsupported example and (b) label widths look
   right with mdp's font stack.
3. Keep #82 and #83 behaviour intact; document that `--dagre` and the
   Iconify packs apply to the fallback path.
4. Leave `pkg/parser` unchanged unless the server-side ASCII option is
   chosen.

## Open Questions

Each question lists **a** as my recommendation and **b…** as
alternatives. Write your choice (or "other: …") next to each.

**1. Adoption shape — how should beautiful-mermaid coexist with mermaid.js?**

- **a.** Hybrid, always on: try beautiful-mermaid for every `mermaid`
  block, fall back to mermaid.js when its parser throws. No new flag;
  every supported diagram gets the new look immediately.
- **b.** Opt-in: keep mermaid.js as default, enable beautiful-mermaid via
  a CLI flag / Neovim option (`--mermaid-engine beautiful`). Safer for
  existing users, but most people never find the flag.
- **c.** Full replacement: drop mermaid.js and the seven unsupported
  types. Rejected by the evidence in Observation 2 — breaks
  `docs/examples` and #83.
- Other:

**2. Where does ASCII rendering run?**

- **a.** Client-side, `renderMermaidASCII` in `preview.js` with
  `colorMode: 'html'` into a `<pre>`. Same engine as the SVG path, five
  types, no Go dependencies, no subprocess, themed via the same slots.
- **b.** Server-side in Go by importing `mermaid-ascii/pkg/render` and
  adding a `NodeRenderer` for `mermaid.Kind` (new
  `parser.WithMermaidASCII(bool)`). Benefits library consumers of
  `pkg/parser` and any future static-export path, but adds seven deps on
  a pseudo-versioned, undocumented API and covers fewer types.
- **c.** Both: server-side for `pkg/parser` consumers, client-side for the
  live preview. Two engines with slightly different output to keep in
  sync.
- Other:

**3. How is ASCII mode selected?**

- **a.** A document-level view mode: CLI flag
  `--mermaid-render svg|ascii`, matching Neovim `opts` key, and a runtime
  toggle in the preview (keyboard shortcut or small toolbar) so you can
  flip without restarting. ASCII is a way of *viewing* a diagram, not a
  property of the diagram.
- **b.** Per-block, in the fence info string (```` ```mermaid ascii ````).
  Author-controlled and survives to other renderers as a normal mermaid
  fence, but there is no way to see everything as ASCII at once.
- **c.** Both — global mode plus per-block override.
- Other:

**4. How are diagram colors wired to themes?**

- **a.** Add seven `--diagram-*` properties (`bg fg line accent muted
  surface border`) to each theme CSS file next to the existing
  `--mermaid-*` block; seed the seven overlapping themes from upstream's
  `THEMES` values verbatim and derive the other eight from `--color-*`.
  JS passes `var(--diagram-…)` strings with `transparent: true`. One
  mechanism, live theme switching, pixel-faithful Tokyo Night, and it
  follows the existing theme-file convention.
- **b.** No new properties: pass `var(--color-canvas-default)` etc.
  directly. Zero CSS work, but `accent` / `line` / `muted` drift from
  upstream on tokyo-night, github-dark, and catppuccin-mocha
  (Observation 4).
- **c.** Look up `THEMES[name]` at runtime by theme name for the seven
  matches and derive for the rest. Couples `preview.js` to theme names
  and needs a name map for `tokyo-night-day`.
- Other:

**5. How is the JavaScript vendored?**

- **a.** Add a bundling step: pin `esbuild` in `mise.toml`, and have
  `make update-vendor` run
  `esbuild --bundle --minify --format=iife` (or `esm`) to emit one
  self-contained `assets/vendor/beautiful-mermaid.min.js`, committed like
  `mermaid.min.js`. First JS toolchain in the repo, but reproducible and
  offline-safe.
- **b.** Vendor jsDelivr's three `+esm` files (`beautiful-mermaid`,
  `elkjs`, `entities`) with curl and `sed` the `/npm/…` import
  specifiers to local paths. No toolchain, but brittle and dependent on
  jsDelivr's bundling output.
- **c.** Load from the CDN at runtime. Rejected — mdp is offline-first and
  embeds every asset.
- Other:

**6. What happens to `--dagre` and the Iconify packs?**

- **a.** Keep both unchanged; they only affect the mermaid.js fallback
  path. Note that in the README and flag help.
- **b.** Remove `--dagre` now on the grounds that the covered types no
  longer use mermaid.js for layout. Still breaks anyone pinning dagre for
  the seven fallback types.
- Other:

**7. Which font does the SVG declare?**

- **a.** Pass `font` = the preview's body font stack so diagram text
  matches prose, and verify label fit on `docs/examples` during the
  spike. Accept the heuristic-width risk in Observation 5.
- **b.** Vendor Inter (woff2) and use it for diagrams only, matching the
  measurement table exactly. Extra asset, and diagrams stop matching the
  page font.
- Other:

**8. Should mermaid.js stay long-term?**

- **a.** Keep it indefinitely as the fallback; revisit only if upstream
  adds the missing types. Cost is the extra ~1.8 MB in the binary.
- **b.** Time-box it: open a tracking issue to re-evaluate dropping it
  when beautiful-mermaid covers `architecture-beta` and the other
  types used in `docs/examples`.
- Other:

**9. Should ASCII output be reachable outside the preview?**

For example a `:MdpAscii` command that inserts the rendered text into
the buffer, or `mdp ascii file.md` on the CLI.

- **a.** Out of scope for this investigation; capture as a separate
  GitHub issue once the preview mode ships.
- **b.** Include it in the DESIGN now, which would push the answer to
  question 2 toward server-side (b or c).
- Other:

## References

- Craft demo page: <https://agents.craft.do/mermaid>
- Source: <https://github.com/lukilabs/beautiful-mermaid>
  (`src/index.ts`, `src/theme.ts`, `src/ascii/index.ts`,
  `src/elk-instance.ts`, `src/text-metrics.ts`, `src/browser.ts`,
  `tsup.config.ts`)
- npm: <https://www.npmjs.com/package/beautiful-mermaid> (1.1.3; sizes
  measured via `data.jsdelivr.com/v1/package/npm/beautiful-mermaid@1.1.3/flat`
  and `cdn.jsdelivr.net/npm/beautiful-mermaid@1.1.3/+esm`)
- ASCII origin: <https://github.com/AlexanderGrooff/mermaid-ascii>
  (`pkg/render`, `pkg/diagram`)
- goldmark-mermaid v0.6.0: `ast.go` (`Kind`, `Block`),
  `server_render.go` (`Compiler`, `CompileRequest`, `CompileResponse`)
- PR #82 — Mermaid v12 with ELK default and `--dagre`
- PR #83 — Iconify packs and `docs/examples/`
- `assets/preview.js` (Mermaid init / `renderClientSide`),
  `assets/preview.html`, `assets/themes/*.css`, `Makefile`
  (`update-vendor`), `internal/server/server.go` (`MermaidTheme`,
  `MermaidLayout`)
