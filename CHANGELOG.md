# Changelog

All notable changes to this project will be documented in this file.

## [0.3.0] - 2026-08-19

### Features

- **parser**: Footnote support per DESIGN-0003 (IMPL-0006) (#76)

## [0.2.2] - 2026-08-18

### Documentation

- Mark IMPL-0005 Completed and INV-0003 Concluded post-v0.2.1 (#66)

### Miscellaneous Tasks

- Deps cleanup and upgrade to 1.26.6 (#74)
- **release**: Update CHANGELOG.md for v0.2.2

## [0.2.1] - 2026-06-18

### Bug Fixes

- **parser**: Serialize Render to work around INV-0003 gm-alert-callouts race (#65)

### Documentation

- Finalize RFC-0001 doc statuses post-v0.2.0 release (#51)

### Miscellaneous Tasks

- **codecov**: Add per-component thresholds for public pkg/ (#52)
- **release**: Update CHANGELOG.md for v0.2.1

## [0.2.0] - 2026-05-25

### Documentation

- Add README ## Library section for v0.2.0 (#50)

### Miscellaneous Tasks

- **release**: Update CHANGELOG.md for v0.2.0

## [0.1.12] - 2026-05-25

### Features

- Harden public pkg API for v0.2.0 (#49)

### Miscellaneous Tasks

- **release**: Update CHANGELOG.md for v0.1.12

## [0.1.11] - 2026-05-25

### Features

- Extract pkg/livereload from internal/server (#48)

### Miscellaneous Tasks

- **release**: Update CHANGELOG.md for v0.1.11

## [0.1.10] - 2026-05-25

### Features

- Lift parser and theme into pkg/ (#46)

### Documentation

- INV/RFC/DESIGN/IMPL set for RFC-0001 public mdp Go library (#45)

### Miscellaneous Tasks

- **release**: Update CHANGELOG.md for v0.1.10

## [0.1.9] - 2026-05-23

### Other

- Deps update (#40)

### Miscellaneous Tasks

- **release**: Update CHANGELOG.md for v0.1.9

## [0.1.8] - 2026-04-03

### Miscellaneous Tasks

- **release**: Update CHANGELOG.md for v0.1.8

## [0.1.7] - 2026-04-03

### Features

- **parser**: Add GitHub-style callout/alert rendering (#25)

### Bug Fixes

- Deps (#17)

### Miscellaneous Tasks

- Disable dependabot (#18)
- **release**: Update CHANGELOG.md for v0.1.7

## [0.1.6] - 2026-03-08

### Miscellaneous Tasks

- **release**: Update CHANGELOG.md for v0.1.6

## [0.1.5] - 2026-03-08

### Features

- **themes**: Rewrite donald theme from authoritative site source files (#13)

## [0.1.4] - 2026-03-07

### Features

- **themes**: Add donald theme; fix keyword/operator contrast across all themes (#12)

## [0.1.3] - 2026-03-07

### Features

- **themes**: Implement first-class theme system with 14 built-in themes (#11)

## [0.1.2] - 2026-02-24

### Features

- Log server output to ~/.local/state/nvim/mdp.log
- Logging, version info, and buffer-close idle shutdown (#9)

## [0.1.1] - 2026-02-24

### Features

- Auto-shutdown server after idle timeout

## [0.1.0] - 2026-02-19

### Features

- Add install script for pre-built binary download
- Add lazy.nvim build.lua and lazy.lua for proper plugin setup

### Bug Fixes

- Move Neovim plugin to standard lua/ directory

### Documentation

- Update lazy.nvim spec to use explicit config function
- Update CONTRIBUTING, README, and CLAUDE.md

## [0.0.1] - 2026-02-18

### Features

- **parser**: Add goldmark parser with GFM and syntax highlighting
- **assets**: Add HTML template, CSS, JS and go:embed
- **server**: Add cobra CLI and HTTP preview server
- **server**: Add WebSocket hub and /ws endpoint for live reload
- **watcher**: Add fsnotify file watcher with 50ms debounce
- **server**: Add SSE fallback, graceful shutdown, connection banner
- **assets**: Vendor Mermaid, KaTeX, and highlight.js libraries
- **parser**: Add goldmark-mermaid and math extensions
- **server**: Add client-side rendering pipeline and theme support
- **parser**: Add AST transformer for source line annotations
- **server**: Add cursor endpoint and scroll sync support
- **server**: Add stdin JSON protocol for editor plugin communication
- **nvim**: Add Neovim Lua plugin with live preview commands
- **cli**: Add --verbose flag for debug logging
- **cli**: Add --version flag with build info via ldflags
- Add relative image paths, custom CSS, network flag, and benchmarks
- **server**: Add auth token for network-exposed servers

### Bug Fixes

- **ci**: Ignore goldmark-mathjax in license check

### Documentation

- Write comprehensive README with install, plugin setup, and CLI reference
- Add CONTRIBUTING.md with vendor update instructions

### Testing

- Add integration tests for parser and HTTP server
- Add live reload tests for WebSocket, SSE, and file watcher
- Add feature tests and update-vendor Makefile target
- **server**: Add scroll sync tests

### Miscellaneous Tasks

- Add Makefile with build, lint, fmt, test targets
- Complete project directory structure
- **release**: Add ldflags and Windows to GoReleaser config
- Update CLAUDE.md with current architecture and commands
- Add Homebrew formula template for tap distribution

[0.3.0]: https://github.com/donaldgifford/mdp/compare/v0.2.2...v0.3.0
[0.2.2]: https://github.com/donaldgifford/mdp/compare/v0.2.1...v0.2.2
[0.2.1]: https://github.com/donaldgifford/mdp/compare/v0.2.0...v0.2.1
[0.2.0]: https://github.com/donaldgifford/mdp/compare/v0.1.12...v0.2.0
[0.1.12]: https://github.com/donaldgifford/mdp/compare/v0.1.11...v0.1.12
[0.1.11]: https://github.com/donaldgifford/mdp/compare/v0.1.10...v0.1.11
[0.1.10]: https://github.com/donaldgifford/mdp/compare/v0.1.9...v0.1.10
[0.1.9]: https://github.com/donaldgifford/mdp/compare/v0.1.8...v0.1.9
[0.1.8]: https://github.com/donaldgifford/mdp/compare/v0.1.7...v0.1.8
[0.1.7]: https://github.com/donaldgifford/mdp/compare/v0.1.6...v0.1.7
[0.1.6]: https://github.com/donaldgifford/mdp/compare/v0.1.5...v0.1.6
[0.1.5]: https://github.com/donaldgifford/mdp/compare/v0.1.4...v0.1.5
[0.1.4]: https://github.com/donaldgifford/mdp/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/donaldgifford/mdp/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/donaldgifford/mdp/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/donaldgifford/mdp/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/donaldgifford/mdp/compare/v0.0.1...v0.1.0

