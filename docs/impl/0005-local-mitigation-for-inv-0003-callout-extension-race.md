---
id: IMPL-0005
title: "local mitigation for INV-0003 callout extension race"
status: Draft
author: Donald Gifford
created: 2026-06-18
---
<!-- markdownlint-disable-file MD025 MD041 -->

# IMPL 0005: local mitigation for INV-0003 callout extension race

**Status:** Draft
**Author:** Donald Gifford
**Date:** 2026-06-18

<!--toc:start-->
- [Objective](#objective)
- [Scope](#scope)
  - [In Scope](#in-scope)
  - [Out of Scope](#out-of-scope)
- [Implementation Phases](#implementation-phases)
  - [Phase 1: Reproduce & lock in the failure mode](#phase-1-reproduce--lock-in-the-failure-mode)
    - [Tasks](#tasks)
    - [Success Criteria](#success-criteria)
  - [Phase 2: Implement the local guard in pkg/parser](#phase-2-implement-the-local-guard-in-pkgparser)
    - [Tasks](#tasks-1)
    - [Success Criteria](#success-criteria-1)
  - [Phase 3: Document the constraint](#phase-3-document-the-constraint)
    - [Tasks](#tasks-2)
    - [Success Criteria](#success-criteria-2)
  - [Phase 4: Benchmark and verify no regression](#phase-4-benchmark-and-verify-no-regression)
    - [Tasks](#tasks-3)
    - [Success Criteria](#success-criteria-3)
  - [Phase 5: Ship](#phase-5-ship)
    - [Tasks](#tasks-4)
    - [Success Criteria](#success-criteria-4)
- [File Changes](#file-changes)
- [Testing Plan](#testing-plan)
- [Rollback / Removal Plan](#rollback--removal-plan)
- [Open Questions](#open-questions)
  - [1. Mutex scope: always lock, or only when callouts are enabled?](#1-mutex-scope-always-lock-or-only-when-callouts-are-enabled)
  - [2. Lock granularity: whole Render call, or wrap only the gm-alert-callouts renderer?](#2-lock-granularity-whole-render-call-or-wrap-only-the-gm-alert-callouts-renderer)
  - [3. Should we add benchmarks (Phase 4) at all?](#3-should-we-add-benchmarks-phase-4-at-all)
  - [4. Concurrent-render test parameters (goroutines × iterations)?](#4-concurrent-render-test-parameters-goroutines--iterations)
  - [5. Commit / PR shape?](#5-commit--pr-shape)
  - [6. Should the regression test stay after the upstream fix?](#6-should-the-regression-test-stay-after-the-upstream-fix)
  - [7. Should we file the upstream issue before, after, or in parallel with this IMPL's PR?](#7-should-we-file-the-upstream-issue-before-after-or-in-parallel-with-this-impls-pr)
- [References](#references)
<!--toc:end-->

## Objective

Stop the `TestRender_GitHubCallout` race in CI **without** waiting for
an upstream `gm-alert-callouts` release, while preserving the
documented "`*Parser` is safe to share across goroutines"
contract. This is a deliberately scoped, fully reversible mitigation
that lives in `pkg/parser` until the upstream fix lands per the
[INV-0003 fix plan](../investigation/0003-fix-plan-upstream-callout-race.md).

**Implements:** [INV-0003](../investigation/0003-callout-extension-race-in-testrendergithubcallout.md)

## Scope

### In Scope

- A per-`Parser` serialization guard inside `pkg/parser.Parser.Render`
  so the shared `cases.Caser` in `gm-alert-callouts@v0.8.0` can no
  longer be hit concurrently.
- A focused regression test (`TestParser_ConcurrentRender_NoRace`)
  that fans `Render` out over N goroutines on callout-heavy input
  and runs cleanly under `-race`.
- A small benchmark (`BenchmarkRender` / `BenchmarkRenderParallel`)
  to quantify the overhead the guard adds.
- Doc updates: `pkg/parser/doc.go` note about serialization,
  CLAUDE.md note about the mutex, INV-0003 link to this IMPL.

### Out of Scope

- Forking `zmtcreative/gm-alert-callouts` and opening the upstream
  PR — that's the [INV-0003 fix plan](../investigation/0003-fix-plan-upstream-callout-race.md).
- Removing the local guard — that's tracked separately and will
  happen after upstream tags a fixed release.
- Reworking goldmark extension wiring (e.g., wrapping the callout
  renderer in our own NodeRenderer).
- Touching `internal/server` — the server only constructs one
  `*Parser` at startup, and the race surfaces in the public API,
  not the CLI path.

## Implementation Phases

Each phase builds on the previous one. A phase is complete when all
its tasks are checked off and its success criteria are met.

---

### Phase 1: Reproduce & lock in the failure mode

Before fixing anything, lock in a test that *fails* under `-race`
on `main`. This is the evidence the fix works and the regression
gate against any future change that drops the guard prematurely.

#### Tasks

- [ ] Confirm `github.com/zmtcreative/gm-alert-callouts` is still at
      `v0.8.0` (`go list -m -versions ...`); if a new version exists,
      check its changelog before continuing.
- [ ] Add `TestParser_ConcurrentRender_NoRace` to
      `pkg/parser/parser_test.go`:
      one shared `*Parser`, callout-heavy markdown, fan out to ~32
      goroutines × ~16 iterations, `sync.WaitGroup` join, fail on
      any non-nil error.
- [ ] Run the new test on a stock checkout (before any fix) with
      `go test -race -count=5 -run TestParser_ConcurrentRender_NoRace ./pkg/parser/`
      and confirm it reliably reports a race / panic.
- [ ] Capture a one-paragraph note in the PR description with the
      stack trace snippet (so reviewers can see "this fails before
      → passes after").

#### Success Criteria

- New `TestParser_ConcurrentRender_NoRace` exists and is checked
  in (no `t.Skip` guards).
- The test deterministically fails under `-race -count=5` on `main`
  (verified by running before applying Phase 2).
- All other existing tests in `pkg/parser` continue to pass.

---

### Phase 2: Implement the local guard in pkg/parser

Add the smallest possible serialization point that closes the race
without leaking implementation detail into the public API.

#### Tasks

- [ ] Add a `mu sync.Mutex` field to the `Parser` struct in
      `pkg/parser/parser.go`.
- [ ] Wrap the body of `Parser.Render` in `p.mu.Lock()` /
      `defer p.mu.Unlock()`.
- [ ] Add a single-line comment above the lock referencing
      INV-0003 and the upstream fix plan, e.g.:
      `// Serialize Convert until gm-alert-callouts ships a fix — see INV-0003.`
- [ ] Run `make fmt` and `make lint` — both must pass clean.
- [ ] Run `TestParser_ConcurrentRender_NoRace` with
      `go test -race -count=20 -run TestParser_ConcurrentRender_NoRace ./pkg/parser/`
      and confirm zero races and zero panics.
- [ ] Run the full `pkg/parser` race suite:
      `go test -race -count=5 ./pkg/parser/` — must be green.

#### Success Criteria

- `Parser.Render` is serialized per `*Parser` instance.
- Public API surface is unchanged (no new exported types,
  fields, or methods).
- `make lint` is clean (gocritic / staticcheck / govet all pass).
- `TestParser_ConcurrentRender_NoRace` passes under `-race -count=20`.
- Full `pkg/parser` test suite passes under `-race -count=5`.

---

### Phase 3: Document the constraint

Make the temporary serialization discoverable so future-us doesn't
either (a) accidentally remove it before upstream fixes the bug, or
(b) forget to remove it after upstream ships.

#### Tasks

- [ ] Update `pkg/parser/doc.go` with a short note: "`Render` is
      currently serialized per `*Parser` to work around a known
      data race in `gm-alert-callouts@v0.8.0` (see INV-0003).
      Throughput-sensitive consumers should construct multiple
      `*Parser` instances."
- [ ] Update `CLAUDE.md`'s `pkg/parser` paragraph to mention the
      mutex and link INV-0003.
- [ ] Update INV-0003 status section: add an entry under
      **References** linking back to this IMPL doc.
- [ ] Update INV-0003 status: `In Progress` → `Concluded` after
      this IMPL merges (Phase 5).

#### Success Criteria

- `go doc github.com/donaldgifford/mdp/pkg/parser Parser.Render`
  surfaces the serialization caveat.
- CLAUDE.md, INV-0003, and IMPL-0005 cross-link cleanly.
- A future grep for `INV-0003` in source + docs returns every
  call site that needs cleanup when upstream merges.

---

### Phase 4: Benchmark and verify no regression

Quantify the mutex overhead so the "negligible" claim in Phase 3
isn't a guess.

#### Tasks

- [ ] Add `BenchmarkRender` to `pkg/parser/parser_test.go`:
      single-goroutine baseline rendering a representative document
      (mix of GFM table + code block + callout).
- [ ] Add `BenchmarkRenderParallel` using `b.RunParallel`:
      same document, all goroutines hitting the same `*Parser`.
- [ ] Run on `main` (pre-fix) and on the fix branch:
      `go test -bench=BenchmarkRender -benchmem -count=10 ./pkg/parser/ | tee bench.out`
      and capture both into the PR description.
- [ ] If single-goroutine `ns/op` regresses >5%, investigate before
      shipping; otherwise note the measured overhead in the PR body.

#### Success Criteria

- Both benchmarks compile and run.
- Single-goroutine `BenchmarkRender` shows <5% `ns/op` regression
  vs `main`.
- Parallel benchmark numbers are captured in the PR description as
  a reference point for the upstream-fix follow-up.

---

### Phase 5: Ship

#### Tasks

- [ ] Create branch `fix/parser-serialize-render-inv-0003` (if not
      already on it) off `main`.
- [ ] Stage and commit per Phase 1-4 with conventional commit messages.
      Suggested split: one commit per phase, OR a single squash-friendly
      commit titled `fix(parser): serialize Render to work around INV-0003 race`.
- [ ] Push the branch and open a PR targeting `main` with:
      the captured stack trace from Phase 1, the benchmark numbers
      from Phase 4, and an explicit "remove this when
      gm-alert-callouts ships a fix — see INV-0003 fix plan" note.
- [ ] Label the PR `patch` (per the project's semver-via-labels
      workflow).
- [ ] Merge once CI is green; the PR's merge commit closes the
      callout flake.

#### Success Criteria

- PR merged to `main` with green CI (lint + test, including `-race`).
- Reruns of `TestRender_GitHubCallout` no longer flake on subsequent
  PRs (sanity check on the next 2-3 unrelated PRs).
- INV-0003 status moved to `Concluded` with a pointer to the merged
  PR + this IMPL.

---

## File Changes

| File | Action | Description |
|------|--------|-------------|
| `pkg/parser/parser.go` | Modify | Add `mu sync.Mutex` field; wrap `Render` body in `Lock`/`Unlock`; add INV-0003 comment. |
| `pkg/parser/parser_test.go` | Modify | Add `TestParser_ConcurrentRender_NoRace`, `BenchmarkRender`, `BenchmarkRenderParallel`. |
| `pkg/parser/doc.go` | Modify | Document serialization caveat with link to INV-0003. |
| `CLAUDE.md` | Modify | Note mutex on `pkg/parser` paragraph; link INV-0003. |
| `docs/investigation/0003-callout-extension-race-in-testrendergithubcallout.md` | Modify | Add IMPL-0005 to References; move status to `Concluded` after merge. |

## Testing Plan

- **Pre-fix evidence:** `TestParser_ConcurrentRender_NoRace` must
  fail on `main` under `-race -count=5`. This is captured in the
  PR description (not a CI gate — the failing-on-`main` baseline
  is one-shot evidence, not a perpetual test).
- **Post-fix gate:** Same test passes on the fix branch under
  `-race -count=20`. CI runs `-race` once per push as it already
  does today; the regression test sits inside that run.
- **Full suite:** `make test` + `make lint` pass.
- **Benchmarks:** Captured in PR description; not a CI gate.

## Rollback / Removal Plan

When `gm-alert-callouts` ships a fixed release per the
[INV-0003 fix plan](../investigation/0003-fix-plan-upstream-callout-race.md):

1. Bump the dependency: `go get github.com/zmtcreative/gm-alert-callouts@vX.Y.Z`.
2. Remove the `mu sync.Mutex` field and `Lock`/`Unlock` in
   `Parser.Render`.
3. Remove the serialization note from `pkg/parser/doc.go`.
4. Remove the CLAUDE.md note.
5. **Keep** `TestParser_ConcurrentRender_NoRace` — it now guards
   against future upstream regressions.
6. Run `go test -race -count=20 ./pkg/parser/` — must stay green
   without the mutex.

## Open Questions

Each question lists `a` as the recommendation; alternatives follow.

### 1. Mutex scope: always lock, or only when callouts are enabled?

The `callouts` config flag is already plumbed; if a consumer
constructs `parser.New(parser.WithCallouts(false))`, there is no
race exposure and no need to serialize.

- **a (Recommended):** Always lock unconditionally. Mutex
  Lock/Unlock on an uncontended sync.Mutex is single-digit
  nanoseconds; the simplicity of "Render is always serialized"
  is worth more than the micro-savings for callouts-off users.
  Also future-proofs against another upstream extension having
  the same shape.
- **b:** Conditionally store the mutex only when `cfg.callouts`
  is true, and skip the lock when nil. Saves a nanosecond per
  call for callouts-off consumers but introduces a code path
  that is harder to test.
- **c:** No mutex at all; instead, document that `*Parser` is
  *not* concurrency-safe and let consumers wrap it themselves.
  Breaks the existing API contract.
- **Other:** _your answer_

### 2. Lock granularity: whole `Render` call, or wrap only the gm-alert-callouts renderer?

- **a (Recommended):** Lock the whole `Render` call (i.e., the
  `p.md.Convert` invocation). Minimal code, zero leaking of
  goldmark internals into our public surface, and the goldmark
  conversion is fast enough that whole-call serialization is
  fine for our workloads.
- **b:** Construct a custom `NodeRenderer` that wraps
  `alertcallouts.NewAlertCallouts(...)` and only locks inside
  the callout-rendering path. Surgically correct, but requires
  reimplementing extension wiring and tracking upstream's
  renderer signature.
- **c:** Use a `sync.Pool` of `goldmark.Markdown` instances per
  `*Parser` to allow parallel renders without sharing state.
  Most performant but biggest refactor; defeats the "smallest
  possible mitigation" objective.
- **Other:** _your answer_

### 3. Should we add benchmarks (Phase 4) at all?

- **a (Recommended):** Yes, add `BenchmarkRender` +
  `BenchmarkRenderParallel`. Cheap to write, and they give us
  concrete numbers to re-check after the upstream fix lands —
  i.e., they prove removing the mutex actually buys back
  parallelism, which is the whole point of removing it.
- **b:** Skip benchmarks; defer until the upstream-fix follow-up
  needs the data. Smaller PR, but means we ship the mitigation
  blind on perf.
- **Other:** _your answer_

### 4. Concurrent-render test parameters (goroutines × iterations)?

- **a (Recommended):** 32 goroutines × 16 iterations on callout-heavy
  input. Empirically reproduces the race in <1s in local runs and
  doesn't meaningfully extend CI wall-clock.
- **b:** 8 × 8. Faster, but may not reliably trigger pre-fix on
  slower CI runners.
- **c:** 128 × 64. Belt-and-suspenders confidence but adds
  noticeable CI time.
- **Other:** _your answer_

### 5. Commit / PR shape?

- **a (Recommended):** Single PR, single squash-friendly commit
  titled `fix(parser): serialize Render to work around INV-0003 race`.
  Reviewers see one diff with the test, the fix, and the docs
  together; revert is a single commit when upstream merges.
- **b:** Multiple commits on the same PR (one per phase). Easier
  to bisect the implementation steps, but the PR squashes anyway
  per the project's merge convention.
- **c:** Two PRs — first the failing regression test (red CI),
  then the fix (green CI). Cleanest "before / after" history but
  intentionally lands red CI for 1+ commits, which is noisy.
- **Other:** _your answer_

### 6. Should the regression test stay after the upstream fix?

- **a (Recommended):** Keep it permanently as an upstream-regression
  gate. If `gm-alert-callouts` (or any future extension) ever
  reintroduces the same class of bug, we catch it immediately.
  Cost is negligible — one extra test under `-race`.
- **b:** Delete it when the mutex comes out. The race is
  upstream's responsibility; keeping a downstream test for an
  upstream invariant duplicates the upstream's own coverage.
- **Other:** _your answer_

### 7. Should we file the upstream issue before, after, or in parallel with this IMPL's PR?

- **a (Recommended):** In parallel — file the upstream issue (not
  the fork PR yet) as soon as IMPL-0005 Phase 1 confirms the
  reproducer. Lets us link the upstream issue from our PR
  description and signals to upstream that there's a real
  downstream consumer asking for the fix.
- **b:** After our PR merges — keeps focus on shipping the local
  mitigation first.
- **c:** Skip filing an issue; jump straight to the fork PR per
  the INV-0003 fix plan.
- **Other:** _your answer_

## References

- [INV-0003](../investigation/0003-callout-extension-race-in-testrendergithubcallout.md) — investigation establishing the root cause
- [INV-0003 fix plan](../investigation/0003-fix-plan-upstream-callout-race.md) — upstream fork / PR playbook
- [RFC-0001](../rfc/0001-public-mdp-go-library.md) — public Go library contract being defended
- `pkg/parser/parser.go` — the `Render` method to be guarded
- `gm-alert-callouts@v0.8.0/internal/renderer/header.go:29,68,151` — shared `titleCaser` field (upstream root cause)
