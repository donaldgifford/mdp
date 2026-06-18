<!-- markdownlint-disable-file MD025 MD041 -->

# INV-0003 — Supplementary Fix Plan: Patch `gm-alert-callouts` Upstream

This is a companion to
[INV-0003](./0003-callout-extension-race-in-testrendergithubcallout.md).
The investigation concluded that the race lives in
`github.com/zmtcreative/gm-alert-callouts@v0.8.0`. This document is
the playbook for fixing it upstream and validating that the fix
resolves our flake — by pointing mdp's `go.mod` at the patched fork
end-to-end before the upstream PR merges.

## Goal

1. Land the fix where it belongs (upstream).
2. Prove the fix actually resolves the `TestRender_GitHubCallout`
   race in mdp by consuming the patched fork via a `replace`
   directive.
3. Switch mdp back to a tagged upstream release once the PR merges.

## The Fix

`internal/renderer/header.go:29,68,151` stores a single
`cases.Caser` on the renderer struct and calls `.String()` on it
from every callout header render. `cases.Caser` wraps a stateful
`golang.org/x/text/transform.Transformer` and is not safe for
concurrent use.

**Recommended fix (Option A — rebuild per call):** keep the
`language.Tag` on the struct, not the `Caser`. `cases.Title(tag, cases.Compact)`
returns a value-typed `Caser` and is cheap enough to build on the
hot path, and it eliminates the shared mutable state entirely.

```go
// header.go — before
type AlertsHeaderHTMLRenderer struct {
    ...
    titleCaser cases.Caser
}

// constructor
titleCaser: cases.Title(tag, cases.Compact),

// renderAlertsHeader
startHTML += r.titleCaser.String(kind)
```

```go
// header.go — after
type AlertsHeaderHTMLRenderer struct {
    ...
    titleTag language.Tag
}

// constructor
titleTag: tag,

// renderAlertsHeader
startHTML += cases.Title(r.titleTag, cases.Compact).String(kind)
```

**Fallback fix (Option B — mutex):** add `sync.Mutex` to the
struct and `Lock`/`Unlock` around the `String` call. Correct, but
introduces contention; only use this if upstream pushes back on
Option A for some perf reason.

## Playbook

### 1. Fork the upstream repo

```bash
gh repo fork zmtcreative/gm-alert-callouts --clone --remote
cd gm-alert-callouts
git checkout -b fix/caser-not-concurrency-safe
```

`gh repo fork` does three things in one command: creates the fork
under your account, clones it locally, and sets `upstream` to
`zmtcreative/gm-alert-callouts` so you can rebase against their
`main` later.

### 2. Apply the fix

Edit `internal/renderer/header.go` per Option A above. Required
imports: keep `golang.org/x/text/cases`, keep
`golang.org/x/text/language`. Remove the `titleCaser` field, add
`titleTag language.Tag`, update the constructor, update the call
site.

### 3. Add a regression test upstream

In `internal/renderer/header_test.go`, add a parallel test that
fans the renderer out across goroutines on input that triggers
`renderAlertsHeader`. Run it under `-race -count=10`:

```bash
go test -race -count=10 -run TestRenderAlertsHeader_Concurrent ./...
```

The test must fail on `main` and pass on the fix branch — that's
the evidence the upstream maintainer needs.

### 4. Verify locally against mdp via `replace`

Push the fix branch to your fork:

```bash
git push -u origin fix/caser-not-concurrency-safe
```

Then in **mdp**, point `go.mod` at the fork:

```bash
cd /path/to/mdp
go mod edit -replace=github.com/zmtcreative/gm-alert-callouts=github.com/<your-gh-handle>/gm-alert-callouts@fix/caser-not-concurrency-safe
go mod tidy
```

`go mod tidy` will resolve the branch to a pseudo-version
(`v0.8.1-0.YYYYMMDDhhmmss-<sha>`). Confirm the entries landed:

```bash
grep gm-alert-callouts go.mod
```

You should see both a `require` line on the pseudo-version and a
`replace` line pointing at your fork.

### 5. Reproduce the flake on baseline, confirm it disappears

```bash
# Sanity check: baseline (no replace) still flakes
git stash  # stash the go.mod / go.sum changes
go test -race -count=20 -run 'TestRender_GitHubCallout' ./pkg/parser/
# expect: race + slice-bounds panic within ~20 runs

git stash pop  # reapply the replace
go test -race -count=20 -run 'TestRender_GitHubCallout' ./pkg/parser/
# expect: PASS, no race, no panic
```

If step 2 still races, the upstream fix is incomplete — go back
to step 2 before opening the PR.

### 6. Open the upstream PR

```bash
cd /path/to/gm-alert-callouts
gh pr create --repo zmtcreative/gm-alert-callouts \
  --title "fix: rebuild cases.Caser per call to avoid concurrent-use panic" \
  --body "$(cat <<'EOF'
## Summary
`AlertsHeaderHTMLRenderer.titleCaser` is shared across goroutines
when consumers fan goldmark `Render` out across goroutines.
`golang.org/x/text/cases.Caser` wraps a stateful
`transform.Transformer` and is not safe for concurrent use, which
produces `panic: runtime error: slice bounds out of range` from
`transform.String` under `-race`.

This PR moves the `Caser` construction onto the hot path
(`cases.Title(tag, cases.Compact)` is cheap) and stores only the
`language.Tag` on the renderer, eliminating the shared mutable
state.

## Reproducer
See `TestRenderAlertsHeader_Concurrent` (added in this PR). Fails
on `main`, passes here.

## Downstream context
Surfaced via the mdp project — see their investigation doc:
https://github.com/donaldgifford/mdp/blob/main/docs/investigation/0003-callout-extension-race-in-testrendergithubcallout.md
EOF
)"
```

### 7. Keep mdp on the fork until upstream merges

The `replace` directive in mdp's `go.mod` is the bridge. CI runs
green against the fork; we don't ship a workaround in `pkg/parser`.
The replace stays in place until step 8.

### 8. After upstream merges and tags

```bash
cd /path/to/mdp
go mod edit -dropreplace=github.com/zmtcreative/gm-alert-callouts
go get github.com/zmtcreative/gm-alert-callouts@vX.Y.Z   # the new tag
go mod tidy
go test -race -count=20 ./pkg/parser/
```

Update [INV-0003](./0003-callout-extension-race-in-testrendergithubcallout.md)
status from `In Progress` to `Concluded`, note the upstream PR
and the released version in **References**, and remove this fix
plan (or leave it as historical record — preference).

## Decision Log

- **Why not just add a mutex in mdp?** Hides an upstream bug,
  silently turns our documented concurrency-safe `*Parser` into a
  serialization point for callout-heavy input, and leaves every
  other goldmark consumer of `gm-alert-callouts` exposed.
- **Why Option A (rebuild) over Option B (mutex) upstream?**
  Zero ongoing contention; the `cases.Title` constructor is cheap
  enough that per-call allocation is invisible in benchmarks.
  Mutex is correct but worse on contention-heavy workloads.
- **Why `replace` instead of `go.work`?** `replace` ships in
  `go.mod`, so CI picks it up automatically without a separate
  workspace file. `go.work` is for local dev only and would let
  CI keep using the broken upstream.
