# Contributing to mdp

## Development Setup

```bash
# Install tool versions
mise install

# Build
make build

# Run tests
make test

# Lint
make lint
```

## Updating Vendored JS Libraries

The `assets/vendor/` directory contains Mermaid, KaTeX, and highlight.js
libraries embedded into the binary. To update them:

```bash
make update-vendor
```

This pulls the latest versions from CDN. After updating:

1. Run `make test` to verify nothing broke
2. Check binary size hasn't grown significantly: `make build && ls -lh mdp`
3. If KaTeX version changed, update fonts too:

```bash
# Download KaTeX fonts (replace VERSION as needed)
cd assets/vendor/katex/fonts/
for font in KaTeX_AMS KaTeX_Caligraphic KaTeX_Fraktur KaTeX_Main KaTeX_Math KaTeX_SansSerif KaTeX_Script KaTeX_Size1 KaTeX_Size2 KaTeX_Size3 KaTeX_Size4 KaTeX_Typewriter; do
  for variant in Regular Bold BoldItalic Italic; do
    url="https://cdn.jsdelivr.net/npm/katex@0.16/dist/fonts/${font}-${variant}.woff2"
    curl -sfL -o "${font}-${variant}.woff2" "$url" 2>/dev/null
  done
done
```

## Commit Conventions

Use conventional commits: `feat:`, `fix:`, `chore:`, `docs:`, `test:`.

PR labels (`major`, `minor`, `patch`) control semantic versioning on release.
