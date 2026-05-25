---
id: IMPL-0004
title: "Phase 3-4 of RFC-0001 — harden public API and tag v0.2.0"
status: Draft
author: Donald Gifford
created: 2026-05-24
---
<!-- markdownlint-disable-file MD025 MD041 -->

# IMPL 0004: Phase 3-4 of RFC-0001 — harden public API and tag v0.2.0

**Status:** Draft
**Author:** Donald Gifford
**Date:** 2026-05-24

<!--toc:start-->
- [Objective](#objective)
- [Scope](#scope)
  - [In Scope](#in-scope)
  - [Out of Scope](#out-of-scope)
- [Prerequisites](#prerequisites)
- [Implementation Phases](#implementation-phases)
  - [Phase 3: Harden public API for v1](#phase-3-harden-public-api-for-v1)
    - [Tasks](#tasks)
    - [Success Criteria](#success-criteria)
  - [Phase 4: Tag v0.2.0](#phase-4-tag-v020)
    - [Tasks](#tasks-1)
    - [Success Criteria](#success-criteria-1)
- [Cross-Phase Concerns](#cross-phase-concerns)
- [Open Questions](#open-questions)
  - [Resolved](#resolved)
- [References](#references)
<!--toc:end-->

## Objective

Finish RFC-0001 by hardening the public packages
(`pkg/parser`, `pkg/theme`, `pkg/livereload`) for their first semver
release and tagging `v0.2.0`. Phase 3 adds the compile-time isolation
proof (per-package `example_test.go`), `doc.go` package docs, the
`WithMermaidRenderMode` option, and a `Theme` field audit. Phase 4
ships the README "Library" section and tags the release.

**Implements:** [RFC-0001](../rfc/0001-public-mdp-go-library.md)
phases 3 and 4. **Depends on:** [IMPL-0003](0003-phase-1-2-of-rfc-0001-lift-parsertheme-and-extract-livereload.md)
phases 0/1/2 merged to `main`, plus RFC-0001 phase 2.5 (docz
`serve` validation) complete with any API friction fed back into
IMPL-0003.

## Scope

### In Scope

- Per-package `doc.go` (package-level GoDoc with usage example) for
  `pkg/parser`, `pkg/theme`, `pkg/livereload`
- Per-package `example_test.go` importing *only* that package
  (compile-time isolation proof + `go doc` examples)
- Cross-package import audit (grep + manual confirmation that no
  exported function in package X takes/returns a type from package
  Y; boundary types are stdlib only)
- Add `parser.WithMermaidRenderMode(mode mermaid.RenderMode) Option`
- Audit `theme.Theme` exported fields; decide whether `IsAuto`,
  `HljsVendorCSS`, `MermaidTheme` stay as fields or move behind
  accessors before the v1 contract freezes
- Document the `pkg/livereload` transport contract in its `doc.go`
  (WS frame format, SSE event framing, payload-is-opaque convention)
- README "Library" section with two worked examples
  (parser-only + parser+theme+livereload)
- Tag `v0.2.0`

### Out of Scope

- New library packages beyond the three established in IMPL-0003
- Changes to `internal/server`, `internal/watcher`, `internal/cli`
  (except passive import-path updates if any audit-driven rename
  happens)
- Mermaid render-mode default change — `WithMermaidRenderMode` is
  additive; default stays `mermaid.RenderModeClient`
- docz `serve` implementation (lives in the docz repo; phase 2.5)
- Snap/Homebrew/distro packaging (covered by the deferred-channel
  list in INV-0002)

## Prerequisites

Phase 3 must not start until:

1. IMPL-0003 phases 0/1/2 are merged to `main`. Specifically:
   - `pkg/parser`, `pkg/theme`, `pkg/livereload` exist and are
     covered by tests
   - `internal/server` consumes `pkg/livereload` (no `hub.go` /
     `sse.go` files left in `internal/server`)
   - `make test-coverage` (the CI-invoked race-detected path) is
     green on `main`
2. RFC-0001 phase 2.5 is complete: docz has a working `serve` command
   built against the un-tagged public packages via a `replace`
   directive. Any API friction surfaced in docz has been fed back into
   `pkg/{parser,theme,livereload}` *before* this IMPL begins. If
   docz's needs reveal a missing option or a wrong field shape, fix
   it in IMPL-0003 follow-ups, not here — phase 3 is for hardening,
   not for finding new API gaps.

## Implementation Phases

### Phase 3: Harden public API for v1

**Branch:** `feat/harden-public-api-for-v1`
**Estimated diff:** 10–15 files. Mostly new (doc.go, example_test.go,
testdata if any), with surgical changes to `pkg/parser/parser.go`
(new option) and possibly `pkg/theme/theme.go` (field audit
outcomes).

#### Tasks

**Cross-package isolation audit**

- [x] Run
      `grep -rn "donaldgifford/mdp/pkg/" pkg/parser/ pkg/theme/ pkg/livereload/ --include='*.go' | grep -v _test.go`
      → returns nothing. No `pkg/X` package imports another `pkg/Y`
      package
- [x] Run
      `go doc -all ./pkg/parser ./pkg/theme ./pkg/livereload` →
      every exported symbol's signature uses only stdlib types or
      types from its own package. One intentional exception:
      `parser.WithMermaidRenderMode(mode mermaid.RenderMode)` takes
      a type from `go.abhg.dev/goldmark/mermaid` since that's the
      mode enum the underlying extender uses; consumers who don't
      set the option don't have to import that package
- [x] Audit surfaced no unintended coupling

**Per-package doc.go (primary docs surface)**

Each package's `doc.go` is the canonical place for usage examples.
The README's Library section (phase 4) defers to GoDoc rather than
duplicating prose. Each `doc.go` holds the package-level GoDoc
comment with usage examples rendered as fenced code blocks.

- [x] `pkg/parser/doc.go`: minimal `parser.New().Render([]byte("# Hi"))`
      example + all-options example including `WithMermaidRenderMode`
- [x] `pkg/theme/doc.go`: `Resolve("github-light")` example reading
      the resolved fields (`MermaidTheme`, `IsAuto()`,
      `HljsVendorCSS`); binary-bloat note included
- [x] `pkg/livereload/doc.go`: transport contract (WS TextMessage,
      SSE `data: ...\n\n` framing, opaque payload); concurrency +
      security sections; single composition example using only
      stdlib + livereload (no typed/JSON example per Q2)

**Per-package example_test.go (compile-time isolation backstop)**

Each public package has one `example_test.go` with a *minimal*
runnable `Example*` function. The point is the compile-time
guarantee: the file's import block must contain only that package
plus stdlib; if anyone later adds a cross-`pkg/` import to the
public surface, the example fails to build. Examples are intentionally
small — the rich examples live in `doc.go`.

- [x] `pkg/parser/example_test.go`:
  - [x] `ExampleNew`: 5-line `parser.New().Render([]byte("# Hi"))`
        with `// Output:` assertion
  - [x] Import block: only `parser` + `fmt`
- [x] `pkg/theme/example_test.go`:
  - [x] `ExampleResolve`: `theme.Resolve("github-light")` and prints
        `MermaidTheme`, `IsAuto()`, `HljsVendorCSS`
  - [x] Import block: only `theme` + `fmt`
- [x] `pkg/livereload/example_test.go`:
  - [x] `ExampleHub`: `NewHub()`, `Count()`, `Close()` — no network
        required
  - [x] Import block: only `livereload` + `fmt`. Notably no
        gorilla/websocket (transitive only)

**pkg/parser: WithMermaidRenderMode option**

- [x] Add `WithMermaidRenderMode(mode mermaid.RenderMode) Option`
      to `pkg/parser/parser.go`. Stored in `config.mermaidMode`,
      defaults to `mermaid.RenderModeClient`
- [x] Updated the mermaid extender block to use `cfg.mermaidMode`
- [x] `TestParser_WithMermaidRenderMode_Server` exercises
      `RenderModeServer` against a fixture with a mermaid block;
      asserts `<svg>` is emitted (not `<pre class="mermaid">`).
      Skipped via `t.Skip` when the `mmdc` CLI is not on `$PATH`
      so the suite stays green on machines without it.
      `TestParser_WithMermaidRenderMode_Client` asserts the default
      client-mode behavior
- [x] `WithMermaidRenderMode` documented in `pkg/parser/doc.go`'s
      all-options code block

**pkg/theme: exported field audit**

- [x] `CSS string` — kept as field (load-bearing).
- [x] `HljsVendorCSS string` — kept as field.
- [x] `MermaidTheme string` — kept as field.
- [x] Converted `IsAuto bool` → `Theme.IsAuto() bool` method.
      Underlying field is now `isAuto bool` (unexported)
- [x] Updated `internal/server/server.go` `pageData.IsAuto`
      initializer to call `s.theme.IsAuto()`
- [x] Updated `pkg/theme/theme_test.go` call sites (4) to use
      `IsAuto()`
- [x] `pkg/theme/doc.go` example uses `IsAuto()`

**Documentation and isolation proofs**

- [x] `go test ./pkg/...` — all `Example*` and unit tests pass
- [x] `go vet ./pkg/...` — clean (covered by `make lint`)
- [x] `go doc github.com/donaldgifford/mdp/pkg/parser` (and theme,
      livereload) — package docs render with sections, examples, and
      method/function lists as expected

**Coverage hardening (100% target on exported symbols)**

Public packages aim for 100% line coverage on exported
functions/methods. Defensive checks that aren't realistically
exercisable (e.g., a `panic` guard on a stdlib call that doesn't fail
in practice) are acceptable exemptions — annotate inline with a
`// coverage: <reason>` comment so the gap is intentional and
auditable.

- [x] Per-package coverage after additional tests:
      `pkg/parser` 97.0%, `pkg/theme` 93.8%, `pkg/livereload` 89.7%.
      Added `TestParser_AllOptionsOff` (covers every disable-side
      With* setter), `TestWrapHandler_HonorsCustomInjectionPoint`,
      `TestWrapHandler_PreservesUnderlyingStatusCode`,
      `TestWrapHandler_WarnsOnCustomPathWithoutCustomClientJS`,
      `TestHub_HandleAfterClose`
- [x] All remaining sub-100 lines annotated with
      `// coverage: <reason>` (defensive guards on goldmark.Convert
      errors, embedded-asset panic, filepath.Abs error, gorilla
      conn.Close errors, websocket upgrade failure, SSE non-Flusher
      writer, SSE drop loop, response write failure, idempotent
      remove guards)
- [ ] Update `.codecov.yml` if needed to assert per-`pkg/` thresholds
      higher than the project-wide 60% *(deferred — codecov is not
      currently wired up; existing project-wide 60% in CLAUDE.md is
      enforced via `make test-coverage` thresholds rather than
      codecov.yml. Tracking a follow-up if codecov is added.)*

**PR**

- [x] Open PR with `patch` label — [PR #49](https://github.com/donaldgifford/mdp/pull/49)
      (the `Theme.IsAuto` field→method is breaking for any external
      consumer; since the in-tree consumer is the only known caller
      and there's no released library yet, patch label fits)
- [x] PR title: `feat: harden public pkg API for v0.2.0`
- [x] PR body references RFC-0001 phase 3 acceptance criteria

#### Success Criteria

- `make build && make test && make lint && make test-coverage` all
  green (test-coverage is the race-detected CI path)
- `go test ./pkg/...` runs and all `Example*` tests pass
- Each `example_test.go` imports *only* its own package + stdlib
  (compile-time enforcement of isolation; the test file fails to
  build if a cross-`pkg/` import is added later)
- `go doc github.com/donaldgifford/mdp/pkg/parser` (and theme,
  livereload) shows readable package docs with at least one example
  in the doc comment
- **100% line coverage on exported symbols** in `pkg/parser`,
  `pkg/theme`, `pkg/livereload`. Any gaps annotated with
  `// coverage: <reason>` and listed in the PR body
- `theme.Theme.IsAuto` is a method (not a field); call sites in
  `internal/server` updated accordingly
- No regressions in `mdp --file` or Neovim plugin behavior

---

### Phase 4: Tag v0.2.0

**Branch:** `chore/release-v0.2.0`
**Estimated diff:** 2 files (README.md, possibly cliff.toml tweak).
**Depends on:** Phase 3 merged to `main`.

#### Tasks

**README.md Library section**

Minimal — README points at GoDoc rather than duplicating examples.
If a richer top-level explainer is needed later, revisit.

- [ ] Add a `## Library` section (probably after `## Install`,
      before `## Development`) with:
  - One-paragraph description: "mdp's markdown parser, theme
    registry, and live-reload primitive are also importable as Go
    packages. See the package docs for usage."
  - GoDoc links: `pkg.go.dev/github.com/donaldgifford/mdp/pkg/parser`,
    `pkg/theme`, `pkg/livereload`
  - A short "use cases" pointer table:
    - markdown → HTML in your own app → `pkg/parser`
    - same with mdp's look → add `pkg/theme`
    - same with browser auto-reload → add `pkg/livereload`
  - One concrete reference to a downstream consumer: docz `serve`
    (link the docz repo)

**Release mechanics**

- [ ] Run `git-cliff --bump --unreleased` locally to preview the
      generated changelog for v0.2.0; confirm the Features section
      lists the `pkg/` additions (`lift parser and theme into pkg/`,
      `extract pkg/livereload`, `harden public pkg API for v0.2.0`)
- [ ] If the cliff output is missing or mis-categorizes any commits,
      tweak `cliff.toml` group regexps. Otherwise leave alone
- [ ] Open PR with `minor` label (this is the v0.2.0 trigger).
      Pre-merge: dry-run the release workflow if possible
- [ ] PR title: `chore: release v0.2.0 — public pkg/ API`
- [ ] PR body: link RFC-0001, IMPL-0003, IMPL-0004; note that this
      tag freezes the v1 public API surface and any future breaking
      change requires a v0 → v1 bump or a `replace`-style migration

**Post-merge**

- [ ] Confirm the release workflow runs cleanly and produces:
      tagged `v0.2.0`, signed checksums, SBOM, CHANGELOG entry
- [ ] Verify the published Go module resolves:
      `go get github.com/donaldgifford/mdp@v0.2.0` from a
      throwaway scratch dir, then
      `import "github.com/donaldgifford/mdp/pkg/parser"`,
      `go build` against a 5-line `main.go`
- [ ] Update RFC-0001 status from `Draft` → `Accepted`; update
      DESIGN-0002 status from `Draft` → `Implemented`; update
      IMPL-0003 and IMPL-0004 from `Draft` → `Completed`. Run
      `docz update` to refresh the indexes

#### Success Criteria

- `git tag` shows `v0.2.0`; the GitHub release exists with
  attached archives, SBOM, and signed checksums
- A throwaway `go.mod` outside the repo can
  `go get github.com/donaldgifford/mdp@v0.2.0` and build a tiny
  program importing `pkg/parser`
- `CHANGELOG.md` has a `v0.2.0` section listing the `pkg/` work
- All four docs (RFC-0001, DESIGN-0002, IMPL-0003, IMPL-0004) have
  their statuses bumped to the terminal state and the docz indexes
  reflect it
- `README.md` has a `## Library` section with the use-cases
  pointer table and GoDoc links

---

## Cross-Phase Concerns

- **Library API is stable from v0.2.0 onward.** After phase 4 tags,
  any breaking change to `pkg/{parser,theme,livereload}` requires a
  major bump (`v0` → `v1`, or `v1` → `v2`). Plan API audits
  carefully *before* this IMPL is marked Completed; after, the cost
  of a rename or signature change goes up sharply.
- **docz drives requirements; mdp owns the API.** If docz's needs
  change post-v0.2.0, additive `With*` options are fine; structural
  changes to existing types are breaking. Use phase 2.5's tight
  feedback loop to catch structural needs before they freeze.
- **No CHANGELOG hand-edits.** Same as IMPL-0003. `git-cliff`
  generates everything; tune `cliff.toml` if categorization is off.
- **Neovim plugin remains untouched.** The Go binary path is
  unchanged. The lazy.nvim spec, build.lua, and lua/mdp/init.lua are
  not affected by anything in this IMPL.

## Open Questions

None remaining. All six questions raised during drafting are
captured below under [Resolved](#resolved).

### Resolved

1. **`theme.Theme.IsAuto` rename.** Convert to a method
   (`Theme.IsAuto() bool`) before v0.2.0 freezes. Reflected in the
   field-audit tasks above. Other fields (`CSS`, `HljsVendorCSS`,
   `MermaidTheme`) stay as fields.
2. **Typed-message example in `pkg/livereload`.** Not included.
   Examples are opaque-bytes only — typed messaging is mdp's
   contract, not livereload's, and including it would muddle the
   "payload is opaque" guarantee. Richer examples (if needed) go in
   the package's `doc.go`.
3. **README "Library" section depth.** Minimal. Section points at
   GoDoc rather than duplicating examples. `doc.go` per package is
   the canonical example surface; special-purpose docs pages can be
   added later if `doc.go` proves insufficient.
4. **`cliff.toml` updates.** Dry-run during phase 4 (already a task);
   adjust only if categorization is off.
5. **docz `replace` cleanup.** Tracked as a docz-repo follow-up in
   the phase 4 post-merge task list (verify `go get @v0.2.0` works
   from a throwaway dir; docz then drops the `replace` and pins
   `v0.2.0`).
6. **Coverage thresholds.** **100% line coverage on exported
   symbols** in `pkg/parser`, `pkg/theme`, `pkg/livereload`.
   Defensive gaps that aren't realistically exercisable get a
   `// coverage: <reason>` annotation. Project-wide threshold stays
   at the current 60% — the bar is raised only for public packages.
   Back-ported to IMPL-0003 phase 2 success criteria.

## References

- [IMPL-0003 — Phase 1-2 of RFC-0001](0003-phase-1-2-of-rfc-0001-lift-parsertheme-and-extract-livereload.md)
  — direct predecessor; phases 0/1/2 must merge before this IMPL
  begins
- [RFC-0001 — Public mdp Go library](../rfc/0001-public-mdp-go-library.md)
  — phases 3 and 4 spec
- [DESIGN-0002 — Refactor mdp internals into public pkg packages](../design/0002-refactor-mdp-internals-into-public-pkg-packages.md)
  — original design; phase 3 hardening was explicitly out of scope
  there
- `pkg/parser/parser.go` (after IMPL-0003) — site of the
  `WithMermaidRenderMode` addition
- `pkg/theme/theme.go` (after IMPL-0003) — site of the field audit
- `pkg/livereload/doc.go` (NEW) — site of the transport contract
  documentation
- `assets/preview.js` — the source of mdp's specific (typed)
  message-handling shape; useful reference for what docz would
  build *separately* from the opaque `pkg/livereload` contract
- `cliff.toml` — informs Open Question 4
- `.github/workflows/release.yml` — release workflow that runs on
  the v0.2.0 PR merge
- `CLAUDE.md` § CI/CD — release mechanics, coverage thresholds
