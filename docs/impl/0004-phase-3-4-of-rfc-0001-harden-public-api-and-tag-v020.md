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
   - `make test-race` is green in CI
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

- [ ] Run
      `grep -rn "donaldgifford/mdp/pkg/" pkg/parser/ pkg/theme/ pkg/livereload/ --include='*.go' | grep -v _test.go`
      to confirm no `pkg/X` package imports another `pkg/Y` package
      (parser must not import theme/livereload; theme must not
      import parser/livereload; livereload must not import
      parser/theme). Allowed cross-deps: `pkg/theme` → `assets`
      (stdlib-of-mdp; already documented)
- [ ] Run
      `go doc -all ./pkg/parser ./pkg/theme ./pkg/livereload`
      and confirm every exported symbol's signature uses only stdlib
      types or types from its own package. Document anything that
      does cross packages and decide whether it's intentional
- [ ] If audit surfaces unintended coupling, file as a follow-up PR
      rather than fixing inline (keeps this PR scoped to hardening)

**Per-package doc.go**

- [ ] `pkg/parser/doc.go`: package-level GoDoc with a 10-line
      "render markdown to HTML" example in the comment block; covers
      Mermaid + KaTeX + highlighting + callouts feature flags
- [ ] `pkg/theme/doc.go`: package-level GoDoc explaining the theme
      registry, `Resolve`/`Names`, and that consumers importing
      `pkg/theme` will pull in `assets`' embedded JS/CSS (binary
      bloat warning)
- [ ] `pkg/livereload/doc.go`: package-level GoDoc with **transport
      contract** spelled out — WS frame format (TextMessage carrying
      opaque `[]byte` payload), SSE event framing (`data: <payload>\n\n`),
      and the explicit statement that `Broadcast` payloads are
      consumer-defined. Include the standard `WrapHandler` + `Hub`
      composition example

**Per-package example_test.go**

- [ ] `pkg/parser/example_test.go`:
  - [ ] `ExampleNew_basic`: minimal `parser.New().Render([]byte("# Hi"))`
  - [ ] `ExampleNew_withOptions`: parser with all current `With*`
        options + the new `WithMermaidRenderMode`
  - [ ] Import block must contain only `parser` (and stdlib).
        Compile-time fail if the audit was wrong
- [ ] `pkg/theme/example_test.go`:
  - [ ] `ExampleResolve`: resolve `"github-light"` and use the
        returned `Theme` fields
  - [ ] `ExampleNames`: print all built-in theme names
  - [ ] Import block: `theme` + stdlib (+ `assets` only if the
        example serves CSS bytes)
- [ ] `pkg/livereload/example_test.go`:
  - [ ] `ExampleHub_broadcast`: minimal hub + WS connect from test
        client + assert receive
  - [ ] `ExampleWrapHandler`: wrap a stdlib `http.HandlerFunc`,
        broadcast, demonstrate auto-reload `<script>` in response
  - [ ] Import block: `livereload` + stdlib + `gorilla/websocket`
        (transitively required) — no `pkg/parser`, no `pkg/theme`

**pkg/parser: WithMermaidRenderMode option**

- [ ] Add `Option` `WithMermaidRenderMode(mode mermaid.RenderMode) Option`
      to `pkg/parser/parser.go`. Stored in `config.mermaidMode`,
      defaults to `mermaid.RenderModeClient` (preserves current
      behavior)
- [ ] Update the existing mermaid extender block (currently
      `mermaid.Extender{RenderMode: mermaid.RenderModeClient}`) to
      use `cfg.mermaidMode`
- [ ] Add a unit test in `pkg/parser/parser_test.go`:
      `TestParser_MermaidRenderMode_Server` exercises
      `mermaid.RenderModeServer` against a fixture markdown with a
      mermaid block; assert the rendered HTML contains an `<svg>`
      (server-side render) rather than a `<pre class="mermaid">`
      placeholder (client-side render)
- [ ] Add the new option to `ExampleNew_withOptions` from above

**pkg/theme: exported field audit**

- [ ] Walk each exported field on `Theme`:
  - [ ] `CSS string` — load-bearing; consumers will inject this into
        their HTML. Keep as field
  - [ ] `HljsVendorCSS string` — path to a vendored stylesheet.
        Consumers serve this from `assets.FS`. Decide: keep as field
        (path is simple to use) vs. accessor `Theme.HljsVendorCSSPath()`.
        Default recommendation: keep field; rename is not a v2
        regret because the field name is precise
  - [ ] `MermaidTheme string` — string passed to `mermaid.initialize()`.
        Consumers using `pkg/livereload` + their own mermaid integration
        need this. Keep as field
  - [ ] `IsAuto bool` — sentinel for the browser-driven auto theme.
        Decide: keep as field vs. `Theme.IsAuto()` accessor.
        Recommendation: accessor — `IsAuto bool` reads oddly when
        someone is checking it in a non-mdp context; accessor makes
        the intent clearer (`theme.Resolve("auto").IsAuto()`).
        If renaming, do it here — last chance before semver
