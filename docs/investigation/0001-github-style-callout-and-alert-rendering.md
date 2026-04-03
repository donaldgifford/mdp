---
id: INV-0001
title: "GitHub-style callout and alert rendering"
status: Concluded
author: Donald Gifford
created: 2026-04-03
---
<!-- markdownlint-disable-file MD025 MD041 -->

# INV 0001: GitHub-style callout and alert rendering

**Status:** In Progress
**Author:** Donald Gifford
**Date:** 2026-04-03

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
- [Approach](#approach)
- [Environment](#environment)
- [Findings](#findings)
  - [GitHub callout syntax and rendering](#github-callout-syntax-and-rendering)
  - [Existing goldmark extensions](#existing-goldmark-extensions)
  - [Current mdp parser pipeline](#current-mdp-parser-pipeline)
  - [CSS considerations](#css-considerations)
- [Conclusion](#conclusion)
- [Recommendation](#recommendation)
- [References](#references)
<!--toc:end-->

## Question

Can we add GitHub-style callout/alert rendering (`> [!NOTE]`, `> [!TIP]`, etc.) to mdp using an existing goldmark extension, or do we need to build our own AST transformer?

## Hypothesis

An existing goldmark extension likely exists that handles GitHub-style alerts since this is a widely-used GitHub feature. We should be able to drop it into our existing goldmark pipeline with minimal changes — the main work will be CSS styling for each theme.

## Context

mdp renders GitHub-flavored markdown but does not currently support GitHub's alert/callout syntax. These are commonly used in README files and documentation. Without support, callouts render as plain blockquotes with the `[!NOTE]` text visible literally, which is a poor experience.

**Triggered by:** User request during development of the theme system.

## Approach

1. Document GitHub's callout syntax and the HTML structure GitHub produces
2. Search for existing goldmark extensions that handle this syntax
3. Evaluate each candidate: maintenance status, HTML output, compatibility with our pipeline
4. Assess CSS work needed per theme (5 callout types × 15 themes)
5. Recommend: adopt an extension or build a custom AST transformer

## Environment

| Component | Version / Value |
|-----------|----------------|
| goldmark | v1.7.16 |
| Go | 1.25.7 |
| goldmark extensions in use | GFM, highlighting/v2, mermaid, mathjax |

## Findings

### GitHub callout syntax and rendering

GitHub supports 5 alert types in blockquote syntax:

```markdown
> [!NOTE]
> Useful information that users should know, even when skimming content.

> [!TIP]
> Helpful advice for doing things better or more easily.

> [!IMPORTANT]
> Key information users need to know to achieve their goal.

> [!WARNING]
> Urgent info that needs immediate user attention to avoid problems.

> [!CAUTION]
> Advises about risks or negative outcomes of certain actions.
```

GitHub renders these as styled `<div>` elements with an SVG icon and a title line, wrapped in a container with class `markdown-alert markdown-alert-{type}`. Each type gets a distinct color (blue for note, green for tip, purple for important, yellow for warning, red for caution).

### Existing goldmark extensions

Four candidates found on pkg.go.dev:

#### 1. `github.com/zmtcreative/gm-alert-callouts` (v0.8.0)

- **Stars:** 1 | **License:** MIT | **Last updated:** Sep 2025
- **How it works:** AST transformer — hooks into goldmark's blockquote parsing, detects `[!TYPE]` pattern, transforms to styled HTML
- **HTML output:** `<div class="callout callout-{type}">` with nested icon, title, and content containers
- **Features:**
  - All 5 GitHub alert types
  - Obsidian-style callouts (any custom type name)
  - Collapsible/foldable callouts via `<details>/<summary>` (`+`/`-` suffix)
  - Configurable icon sets: `UseGFMStrictIcons()`, `UseHybridIcons()`, `UseObsidianIcons()`
  - Custom icon support via `WithIcon()`/`WithIcons()`
- **Requires:** Go 1.23+, goldmark 1.4.6+
- **Assessment:** Most feature-rich. Active development. Functional options API. Obsidian support is a bonus but not needed. Clean AST transformer approach fits our pipeline.

#### 2. `github.com/thiagokokada/goldmark-gh-alerts`

- **Stars:** ~0 | **License:** MIT | **Last updated:** 2025
- **How it works:** AST transformer
- **HTML output:** `<div class="markdown-alert markdown-alert-{type}">` (matches GitHub's class names)
- **Features:** 5 GitHub alert types, customizable SVG icons
- **Assessment:** Author notes: "mostly created to be used by gh-gfm-preview" and "API is not guaranteed to be stable." Recommends pinning a commit or forking. Not suitable for production dependency.

#### 3. `gitlab.com/staticnoise/goldmark-callout`

- **License:** MIT | **Last updated:** 2024
- **Stars:** 2 imports on pkg.go.dev
- **Assessment:** GitLab-hosted, limited documentation available. Supports GitHub alerts and Obsidian callouts. Less visibility than GitHub-hosted alternatives.

#### 4. `github.com/omar0ali/goldmark-alerts` (v0.1.0)

- **License:** None specified
- **Assessment:** No license — cannot use.

### Current mdp parser pipeline

The parser at `internal/parser/parser.go` builds a goldmark instance with extensions:

```go
goldmark.New(
    goldmark.WithExtensions(
        extension.GFM,
        highlighting.NewHighlighting(...),
        mathjax.MathJax,
        &mermaid.Extender{},
        // callout extension would go here
    ),
    goldmark.WithParserOptions(...),
    goldmark.WithRendererOptions(...),
)
```

Adding a new extension is a one-line change — the goldmark `Extender` interface is all that's needed. The callout extension registers its own AST node types and renderer, so it won't conflict with existing extensions.

### CSS considerations

Each callout type needs distinct styling:
- **Container:** border-left color, background tint, padding, border-radius
- **Icon:** SVG inline or CSS-based, colored per type
- **Title:** bold, colored per type

Per theme, we need 5 color sets (note=blue, tip=green, important=purple, warning=yellow, caution=red) that harmonize with the theme's palette. This means adding CSS custom properties to each theme file:

```css
[data-theme="<name>"] {
  --callout-note-color:      #...; /* blue family */
  --callout-tip-color:       #...; /* green family */
  --callout-important-color: #...; /* purple family */
  --callout-warning-color:   #...; /* yellow family */
  --callout-caution-color:   #...; /* red family */
}
```

The base callout styling (layout, padding, border-radius) goes in `assets/preview.css`. Theme-specific colors go in each theme CSS file. This is the same pattern used for mermaid variables.

The `gm-alert-callouts` extension uses `.callout` and `.callout-{type}` classes. If we use `goldmark-gh-alerts` instead, it uses `.markdown-alert` and `.markdown-alert-{type}` (matching GitHub). Either way, the CSS selector structure is straightforward.

If the extension embeds SVG icons in the HTML output (both candidates do), the icons are already styled — we just need the container colors. If we want CSS-only icons, that's more work but avoids inline SVGs.

## Conclusion

**Answer:** Yes — an existing extension can handle this. `gm-alert-callouts` is the best candidate.

**Rationale:**
- Only extension with active development, functional options API, and proper versioning (v0.8.0)
- AST transformer approach integrates cleanly with our goldmark pipeline (one-line addition)
- `UseGFMStrictIcons()` option gives us exactly the 5 GitHub types without Obsidian complexity
- MIT license, Go 1.23+ / goldmark 1.4.6+ requirements are compatible with our setup
- The `goldmark-gh-alerts` alternative explicitly warns against stable API use
- Building our own would duplicate well-tested AST transformation logic for no benefit

**Work estimate:**
1. Add `gm-alert-callouts` dependency — trivial
2. Wire into parser pipeline — one line
3. Base callout CSS in `preview.css` — ~30 lines (layout, padding, border)
4. Per-theme callout colors — ~10 lines per theme × 15 themes = ~150 lines
5. Tests — parser test with callout input/output, server integration test

## Recommendation

1. Add `github.com/zmtcreative/gm-alert-callouts` as a dependency
2. Wire it into `internal/parser/parser.go` with `UseGFMStrictIcons()` (no folding, no Obsidian)
3. Add callout color CSS variables to each theme file
4. Add base callout layout CSS to `assets/preview.css`
5. Create IMPL doc for the phased rollout if desired, or implement directly on a feature branch

## References

- [GitHub Docs: Alerts syntax](https://docs.github.com/en/get-started/writing-on-github/getting-started-with-writing-and-formatting-on-github/basic-writing-and-formatting-syntax#alerts)
- [gm-alert-callouts](https://github.com/zmtcreative/gm-alert-callouts) — recommended extension
- [goldmark-gh-alerts](https://github.com/thiagokokada/goldmark-gh-alerts) — alternative (unstable API)
- [goldmark-callout](https://gitlab.com/staticnoise/goldmark-callout) — GitLab alternative
- [GitHub community discussion on alerts](https://github.com/orgs/community/discussions/16925)
