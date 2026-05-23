---
id: INV-0002
title: "Package mdp for distribution"
status: Open
author: Donald Gifford
created: 2026-05-23
---
<!-- markdownlint-disable-file MD025 MD041 -->

# INV 0002: Package mdp for distribution

**Status:** Open
**Author:** Donald Gifford
**Date:** 2026-05-23

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
- [Approach](#approach)
- [Environment](#environment)
- [Findings](#findings)
  - [Current distribution surface](#current-distribution-surface)
  - [Gap analysis](#gap-analysis)
  - [Option matrix](#option-matrix)
  - [Notes per option](#notes-per-option)
- [Conclusion](#conclusion)
- [Recommendation](#recommendation)
- [References](#references)
<!--toc:end-->

## Question

Which packaging channels should `mdp` ship through so that non-Neovim users
(and Neovim users on platforms or workflows that don't suit the lazy.nvim
`build.lua` path) can install it with a single command idiomatic to their
environment — and what is the smallest, lowest-maintenance set of additions
to reach "good enough" coverage?

## Hypothesis

Most of the value comes from three additions on top of today's GoReleaser
tarballs:

1. A **Homebrew tap** (`donaldgifford/homebrew-tap`) wired through GoReleaser's
   `brews:` block — the README already advertises `brew install
   donaldgifford/tap/mdp`, so this is the most visible gap.
2. **Windows binaries** in the GoReleaser build matrix — the install script
   already detects Windows, but no artifact exists for it to download.
3. **Linux native packages** (`.deb` / `.rpm` / `.apk`) via GoReleaser's
   `nfpms:` block — cheap to add, broad reach, no extra registries to claim.

A **container image** and **Mason.nvim registry entry** are likely
nice-to-haves rather than must-haves; Nix/AUR/Scoop/Winget can be left to
community submission.

## Context

Today's installation flows assume either (a) a Neovim user who lets
`lua/mdp` and `build.lua` handle binary acquisition, or (b) a manual
GitHub-release download. The README advertises `brew install
donaldgifford/tap/mdp` but the tap has never been set up, so the
instruction is misleading. As `mdp` is otherwise a generic CLI markdown
preview server (it can run standalone via `--file` and watch via fsnotify),
there is value in making it installable through the channels people
already use for CLIs.

**Triggered by:** project-level desire to broaden audience beyond the
Neovim plugin path. No parent RFC/DESIGN exists yet — this investigation
is the input for one.

## Approach

1. Inventory current release surface (GoReleaser config, CI, install scripts,
   README claims).
2. Enumerate plausible packaging channels and score each on:
   reach, integration cost in `.goreleaser.yml`, ongoing maintenance,
   external account/registry requirements.
3. Identify which channels GoReleaser can produce as a side effect of an
   existing release run (no new CI jobs) vs. which require external
   infrastructure (taps, AUR keys, registry accounts).
4. Recommend a tiered rollout (must-have, nice-to-have, defer-to-community).

## Environment

| Component | Version / Value |
|-----------|----------------|
| GoReleaser | v2 (`.goreleaser.yml` `version: 2`) |
| Build targets today | `linux`, `darwin` × `amd64`, `arm64` |
| Archive format | `tar.gz`, name `mdp_<os>_<arch>` |
| Signing | GPG-signed `checksums.txt` |
| SBOM | Syft, per-archive SPDX JSON |
| Go module path | `github.com/donaldgifford/mdp` |
| `go install` works today | Yes — entrypoint is `./cmd/mdp` |
| Neovim install path | `lazy.nvim` -> `build.lua` -> binary at `<plugin>/bin/mdp` |

## Findings

### Current distribution surface

| Channel | Status | Source of truth |
|---------|--------|-----------------|
| GitHub Releases (tarballs) | Working | `.goreleaser.yml` `archives:` |
| GPG-signed checksums | Working | `.goreleaser.yml` `signs:` |
| SBOM (SPDX) | Working | `.goreleaser.yml` `sboms:` |
| `go install ...@latest` | Working (package-path-based) | `cmd/mdp/main.go` + `go.mod` |
| Neovim via lazy.nvim | Working | `lazy.lua`, `build.lua`, `lua/mdp/init.lua` |
| Manual shell installer | Working | `scripts/install.sh` |
| **Homebrew tap** | **Advertised in README, not implemented** | n/a |
| **Windows binaries** | **Detected in installers, not built** | `.goreleaser.yml` (no `windows`) |
| Linux native packages (deb/rpm/apk) | Not built | n/a |
| Container image | Not built | n/a |
| Mason.nvim registry | Not submitted | n/a |
| Nix / AUR / Scoop / Winget / Chocolatey | Not submitted | n/a |

### Gap analysis

1. **`README.md:200-201`** says `brew install donaldgifford/tap/mdp`. The
   tap doesn't exist. Either set it up or remove the line — the current
   state misleads users.
2. **`scripts/install.sh:23-35`** detects `MINGW*|MSYS*|CYGWIN*` and prints
   `windows`, but `.goreleaser.yml:8-10` only builds `linux` and `darwin`.
   The Windows branch will always 404 on download. Same for `build.lua`
   (Lua side returns `nil` for non-darwin/linux, so it falls back to
   `go build`, which is acceptable — but a real Windows release would
   close the gap).
3. **No native Linux packages.** GoReleaser's `nfpms:` block produces
   `.deb`/`.rpm`/`.apk` in the same release run, no new infrastructure.
4. **No container image.** GoReleaser's `dockers:` block plus a thin
   `Dockerfile` could publish to `ghcr.io/donaldgifford/mdp` on every
   release. Probably low-value for a CLI preview server, but cheap.
5. **No Mason.nvim entry.** Neovim users frequently use Mason to install
   non-Neovim CLI deps. Submission is a PR to
   `mason-org/mason-registry`.
6. **No Nix expression.** Real demand-signal-driven — defer.

### Option matrix

Scoring legend: **L**ow / **M**edium / **H**igh.

| Channel | Reach | Setup cost | Maintenance | External deps | Verdict |
|---------|-------|-----------|-------------|---------------|---------|
| Homebrew tap | H (macOS + Linux brew users) | L (one tap repo + goreleaser block) | L (auto-bumped per release) | New repo `homebrew-tap` + `HOMEBREW_TAP_GITHUB_TOKEN` | **Must-have** — already advertised |
| Windows binaries | M | L (`goos: [windows]` + `.zip` archive) | L | None | **Must-have** — closes installer gap |
| nfpms (deb/rpm/apk) | M-H | L (one block in `.goreleaser.yml`) | L | None | **Should-have** — same release run, broad reach |
| Container image (`ghcr.io`) | L-M | M (Dockerfile + `dockers:` + multi-arch buildx) | L | `GHCR` already available in workflow | Nice-to-have, defer until demand |
| Mason.nvim registry | M (targeted at Neovim crowd) | L (single PR upstream) | L (rare bumps) | mason-org PR | **Should-have** — high audience fit |
| Scoop bucket | L-M | L (goreleaser `scoops:` + bucket repo) | L | New repo `scoop-bucket` + token | Defer — bundle with Homebrew work if Windows lands |
| Winget | L-M (Windows official) | M (manifest PR per release, automatable via winget-create) | M | winget-pkgs PR | Defer |
| Chocolatey | L | M (nuspec + API key) | M | Choco account | Defer |
| AUR | L | L (PKGBUILD repo) | M (manual bumps unless automated) | AUR account + SSH key | Defer to community submission |
| Snap | L | M (snapcraft.yaml + store account) | M | Snapcraft store | Defer |
| Nix flake / nixpkgs | L | M (flake) / H (nixpkgs PR) | L (flake) / M (nixpkgs) | None for flake | Defer to community |

### Notes per option

**Homebrew tap.** GoReleaser's `brews:` block writes a formula directly
into a `homebrew-tap` repo on each release. Requirements: create
`donaldgifford/homebrew-tap`, add a fine-grained PAT with `contents:write`
on that repo as `HOMEBREW_TAP_GITHUB_TOKEN` (or reuse a PAT secret), and
add a `brews:` entry to `.goreleaser.yml`. Formula tests can be as
simple as `system "#{bin}/mdp", "--version"`.

**Windows.** Add `windows` to `builds[0].goos`, add a Windows-specific
archive entry (`format_overrides: [{ goos: windows, format: zip }]`), and
the existing `scripts/install.sh` and `build.lua` Windows paths become
real. The Lua side will need a small patch to recognize `windows_amd64`
and append `.exe`.

**nfpms.** GoReleaser produces deb/rpm/apk from the same compiled binaries
— no extra build matrix. Distros aren't claimed (these go into GitHub
Releases, not into Debian/Fedora/Alpine archives), but users can
`dpkg -i mdp_*.deb` directly. Add `nfpms:` with `formats: [deb, rpm, apk]`,
`maintainer: "Donald Gifford <...>"`, `description`, `license: MIT`.

**Container image.** Would publish to `ghcr.io/donaldgifford/mdp:<tag>`
and `:latest`. Multi-arch requires `dockers:` × `docker_manifests:`. The
release server only serves rendered HTML over HTTP; running it in a
container is unusual (the typical use case binds to localhost from a
Neovim instance on the same host). Low priority.

**Mason.nvim.** Neovim users frequently install CLI tools via
`:MasonInstall mdp`. Submission is one PR to `mason-org/mason-registry`
with a package definition pointing at GitHub releases. Once merged,
Mason handles version detection automatically.

**Distro-native packages (AUR, Debian, Fedora, Nix, Alpine, openSUSE).**
Each has a different submission process and ongoing maintenance burden.
For a small project, the right move is to ship good tarballs/deb/rpm in
GitHub Releases and let community packagers submit (they will, if there's
demand). Premature distro packaging is a maintenance trap.

## Conclusion

**Answer:** Partial — the investigation surfaces a concrete, tiered plan
rather than a single yes/no.

The hypothesis holds: Homebrew tap, Windows binaries, and nfpms (deb/rpm/apk)
are the three highest-leverage additions, all delivered through a single
`.goreleaser.yml` extension on the existing release pipeline. Mason.nvim
deserves promotion from "nice-to-have" to "should-have" given mdp's
Neovim-adjacent audience. Container image and other Windows package
managers should wait for demand signals.

## Recommendation

Promote this investigation into an **RFC** (e.g., `RFC-XXXX: Broaden mdp
distribution channels`) or a focused **PLAN** with the following phases:

1. **Phase 1 — Fix advertised channels** (must-have):
   - Create `donaldgifford/homebrew-tap` repo.
   - Add `brews:` block to `.goreleaser.yml`; wire `HOMEBREW_TAP_GITHUB_TOKEN`
     in release workflow.
   - Add `windows` to `goos`, plus `format_overrides` for `.zip`.
   - Patch `build.lua` to handle `windows_amd64` and `.exe` suffix.
   - Update `README.md:200-205` to reflect actual install paths.

2. **Phase 2 — Broaden Linux reach** (should-have):
   - Add `nfpms:` block (deb/rpm/apk) to `.goreleaser.yml`.
   - Verify artifacts attach to GitHub Releases.

3. **Phase 3 — Neovim ecosystem reach** (should-have):
   - Submit Mason registry PR.

4. **Phase 4 — Defer until demand**: container image, Scoop, Winget,
   Chocolatey, AUR, Nix.

A follow-up **ADR** may be useful to record the *decision to not* pursue
distro-native packaging in-tree (sets community expectations).

## References

- `.goreleaser.yml` — current release config
- `.github/workflows/release.yml` — release workflow
- `scripts/install.sh:23-35` — Windows detection that has no artifact today
- `build.lua` — Neovim plugin install path
- `lazy.lua`, `lua/mdp/init.lua` — Neovim plugin spec & binary resolution
- `README.md:186-205` — install section (Homebrew claim is currently aspirational)
- GoReleaser docs — `brews:`, `nfpms:`, `dockers:`, format_overrides
- Mason.nvim registry — `mason-org/mason-registry` (GitHub)
