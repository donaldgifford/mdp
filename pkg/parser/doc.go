// Package parser converts markdown to HTML using a configurable
// goldmark pipeline. The default Parser enables GFM extensions
// (tables, strikethrough, task lists, autolinks), syntax highlighting
// with chroma's github style, client-side Mermaid diagrams, MathJax,
// GitHub-style callouts, and extended-syntax footnotes.
//
// Every block-level element in the output carries a data-source-line
// attribute pointing at its 1-indexed line in the source, which
// downstream consumers can use for scroll-sync or other cursor-aware
// integrations.
//
// # data-source-line ordering
//
// With footnotes enabled, data-source-line values are NOT guaranteed
// to be non-decreasing in document order. Footnote definitions are
// collected into a <div class="footnotes"> endnote list rendered last,
// but each entry keeps the source line where it was defined. Two
// shapes produce out-of-order values:
//
//   - A definition placed mid-document renders after body content that
//     appears later in the source.
//   - Footnotes are numbered by first-reference order, so when
//     reference order differs from definition order the entries
//     themselves are out of source order — even with every definition
//     at the end of the file.
//
// Consumers that map a cursor line to an element by scanning in
// document order and stopping at the first larger value must skip the
// .footnotes subtree, or they will select a footnote instead of the
// intended block. Outside that subtree the values are non-decreasing.
//
// # Minimal usage
//
// The zero-config Parser is suitable for most callers:
//
//	p := parser.New()
//	html, err := p.Render([]byte("# Hello"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("%s", html)
//
// # All options
//
// Each feature has a With* toggle. Pass mermaid.RenderModeServer to
// WithMermaidRenderMode to render Mermaid diagrams to inline <svg>
// at parse time (requires the mmdc CLI). The default
// RenderModeClient emits <pre class="mermaid"> placeholders for the
// browser to render with mermaid.js.
//
//	import "go.abhg.dev/goldmark/mermaid"
//
//	p := parser.New(
//	    parser.WithGFM(true),
//	    parser.WithSyntaxHighlighting(true),
//	    parser.WithMermaid(true),
//	    parser.WithMermaidRenderMode(mermaid.RenderModeClient),
//	    parser.WithMath(true),
//	    parser.WithCallouts(true),
//	    parser.WithFootnotes(true),
//	)
//
// # Concurrency
//
// Parser.Render is safe for concurrent use. A single Parser may be
// shared across goroutines.
//
// Note: as a temporary workaround for a known data race in
// gm-alert-callouts@v0.8.0 (see INV-0003), Render currently
// serializes goldmark conversion behind a per-Parser mutex.
// Throughput-sensitive callers that fan out across many goroutines
// should construct multiple Parser instances until the upstream
// fix lands.
package parser
