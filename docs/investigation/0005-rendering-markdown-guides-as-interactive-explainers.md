---
id: INV-0005
title: "Rendering markdown guides as interactive explainers"
status: Concluded
author: Donald Gifford
created: 2026-10-04
---

<!-- markdownlint-disable-file MD025 MD041 -->

# INV-0005: Rendering markdown guides as interactive explainers

<!--toc:start-->
- [Question](#question)
- [Hypothesis](#hypothesis)
- [Context](#context)
- [Approach](#approach)
- [Environment](#environment)
- [Findings](#findings)
  - [Observation 1: mdxcn is a component kit, not a renderer](#observation-1-mdxcn-is-a-component-kit-not-a-renderer)
  - [Observation 2: Comark is the plain-markdown path, and it is JavaScript only](#observation-2-comark-is-the-plain-markdown-path-and-it-is-javascript-only)
  - [Observation 3: the explainer is an authored artifact, not a rendering](#observation-3-the-explainer-is-an-authored-artifact-not-a-rendering)
  - [Observation 4: what a swap would do to mdp](#observation-4-what-a-swap-would-do-to-mdp)
  - [Observation 5: docz already plans docz view, and docz-site already has mdxcn's runtime](#observation-5-docz-already-plans-docz-view-and-docz-site-already-has-mdxcns-runtime)
  - [Observation 6: a Go-side block parser is small](#observation-6-a-go-side-block-parser-is-small)
  - [Observation 7: the Neovim plugin question is small](#observation-7-the-neovim-plugin-question-is-small)
- [Conclusion](#conclusion)
- [Recommendation](#recommendation)
- [Decisions](#decisions)
- [Open Questions](#open-questions)
- [References](#references)
<!--toc:end-->

## Question

Can a markdown guide such as `bazel-go-monorepo-guide.md` be rendered as
something close to `bazel-go-monorepo-explainer.html` (an interactive page
with a clickable dependency graph, step-through workflows, tabbed
comparisons, and copyable commands) by replacing or extending mdp's
embedded client-side JavaScript with a component layer such as
[mdxcn](https://www.mdxcn.dev/)? And if so, where should that capability
live:

1. in mdp, as a generic feature of the preview, or
2. in docz v2, as a `docz view <type> <id>` command plus a docz Neovim
   plugin, where docz knows the document types and can render each one in
   a more tailored way?

A useful answer names what is achievable generically, what needs an
author's hand, and which repository owns each part.

## Hypothesis

- **mdxcn is not a drop-in replacement for mdp's JavaScript.** The site
  reads as a component kit for React and MDX, which is the opposite of
  mdp's design (a Go binary with no JavaScript build step and all assets
  embedded). Expect a category mismatch rather than a swap.
- **A generic markdown-to-explainer transformation does not exist.** The
  explainer looks hand-built: its data (the example repo's targets and
  edges, the five pipeline stages, the workflow steps) is not present in
  the guide in any machine-readable form. At best, the guide's tables,
  numbered lists, and command blocks map onto generic interactive
  components; the simulators do not.
- **The authoring syntax can be shared even if the runtime cannot.** If
  interactive blocks are written in the guide using one grammar (the
  `::graph-*` blocks mdxcn already defines for plain `.md`), mdp can render
  them with small vanilla renderers and docz-site can render the same
  blocks with the real mdxcn components. That gives the "fairly generic"
  option in mdp and the "dialed in" option in docz without two authoring
  formats.
- **`docz view` is already on docz's roadmap** and is designed to consume
  mdp as a library, so the split is a layering question, not a choice
  between repositories.

## Context

mdp renders markdown server-side with goldmark and hands three things to
the browser for client-side rendering: Mermaid diagrams, KaTeX math, and
highlight.js. Everything else is static HTML with `data-source-line`
attributes for scroll sync. The user wrote a long Bazel guide in
markdown and, separately, an interactive single-file HTML explainer of
the same material, and asked whether the preview could produce the
second from the first.

The two sample files were read from the repository root and were not
committed, since the investigation concluded against doing this work in
mdp:

- `bazel-go-monorepo-guide.md` (1,184 lines: eighteen numbered sections
  plus a contents list, eleven tables, fifty-seven fenced code blocks)
- `bazel-go-monorepo-explainer.html` (880 lines: 209 lines of CSS, 475
  lines of vanilla JavaScript, six interactive panels)

Two docz documents bear directly on the "where" question and are read in
Observation 5: docz INV-0004 (v1 release plan: TUI, markdown preview, CLI
parity, status Open) and docz INV-0005 (docz-api and docz-site), whose
Decision 3 scopes mdp to single-user Neovim and terminal use.

**Triggered by:** user request on 2026-10-04, after reviewing
<https://www.mdxcn.dev/>. Related: RFC-0001 (public mdp Go library),
DESIGN-0004 (diagram skin), docz INV-0004, docz INV-0018 (step-aware
runbook view in docz-site).

## Approach

1. Inventory mdxcn from its site, its agent reference (`llms.txt`), its
   GitHub repository, and one registry item (`graph-timeline.json`):
   what it renders, what it needs at runtime, how it is distributed, and
   what it does with a plain `.md` file. **Done.**
2. Inventory Comark, the plain-markdown component syntax mdxcn adapts
   to, since that is the only path that does not require MDX. **Done.**
3. Read the explainer and classify each panel: generic component, or
   bespoke logic that needs authored data. Then walk the guide and
   record which sections carry data a generic component could consume.
   **Done.**
4. Compare mdxcn's runtime requirements with mdp's architecture
   (`assets/`, `pkg/parser`, `pkg/livereload`, the `data-source-line`
   contract, live reload). **Done.**
5. Read docz INV-0004, docz INV-0005 Decision 3, docz DESIGN-0009, and
   docz-site's dependency list to establish what docz already plans for
   viewing documents and what runtime docz-site has. **Done.**
6. Check what goldmark offers for a `::name` or `:::name` block syntax,
   so a Go-side parser for component blocks is sized realistically.
   **Done.**
7. Prototype one block end to end in mdp: a `::graph-steps` block parsed
   by a goldmark block parser, emitted as a typed element, hydrated by
   vanilla JavaScript in `preview.js`, with `data-source-line` intact
   and state surviving a live-reload update. **Not pursued; see
   Conclusion.**
8. Write the same block in a docz-site fixture using the mdxcn
   component to confirm the shared-grammar claim. **Not pursued here;
   belongs to docz if picked up.**

## Environment

Checked on 2026-10-04.

| Component | Version / Value |
| --------- | --------------- |
| mdp | `main` after v0.6.1; goldmark v1.8.6; `go.abhg.dev/goldmark/mermaid` v0.6.0 |
| mdp vendored client assets | `mermaid.min.js` 5.4 MB, KaTeX 596 KB, highlight.js 136 KB, fonts 204 KB, `preview.js` 24 KB |
| mdxcn | GitHub `keshav-exe/mdxcn`, MIT, 373 stars, 35 commits, Next.js app; registry at `https://mdxcn.dev/r/<slug>.json` |
| mdxcn `graph-timeline` registry item | npm dependency `motion`; files `graph-frame.tsx` (~280 lines), `graph-markdown.ts` (~680 lines), `graph-motion.ts` (~160 lines), `graph-timeline.tsx` (~120 lines) |
| Comark | <https://comark.dev>, MIT, maintained by Vercel, TypeScript on `markdown-exit`; renderers for HTML, ANSI, React, Vue, Svelte, Angular; no Go implementation |
| docz | `/v2` development line; INV-0004 Open; `pkg/` holds `doczcore`, `adr`, `design`, `impl`, `investigation`, `rfc`, `runbook`, `wiki` |
| docz-site | Vite, React 19.2, Tailwind CSS 4.3, unified 11 with `remark-gfm`, `remark-rehype`, `rehype-sanitize`, `rehype-raw`, `rehype-slug`, `@shikijs/rehype`, `mermaid` 12 |
| goldmark directive-style extensions | `github.com/stefanfritsch/goldmark-fences` v1.0.0 (`:::name` fenced divs) |

## Findings

### Observation 1: mdxcn is a component kit, not a renderer

mdxcn is a shadcn registry of about fifty React "graph" components:
content blocks with markdown bodies (Callout, Quote, Steps, Terminal,
Changelog, Annotate, Decision, Chat, Env, Endpoint, Keys, FAQ) and
data-driven graphs (Timeline, Tree, Gantt, Board, Stat, Spec, Check,
Diff, KPI, Table, Sheet, Compare, Matrix, Flow, Slope, Heatmap, and
others). All of them share one markdown grammar: `**bold**` marks the
current state, `*italic*` the next or rejected one, `- label: value`
makes rows, `a → b → c` makes paths, nested lists make children, and
`###` headings make columns.

It can be written three ways:

```jsx
<GraphTimeline title="SHIPPED">
- Mar 12: CLI copies the files
- **Mar 18: Docs, live previews**
</GraphTimeline>
```

```markdown
::graph-timeline
---
title: SHIPPED
events:
  - { date: "Mar 12", label: "CLI copies the files" }
  - { date: "Mar 18", label: "Docs, live previews", state: now }
---
::
```

```text
+------------------ [ SHIPPED ] -------------------+
| ●  Mar 12  CLI copies the files                  |
| ●  Mar 18  Docs, live previews                   |
+--------------------------------------------------+
```

The first is MDX (needs a React host), the second is a Comark block in a
plain `.md` file (needs a Comark host), the third is the ASCII fallback
that any markdown viewer shows as a code block. The agent reference is
explicit that mdxcn "renders its own diagram blocks only, not arbitrary
Markdown": the host application renders the prose.

The runtime is not optional. The `graph-timeline` registry item pulls in
`motion`, imports `cn` from `@/lib/utils` (the shadcn Tailwind helper),
and ships a 680-line TypeScript markdown grammar parser alongside the
component. Installing it means a React application with a bundler and
Tailwind, into which the shadcn CLI copies source files. There is no
prebuilt browser bundle to vendor the way mdp vendors Mermaid.

### Observation 2: Comark is the plain-markdown path, and it is JavaScript only

Comark is Vercel's MIT markdown library with a component syntax:
`::alert{type="info"}` ... `::` for blocks and `:button[Submit]{type="primary"}`
inline. It is a standalone TypeScript parser (a rewrite of markdown-it),
not a remark plugin, with renderers for HTML, ANSI, React, Vue, Svelte,
and Angular. No Go implementation is documented.

For mdp, which parses on the server in Go, this means the `::graph-*`
syntax can be adopted as a *grammar* but not as a *library*. A goldmark
block parser would have to recognise the fence and hand the YAML body to
the client, the same pattern mdp already uses for Mermaid (`<pre
class="mermaid">` placeholder) and, since PR #95, for math (`data-math`
elements hydrated by KaTeX). Observation 6 sizes that.

### Observation 3: the explainer is an authored artifact, not a rendering

The explainer's six panels and what drives each:

| Panel | Behaviour | Data source | Generic component? |
| --- | --- | --- | --- |
| What rebuilds | SVG graph of 15 targets and 14 edges; click a node, see downstream targets rerun; three cache modes change the result | Hand-written `nodes` and `edges` arrays, a reachability walk | No. Needs the graph as data and a simulation rule |
| One command, step by step | Five stage tabs with what each reads and when it is slow | Hand-written `stages` array | Partly. Steps or Timeline could show it; the "slow when" field is custom |
| Cache hits | Checklist of inputs and non-inputs; an FNV hash of the ticked set shows whether the key changed | Hand-written lists plus a hash | No. A simulator with domain rules |
| Labels and patterns | Text input; a `matcher()` parses `//pkg/...`, `:all`, labels and highlights the matching targets | Hand-written target list and a Bazel pattern grammar | No. Domain-specific parser |
| Workflows | Tabs of workflows, each a step list; a file strip shows which files each step reads or writes | Hand-written `flows` with `w` and `r` file sets | Partly. Steps plus a per-step "touches" annotation |
| mise, Nix, Bazelisk | Tabs with a layer diagram, a config block, and a note | Hand-written `tools` array | Yes. Tabs or Compare over three columns |

None of that data exists in the guide as data. The guide has the
"You get / You pay" table, the Bazel-term table, the version table, the
pattern table, the failure-modes table, numbered bootstrap and adoption
steps, and fifty-seven fenced blocks, most of them commands or config
files. A generic layer could turn those
into Compare, Spec, Table, Steps, and Terminal components with copy
buttons, which is roughly the "mise, Nix, Bazelisk" and "Workflows"
panels. It cannot produce the graph simulator, the cache-key simulator,
or the pattern matcher, because they encode knowledge an author added
when writing the explainer. Those are, in mdxcn's own terms, custom
components with authored data.

So the realistic target is not "render the guide as the explainer" but
"let an author put explainer blocks in the guide and have them render":
the `.md` carries the data for a Flow or Tree block in YAML, and prose
around it. That is also mdxcn's stated usage pattern ("drop a figure next
to prose", at most two graphs per section).

### Observation 4: what a swap would do to mdp

mdp's design, from RFC-0001 and `CLAUDE.md`: one Go binary, goldmark on
the server, `assets/` embedded with `go:embed`, no JavaScript build,
24 KB of hand-written `preview.js`, and three vendored libraries. The
client contract is small and documented: `data-source-line` for scroll
sync, `<pre class="mermaid">` for diagrams, `[data-math]` for KaTeX, a
`wsMessage` JSON shape for live reload.

Replacing that with mdxcn would mean:

- adding Node, a bundler, React, Tailwind, and `motion` to a Go
  repository, and a build step that produces the embedded bundle;
- keeping Mermaid, KaTeX, and highlight.js anyway, since mdxcn renders
  none of those (its host does);
- re-establishing scroll sync, because a React render tree replaces the
  server HTML that carries `data-source-line`;
- losing live reload's simplicity: today `updateContent` swaps
  `innerHTML` and re-runs three renderers; a React tree needs a
  reconciliation or state-preservation story for every interactive
  block.

In other words the "baked-in JS" (Mermaid, KaTeX, highlight.js) is
orthogonal to mdxcn; what mdxcn would replace is the *absence* of a
frontend build. That is the design decision docz INV-0005 Decision 3
already recorded from the other side: mdp is intentionally small for
single-user Neovim and terminal use, and the multi-user, React-rendered
viewer is docz-site.

### Observation 5: docz already plans `docz view`, and docz-site already has mdxcn's runtime

docz INV-0004 (Open) answers the second half of the question before this
investigation asked it. Its plan is a Bubble Tea TUI inside the `docz`
binary with a View screen: Glamour renders the document in the terminal,
and the `p` key opens it in the browser through mdp. The integration
decision is resolved there: link `github.com/donaldgifford/mdp` as a Go
dependency and build `internal/preview/` on `mdp/pkg/parser`,
`mdp/pkg/theme`, and `mdp/pkg/livereload`. No subprocess, no `PATH`
dependency. IMPL-0010 (preview integration) and IMPL-0011 (full TUI) are
the named follow-ups.

So `docz view rfc 0001` is a thin command over that: resolve the id to a
file through `pkg/doczcore`, render with mdp's packages, serve with
`pkg/livereload`. docz does not need a second renderer, and it inherits
whatever block support mdp's parser gains.

docz-site is the other consumer, and it is where mdxcn fits naturally.
It is React 19 with Tailwind 4 and a unified pipeline (`remark-parse`,
`remark-gfm`, `remark-rehype`, `rehype-sanitize`, `@shikijs/rehype`,
`mermaid`). docz INV-0018 is already adding type-aware interactive
rendering there (step anchors, copy buttons, and rail navigation for
runbooks) as remark plugins inside that pipeline. An mdxcn Comark block
is one more remark plugin that recognises `::graph-*`, parses the YAML,
and renders the installed component. Two cautions from INV-0018 carry
over: `rehype-sanitize` strips what it does not know, so the block must
be transformed at the remark stage, and a grammar implemented twice (Go
in mdp, TypeScript in docz-site) drifts unless both are pinned to shared
fixtures.

### Observation 6: a Go-side block parser is small

goldmark has no built-in directive syntax. `goldmark-fences` v1.0.0
gives `:::name` fenced divs with attributes but no YAML body. A purpose-
built block parser for `::graph-<kind>` with a `---` YAML body and a
closing `::` is the same shape as `mathBlockParser` in
`pkg/parser/math.go` (about 300 lines with tests, landed in PR #95):
trigger on `:`, open on the fence line, append lines until the closer,
render a typed element. Emitting

```html
<div class="graph" data-graph="timeline" data-source-line="42">
  <script type="application/yaml">...</script>
  <pre>ASCII fallback, if the author supplied one</pre>
</div>
```

keeps scroll sync (the wrapper carries `data-source-line`), degrades to
the fallback with JavaScript off, and hands hydration to `preview.js`
the same way `[data-math]` does. A YAML parser is the one new direct Go
dependency; `gopkg.in/yaml.v3` and `go.yaml.in/yaml/v3` are already in
the module graph as indirect dependencies, so promoting one is a
one-line `go.mod` change.

The client side is the real cost. Each kind needs a vanilla renderer in
`preview.js`. Timeline, Steps, Spec, Compare, Table, and Terminal with a
copy button are each under a hundred lines of DOM code. Flow and Tree
need layout; the explainer draws its graph with hand-placed columns and
Bezier edges in 60 lines, which is fine for a declared column layout and
not fine for arbitrary graphs. Anything the ASCII fallback covers well
(Timeline, Steps, Check) can ship fallback-first.

Live reload adds one problem mdp has not had to solve: interactive state.
`updateContent` replaces `innerHTML`, so a selected tab or an edited
graph node resets on every keystroke in the editor. Keying state by
`data-source-line` plus kind and restoring it after hydration is enough
for tabs and selections; it is the first thing the prototype in Approach
step 7 must prove.

### Observation 7: the Neovim plugin question is small

mdp already ships `lua/mdp/init.lua`, which resolves the binary, pipes
the buffer over stdin, and syncs the cursor. A docz Neovim plugin that
opens `docz view` would duplicate that transport. The thin version is a
plugin (or a few commands added to mdp's) that calls `docz` to resolve
`rfc 0001` to a path and then invokes the existing `:MdpPreview` on it.
docz-specific value lives in resolution, listing, and status changes,
not in preview plumbing.

## Conclusion

**Answer:** No for mdp. Reviewed on 2026-10-04: mdp is not the right
place for this, and the work will not be done here. The investigation is
kept as the record of why. The findings below stand for anyone picking
the idea up in docz.

The detailed answer is: partly, and the parts belong in different places.

- **Swapping mdp's embedded JavaScript for mdxcn: no.** mdxcn is a React
  and Tailwind component kit installed by copying source into a bundled
  application. mdp has no bundler by design, and the libraries it
  embeds (Mermaid, KaTeX, highlight.js) are not what mdxcn replaces.
- **Rendering the guide as the explainer generically: no.** The explainer
  is a second authored artifact. Its simulators run on hand-written data
  that the guide does not contain. Roughly two of its six panels have
  generic equivalents (tabbed comparisons, step lists with command copy
  and a file-touch strip); the rest are custom components.
- **Rendering authored explainer blocks inside a guide: yes, in two
  tiers.** Adopt mdxcn's Comark grammar (`::graph-<kind>` with a YAML
  body and optional ASCII fallback) as the authoring format. mdp parses
  it in Go and hydrates a small set of kinds with vanilla renderers: the
  generic tier. docz-site renders the same blocks with the real mdxcn
  components inside its existing remark pipeline: the dialed-in tier.
  One syntax, two runtimes, no MDX.
- **`docz view` lives in docz and consumes mdp.** docz INV-0004 already
  decided this; this investigation adds nothing to that decision except
  that any block support must land in `pkg/parser` so docz inherits it.
  A docz Neovim plugin should resolve ids and delegate preview to
  mdp.nvim.

## Recommendation

1. **No change to mdp.** No mdxcn swap, no component-block parser, no
   new client contract. mdp keeps its scope: a small Go binary with
   embedded assets for single-user Neovim and terminal preview, as docz
   INV-0005 Decision 3 already records from the docz side.
2. **If the idea returns, it is docz work.** The natural shape, for the
   record: a `::graph-*` remark plugin in docz-site that renders mdxcn's
   components inside the existing unified pipeline (transformed at the
   remark stage because of `rehype-sanitize`, as docz INV-0018 notes),
   and `docz view <type> <id>` as the thin command docz INV-0004 already
   plans on top of mdp's `pkg/`. mdp would only ever need to pass
   `::graph-*` blocks through untouched, which it does today: they
   render as paragraphs and the ASCII fallback renders as a code block.
3. **The sample files stay out of the repository.** They are not
   committed; keep them locally or delete them.

## Decisions

Resolved by user review on 2026-10-04.

| # | Topic | Options | Decision |
| --- | --- | --- | --- |
| 1 | mdxcn in mdp | (a) swap in mdxcn with a frontend build; (b) adopt its grammar only, render with vanilla JS; (c) do nothing | **(c)**. mdp is not the right place for this |
| 2 | Authoring syntax | (a) Comark `::graph-*` with YAML; (b) `:::name` fenced divs via goldmark-fences; (c) HTML comments or a custom fence language | Not decided here. If docz picks it up, (a) keeps mdxcn's components usable unchanged |
| 3 | Where interactive rendering lives | (a) mdp; (b) docz-site; (c) both | **(b)**, if anywhere |
| 4 | `docz view` | (a) new renderer in docz; (b) docz links mdp `pkg/` per docz INV-0004 | **(b)**, already decided in docz INV-0004; no mdp change needed |
| 5 | Neovim | (a) new docz.nvim with its own preview; (b) docz resolves, mdp.nvim previews | **(b)**, if docz builds a plugin |
| 6 | Sample files | (a) `docs/examples/explainer/`; (b) not committed | **(b)** |

## Open Questions

None open for mdp. The questions below pass to docz with the idea, if it
is picked up there:

1. Which kinds would docz-site render first? Steps, Terminal, Compare,
   Spec, and Timeline cover most of what a reference document wants;
   Flow is the one the explainer makes the strongest case for.
2. Should every block carry the ASCII fallback so a document stays
   readable on GitHub, in mdp, and in the terminal? mdxcn treats it as
   optional.
3. Does Glamour (docz's planned in-TUI renderer) show the ASCII fallback
   cleanly? If so the TUI gets the blocks for free.

## References

- `bazel-go-monorepo-guide.md` and `bazel-go-monorepo-explainer.html`
  (untracked, repository root)
- mdxcn: <https://www.mdxcn.dev/>, agent reference
  <https://mdxcn.dev/llms.txt>, registry example
  <https://mdxcn.dev/r/graph-timeline.json>, source
  <https://github.com/keshav-exe/mdxcn>
- Comark: <https://comark.dev>, <https://github.com/comarkdown/comark>
- goldmark-fences: <https://github.com/stefanfritsch/goldmark-fences>
- mdp RFC-0001 (public mdp Go library), DESIGN-0004 (diagram skin),
  INV-0004 (beautiful-mermaid), PR #95 (`pkg/parser/math.go` block
  parser and the `data-math` client contract)
- docz INV-0004: v1 release plan, TUI, markdown preview, CLI parity
  (`docs/investigation/0004-v1-release-plan-tui-markdown-preview-and-cli-parity.md`
  in `donaldgifford/docz`)
- docz INV-0005 Decision 3 and DESIGN-0009: mdp scoped to single-user
  Neovim and terminal use; docz-site renders markdown client-side
- docz INV-0018: docz-site step-aware runbook view (remark-stage
  transforms, sanitize caveat, grammar-drift caveat)
- `CLAUDE.md` in this repository: math contract, Mermaid contract,
  `data-source-line` scroll-sync contract
