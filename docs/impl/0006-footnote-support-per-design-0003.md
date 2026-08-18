---
id: IMPL-0006
title: "Footnote support per DESIGN-0003"
status: Draft
author: Donald Gifford
created: 2026-08-18
---
<!-- markdownlint-disable-file MD025 MD041 -->

# IMPL 0006: Footnote support per DESIGN-0003

**Status:** Draft
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

- [ ] 1. Change the selector at `assets/preview.js:102` to exclude the
  footnotes subtree (DESIGN-0003 Decision 2):

  ```javascript
  var elements = content.querySelectorAll(
    "[data-source-line]:not(.footnotes [data-source-line])"
  );
  ```

  [Decision 1](#resolved-decisions) confirms this form over the
  `contains()` / `closest()` alternatives
- [ ] 2. Add a short comment above the selector explaining *why* the
  exclusion exists (footnote definitions relocate to the end of the
  render while keeping their source line, which breaks the `break`
  below)
- [ ] 3. Reword the comment on the `seg.Start < 0` guard at
  `pkg/parser/lineannotator.go:34-40`: drop the `// coverage:`
  annotation and describe the real case — container nodes such as
  `FootnoteList` legitimately have neither their own segment nor a
  first child with one, and are intentionally left unannotated
  (DESIGN-0003 Decision 7)
- [ ] 4. Add `TestLineAnnotator_FootnoteOrdering` to
  `pkg/parser/lineannotator_test.go` covering the **mid-document
  definition** case; assert the document-order sequence is
  `1, 3, 7, 9, 5, 5`
- [ ] 5. Add a sub-case for the **reference-order vs definition-order**
  shape found during re-verification; assert the `<li>` source lines
  come out `4, 3` (descending) even though both definitions sit at the
  end of the file
- [ ] 6. Add `TestLineAnnotator_NonFootnoteOrderIsMonotonic` asserting
  the invariant `findScrollTarget` actually depends on: **the
  subsequence of `data-source-line` values outside the `.footnotes`
  subtree is non-decreasing.** Implement by splitting the rendered
  HTML on `<div class="footnotes"` and scanning only the prefix
  ([Decision 2](#resolved-decisions)); also assert the marker appears
  **at most once**, so the test fails loudly if goldmark ever moves
  the footnote list
- [ ] 7. Verify `<div class="footnotes">` itself carries no
  `data-source-line` attribute
- [ ] 8. Run `make test-coverage` (race detector) — no new races

#### Success Criteria

- All three new/extended annotator tests pass
- `TestLineAnnotator_NonFootnoteOrderIsMonotonic` **fails** if the
  `:not(...)` exclusion is reverted — verify by temporarily reverting,
  re-running, then restoring
- `make test-coverage` passes under `-race`
- Manual check in Neovim: open a file with a **mid-document** footnote
  definition, place the cursor on a line *after* it, confirm the
  preview scrolls to that line and **not** to the endnote list
- Manual check: cursor movement in a document with **no** footnotes
  behaves exactly as before

---

### Phase 3: Footnote styling

Add the CSS. No theme file is touched — every value resolves through
custom properties all 13 themes already define (verified).

#### Tasks

- [ ] 1. Append a `/* Footnotes */` block to `assets/preview.css` after
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

- [ ] 2. Confirm `.footnotes hr` overrides the global `hr` rule at
  `assets/preview.css:228-234` (`height: 0.25em`) — later in file plus
  higher specificity, so no `!important` needed
- [ ] 3. `make build` and open a document with footnotes
- [ ] 4. Visually verify two dark themes (`github` in dark mode,
  `tokyo-night`) and two light themes (`rose-pine-dawn`,
  `catppuccin-latte`)
- [ ] 5. Click a `.footnote-ref` — confirm it jumps to the definition
  and `:target` highlighting fires
- [ ] 6. Click a `↩︎` backlink — confirm it returns to the reference
- [ ] 7. Verify a document containing **no** footnotes is visually
  unchanged (the new rules are inert)

#### Success Criteria

- The endnote list renders as muted, smaller text below a **hairline**
  rule — not the global thick bar
- Reference superscripts and backlinks are legible in all four
  spot-checked themes
- `git diff --stat assets/themes/` is **empty** — zero theme files
  modified
- Anchor navigation works in both directions
- No visual change to documents without footnotes

---

### Phase 4: Fixtures and coverage verification

Extend the shared fixture and confirm the component coverage gate
holds.

#### Tasks

- [ ] 1. Add a `## Footnotes` section to
  `pkg/parser/testdata/fixture.md` exercising a reference, a
  definition containing a link, and a repeated reference
- [ ] 2. Add `{"footnote", "class=\"footnotes\""}` to the `checks`
  table in `TestRender_MarkdownFixture` (`parser_test.go:215-250`)
- [ ] 3. Run `make test-coverage` and inspect `pkg/parser` line
  coverage
- [ ] 4. Confirm the reworded guard at `lineannotator.go:34-40` now
  shows as **covered** (it was previously annotated as unreachable;
  `FootnoteList` reaches it on every footnote render)
- [ ] 5. Confirm no remaining `// coverage:` annotation in `pkg/parser`
  describes a now-reachable branch
- [ ] 6. Run `make ci` (lint + test + build + license-check)

#### Success Criteria

- `make ci` passes with zero errors
- `pkg/parser` coverage **≥ 95%**, and not more than 1% below the
  97.1% baseline (the codecov component threshold)
- `TestRender_MarkdownFixture` asserts footnote output
- `go test -race ./...` passes, including
  `TestParser_ConcurrentRender_NoRace`
- `make license-check` passes with no new entries (no new dependency)

---

### Phase 5: Documentation and release preparation

#### Tasks

- [ ] 1. `README.md:120-131` — add a bullet to "Supported Markdown
  Features": footnote references with definitions collected into an
  endnote list
- [ ] 2. `README.md:214-216` — add "footnotes" to the `pkg/parser`
  feature list in the Library section
- [ ] 3. `CLAUDE.md` — record the DESIGN-0003 Decision-2 invariant near the
  existing `pkg/parser` notes: `data-source-line` is not monotonic in
  document order once footnotes are enabled, and `findScrollTarget` in
  `assets/preview.js` depends on excluding the `.footnotes` subtree.
  Note that removing the exclusion silently breaks scroll sync
- [ ] 4. `docs/impl/mvp.md:283` — check off "Footnote support via
  goldmark extension"
- [ ] 5. Set this document's status to `Completed`
- [ ] 6. Set DESIGN-0003's status to `Implemented`
- [ ] 7. Replace DESIGN-0003's "No follow-up IMPL doc proposed" line
  with a pointer to IMPL-0006 ([Decision 6](#resolved-decisions))
- [ ] 8. Run `docz update` to regenerate the README index tables
- [ ] 9. Open the PR against `main` with the **`minor`** label
  (DESIGN-0003 Decision 8), as a single PR with one commit per phase
  ([Decision 7](#resolved-decisions))
- [ ] 10. Reference issue
  [#75](https://github.com/donaldgifford/mdp/issues/75) in the PR
  description as deferred follow-up work
  ([Decision 4](#resolved-decisions))

#### Success Criteria

- `docz update --dry-run` reports no drift
- README documents footnotes in both the features list and the library
  section
- `CLAUDE.md` carries the scroll-sync invariant
- DESIGN-0003 is `Implemented` and no longer says an IMPL doc is
  unnecessary; IMPL-0006 is `Completed`
- CI green on the PR: lint, test, build, license-check, security scan
- PR carries exactly one release label (`minor`), five commits, and a
  link to issue #75

---

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `pkg/parser/parser.go` | Modify | `footnotes` config field, `WithFootnotes` option, `extension.Footnote` registration |
| `pkg/parser/doc.go` | Modify | `WithFootnotes` in the options block; non-monotonic `data-source-line` warning |
| `pkg/parser/lineannotator.go` | Modify | Reword the `seg.Start < 0` guard comment; drop the stale `// coverage:` annotation |
| `pkg/parser/parser_test.go` | Modify | Seven new footnote tests; amend `TestParser_AllOptionsOff` and `TestRender_MarkdownFixture` |
| `pkg/parser/lineannotator_test.go` | Modify | Ordering regression tests (mid-document, reference-order, monotonic-outside-footnotes) |
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

- [ ] Unit: reference + definition produce `.footnote-ref`, `#fn:1`,
  `.footnotes`
- [ ] Unit: link inside a definition renders as an `<a href>`
- [ ] Unit: multi-paragraph definition (4-space continuation), backlink
  on the last paragraph
- [ ] Unit: repeated reference → `fnref:1` + `fnref1:1`, two backlinks
- [ ] Unit: named labels numbered by first-reference order
- [ ] Unit: `WithFootnotes(false)` leaves `[^1]` literal
- [ ] Unit: undefined reference `[^missing]` stays literal, no empty
  `.footnotes` div
- [ ] Regression: document-order sequence for the mid-document case
  (`1, 3, 7, 9, 5, 5`)
- [ ] Regression: descending `<li>` lines (`4, 3`) when reference order
  differs from definition order
- [ ] Invariant: values outside `.footnotes` are non-decreasing
- [ ] Fixture: `TestRender_MarkdownFixture` asserts footnote output
- [ ] Race: `make test-coverage` under `-race`
- [ ] Manual: Neovim cursor sync past a mid-document definition
- [ ] Manual: visual check across 2 dark + 2 light themes
- [ ] Manual: forward and backward anchor navigation

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
Question 8 was discovered during implementation and is **open**.

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