- [ ] If any field changes from field to method, update
      `internal/server/server.go` call sites and the affected
      `pkg/theme` tests
- [ ] Update `pkg/theme/example_test.go` to use whatever shape
      survives the audit

**Documentation and isolation proofs**

- [ ] Run `go test ./pkg/...` — all examples compile and run cleanly
- [ ] Run `go vet ./pkg/...` — no issues
- [ ] Confirm with
      `go doc github.com/donaldgifford/mdp/pkg/parser`
      (and theme, livereload) that the package docs render as
      expected on the command line

**PR**

- [ ] Open PR with `patch` label (the only user-facing API change is
      additive: `WithMermaidRenderMode` and possibly the `IsAuto`
      accessor rename)
- [ ] PR title: `feat: harden public pkg API for v0.2.0`
- [ ] PR body references RFC-0001 phase 3 acceptance criteria

#### Success Criteria

- `make build && make test && make lint && make test-race` all green
- `go test ./pkg/...` runs and all `Example*` tests pass
- Each `example_test.go` imports *only* its own package + stdlib
  (compile-time enforcement of isolation; the test file fails to
  build if a cross-`pkg/` import is added later)
- `go doc github.com/donaldgifford/mdp/pkg/parser` (and theme,
  livereload) shows readable package docs with at least one example
- `theme.Theme` field audit is documented (which fields stayed,
  which became accessors, and why — captured in the PR body)
- No regressions in `mdp --file` or Neovim plugin behavior

---

### Phase 4: Tag v0.2.0

**Branch:** `chore/release-v0.2.0`
**Estimated diff:** 2 files (README.md, possibly cliff.toml tweak).
**Depends on:** Phase 3 merged to `main`.

#### Tasks

**README.md Library section**

- [ ] Add a `## Library` section (probably after `## Install`,
      before `## Development`) describing the public packages
- [ ] Worked example 1 — parser-only (10 lines):
      `import "github.com/donaldgifford/mdp/pkg/parser"`,
      `p := parser.New()`, `html, _ := p.Render([]byte("# Hi"))`,
      print
- [ ] Worked example 2 — parser + theme + livereload (docz `serve`
      shape, ~25 lines): set up a `Hub`, wire a stdlib
      `http.ServeMux` that renders some markdown via `parser` +
      `theme`, wrap with `livereload.WrapHandler`, start server.
      Mention this is the shape `docz serve` uses
- [ ] Link the worked examples to the corresponding
      `example_test.go` files so they stay in sync (Go's `go test
      -run Example` will catch drift if the README examples are
      updated to match the test files verbatim)

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
- `README.md` has a `## Library` section with both worked examples

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

Implementation-specific questions to resolve before phase 3 starts.

1. **`theme.Theme.IsAuto` → `Theme.IsAuto()` accessor rename.**
   Recommendation in the task list is to rename to a method before
   v0.2.0 freezes the API. Confirm — or keep as a field if you
   prefer field syntax at call sites. The same question applies
   (less urgently) to `HljsVendorCSS`. If kept as a field, the
   audit task simply documents the decision rather than refactoring.

2. **Whether to include a `pkg/livereload` example that broadcasts a
   typed message** vs. only the opaque-bytes example. A typed
   example would show consumers what the docz wire shape looks like
   in practice, but risks implying that `pkg/livereload` cares about
   the bytes (it doesn't). Recommendation: opaque only; let docz's
   own examples (in the docz repo) cover the typed case.

3. **README "Library" section depth.** Two worked examples (per the
   tasks above) is a reasonable starting point. Consider also
   adding a one-line "use cases" table (static site, lambda,
   docz-style preview) so prospective consumers can self-route to
   the right composition. Optional — defer if it adds churn.

4. **Whether the `cliff.toml` group regexps need updating** for the
   new `pkg/` work. The current regexps in `.cliff.toml` (or similar)
   probably already capture `feat:` prefixes; worth a dry-run
   before phase 4 PR to confirm.

5. **`replace` directive cleanup in docz.** After v0.2.0 tags,
   docz's `replace github.com/donaldgifford/mdp => ../mdp` should be
   removed and replaced with a real `require` on `v0.2.0`. This is
   technically docz-repo work, but worth flagging as a follow-up so
   it doesn't get forgotten and leak through to a docz release.

6. **Coverage thresholds for the new packages.** CLAUDE.md cites a
   60% target and 40% minimum overall. After phase 3 lands the
   `example_test.go` files, coverage for `pkg/parser` and `pkg/theme`
   should naturally cross 70%+. `pkg/livereload` was set to ≥70%
   in IMPL-0003 — confirm this target survives the phase 3
   additions and reconcile against the per-package `.golangci.yml`
   or `.codecov.yml` configs if they exist.

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
