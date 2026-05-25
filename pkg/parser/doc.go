// Package parser converts markdown to HTML using a configurable
// goldmark pipeline. The default Parser enables GFM extensions
// (tables, strikethrough, task lists, autolinks), syntax highlighting
// with chroma's github style, client-side Mermaid diagrams, MathJax,
// and GitHub-style callouts.
//
// Every block-level element in the output carries a data-source-line
// attribute pointing at its 1-indexed line in the source, which
// downstream consumers can use for scroll-sync or other cursor-aware
// integrations.
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
//	)
//
// # Concurrency
//
// Parser.Render is safe for concurrent use. A single Parser may be
// shared across goroutines.
package parser
