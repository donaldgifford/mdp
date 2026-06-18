---
id: INV-0003
title: "callout extension race in TestRender_GitHubCallout"
status: Concluded
author: Donald Gifford
created: 2026-06-18
---
<!-- markdownlint-disable-file MD025 MD041 -->

# INV 0003: callout extension race in TestRender_GitHubCallout

**Status:** Concluded
**Author:** Donald Gifford
**Date:** 2026-06-18

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
- [Approach](#approach)
- [Environment](#environment)
- [Findings](#findings)
  - [Observation 1 — the panic](#observation-1--the-panic)
  - [Observation 2 — the shared mutable state](#observation-2--the-shared-mutable-state)
  - [Observation 3 — concurrency exposure](#observation-3--concurrency-exposure)
- [Conclusion](#conclusion)
- [Recommendation](#recommendation)
- [References](#references)
<!--toc:end-->

## Question

Why does `TestRender_GitHubCallout` (and its siblings under
`pkg/parser`) intermittently race under `go test -race`, sometimes
producing `panic: runtime error: slice bounds out of range [9:4]`
inside the `gm-alert-callouts` extension, and what is the smallest
correct fix on mdp's side?

## Hypothesis

The `gm-alert-callouts` extension caches a single
`golang.org/x/text/cases.Caser` on its renderer struct and reuses
it for every callout header. `cases.Caser` is **not** safe for
concurrent use — calling `Caser.String` from multiple goroutines
mutates internal `transform.Transformer` state and trips the
`slice bounds out of range` panic we see in CI. The race is in
the upstream extension, not in mdp's `pkg/parser`.

## Context

`TestRender_GitHubCallout` started flaking on PR CI runs during
the RFC-0001 hardening pass (IMPL-0004). The same job passes on
rerun, which is the classic shape of a data race. We need to
decide whether to:

1. Patch / wait on upstream `gm-alert-callouts`.
2. Serialize calls in `pkg/parser.Parser.Render` ourselves.
3. Stop sharing one `*Parser` across goroutines and document that.

**Triggered by:** [RFC-0001](../rfc/0001-public-mdp-go-library.md),
[IMPL-0004](../impl/0004-phase-3-4-of-rfc-0001-harden-public-api-and-tag-v020.md)

## Approach

1. Force the race by running the affected tests with
   `-race -count=5` and capture the full stack.
2. Walk the stack from the panic site back to mdp code to identify
   which third-party package owns the unsafe access.
3. Read the upstream source at the indicated file/line to confirm
   the shared mutable field.
4. Cross-reference `golang.org/x/text/cases` documentation to
   confirm `cases.Caser` is not safe for concurrent use.
5. Evaluate fix options against mdp's public API contract.

## Environment

| Component | Version / Value |
|-----------|-----------------|
| Go        | 1.24 (project `mise.toml`) |
| OS        | macOS (Darwin 25.5.0) |
| `pkg/parser` | post-IMPL-0004 (v0.2.0) |
| `github.com/zmtcreative/gm-alert-callouts` | v0.8.0 |
| `golang.org/x/text` | v0.27.0 |
| Test     | `go test -race -count=5 -run 'TestRender_GitHubCallout' ./pkg/parser/` |

## Findings

### Observation 1 — the panic

`go test -race -count=5` against the callout tests reliably trips
both the race detector and a runtime panic:

```
testing.go:1712: race detected during execution of test
panic: runtime error: slice bounds out of range [9:4] [recovered, repanicked]

goroutine ... [running]:
golang.org/x/text/transform.String(...)
    .../golang.org/x/text@v0.27.0/transform/transform.go:650 +0x8b4
golang.org/x/text/cases.Caser.String(...)
    .../golang.org/x/text@v0.27.0/cases/cases.go:51
github.com/zmtcreative/gm-alert-callouts/internal/renderer.(*AlertsHeaderHTMLRenderer).renderAlertsHeader(0xc0002fe9c0, ...)
    .../gm-alert-callouts@v0.8.0/internal/renderer/header.go:151 +0x52c
```

The crash is deterministic in shape (`[9:4]` style slice-bounds
panic from `transform.String`), which is the signature of two
goroutines stepping on the same `Transformer` buffer indices.

### Observation 2 — the shared mutable state

`gm-alert-callouts@v0.8.0/internal/renderer/header.go`:

```go
// line 29 — field on the renderer struct
type AlertsHeaderHTMLRenderer struct {
    ...
    titleCaser cases.Caser
}

// line 68 — built once at constructor time
titleCaser: cases.Title(tag, cases.Compact),

// line 151 — used from the renderAlertsHeader hot path
startHTML += r.titleCaser.String(kind)
```

A single `cases.Caser` value is created when goldmark builds the
extension and is then called from every callout header render.
The `golang.org/x/text/cases` package documents that `Caser`
values **wrap a stateful `transform.Transformer`** and are not
safe for concurrent use without external synchronization.

### Observation 3 — concurrency exposure

`pkg/parser.Parser.Render` does not synchronize calls; the public
API doc explicitly invites concurrent use ("a `*Parser` is safe
to share across goroutines once constructed"). The race surfaces
in tests because `t.Parallel()` is enabled across the callout
table and `-count=5` multiplies the parallelism, but the same
exposure exists for any embedder that fans `Render` out across
goroutines (which is the documented usage pattern).

## Conclusion

**Answer:** Confirmed — the race lives in
`github.com/zmtcreative/gm-alert-callouts@v0.8.0`, specifically a
shared `cases.Caser` on `AlertsHeaderHTMLRenderer` that is called
without synchronization. It is not a bug in mdp's `pkg/parser`,
but mdp's documented "concurrency-safe `*Parser`" contract is
violated whenever GitHub-style callouts are rendered concurrently.

## Recommendation

Short-term (this repo, no upstream wait):

1. Open an issue against `zmtcreative/gm-alert-callouts` with the
   captured stack and a minimal reproducer (`cases.Caser` field
   needs to be either rebuilt per-call or guarded by a `sync.Mutex`,
   ideally the former since `cases.Title` is cheap).
2. Until upstream ships a fix, guard the call inside
   `pkg/parser.Parser.Render` with a per-Parser `sync.Mutex` so
   our public API holds its concurrency contract. Document the
   serialization in `doc.go` with a `// coverage:` note pointing
   back to this INV.
3. Add a focused regression test (`TestParser_ConcurrentRender_NoRace`)
   that fans `Render` out over N goroutines on callout-heavy input
   so we catch upstream regressions immediately.

Longer-term (after upstream fix):

4. Bump `gm-alert-callouts`, remove the mutex, keep the regression
   test, and close this INV with a follow-up doc/commit reference.

## References

- [RFC-0001](../rfc/0001-public-mdp-go-library.md) — public Go library contract
- [IMPL-0004](../impl/0004-phase-3-4-of-rfc-0001-harden-public-api-and-tag-v020.md) — hardening pass that surfaced the flake
- [IMPL-0005](../impl/0005-local-mitigation-for-inv-0003-callout-extension-race.md) — local mitigation (per-`Parser` mutex)
- [INV-0003 fix plan](./0003-fix-plan-upstream-callout-race.md) — upstream fork/PR playbook
- `gm-alert-callouts@v0.8.0/internal/renderer/header.go:29,68,151` — shared `titleCaser` field
- `golang.org/x/text@v0.27.0/cases/cases.go` — `cases.Caser` docs
- `golang.org/x/text@v0.27.0/transform/transform.go:650` — panic site
- Upstream repo: <https://github.com/zmtcreative/gm-alert-callouts>
