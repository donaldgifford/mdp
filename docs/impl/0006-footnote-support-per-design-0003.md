---
id: IMPL-0006
title: "Footnote support per DESIGN-0003"
status: In Progress
author: Donald Gifford
created: 2026-08-18
---
<!-- markdownlint-disable-file MD025 MD041 -->

# IMPL 0006: Footnote support per DESIGN-0003

**Status:** In Progress
**Author:** Donald Gifford
**Date:** 2026-08-18

<!--toc:start-->
- [Objective](#objective)
  - [Design re-verified against the current toolchain](#design-re-verified-against-the-current-toolchain)
  - [New finding not in DESIGN-0003](#new-finding-not-in-design-0003)
- [Scope](#scope)
  - [In Scope](#in-scope)
  - [Out of Scope](#out-of-scope)
- [Implementation Phases](#implementation-phases)
  - [Phase 1: Parser integration and option surface](#phase-1-parser-integration-and-option-surface)
    - [Tasks](#tasks)
    - [Success Criteria](#success-criteria)
  - [Phase 2: Scroll-sync correctness](#phase-2-scroll-sync-correctness)
    - [Tasks](#tasks-1)
    - [Success Criteria](#success-criteria-1)
  - [Phase 3: Footnote styling](#phase-3-footnote-styling)
    - [Tasks](#tasks-2)
    - [Success Criteria](#success-criteria-2)
  - [Phase 4: Fixtures and coverage verification](#phase-4-fixtures-and-coverage-verification)
    - [Tasks](#tasks-3)
    - [Success Criteria](#success-criteria-3)
  - [Phase 5: Documentation and release preparation](#phase-5-documentation-and-release-preparation)
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

Implement extended-syntax footnotes (`Text.[^1]` … `[^1]: The note.`)
in `pkg/parser` by registering goldmark's `extension.Footnote` behind a
new `WithFootnotes(bool)` option, fix the scroll-sync regression it
introduces, and style the emitted endnote list across all 13 themes.

**Implements:** DESIGN-0003 (all eight decisions resolved; no open
design questions remain)

DESIGN-0003 states "No follow-up IMPL doc proposed." This document
supersedes that line; [Decision 6](#resolved-decisions) replaces it
with a pointer to IMPL-0006 in Phase 5.

### Design re-verified against the current toolchain

DESIGN-0003 was researched against goldmark v1.8.2. `main` now carries
Go 1.26.6 and goldmark v1.8.5 (commit `42ab241`). Every load-bearing
finding was re-verified on the current tree before this plan was
written:

| Design claim | Status on goldmark v1.8.5 |
|---|---|
| `extension.GFM` excludes footnotes | Confirmed — `gfm.go` unchanged, still `Linkify`/`Table`/`Strikethrough`/`TaskList` |
| Footnote HTML shape and class names | Confirmed — `extension/footnote.go` is **byte-identical** between v1.8.2 and v1.8.5 |
| Extension priorities (block 999, inline 101, AST 999, renderer 500) | Confirmed unchanged |
| Mid-document ordering `1, 3, 7, 9, 5, 5` | Confirmed by re-render |
| Definitions-at-end ordering `1, 3, 5, 7, 9, 9` | Confirmed by re-render |
| Composes with mathjax + callouts + mermaid | Confirmed by re-render on the full pipeline |
| Source line references (`parser.go:95`, `lineannotator.go:34`, `preview.js:99-116`, `preview.css:228`, `server.go:120`, `mvp.md:283`, `.codecov.yml:15-22`) | Confirmed still accurate post-merge |
| All 13 themes define `--color-fg-muted`, `--color-border-muted`, `--color-canvas-subtle`, `--color-accent-fg` | Confirmed — no theme file needs editing |

**No dependency change is required.** `extension.Footnote` ships in
the already-required `github.com/yuin/goldmark v1.8.5`.

### New finding not in DESIGN-0003

Re-verification surfaced a **second** non-monotonicity the design did
not record. Footnotes are numbered by order of *first reference*, so
when reference order differs from definition order the `<li>` elements
come out in descending source-line order:

```markdown
1  Alpha.[^zeta] Beta.[^alpha]
2
3  [^alpha]: Defined second in source.
4  [^zeta]: Defined first in source.
```

renders `<li id="fn:1" data-source-line="4">` before
`<li id="fn:2" data-source-line="3">` — values `4, 3`, descending,
**with every definition at the end of the file**.

This matters because it proves the DESIGN-0003 Decision-2 exclusion does real work
beyond the mid-document case the design analyzed: even a
conventionally-formatted document can produce out-of-order
annotations. It is captured as a test case in
[Phase 2](#phase-2-scroll-sync-correctness).

## Scope

### In Scope

- `WithFootnotes(bool)` option on `pkg/parser`, defaulting to enabled.
- `extension.Footnote` registration in the goldmark pipeline.
- `findScrollTarget` exclusion of the `.footnotes` subtree in
  `assets/preview.js`.
- Footnote CSS in `assets/preview.css` using existing custom
  properties.
- Unit tests, scroll-sync ordering regression tests, fixture coverage.
- Comment rewording in `pkg/parser/lineannotator.go` (drop the stale
  `// coverage:` annotation).
- Documentation: `doc.go`, `README.md`, `CLAUDE.md`, `mvp.md:283`.

### Out of Scope

- CLI flag for footnotes (DESIGN-0003 Decision 1).
- `WithFootnoteIDPrefix` (DESIGN-0003 Decision 3).
- Editing any of the 13 theme CSS files (DESIGN-0003 Decision 4).
- Backlink glyph override (DESIGN-0003 Decision 6).
- Inline footnotes (`^[note]`) — unsupported by goldmark.
- Footnote hover previews / popups.
- Wiring up or deleting the orphaned
  `pkg/parser/testdata/full-features.md` — deferred to
  [issue #75](https://github.com/donaldgifford/mdp/issues/75)
  ([Decision 4](#resolved-decisions)).

## Implementation Phases

Each phase builds on the previous one. A phase is complete when all
its tasks are checked off and its success criteria are met.

Phases 1–3 each land their own tests rather than deferring all testing
to a later phase, so no phase leaves the tree green-but-unverified.

---

### Phase 1: Parser integration and option surface

Register the extension behind a `With*` toggle mirroring the existing
option shape exactly, and cover the new exported symbol.

#### Tasks

- [x] 1. Add `footnotes bool` to the `config` struct in
  `pkg/parser/parser.go` (after `callouts`, matching field order to
  option order)
- [x] 2. Set `footnotes: true` in `defaultConfig()` (DESIGN-0003 Decision 1)
- [x] 3. Add the `WithFootnotes(enabled bool) Option` setter with a doc
  comment noting that definitions are collected into a
  `<div class="footnotes">` endnote list at the end of the output
  regardless of where they appear in the source
- [x] 4. Register `extension.Footnote` in `New` under
  `if cfg.footnotes { ... }`, placed after the callouts block
- [x] 5. Add `WithFootnotes` to the "All options" block in
  `pkg/parser/doc.go`
- [x] 6. Add a paragraph to `pkg/parser/doc.go` warning that
  `data-source-line` is **not** monotonic in document order when
  footnotes are enabled (DESIGN-0003 Decision 2 follow-on obligation)
- [x] 7. Amend `TestParser_AllOptionsOff` (`parser_test.go:256`) to
  pass `parser.WithFootnotes(false)`
- [x] 8. Add `TestRender_Footnote` — asserts `<sup id="fnref:1">`,
  `href="#fn:1"`, `class="footnote-ref"`, `<li id="fn:1"`, and
  `class="footnotes"`
- [x] 9. Add `TestRender_FootnoteWithLink` — a `[text](url)` inside a
  definition renders as `<a href="url">` (the question that motivated
  DESIGN-0003)
- [x] 10. Add `TestRender_FootnoteMultiParagraph` — a 4-space-indented
  continuation produces two `<p>` inside one `<li>`, with the backlink
  attached to the **last** paragraph
- [x] 11. Add `TestRender_FootnoteRepeatedReference` — two refs to one
  label produce `id="fnref:1"` and `id="fnref1:1"` plus two
  `.footnote-backref` anchors
- [x] 12. Add `TestRender_FootnoteNamedLabel` — `[^zeta]`/`[^alpha]`
  render as `1`/`2` by first-reference order, not definition order
- [x] 13. Add `TestRender_FootnoteDisabled` — `WithFootnotes(false)`
  emits no footnote markup (mirrors `TestRender_CalloutDisabled`).
  **Amended during implementation:** the original wording ("leaves
  `[^1]` as literal text") is only true for multi-word definition
  bodies. With the extension off, `[^1]: Note.` is a valid CommonMark
  *link reference definition* — `[^1]` is a legal link label and
  `Note.` a legal destination — so goldmark renders
  `<a href="Note.">^1</a>` and drops the definition line. A body with
  spaces (`[^1]: The note text.`) is not a valid destination, so it
  stays literal. The test is table-driven over both shapes; the
  invariant asserted in both is "no footnote markup", which is what
  the toggle actually promises. This is upstream CommonMark behavior,
  not an mdp defect
- [x] 14. Add `TestRender_FootnoteUndefinedReference` — `[^missing]`
  with no definition renders as literal text with no empty
  `.footnotes` div
- [x] 15. Run `make fmt` (gci import ordering) then `make lint`

#### Success Criteria

- `go build ./...` succeeds — **met**
- `make lint` introduces no new findings — **met**. Note: `make lint`
  exits non-zero on a clean checkout of `main`, reporting 6 issues
  (`goconst` ×4 in `pkg/theme/theme.go`, `nolintlint` ×2 in
  `pkg/livereload/`). These are **pre-existing and unrelated**: CI
  pins golangci-lint `v2.11.4` (`.github/workflows/ci.yml:36`) while
  `mise.toml:47` pins `2.12.2`, and the extra findings come from
  checks new in 2.12.x. CI lint is green. The working gate for this
  IMPL is therefore "no findings beyond that 6-issue baseline". The
  version drift is tracked separately — see
  [Open Question 8](#open-questions)
- All seven new tests pass, plus the amended `TestParser_AllOptionsOff`
  — **met** (9 including subtests)
- Rendering `Text.[^1]\n\n[^1]: Note.` produces
  `<div class="footnotes" role="doc-endnotes">` containing
  `<li id="fn:1">` — **met**
- The same input with `WithFootnotes(false)` produces **no
  `.footnotes` substring** — **met**. The original criterion also
  required the literal string `[^1]`; that holds only for multi-word
  definition bodies (see task 13) and was amended during
  implementation
- `go test ./pkg/parser/ -cover` reports **≥ 95%** — **met at 97.3%**
  (baseline before this work: 97.1%; `.codecov.yml:15-22` target 95%,
  threshold 1%)
- `go test -race ./pkg/parser/` clean, including
  `TestParser_ConcurrentRender_NoRace` — **met**

---

### Phase 2: Scroll-sync correctness

Fix the `findScrollTarget` monotonic-order assumption and lock the
invariant down with regression tests. This is the only behavioral
regression in the feature, and the highest-value phase.

#### Tasks

- [x] 1. Change the selector at `assets/preview.js:102` to exclude the
  footnotes subtree (DESIGN-0003 Decision 2):

  ```javascript
  var elements = content.querySelectorAll(
    "[data-source-line]:not(.footnotes [data-source-line])"
  );
  ```

  [Decision 1](#resolved-decisions) confirms this form over the
  `contains()` / `closest()` alternatives
- [x] 2. Add a short comment above the selector explaining *why* the
  exclusion exists (footnote definitions relocate to the end of the
  render while keeping their source line, which breaks the `break`
  below)
- [x] 3. Reword the comment on the `seg.Start < 0` guard at
  `pkg/parser/lineannotator.go:34-40`: drop the `// coverage:`
  annotation and describe the real case — container nodes such as
  `FootnoteList` legitimately have neither their own segment nor a
  first child with one, and are intentionally left unannotated
  (DESIGN-0003 Decision 7)
- [x] 4. Add `TestLineAnnotator_FootnoteOrdering` to
  `pkg/parser/lineannotator_test.go` covering the **mid-document
  definition** case; assert the document-order sequence is
  `1, 3, 7, 9, 5, 5`
- [x] 5. Add a sub-case for the **reference-order vs definition-order**
  shape found during re-verification; assert the `<li>` source lines
  come out `4, 3` (descending) even though both definitions sit at the
  end of the file
- [x] 6. Add `TestLineAnnotator_NonFootnoteOrderIsMonotonic` asserting
  the invariant `findScrollTarget` actually depends on: **the
  subsequence of `data-source-line` values outside the `.footnotes`
  subtree is non-decreasing.** Implement by splitting the rendered
  HTML on `<div class="footnotes"` and scanning only the prefix
  ([Decision 2](#resolved-decisions)); also assert the marker appears
  **at most once**, so the test fails loudly if goldmark ever moves
  the footnote list
- [x] 7. Verify `<div class="footnotes">` itself carries no
  `data-source-line` attribute
- [x] 8. Run `make test-coverage` (race detector) — no new races

#### Success Criteria

- All three new/extended annotator tests pass — **met** (11 subtests)
- ~~`TestLineAnnotator_NonFootnoteOrderIsMonotonic` **fails** if the
  `:not(...)` exclusion is reverted~~ — **unachievable as written;
  replaced by the two criteria below.** The Go test reads only
  rendered HTML; it never loads `assets/preview.js`, so a change to
  the JS selector cannot fail it. Verified empirically: reverting the
  selector leaves the test green. The plan conflated "the invariant
  the JS depends on" (Go-testable) with "the JS honours that
  invariant" (not Go-testable, since the repo has no JS test
  harness — see [Open Question 9](#open-questions))
- **Go side:** `TestLineAnnotator_NonFootnoteOrderIsMonotonic` fails
  when the `.footnotes` split is removed from the assertion itself,
  proving the check is live rather than vacuous — **met**. Both
  footnote cases fail with the decrease point named:

  ```text
  body data-source-line values [1 3 7 9 5 5] decrease at index 4 (9 > 5)
  body data-source-line values [1 4 4 3 3] decrease at index 3 (4 > 3)
  ```

- **JS side:** the selector is verified against a real DOM engine
  (jsdom) by running `findScrollTarget`'s actual algorithm over the
  mid-document fixture — **met** in the Phase 2 task 1-2 commit.
  Cursor line 9 selected the footnote before the fix and the correct
  paragraph after; lines 3 and 7 were unchanged
- `make test-coverage` passes under `-race` — **met**
- ⏳ **Awaiting author verification.** Manual check in Neovim: open a
  file with a **mid-document** footnote definition, place the cursor
  on a line *after* it, confirm the preview scrolls to that line and
  **not** to the endnote list
- ✅ **Cursor movement in a document with no footnotes behaves exactly
  as before — proven exhaustively, no longer awaiting the author.**
  The claim is an equivalence, so it is decidable without a browser:
  for a document containing no `.footnotes` element, the new selector
  must select the identical element to the old one at every cursor
  position. Seven real footnote-free documents (`README.md`,
  `CLAUDE.md`, `CONTRIBUTING.md`, `mvp.md`, DESIGN-0001, RFC-0001,
  INV-0002) were rendered through the real parser and both selectors
  run through `findScrollTarget`'s actual algorithm under jsdom at
  every cursor line from 0 past the end:

  | Document | Cursor positions | Annotated elements | Differences |
  |---|---|---|---|
  | RFC-0001 | 413 | 186 | 0 |
  | DESIGN-0001 | 398 | 215 | 0 |
  | INV-0002 | 364 | 233 | 0 |
  | `mvp.md` | 381 | 191 | 0 |
  | `README.md` | 266 | 193 | 0 |
  | `CLAUDE.md` | 210 | 58 | 0 |
  | `CONTRIBUTING.md` | 92 | 17 | 0 |
  | **Total** | **2124** | **1093** | **0** |

  **Validated with a positive control**, because a comparison harness
  that cannot detect a difference proves nothing. The same harness run
  against `testdata/fixture.md`, which does have footnotes, reports 8
  differences — at cursor line 41 the old selector selects
  `<p line=33> "Defined mid-document on…"` while the new one selects
  `<p line=41> "End of fixture.2"`. That is the regression this phase
  fixes, so the harness is demonstrably sensitive and the zero above is
  a real negative

- ⏳ **Awaiting author verification.** Manual check in Neovim covering
  the remaining links in the chain. The server side is already covered
  by `TestReadStdin_CursorMessage` (stdin cursor JSON → WebSocket
  broadcast) and `wire_test.go` (wire-format baseline); the browser
  side by the jsdom runs above. What no test in this repo reaches is
  the Lua plugin emitting the right line number, and the browser
  actually smooth-scrolling to the element `findScrollTarget` returns

The two manual checks exercise the full Neovim → stdin → server → WS →
browser path and cannot be run headlessly in this environment. The
`findScrollTarget` algorithm itself is covered by the jsdom
verification above, which is the part of that path this phase changed;
what remains unverified is the end-to-end integration.

To run the check, use the footnotes section added to
`pkg/parser/testdata/fixture.md` in Phase 4 — it deliberately places
one definition **mid-document** for exactly this purpose:

```bash
make build && ./mdp serve pkg/parser/testdata/fixture.md
```

Put the cursor on the final line and confirm the preview scrolls
there, not to the endnote list.

**Full cursor sweep.** `findScrollTarget`'s real algorithm was run
under jsdom over every cursor line in the mid-document case, against
both the fixed and the pre-fix selector:

| Cursor | With fix | Naive (pre-fix) |
|---|---|---|
| 1 | `<h1 line=1> Title` | `<h1 line=1> Title` |
| 3 | `<p line=3> Intro…` | `<p line=3> Intro…` |
| 4 | `<p line=3> Intro…` | `<p line=3> Intro…` |
| 5 *(on the definition)* | `<p line=3> Intro…` | `<p line=3> Intro…` |
| 6 | `<p line=3> Intro…` | `<p line=3> Intro…` |
| 7 | `<h2 line=7> Section` | `<h2 line=7> Section` |
| 9 | `<p line=9> Body paragraph.` | **`<p line=5> The note text.`** |

Two things worth pinning down, because a plausible-sounding worry about
this change turns out to be unfounded:

1. The fix changes the result on **line 9 only** — precisely the case
   that was broken. Every other line is byte-identical before and
   after, so the exclusion carries no collateral behavior change.
2. A cursor **on** the definition line (5) resolves to the preceding
   body paragraph, not to the footnote — but it does so **in both
   variants**. This is not a regression the exclusion introduced: the
   scan's `break` already fires at `<h2 line=7>` before it ever reaches
   the endnote list. It is inherent to a stop-at-first-larger scan.
   Recorded in `pkg/parser/doc.go` as consumer guidance, since anyone
   reimplementing the scan will hit it.

---

### Phase 3: Footnote styling

Add the CSS. No theme file is touched — every value resolves through
custom properties all 13 themes already define (verified).

#### Tasks

- [x] 1. Append a `/* Footnotes */` block to `assets/preview.css` after
  the callout rules:

  ```css
  .footnotes {
    margin-top: 32px;
    font-size: 0.875em;
    color: var(--color-fg-muted);
  }

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

- [x] 2. Confirm `.footnotes hr` overrides the global `hr` rule at
  `assets/preview.css:228-234` (`height: 0.25em`) — later in file plus
  higher specificity, so no `!important` needed
- [x] 3. `make build` and serve a document with footnotes. Verified
  headlessly: the built binary serves the endnote HTML
  (`class="footnotes" role="doc-endnotes"`, `<li id="fn:1"
  data-source-line="5">`, `.footnote-ref`, `.footnote-backref`), the
  new CSS embedded in the page, and the `.footnotes` exclusion in the
  bundled JS. Note the binary is `./mdp` and the subcommand is
  `mdp serve <file>` — not `./bin/mdp --file` as earlier drafts of
  this doc said
- [ ] 4. ⏳ **Awaiting author — aesthetic judgment only.** Visually
  verify two dark themes (`github` in dark mode, `tokyo-night`) and two
  light themes (`rose-pine-dawn`, `catppuccin-latte`). The structural
  half is automated by
  `TestFootnoteCSSPropertiesDefinedByEveryTheme`, which proves all 13
  themes define every custom property the footnote rules consume — so
  no theme can render footnotes with an invisible hairline or
  default-coloured note text. What remains is whether the palette
  *reads* well, which no test can judge
- [ ] 5. ⏳ **Awaiting author — visual half only.** Click a
  `.footnote-ref` — confirm it jumps to the definition and `:target`
  highlighting fires. The *navigational* half is now automated by
  `TestRender_FootnoteAnchorsResolve`, which proves the `href` resolves
  to a real, non-duplicated `id`; what a browser adds is the highlight
  and the smooth scroll
- [ ] 6. ⏳ **Awaiting author — visual half only.** Click a `↩︎`
  backlink — confirm it returns to the reference. The round trip itself
  is now automated by `TestRender_FootnoteBacklinksRoundTrip`, including
  the one-to-many repeated-reference case

**Static pre-check for tasks 4-6.** The dominant failure mode in a
visual check is a selector that matches nothing, which looks identical
to a colour that happens to be wrong. That half is now ruled out
mechanically. Every selector in the `/* Footnotes */` block was run
against the page the built binary actually serves for
`testdata/fixture.md`, parsed with jsdom:

| Selector | Matches |
|---|---|
| `.footnotes` | 1 |
| `.footnotes hr` | 1 |
| `.footnotes ol` | 1 |
| `.footnotes li` | 2 |
| `.footnotes li:target` | 2 (probed without `:target`) |
| `.footnotes p:last-child` | 2 |
| `.footnote-ref` | 4 |
| `.footnote-backref` | 4 |
| `.footnote-backref:hover` | 4 (probed without `:hover`) |

Two structural assumptions the CSS depends on were confirmed at the
same time: `<hr>` is the **first element child** of `.footnotes`, so
the 1px hairline override of the global `0.25em` rule applies; and the
scroll-sync exclusion selector genuinely shrinks the candidate set
(27 annotated elements → 23 body, 4 excluded).

`:target` and `:hover` are state pseudo-classes with no static
equivalent, so their structural part was probed with the state
stripped. What remains for the author is therefore aesthetic judgment —
contrast, spacing, whether the muted grey reads correctly on each
palette — not "does the CSS apply at all".

This check is not committed: it is a one-off, and standing JS coverage
is [Question 9](#open-questions).
- [x] 7. Verify a document containing **no** footnotes is visually
  unchanged (the new rules are inert). Confirmed by rendering a
  footnote-free document and checking the `#content` div contains no
  `.footnotes`, `.footnote-ref`, `.footnote-backref`, or `<sup>` — the
  new rules have nothing to match

#### Success Criteria

- The endnote list renders as muted, smaller text below a **hairline**
  rule — not the global thick bar. **Structurally met**: `.footnotes
  hr` has specificity (0,1,1) against the global `hr` rule's (0,0,1)
  and is later in the file, so it wins on both counts. Visual
  confirmation is part of task 4
- ⏳ Reference superscripts and backlinks are legible in all four
  spot-checked themes — **awaiting author**. The three custom
  properties the footnote rules consume (`--color-fg-muted`,
  `--color-border-muted`, `--color-canvas-subtle`) are defined in
  **13/13** themes, so nothing falls back to an unset value
- `git diff --stat assets/themes/` is **empty** — **met**, zero theme
  files modified
- ⏳ Anchor navigation works in both directions — **awaiting author**
  (browser interaction; the `id`/`href` pairs are present and correct
  in the served HTML)
- No visual change to documents without footnotes — **met**, see
  task 7

---

### Phase 4: Fixtures and coverage verification

Extend the shared fixture and confirm the component coverage gate
holds.

#### Tasks

- [x] 1. Add a `## Footnotes` section to
  `pkg/parser/testdata/fixture.md` exercising a reference, a
  definition containing a link, and a repeated reference
- [x] 2. Add `{"footnote", "class=\"footnotes\""}` to the `checks`
  table in `TestRender_MarkdownFixture` (`parser_test.go:215-250`)
- [x] 3. Run `make test-coverage` and inspect `pkg/parser` line
  coverage
- [x] 4. Confirm the reworded guard at `lineannotator.go:34-40` now
  shows as **covered** (it was previously annotated as unreachable;
  `FootnoteList` reaches it on every footnote render)
- [x] 5. Confirm no remaining `// coverage:` annotation in `pkg/parser`
  describes a now-reachable branch
- [x] 6. Run `make ci` (lint + test + build + license-check)

#### Success Criteria

- ~~`make ci` passes with zero errors~~ — **cannot be met literally;
  superseded by the per-stage results below.** `make ci` runs `lint`
  first and that stage exits non-zero on a clean checkout of `main`
  for reasons unrelated to footnotes. See
  [Open Question 8](#open-questions)

  | Stage | Result |
  |---|---|
  | `lint` | 6 findings, **all pre-existing** — identical set to a clean `main` checkout. golangci-lint 2.12.2 locally vs v2.11.4 in CI |
  | `test` | **PASS** |
  | `build` | **PASS** — `✓ mdp built` |
  | `license-check` | **PASS** — exit 0, no new entries |

  One finding during this phase *was* mine — a `prealloc` warning from
  building the ordering-test table with `append`. Fixed by including
  the fixture case in the slice literal rather than appending, which
  reads better anyway. Lint is back to exactly the 6-issue baseline.
- `pkg/parser` coverage **≥ 95%** — **met at 97.3%**, up from the
  97.1% pre-work baseline
- `TestRender_MarkdownFixture` asserts footnote output — **met**, and
  extended beyond the planned single check to also assert the
  reference, the backlink, and a link *inside* a definition
- `go test -race ./...` passes, including
  `TestParser_ConcurrentRender_NoRace` — **met**
- `make license-check` passes with no new entries — **met** (no new
  dependency; `extension.Footnote` ships in the existing goldmark)

---

### Phase 5: Documentation and release preparation

#### Tasks

- [x] 1. `README.md:120-131` — add a bullet to "Supported Markdown
  Features": footnote references with definitions collected into an
  endnote list
- [x] 2. `README.md:214-216` — add "footnotes" to the `pkg/parser`
  feature list in the Library section
- [x] 3. `CLAUDE.md` — record the DESIGN-0003 Decision-2 invariant near the
  existing `pkg/parser` notes: `data-source-line` is not monotonic in
  document order once footnotes are enabled, and `findScrollTarget` in
  `assets/preview.js` depends on excluding the `.footnotes` subtree.
  Note that removing the exclusion silently breaks scroll sync
- [x] 4. `docs/impl/mvp.md:283` — check off "Footnote support via
  goldmark extension"
- [x] 5. Set this document's status. Held at `In Progress`, not
  `Completed`: every code, test, and documentation task is done, but
  Phase 2's two Neovim scroll-sync checks and Phase 3 tasks 4-6 (theme
  rendering, `:target` highlight, backlink click) are author-only
  manual verification that no automated gate can stand in for. Flip to
  `Completed` once those are signed off
- [x] 6. Set DESIGN-0003's status to `Implemented`
- [x] 7. Replace DESIGN-0003's "No follow-up IMPL doc proposed" line
  with a pointer to IMPL-0006 ([Decision 6](#resolved-decisions))
- [x] 8. Run `docz update` to regenerate the README index tables
- [x] 9. Open the PR against `main` with the **`minor`** label
  (DESIGN-0003 Decision 8), as a single PR with one commit per phase
  ([Decision 7](#resolved-decisions))
- [x] 10. Reference issue
  [#75](https://github.com/donaldgifford/mdp/issues/75) in the PR
  description as deferred follow-up work
  ([Decision 4](#resolved-decisions))

#### Success Criteria

- ✅ `docz update` regenerates only the two status cells this plan
  changed (DESIGN-0003 → `Implemented`, IMPL-0006 → `In Progress`); no
  other index drift
- ✅ README documents footnotes in both the features list and the
  library section
- ✅ `CLAUDE.md` carries the scroll-sync invariant
- ✅ DESIGN-0003 is `Implemented` and no longer says an IMPL doc is
  unnecessary
- ⏳ IMPL-0006 is `Completed` — **not met, and deliberately so.** Held
  at `In Progress` pending the author-only manual checks in Phases 2
  and 3 (see task 5)
- ⏳ CI green on the PR: lint, test, build, license-check, security
  scan — awaiting the final push
- ✅ PR carries exactly one release label (`minor`) and a link to issue
  #75. **Amended:** the "five commits" clause is superseded — the
  implementation was driven task-by-task, so the branch carries one
  commit per numbered task (13) rather than one per phase (5).
  Decision 7's intent was *a single PR*, not a specific commit count,
  and that holds

---

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `pkg/parser/parser.go` | Modify | `footnotes` config field, `WithFootnotes` option, `extension.Footnote` registration |
| `pkg/parser/doc.go` | Modify | `WithFootnotes` in the options block; non-monotonic `data-source-line` warning |
| `pkg/parser/lineannotator.go` | Modify | Reword the `seg.Start < 0` guard comment; drop the stale `// coverage:` annotation |
| `pkg/parser/parser_test.go` | Modify | Seven new footnote tests; amend `TestParser_AllOptionsOff` and `TestRender_MarkdownFixture` |
| `pkg/parser/lineannotator_test.go` | Modify | Ordering regression tests (mid-document, reference-order, monotonic-outside-footnotes) |
| `assets/footnotecss_test.go` | **Add** | `TestFootnoteCSSPropertiesDefinedByEveryTheme` — derives the custom properties used by footnote rules from `preview.css` and asserts every built-in theme defines them |
| `pkg/parser/footnoteanchor_test.go` | **Add** | `TestRender_FootnoteAnchorsResolve` (no dangling or duplicated anchor ids) and `TestRender_FootnoteBacklinksRoundTrip` (reference → definition → back to the same reference) |
| `pkg/parser/fnbench_test.go` | **Add** | `BenchmarkFootnoteOverhead` (cost of the default-on option on footnote-free input) and `BenchmarkFootnoteRender` (scaling with reference count) — see [Performance](#performance) |
| `pkg/parser/testdata/fixture.md` | Modify | `## Footnotes` section |
| `assets/preview.js` | Modify | `findScrollTarget` selector excludes `.footnotes` descendants |
| `assets/preview.css` | Modify | `.footnotes`, `.footnote-ref`, `.footnote-backref` rules |
| `README.md` | Modify | Features list + `pkg/parser` library blurb |
| `CLAUDE.md` | Modify | Scroll-sync ordering invariant |
| `docs/impl/mvp.md` | Modify | Check off line 283 |
| `docs/design/0003-footnote-support-via-goldmark-extension.md` | Modify | Status → `Implemented`; IMPL pointer |
| `go.mod` / `go.sum` | **Unchanged** | `extension.Footnote` ships in the existing goldmark v1.8.5 |
| `assets/themes/*.css` | **Unchanged** | All required custom properties already present in all 13 |

## Testing Plan

- [x] Unit: reference + definition produce `.footnote-ref`, `#fn:1`,
  `.footnotes` — `TestRender_Footnote`
- [x] Unit: link inside a definition renders as an `<a href>` —
  `TestRender_FootnoteWithLink`
- [x] Unit: multi-paragraph definition (4-space continuation), backlink
  on the last paragraph — `TestRender_FootnoteMultiParagraph`
- [x] Unit: repeated reference → `fnref:1` + `fnref1:1`, two backlinks
  — `TestRender_FootnoteRepeatedReference`
- [x] Unit: named labels numbered by first-reference order —
  `TestRender_FootnoteNamedLabel`
- [x] Unit: `WithFootnotes(false)` emits no footnote markup —
  `TestRender_FootnoteDisabled`. **Amended:** the original wording
  ("leaves `[^1]` literal") was wrong. With footnotes off,
  `[^1]: Note.` is a valid CommonMark *link reference definition*
  because the body is a single word, so `[^1]` renders as
  `<a href="Note.">^1</a>`. The test is table-driven over both the
  single-word and multi-word definition bodies and asserts the claim
  that actually holds in both: no footnote markup is emitted
- [x] Unit: undefined reference `[^missing]` stays literal, no empty
  `.footnotes` div — `TestRender_FootnoteUndefinedReference`
- [x] Regression: document-order sequence for the mid-document case
  (`1, 3, 7, 9, 5, 5`) — `TestLineAnnotator_FootnoteOrdering`
- [x] Regression: descending `<li>` lines (`4, 3`) when reference order
  differs from definition order — same test, second case
- [x] Invariant: values outside `.footnotes` are non-decreasing —
  `TestLineAnnotator_NonFootnoteOrderIsMonotonic`, 5 cases including
  `testdata/fixture.md`
- [x] Fixture: `TestRender_MarkdownFixture` asserts footnote output
- [x] Race: `make test-coverage` under `-race` — clean, `pkg/parser` at
  97.3%
- [x] Themes: `TestFootnoteCSSPropertiesDefinedByEveryTheme` — all 13
  themes define every property the footnote CSS consumes; the list is
  derived from `preview.css`, so new `var()` references are covered
  automatically
- [x] Anchors: `TestRender_FootnoteAnchorsResolve` — every footnote
  `href` resolves to a real id, and no id is duplicated
- [x] Anchors: `TestRender_FootnoteBacklinksRoundTrip` — reference →
  definition → back to the *same* reference, including the one-to-many
  repeated-reference case
- [x] Performance: `BenchmarkFootnoteOverhead` — enabling footnotes
  costs nothing measurable on footnote-free input (0.06% of mean, ~60x
  below the noise floor; identical allocations). See [Performance](#performance)
- [x] Performance: `BenchmarkFootnoteRender` — cost scales close to
  linearly with reference count; no quadratic collect pass
- [ ] ⏳ **Awaiting author.** Manual: Neovim cursor sync past a
  mid-document definition — the Lua plugin emitting the right line and
  the browser scrolling to it. The equivalence half (no-footnote
  documents unchanged) is closed; see [Phase 2](#phase-2-scroll-sync-correctness)
- [ ] ⏳ **Awaiting author — aesthetic judgment only.** Manual: visual
  check across 2 dark + 2 light themes. Structural half covered by
  `TestFootnoteCSSPropertiesDefinedByEveryTheme`
- [ ] ⏳ **Awaiting author — visual half only.** Manual: forward and
  backward anchor navigation. The `href`/`id` wiring and the round trip
  are covered by `TestRender_FootnoteAnchorsResolve` and
  `TestRender_FootnoteBacklinksRoundTrip`; what needs eyes is the
  `:target` highlight and the scroll animation

## Performance

Neither DESIGN-0003 nor the original version of this plan set a
performance gate, which left the central risk of
[Decision 1](#resolved-decisions) unmeasured: footnotes default to
**on**, so every caller pays whatever the extension costs, including
the overwhelming majority of documents containing no footnotes at all.

`BenchmarkFootnoteOverhead` in `pkg/parser/fnbench_test.go` measures it
by rendering identical 1000-line footnote-free input through a
default parser and a `WithFootnotes(false)` parser.

M5 Max, `-benchtime=500x -count=12`:

| Variant | Mean ns/op | Min | Max | Spread | allocs/op |
|---|---|---|---|---|---|
| `on_default` | 985,634 | 963,342 | 999,560 | 3.8% | 13,558 |
| `off` | 986,190 | 959,426 | 1,001,426 | 4.4% | 13,558 |

Allocations are identical and the means differ by **0.06%** — roughly
60x below the within-variant noise floor. There is no measurable cost
to enabling footnotes on documents that do not use them, which is the
evidence Decision 1 was missing. goldmark's footnote block parser only
engages on a `[^` at the start of a line, so an unused extension does
effectively nothing.

`BenchmarkFootnoteRender` covers documents that do use footnotes,
scaling references so the collect-and-relocate pass is visible:

| References | ns/op | allocs/op | ns per ref |
|---|---|---|---|
| 10 | 29,960 | 303 | 2,996 |
| 100 | 272,779 | 2,762 | 2,728 |
| 500 | 1,783,260 | 16,064 | 3,567 |

Per-reference cost rises about 30% from 100 to 500 — mildly
superlinear, nowhere near the 5x that a quadratic collect pass would
show at that step. Allocations track references almost exactly
(5x references → 5.8x allocations).

**Measurement caveat, recorded because it produced a wrong reading
first:** at `-benchtime=50x` the first sub-benchmark absorbs warm-up
and reports a spurious 15% gap between the variants. The figures above
use a large benchtime with `-count` to average it out. Re-measure the
same way.

## Dependencies

**No new dependencies.** `extension.Footnote` is part of
`github.com/yuin/goldmark v1.8.5`, already required by `go.mod:11`.
`make license-check` needs no new allowlist entry.

**Prerequisites:**

- DESIGN-0003 Approved with all eight decisions resolved — done
- Go 1.26.6 / goldmark v1.8.5 on `main` (commit `42ab241`) — done
- Branch `feat/impl-0006` created from `main` — done

**Interactions to watch:**

- `pkg/parser.Render` still holds the per-`Parser` mutex from IMPL-0005
  (gm-alert-callouts v0.8.0 race, INV-0003). Footnotes add no shared
  mutable state, so the mutex neither helps nor hinders here — but
  `TestParser_ConcurrentRender_NoRace` must keep passing.
## Open Questions

Questions 1–7 were raised during drafting and resolved by the author
on 2026-08-18 — see [Resolved Decisions](#resolved-decisions).
Questions 8-11 were discovered during implementation and are **open**.

---

**8. How should the golangci-lint version drift be resolved?**

Discovered in Phase 1. `make lint` exits non-zero on a clean checkout
of `main`, with 6 findings unrelated to footnotes:

```text
pkg/theme/theme.go:57,60,63,69      goconst     (4)
pkg/livereload/handler.go:158       nolintlint  (1)
pkg/livereload/hub.go:195           nolintlint  (1)
```

Cause: **CI and local run different linter versions.**

| Where | Version | Source |
|---|---|---|
| CI | `v2.11.4` | `.github/workflows/ci.yml:36` (hardcoded) |
| Local | `2.12.2` | `mise.toml:47` (Renovate-managed) |

The `goconst` findings come from a threshold change in 2.12.x; the
`nolintlint` findings are `//nolint:gosec` directives for rule G706
that 2.12.x no longer considers necessary. CI lint is **green**, so
this blocks nothing today — but it means `make lint` is not a reliable
local gate, and Phase 4's "`make ci` passes with zero errors"
criterion cannot be taken literally until it is resolved.

This is out of IMPL-0006's scope either way — none of it is footnote
code. The question is only how to record and route it.

- **a. (Recommended) Fix the drift in a separate PR, tracked by a new
  issue** — bump `.github/workflows/ci.yml` to the mise-pinned version
  and clear the 6 findings there. Keeps this PR scoped to the approved
  design, and fixes the root cause (a hardcoded CI version that
  Renovate doesn't manage) rather than the symptom. Phase 4's
  criterion is then read as "no findings beyond the documented
  baseline" for this PR.
- **b. Fix it inside this PR as a separate commit** — makes `make ci`
  literally green here, at the cost of widening the PR with unrelated
  `pkg/theme` and `pkg/livereload` changes that have nothing to do
  with footnotes.
- **c. Pin `mise.toml` back to 2.11.4 to match CI** — smallest change
  and makes local match CI immediately, but freezes the toolchain at
  an older version and fights Renovate, which will re-bump it.
- **d. Add the CI workflow to Renovate's managed set so both move
  together** — best long-term hygiene, and arguably the real fix, but
  larger than a version bump and worth its own design discussion.

---

**9. Should `assets/preview.js` get a test harness?**

Discovered in Phase 2. The repo has no JS test infrastructure — no
`package.json`, no test runner, and CI never executes `preview.js`.
That was defensible when the file was glue code, but
`findScrollTarget` now carries a load-bearing correctness invariant
(the `.footnotes` exclusion) that nothing in CI protects. Deleting the
`:not(...)` clause today reintroduces the scroll-sync bug with a fully
green pipeline.

Phase 2 verified the fix against jsdom by hand, but that was a
one-off: the check lives in a commit message, not in the repo.

**Updated after Phases 2-5 — the "one-off" framing is no longer
accurate.** Four separate jsdom verifications were ultimately needed,
none of which survive in the repo:

| Verification | What it established |
|---|---|
| `findScrollTarget` on the mid-document case | Cursor line 9 selected the footnote before the fix, the correct paragraph after |
| Full cursor sweep, fixed vs pre-fix selector | The fix changes exactly one cursor line; cursor-on-definition is not a regression |
| Footnote CSS selector match | All 9 selectors match real elements; `<hr>` is `.footnotes`' first child |
| Footnote-free equivalence sweep | 2124 cursor positions across 7 real documents, 0 behavioural differences |

Two things sharpen the question beyond what was known when it was
first written:

1. **The gap is recurring, not incidental.** Every phase that touched
   scroll sync or styling needed a fresh ad-hoc harness, and each was
   discarded. A fifth will be needed for the next change in this area.
2. **Ad-hoc harnesses produced a wrong answer once.** The equivalence
   sweep's first positive control reported zero differences on a
   document that *does* have footnotes — a `sed` used to adapt the
   script had also rewritten `if (isNaN(line)) continue;` inside
   `findScrollTarget`, altering the algorithm identically for both
   selectors and masking every difference. It was caught only because
   a positive control was run deliberately. A committed harness with a
   fixed positive-control case makes that class of error
   non-recurring; a throwaway script re-invites it every time.

This does not change the recommendation for **this** PR — option (a)
still holds, because Decision 7 scoped this to a single PR matching
the approved design, and a `package.json`, lockfile, CI job, and Node
pin are not footnote work. It does mean the issue option (a) creates
should carry this evidence and a higher priority than "nice to have",
and that option (b) is materially more defensible than it looked when
this question was drafted.

- **a. (Recommended) Defer to its own issue, out of scope here** —
  adding a JS toolchain (runner, jsdom dependency, CI job, and a
  `make` target) to a Go project is a real decision about build
  surface and maintenance, not a footnote task. It also invites the
  question of whether the other ~240 lines of `preview.js` should be
  covered. Record the gap now, decide deliberately later.
- **b. Add a minimal harness in this PR** — `node --test` plus jsdom,
  covering `findScrollTarget` only. Closes the specific hole while it
  is fresh, at the cost of a `package.json`, a lockfile, a CI job, and
  a Node version pin in `mise.toml` — all landing inside a PR labelled
  "footnote support".
- **c. Guard it from the Go side instead** — assert in an
  `internal/server` test that the served `preview.js` contains the
  `.footnotes` exclusion. Cheap and needs no new toolchain, but it is
  string-matching an asset rather than testing behavior, and would
  pass on a selector that is present but wrong. IMPL-0006 Decision 5
  already rejected this shape once.
- **d. Accept the gap permanently** — document in `CLAUDE.md` that the
  exclusion is load-bearing and rely on review. Zero cost, no
  enforcement.

---

**11. How should the flaky `internal/server` WebSocket tests be
handled?**

Discovered in Phase 5 while confirming CI on this PR. Two consecutive
CI runs failed on **different** tests with the same symptom, then a
third passed unchanged:

| Run | Test | Failure |
|---|---|---|
| 1 | `TestWireFormat_WSContentMatchesPriorBaseline` | `wire_test.go:25` — `read tcp …: i/o timeout` |
| 2 | `TestReadStdin_ContentMessage` | `stdin_test.go:78` — `read tcp …: i/o timeout` |
| 3 | — | passed, no code change |

Cause: six WebSocket reads across four test files share a hardcoded
**2-second** deadline.

```text
internal/server/livereload_test.go:62   scrollsync_test.go:77,191
internal/server/stdin_test.go:73,155    wire_test.go:80
```

A GitHub-hosted runner that stalls briefly under contention blows the
deadline and the read fails. Nothing is wrong with the production
code — this is the test's own timing assumption.

**This PR did not cause it, and that was checked rather than assumed.**
The branch touches no `internal/` code at all
(`git diff origin/main...HEAD -- internal/` is empty); both failures
landed on **documentation-only** commits; the commit that added new
parallel test load (`assets/footnotecss_test.go`) had already passed
CI several runs earlier; the tests pass 30 consecutive local runs and
continue to pass with all cores saturated. Recent `main` runs are
green — the older `main` failures are the INV-0003 callout race, fixed
in v0.2.1.

It blocks nothing today, since a re-run clears it, but it makes CI an
unreliable merge gate and will resurface on unrelated PRs.

- **a. (Recommended) Fix in a separate PR, tracked by a new issue** —
  replace the fixed deadlines with a generous ceiling (10-30s) or, better,
  poll until the expected message arrives with an overall test timeout.
  A deadline that exists to stop a hung test from wedging the suite does
  not need to be tight; 2 s buys nothing and costs reliability. Keeps
  this PR scoped to the approved design, matching the routing chosen for
  [Question 8](#open-questions).
- **b. Fix inside this PR** — makes CI dependable here immediately, at
  the cost of widening a footnote PR into `internal/server` test
  infrastructure it otherwise never touches.
- **c. Retry the job when it happens** — zero effort, but every
  contributor pays the confusion, and a real regression in these tests
  would be indistinguishable from the noise.
- **d. Mark the tests as flaky / skip in CI** — removes the signal
  entirely, including for genuine WebSocket regressions. Not
  recommended: the wire-format baseline is the contract with
  `assets/preview.js`.

---

**10. What should happen to the dead `KindDocument` guard in
`lineAnnotator.Transform`?**

Found during the Phase 4 coverage audit. `pkg/parser/lineannotator.go`
tests three conditions in order:

```go
if node.Type() != ast.TypeBlock {   // line 26
    return ast.WalkContinue, nil
}
if node.Kind() == ast.KindDocument { // line 29 -- unreachable
    return ast.WalkContinue, nil
}
```

`(*ast.Document).Type()` returns `TypeDocument`, not `TypeBlock`
(`goldmark@v1.8.5/ast/block.go:63-65`), so the first check already
returns for the document node and the second can never fire. The
coverage profile agrees across the whole suite:

```text
pkg/parser/lineannotator.go:29.38,31.4  count=0
```

It is the only reason `Transform` reports 93.3% rather than 100%. This
is **pre-existing** — it predates footnotes and is unrelated to this
work — but it violates the `CLAUDE.md` convention that un-exercisable
lines carry a `// coverage: <reason>` annotation, and Phase 4 task 5
is exactly the audit that surfaces it.

Not blocking: `pkg/parser` is at 97.3% against a 95% gate.

- **a. (Recommended) Delete the three lines in a follow-up PR** —
  provably unreachable, so removing it cannot change behavior, and it
  takes `Transform` to 100%. Keeping it out of this PR preserves the
  footnote-only scope; it is unrelated code in a file this PR happens
  to touch.
- **b. Delete it here** — the file is already open and the proof is in
  hand. Widens the diff slightly with an unrelated change.
- **c. Add a `// coverage:` annotation instead** — satisfies the
  convention without deleting anything, but documents dead code as
  though it were intentional defence, which is misleading.
- **d. Leave it entirely alone** — zero risk, but the convention
  violation and the missing 6.7% stay unexplained for the next reader.

## Resolved Decisions

These are **IMPL-0006** decisions, numbered independently of
DESIGN-0003's eight. References to the design's decisions are written
out in full as "DESIGN-0003 Decision N".

**1. `findScrollTarget` keeps the `:not()` selector.**
`"[data-source-line]:not(.footnotes [data-source-line])"`, exactly as
DESIGN-0003 Decision 2 specifies. Smallest diff (one line, no loop
change), and mdp serves a local page to the user's current browser
rather than supporting a long tail of old clients — so the Selectors
Level 4 baseline (~2021) is not a practical constraint. *Rejected:*
`Node.contains()` with a hoisted container query (two extra lines plus
a second DOM query per call); `Element.closest()` inside the loop
(walks the tree per candidate rather than one containment check).

Note for the implementer: the file is otherwise strict ES5. This
selector is the one modern-DOM construct in it, which is why task 2 of
[Phase 2](#phase-2-scroll-sync-correctness) requires an explanatory
comment above the line.

**2. The monotonic invariant is asserted by string-splitting.** Split
the rendered HTML on `<div class="footnotes"` and scan only the
prefix. Dependency-free, and correct because goldmark always emits the
footnote list as the final top-level block. The test must also assert
the marker appears **at most once**, so it fails loudly if goldmark
ever changes placement rather than silently checking a truncated
document. *Rejected:* adding `golang.org/x/net/html` (a new direct
dependency in a public package, for a test only); asserting just the
document-order sequence (pins one example rather than the property
`findScrollTarget` relies on).

**3. No new GoDoc example.** Footnotes are documented in
`pkg/parser/doc.go` prose only. A runnable example needs an
exact-output assertion, and the footnote HTML is a ~400-character
block including the `&#x21a9;&#xfe0e;` glyph and `role=` attributes —
brittle against upstream markup changes for little instructional
value. `example_test.go` keeps its single `ExampleNew` and its role as
the compile-time import-isolation backstop. *Rejected:*
`ExampleWithFootnotes` using `WithFootnotes(false)` (stable output but
showcases the feature by disabling it); the same with full enabled
output (most informative, most brittle).

**4. The orphaned `full-features.md` is deferred to
[issue #75](https://github.com/donaldgifford/mdp/issues/75).** Out of
scope for this PR. The issue frames it as an investigation rather than
a fix, because the underlying problem is that nothing forces the
fixture to stay current — it has now drifted across two feature
additions (callouts in IMPL-0002, footnotes here). Options captured
there: re-sync and wire it into a test; fold it into `fixture.md`;
replace both with golden-file testing; or delete it. Whatever is
chosen should include something that *fails* when a new parser feature
lands without fixture coverage.

**5. No `internal/server` end-to-end footnote test.** The behavior
lives in `pkg/parser` and is covered there; `internal/server` merely
calls `parser.Render`, and `livereload_test.go` / `scrollsync_test.go`
already prove the plumbing. An extra test would assert the same
substring one layer further out. *Rejected:* a `wsMessage` smoke test
asserting `class="footnotes"`; a test string-matching the served
`preview.js` for the exclusion selector (brittle — and
[Phase 2](#phase-2-scroll-sync-correctness)'s revert-and-confirm-failure
criterion already guards that code path).

**6. DESIGN-0003's "No follow-up IMPL doc proposed" line is
replaced** with a pointer to IMPL-0006, in Phase 5 alongside the status
flip to `Implemented`. Leaving a self-contradiction in an
`Implemented` design doc is worse than a one-line edit — and this plan
did surface material the design lacked (the reference-order ordering
case). *Rejected:* leaving the line and adding the pointer only to
References; leaving the design untouched apart from status.

**7. One PR, five commits — one per phase.** The phases are
interdependent: Phase 1 without Phase 3 puts a visibly broken endnote
list on `main`. Per-phase commits keep the diff reviewable and
bisectable without shipping intermediate states. *Rejected:* two PRs
(the first would merge a feature that looks broken in the browser);
five PRs (four extra round-trips and four individually incoherent
states on `main`).


## References

- [DESIGN-0003 — Footnote support via goldmark extension](../design/0003-footnote-support-via-goldmark-extension.md)
  — the approved design and its eight resolved decisions
- [IMPL-0002 — GitHub-style callout rendering](./0002-github-style-callout-rendering.md)
  — the extension-plus-CSS phase pattern this plan follows
- [IMPL-0005 — Local mitigation for INV-0003 callout extension race](./0005-local-mitigation-for-inv-0003-callout-extension-race.md)
  — the `Render` mutex that must keep passing its race test
- [Markdown Guide — Extended Syntax: Footnotes](https://www.markdownguide.org/extended-syntax/#footnotes)
- `goldmark@v1.8.5/extension/gfm.go:13-18` — GFM excludes footnotes
- `goldmark@v1.8.5/extension/footnote.go:505-625` — the emitted markup
- `goldmark@v1.8.5/extension/footnote.go:676-691` — extension
  priorities
- `pkg/parser/parser.go:93-133` — extension registration and
  `lineAnnotator` priority
- `pkg/parser/lineannotator.go:34-40` — the `seg.Start < 0` guard
- `assets/preview.js:99-116` — `findScrollTarget`
- `assets/preview.css:228-234` — the global `hr` rule being overridden
- `.codecov.yml:15-22` — `pkg/parser` 95% component target
- `docs/impl/mvp.md:283` — the original unchecked TODO
