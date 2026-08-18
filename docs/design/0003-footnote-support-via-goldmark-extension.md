---
id: DESIGN-0003
title: "Footnote support via goldmark extension"
status: Approved
author: Donald Gifford
created: 2026-08-18
---
<!-- markdownlint-disable-file MD025 MD041 -->

# DESIGN 0003: Footnote support via goldmark extension

**Status:** Approved
**Author:** Donald Gifford
**Date:** 2026-08-18

<!--toc:start-->
- [Overview](#overview)
- [Goals and Non-Goals](#goals-and-non-goals)
  - [Goals](#goals)
  - [Non-Goals](#non-goals)
- [Background](#background)
  - [Current state](#current-state)
  - [Prior art in this repo](#prior-art-in-this-repo)
  - [What goldmark actually emits](#what-goldmark-actually-emits)
  - [Interaction with lineAnnotator](#interaction-with-lineannotator)
- [Detailed Design](#detailed-design)
  - [Parser integration](#parser-integration)
  - [Scroll sync](#scroll-sync)
  - [Styling](#styling)
- [API / Interface Changes](#api--interface-changes)
- [Data Model](#data-model)
- [Testing Strategy](#testing-strategy)
- [Migration / Rollout Plan](#migration--rollout-plan)
- [Open Questions](#open-questions)
- [Resolved Decisions](#resolved-decisions)
- [References](#references)
<!--toc:end-->

## Overview

Enable [PHP Markdown Extra / "extended syntax"
footnotes](https://www.markdownguide.org/extended-syntax/#footnotes)
(`Text.[^1]` … `[^1]: The note.`) in `pkg/parser` by adding goldmark's
`extension.Footnote` to the extension slice, gated behind a new
`WithFootnotes(bool)` option that follows the existing
`WithMermaid`/`WithMath`/`WithCallouts` pattern.

The parser change itself is three lines. The load-bearing parts of
this design are (1) a **scroll-sync regression** caused by goldmark
relocating footnote definitions to the end of the rendered document
while keeping their original source line numbers, and (2) the CSS
needed so the emitted `<div class="footnotes">` doesn't look broken
under mdp's existing `hr` styling.

## Goals and Non-Goals

### Goals

- Render `[^label]` references and `[^label]:` definitions as linked
  superscripts and an endnote list, matching GitHub's rendering.
- Support arbitrary block content inside footnote definitions —
  specifically **links**, but also emphasis, inline code, math, and
  multi-paragraph bodies.
- Keep scroll sync correct: a cursor position in the buffer must not
  jump the preview to an unrelated footnote.
- Style `.footnotes`, `.footnote-ref`, and `.footnote-backref` so they
  read correctly across all 13 built-in themes **without editing any
  theme file**.
- Keep the option additive and non-breaking for `pkg/parser`
  consumers.

### Non-Goals

- A CLI flag to toggle footnotes. No parser feature has one today
  (`internal/server/server.go:120` calls a bare `parser.New()`); adding
  the first one is a separate decision.
- Inline footnotes (`^[note text]`) — not supported by goldmark's
  extension and not part of the Markdown Guide's extended syntax.
- Footnote popups / hover previews in the browser.
- Changing the WS/SSE wire protocol or the stdin protocol.
- Editing the 13 theme CSS files (see [Styling](#styling) — this
  design deliberately reuses existing custom properties).
- Neovim plugin changes.

## Background

### Current state

`pkg/parser/parser.go:95` registers `extension.GFM`, which in
goldmark v1.8.2 expands to exactly four extensions
(`goldmark@v1.8.2/extension/gfm.go:13-18`):

```go
func (e *gfm) Extend(m goldmark.Markdown) {
	Linkify.Extend(m)
	Table.Extend(m)
	Strikethrough.Extend(m)
	TaskList.Extend(m)
}
```

Footnotes are **not** in that set — `extension.Footnote` is a separate
extender. mdp registers GFM, chroma highlighting, mermaid, mathjax,
and `gm-alert-callouts`, so footnote syntax falls through to the
default paragraph parser. Verified against the current build:

```text
input:  Here is a footnote ref.[^1]
        [^1]: See the [Markdown Guide](https://...) for details.

output: <p data-source-line="1">Here is a footnote ref.[^1]</p>
        <p data-source-line="3">[^1]: See the <a href="https://...">Markdown Guide</a> ...</p>
```

The `[^1]` markers pass through verbatim. The link renders only
because the whole line is being treated as an ordinary paragraph.

This is a known gap, tracked as an unchecked item at
`docs/impl/mvp.md:283` ("Footnote support via goldmark extension").

### Prior art in this repo

IMPL-0002 (GitHub-style callouts) is the closest analogue: a goldmark
extension added default-on behind a `With*` toggle, plus base CSS in
`assets/preview.css`, plus per-theme color variables. This design
follows that shape but **skips the per-theme step** — callouts needed
five semantic accent colors per theme; footnotes need only muted text
and a border, both of which already exist as theme-provided custom
properties.

### What goldmark actually emits

Confirmed empirically by rendering with `extension.Footnote` added to
the current pipeline (callouts, mathjax, and mermaid all active):

```html
<p data-source-line="1">Intro.<sup id="fnref:1"><a href="#fn:1" class="footnote-ref" role="doc-noteref">1</a></sup></p>
<p data-source-line="3">Math <span class="math inline">\(x^2\)</span> and again.<sup id="fnref1:1"><a href="#fn:1" class="footnote-ref" role="doc-noteref">1</a></sup></p>
<div class="footnotes" role="doc-endnotes">
<hr>
<ol>
<li id="fn:1" data-source-line="8">
<p data-source-line="8">See <a href="https://www.markdownguide.org/extended-syntax/#footnotes">the guide</a> and <span class="math inline">\(y^2\)</span>.&#160;<a href="#fnref:1" class="footnote-backref" role="doc-backlink">&#x21a9;&#xfe0e;</a>&#160;<a href="#fnref1:1" class="footnote-backref" role="doc-backlink">&#x21a9;&#xfe0e;</a></p>
</li>
</ol>
</div>
```

Findings that shape the rest of this design:

1. **Links inside footnote definitions work.** Definition bodies are
   parsed as full block content, so links, `<strong>`, `<code>`, math
   spans, and multi-paragraph bodies all render. The only thing that
   can't contain a link is the *label* (`[^1]`), which is a plain-text
   identifier by spec.
2. **Numbering follows first-reference order, not definition order** —
   matching GitHub and the Markdown Guide. Named labels (`[^note]`)
   render as sequential integers.
3. **Repeat references produce multiple backlinks.** A second
   reference to `[^1]` gets `id="fnref1:1"` and the definition grows a
   second `↩︎` backlink.
4. **The extension composes cleanly with the existing pipeline** — no
   conflict with mathjax `$...$`, callouts, or heading IDs.
5. **IDs are unprefixed** (`fn:1`, `fnref:1`). Fine for mdp, which
   renders one document per page. Relevant only if a `pkg/parser`
   consumer composes several rendered documents into one page — see
   [Decision 3](#resolved-decisions).

### Interaction with `lineAnnotator`

goldmark registers the footnote AST transformer at priority 999
(`goldmark@v1.8.2/extension/footnote.go:684-686`); mdp's
`lineAnnotator` is registered at priority 0
(`pkg/parser/parser.go:124`), so it runs **first**. Probing the AST at
both priorities shows the `FootnoteList` node is already attached
during block parsing — the transformer at 999 only appends backlinks
and fixes ref counts:

```text
--- probe prio-0 (lineAnnotator slot) ---
  kind=Paragraph      lines=1 firstSeg.Start=0
  kind=FootnoteList   lines=0 firstSeg.Start=-1
  kind=Footnote       lines=0 firstSeg.Start=17
  kind=Paragraph      lines=1 firstSeg.Start=17
--- probe prio-1500 (after footnote xform) ---
  (identical)
```

Two consequences:

- **`FootnoteList` hits the `seg.Start < 0` guard** at
  `pkg/parser/lineannotator.go:34-40`. That branch carries a
  `// coverage: block nodes always have a segment or a child segment
  after the goldmark default parser runs` annotation, which becomes
  factually wrong once footnotes ship. Confirmed with a coverage run
  against a patched build — a single footnote render marks the block
  covered:

  ```text
  github.com/donaldgifford/mdp/pkg/parser/lineannotator.go:34.20,40.4 1 1
  ```

  Net effect: `<div class="footnotes">` gets **no**
  `data-source-line`, while the `<li>` and its inner `<p>` do (both
  inherited from the definition's first line).
- **The `// coverage:` annotation must be updated** — resolved in
  [Decision 7](#resolved-decisions): drop the annotation and reword the
  comment to describe the real case.

## Detailed Design

### Parser integration

Mirror the existing option shape exactly:

```go
type config struct {
	gfm                bool
	syntaxHighlighting bool
	mermaid            bool
	mermaidMode        mermaid.RenderMode
	math               bool
	callouts           bool
	footnotes          bool // NEW
}

func defaultConfig() config {
	return config{
		// ...
		footnotes: true, // default on — Decision 1
	}
}

// WithFootnotes enables or disables extended-syntax footnotes
// ([^1] references and [^1]: definitions). Definitions are collected
// into a <div class="footnotes"> endnote list at the end of the
// rendered output, regardless of where they appear in the source.
func WithFootnotes(enabled bool) Option {
	return func(c *config) { c.footnotes = enabled }
}
```

And in `New`:

```go
if cfg.footnotes {
	extensions = append(extensions, extension.Footnote)
}
```

Placement in the slice does not matter — goldmark sorts parsers and
renderers by their own registered priorities, not by extender order.
Appending after the callouts block keeps the source reading in the
same order as the `config` struct.

No new dependency: `extension.Footnote` ships in
`github.com/yuin/goldmark v1.8.2`, already in `go.mod:11`.

### Scroll sync

This is the one behavioral regression in the design.

`assets/preview.js:99-116` picks a scroll target by walking
`[data-source-line]` elements **in document order** and breaking at
the first element whose line exceeds the target:

```javascript
for (var i = 0; i < elements.length; i++) {
  var line = parseInt(elements[i].getAttribute("data-source-line"), 10);
  if (isNaN(line)) continue;
  if (line <= targetLine) {
    best = elements[i];          // last match wins
  } else {
    break;                       // assumes monotonic order
  }
}
```

That assumes `data-source-line` is non-decreasing in document order.
Footnotes break the assumption whenever a definition appears anywhere
other than the end of the file, because goldmark renders the endnote
list last while preserving the definition's *source* line.

Two cases, measured against a real render:

| Source layout | Document-order `data-source-line` values | Monotonic? |
|---|---|---|
| Definitions at end of file | `1, 3, 5, 7, 9, 9` | yes — benign |
| Definition mid-document | `1, 3, 7, 9, 5, 5` | **no** |

The mid-document case, concretely:

```markdown
1  # Title
2
3  Intro with a note.[^a]
4
5  [^a]: The note text.
6
7  ## Section
8
9  Body paragraph.
```

With the cursor on line 9, `findScrollTarget(9)` never hits the
`break` (every value is `<= 9`), so `best` ends up as the footnote's
inner `<p data-source-line="5">`. **The preview scrolls to the
footnote at the bottom of the page instead of "Body paragraph."**

Placing definitions at the end of the file — the conventional style,
and what the Markdown Guide shows — avoids this entirely. But mdp
can't assume that about arbitrary buffers.

**Resolved ([Decision 2](#resolved-decisions)): fix it client-side.**
`findScrollTarget` excludes the `.footnotes` subtree, which keeps the
`data-source-line` data intact for `pkg/parser` consumers that may
legitimately want to locate a footnote definition:

```javascript
var elements = content.querySelectorAll(
  "[data-source-line]:not(.footnotes [data-source-line])"
);
```

Because the annotations stay, `pkg/parser/doc.go` must gain a sentence
warning that `data-source-line` is **not** guaranteed monotonic in
document order once footnotes are enabled — the current doc comment
implies a straightforward source-order mapping. `CLAUDE.md` gets the
same invariant so the next change to `findScrollTarget` doesn't
silently reintroduce the `break`.

### Styling

goldmark emits three hooks: `.footnotes` (wrapper `div`, containing an
`<hr>` and an `<ol>`), `.footnote-ref` (the `<a>` inside `<sup>`), and
`.footnote-backref` (the `↩︎` link).

Only `assets/preview.css` changes. Every value below resolves through
custom properties that all 13 theme files already define (per the
Theme CSS Format contract in `CLAUDE.md`), so **no theme file is
touched and no new variable is introduced**:

```css
/* Footnotes */
.footnotes {
  margin-top: 32px;
  font-size: 0.875em;
  color: var(--color-fg-muted);
}

/* Override the global thick `hr` (preview.css:228-234) with a
   hairline separator. */
.footnotes hr {
  height: 1px;
  margin: 0 0 16px 0;
  background-color: var(--color-border-muted);
}

.footnotes ol { margin-bottom: 0; }
.footnotes li { scroll-margin-top: 48px; }
.footnotes li:target { background-color: var(--color-canvas-subtle); }
.footnotes p:last-child { margin-bottom: 0; }

.footnote-ref { font-weight: 600; }

.footnote-backref {
  text-decoration: none;
  font-family: ui-monospace, SFMono-Regular, "SF Mono", Menlo, monospace;
}
.footnote-backref:hover { text-decoration: none; }
```

The `hr` override is not optional. `assets/preview.css:228-234` styles
`hr` as `height: 0.25em` on `--color-border-default` — a heavy bar.
Rendered directly above the endnote list it reads as a document break
rather than a footnote separator.

`:target` highlighting gives visual feedback when the reader clicks a
`.footnote-ref`, matching the `.scroll-target` affordance that already
exists for cursor sync.

Anchor navigation itself needs no JS: `href="#fn:1"` → `id="fn:1"`
works natively. Note the colon makes `fn:1` an *invalid CSS selector*,
so any future JS must use `getElementById("fn:1")`, never
`querySelector("#fn:1")`. `assets/preview.js` currently does no
hash-based lookups, so nothing breaks today.

## API / Interface Changes

| Surface | Change | Breaking? |
|---|---|---|
| `pkg/parser` | New exported `WithFootnotes(bool) Option` | No — additive |
| `pkg/parser` | Default `parser.New()` output now contains `<div class="footnotes">` for documents that use footnote syntax | Behavioral, opt-out via `WithFootnotes(false)` |
| `pkg/parser/doc.go` | Document the option; note non-monotonic `data-source-line` | No |
| CLI (`internal/cli`) | None | — |
| WS/SSE wire protocol | None | — |
| Stdin protocol | None | — |
| `pkg/theme` | None | — |
| `assets/preview.css` | New `.footnotes*` rules | — |
| `assets/preview.js` | `findScrollTarget` excludes the `.footnotes` subtree | — |
| `go.mod` | None — `extension.Footnote` is already vendored with goldmark v1.8.2 | — |

`TestParser_AllOptionsOff` (`pkg/parser/parser_test.go:256`) must gain
`parser.WithFootnotes(false)` — it exists specifically to exercise the
false branch of every toggle.

## Data Model

No changes. Footnote state is per-`Render` call and lives entirely in
goldmark's parser context; nothing is persisted, cached, or added to
`wsMessage`.

One property worth stating explicitly: footnote IDs (`fn:N`,
`fnref:N`) are **per-render and position-dependent**. Adding a
footnote reference earlier in the buffer renumbers every subsequent
one. Since mdp replaces `#content` wholesale on each content
broadcast, there is no stale-ID or ID-collision risk in the live
preview.

## Testing Strategy

`pkg/parser` sits at **97.1%** coverage against a codecov component
target of 95% / threshold 1% (`.codecov.yml:15-22`). New exported
symbols need direct coverage to hold that line.

**New unit tests in `pkg/parser/parser_test.go`:**

| Test | Asserts |
|---|---|
| `TestRender_Footnote` | `[^1]` → `<sup id="fnref:1">` + `<a href="#fn:1" class="footnote-ref">`; definition → `<li id="fn:1">` inside `<div class="footnotes">` |
| `TestRender_FootnoteWithLink` | A `[text](url)` inside a definition renders as `<a href="url">` — the specific question that motivated this design |
| `TestRender_FootnoteMultiParagraph` | Indented continuation blocks produce multiple `<p>` inside one `<li>`; backlink attaches to the last paragraph |
| `TestRender_FootnoteRepeatedReference` | Two refs to one label → `fnref:1` and `fnref1:1`, two `.footnote-backref` anchors |
| `TestRender_FootnoteNamedLabel` | `[^note]` renders as a sequential integer, numbered by first-reference order |
| `TestRender_FootnoteDisabled` | `WithFootnotes(false)` leaves `[^1]` as literal text (mirrors `TestRender_CalloutDisabled`) |
| `TestRender_FootnoteUndefinedReference` | `[^missing]` with no definition renders as literal text, no empty `.footnotes` div |

**Amended tests:**

| Test | Change |
|---|---|
| `TestParser_AllOptionsOff` (`parser_test.go:256`) | Add `parser.WithFootnotes(false)` |
| `TestRender_MarkdownFixture` (`parser_test.go:215`) | Add a `{"footnote", "class=\"footnotes\""}` check |

**Scroll-sync regression test** (new, `lineannotator_test.go`) — the
highest-value test here, and the one that would have caught the
ordering bug:

- Render the mid-document-definition fixture from
  [Scroll sync](#scroll-sync).
- Assert the *document-order* sequence of `data-source-line` values —
  pinning the known-non-monotonic `1, 3, 7, 9, 5, 5` shape so a future
  change to `lineAnnotator` can't silently alter it.
- Assert the invariant Decision 2 settles on: **the subsequence of
  `data-source-line` values outside the `.footnotes` subtree is
  non-decreasing.** That is exactly the precondition
  `findScrollTarget`'s `break` relies on.

**Fixtures:** `pkg/parser/testdata/fixture.md` gains a footnote
section. Note that `pkg/parser/testdata/full-features.md` is currently
**referenced by nothing in the repo** — an orphan. Wiring it up (or
deleting it) is out of scope here but worth a follow-up.

**Manual verification:**

1. `make build && make test && make lint`
2. `./bin/mdp --file <doc-with-footnotes>.md` — check the endnote
   list, click a `.footnote-ref`, click the `↩︎` backlink.
3. Spot-check one theme per family (github, tokyo-night,
   rose-pine-dawn, catppuccin-latte) — two light, two dark — to
   confirm the muted text and hairline rule read correctly.
4. In Neovim: open a file with a **mid-document** footnote definition,
   move the cursor to a line *after* it, confirm the preview scrolls
   to the right place and not to the endnote list.

## Migration / Rollout Plan

Single PR, no phasing — the change is small and the pieces are
interdependent (shipping the parser change without the CSS produces
visibly broken output).

**Branch:** `feat/footnote-support`

**Diff shape:**

| File | Action | Description |
|---|---|---|
| `pkg/parser/parser.go` | Modify | `footnotes` config field, `WithFootnotes` option, `extension.Footnote` registration |
| `pkg/parser/doc.go` | Modify | Add `WithFootnotes` to the options block; note non-monotonic `data-source-line` |
| `pkg/parser/lineannotator.go` | Modify | Drop the `// coverage:` annotation on the `seg.Start < 0` guard, reword the comment to name `FootnoteList` as the real case. No behavior change — annotations stay on footnote elements |
| `pkg/parser/parser_test.go` | Modify | New footnote tests; amend `TestParser_AllOptionsOff` and `TestRender_MarkdownFixture` |
| `pkg/parser/lineannotator_test.go` | Modify | Scroll-sync ordering regression test |
| `pkg/parser/testdata/fixture.md` | Modify | Footnote section |
| `assets/preview.css` | Modify | `.footnotes`, `.footnote-ref`, `.footnote-backref` rules |
| `assets/preview.js` | Modify | `findScrollTarget` selector excludes `.footnotes` descendants |
| `README.md` | Modify | Add footnotes to "Supported Markdown Features" (`README.md:120-127`) and to the `pkg/parser` blurb (`README.md:215-216`) |
| `docs/impl/mvp.md` | Modify | Check off line 283 |
| `CLAUDE.md` | Modify | Record the non-monotonic `data-source-line` invariant and the `findScrollTarget` exclusion it depends on |

**Rollback:** `WithFootnotes(false)` restores prior parser behavior
without a revert. The CSS additions are inert on documents that have
no footnotes.

**PR label:** `minor` ([Decision 8](#resolved-decisions)) — new
user-visible rendering feature plus a new exported symbol on a public
package.

**No follow-up IMPL doc proposed** — the phase tracking in an IMPL
would just restate the file table above. If the scroll-sync work grows
past a one-line selector change, that's the trigger to split it out.

## Open Questions

None remaining. All eight questions raised during drafting were
resolved by the author on 2026-08-18 — every one settled on the
recommended option. See [Resolved Decisions](#resolved-decisions).

## Resolved Decisions

**1. Footnotes default to on.** `footnotes: true` in
`defaultConfig()`. Consistent with GFM, highlighting, mermaid, math,
and callouts, all of which default on. GitHub renders footnotes by
default, so a preview tool that doesn't would read as broken, and the
syntax is inert in documents that don't use it. *Rejected:* default
off (would force `internal/server/server.go:120` to pass the option
and diverge from GitHub); a `--no-footnotes` CLI flag (would be the
first per-feature flag, raising the question of why mermaid/math/
callouts have none).

**2. Scroll sync is fixed client-side.** `findScrollTarget` in
`assets/preview.js` excludes the `.footnotes` subtree via
`"[data-source-line]:not(.footnotes [data-source-line])"`. The
monotonic-order assumption in `preview.js` is where the bug actually
lives, so this fixes it at the source, and `data-source-line` stays on
footnote elements for `pkg/parser` consumers that want to map a
definition back to its source line. *Rejected:* server-side
suppression in `lineAnnotator` (fixes every consumer at once but
destroys the line mapping); doing both (the client guard becomes dead
code); accepting the limitation (mid-document definitions are legal
markdown and the failure is silent).

Follow-on obligations from this decision, both already reflected in
[Migration / Rollout Plan](#migration--rollout-plan):

- `pkg/parser/doc.go` documents that `data-source-line` is **not**
  monotonic in document order once footnotes are enabled.
- `CLAUDE.md` records the same invariant plus the `findScrollTarget`
  exclusion that depends on it.

**3. `WithFootnoteIDPrefix` is not exposed.** mdp renders one document
per page, so `fn:N` / `fnref:N` cannot collide today. Adding the
option later is purely additive and non-breaking, and every existing
`With*` toggles something mdp itself uses. *Rejected:* adding it now
for the hypothetical docz multi-document case; auto-deriving a
per-render prefix (non-deterministic output would break the
exact-output assertion in `example_test.go`).

**4. Footnote styling lives in `assets/preview.css` only.** It reuses
`--color-fg-muted`, `--color-border-muted`, and
`--color-canvas-subtle`, all of which every theme already defines. No
theme file is edited, no new custom property is introduced, and future
themes work automatically. Footnotes need muted text and a hairline,
not semantic accent colors — which is what made the per-theme step
necessary for callouts in IMPL-0002 and unnecessary here. *Rejected:*
`--footnote-*` variables across all 13 themes; `preview.css` defaults
with optional theme overrides (adds variables nothing overrides).

**5. The `<hr>` is overridden to a 1px hairline** on
`--color-border-muted`. Keeps the semantic separator goldmark
hardcodes, matches GitHub's footnote treatment, stays a pure CSS
change. *Rejected:* hiding it and using a `border-top` on `.footnotes`
(leaves a hidden element in the DOM); leaving the global `0.25em` bar
(reads as a document break rather than an appendix).

**6. The backlink glyph keeps goldmark's default**
(`&#x21a9;&#xfe0e;`), styled via `.footnote-backref` in CSS. The
variation selector already forces text rather than emoji
presentation, which is the usual reason to override. *Rejected:*
matching GitHub's exact markup (pins mdp to an upstream HTML detail
that could drift); a text label like `↩ back` (noisy in a long
endnote list).

**7. The `// coverage:` annotation is removed and the comment
reworded.** The guard at `pkg/parser/lineannotator.go:34-40` stops
being a defensive-only branch once footnotes ship — `FootnoteList`
reaches it on every render containing a footnote. The replacement
comment should explain the real case: container nodes such as
`FootnoteList` legitimately have neither their own segment nor a first
child with one, and are intentionally left unannotated. *Rejected:*
keeping the annotation (it would document something false); recursing
in `firstSegment` so the guard becomes unreachable again (would
annotate the wrapper `<div>` with a line number pointing into the
middle of the document, working directly against Decision 2).

**8. The PR carries the `minor` label.** New user-visible rendering
feature plus a new exported symbol (`WithFootnotes`) on a public
package — the same basis on which v0.2.0 was cut for RFC-0001.
*Rejected:* `patch` (the feature was planned, but it is not a bug
fix); batching behind `dont-release` with the rest of
`chore/cleanup-and-inv`.

## References

- [Markdown Guide — Extended Syntax: Footnotes](https://www.markdownguide.org/extended-syntax/#footnotes)
  — the syntax this design implements
- [goldmark `extension.Footnote`](https://pkg.go.dev/github.com/yuin/goldmark/extension#Footnote)
  — `goldmark@v1.8.2/extension/footnote.go`
- `goldmark@v1.8.2/extension/gfm.go:13-18` — proof that GFM excludes
  footnotes
- `goldmark@v1.8.2/extension/footnote.go:505-625` —
  `FootnoteHTMLRenderer`, the exact emitted markup
- `goldmark@v1.8.2/extension/footnote.go:272-302` — `FootnoteConfig`
  (`IDPrefix`, `BacklinkHTML`, `LinkClass`, …)
- [IMPL-0002 — GitHub-style callout rendering](../impl/0002-github-style-callout-rendering.md)
  — the extension-plus-CSS pattern this design follows
- [DESIGN-0002 — Refactor mdp internals into public pkg packages](./0002-refactor-mdp-internals-into-public-pkg-packages.md)
  — establishes the `pkg/parser` public-API and coverage commitments
- `pkg/parser/parser.go:93-133` — extension registration and
  `lineAnnotator` priority
- `pkg/parser/lineannotator.go:34-40` — the `seg.Start < 0` guard
- `assets/preview.js:99-116` — `findScrollTarget`, the monotonic-order
  assumption
- `assets/preview.css:228-234` — the global `hr` rule that needs
  overriding
- `.codecov.yml:15-22` — `pkg/parser` 95% component target
- `docs/impl/mvp.md:283` — the original unchecked TODO
