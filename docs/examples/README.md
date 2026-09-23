# Mermaid example diagrams

Copy-paste fixtures for testing mdp's vendored Mermaid (v12) rendering.
Each file holds one diagram type; `all.md` combines everything.
`all.md` is also the screenshot corpus for the diagram skin (DESIGN-0004).

Preview any file with a local build:

```bash
make build && ./mdp serve docs/examples/<file>.md
```

All blocks are validated with the vendored parser (`mermaid.parse()`), so a
render failure in the browser points at preview/runtime, not syntax.
New diagrams must follow the v12 grammar notes in each file's header.
