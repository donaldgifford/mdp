# Changelog

All notable changes to this project will be documented in this file.

## [0.1.7] - 2026-04-03

### Added

- **parser**: Add GitHub-style callout/alert rendering (#25)

### Fixed

- Deps (#17)

### Miscellaneous

- Disable dependabot (#18)

## [0.1.6] - 2026-03-08

### Miscellaneous

- **changelog**: Add git-cliff and automated CHANGELOG.md generation (#14)

## [0.1.5] - 2026-03-08

### Added

- **themes**: Rewrite donald theme from authoritative site source files (#13)

## [0.1.4] - 2026-03-07

### Added

- **themes**: Add donald theme; fix keyword/operator contrast across all themes (#12)

## [0.1.3] - 2026-03-07

### Added

- **themes**: Implement first-class theme system with 14 built-in themes (#11)

## [0.1.2] - 2026-02-24

### Added

- Log server output to ~/.local/state/nvim/mdp.log
- Logging, version info, and buffer-close idle shutdown (#9)

## [0.1.1] - 2026-02-24

### Added

- Auto-shutdown server after idle timeout

## [0.1.0] - 2026-02-19

### Added

- Add install script for pre-built binary download
- Add lazy.nvim build.lua and lazy.lua for proper plugin setup

### Documentation

- Update lazy.nvim spec to use explicit config function
- Update CONTRIBUTING, README, and CLAUDE.md

### Fixed

- Move Neovim plugin to standard lua/ directory

## [0.0.1] - 2026-02-18

### Added

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

### Documentation

- Write comprehensive README with install, plugin setup, and CLI reference
- Add CONTRIBUTING.md with vendor update instructions

### Fixed

- **ci**: Ignore goldmark-mathjax in license check

### Miscellaneous

- Add Makefile with build, lint, fmt, test targets
- Complete project directory structure
- Update CLAUDE.md with current architecture and commands
- Add Homebrew formula template for tap distribution

### Testing

- Add integration tests for parser and HTTP server
- Add live reload tests for WebSocket, SSE, and file watcher
- Add feature tests and update-vendor Makefile target
- **server**: Add scroll sync tests

[0.1.7]: https://github.com/donaldgifford/mdp/compare/v0.1.6...v0.1.7
[0.1.6]: https://github.com/donaldgifford/mdp/compare/v0.1.5...v0.1.6
[0.1.5]: https://github.com/donaldgifford/mdp/compare/v0.1.4...v0.1.5
[0.1.4]: https://github.com/donaldgifford/mdp/compare/v0.1.3...v0.1.4
[0.1.3]: https://github.com/donaldgifford/mdp/compare/v0.1.2...v0.1.3
[0.1.2]: https://github.com/donaldgifford/mdp/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/donaldgifford/mdp/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/donaldgifford/mdp/compare/v0.0.1...v0.1.0

