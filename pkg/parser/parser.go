package parser

import (
	"bytes"
	"sync"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	mathjax "github.com/litao91/goldmark-mathjax"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
	alertcallouts "github.com/zmtcreative/gm-alert-callouts"
	"go.abhg.dev/goldmark/mermaid"
)

// Parser converts markdown to HTML using goldmark.
type Parser struct {
	md goldmark.Markdown
	mu sync.Mutex
}

// Option configures a Parser.
type Option func(*config)

type config struct {
	gfm                bool
	syntaxHighlighting bool
	mermaid            bool
	mermaidMode        mermaid.RenderMode
	math               bool
	callouts           bool
	footnotes          bool
}

func defaultConfig() config {
	return config{
		gfm:                true,
		syntaxHighlighting: true,
		mermaid:            true,
		mermaidMode:        mermaid.RenderModeClient,
		math:               true,
		callouts:           true,
		footnotes:          true,
	}
}

// WithGFM enables or disables GitHub Flavored Markdown extensions
// (tables, strikethrough, task lists, autolinks).
func WithGFM(enabled bool) Option {
	return func(c *config) { c.gfm = enabled }
}

// WithSyntaxHighlighting enables or disables syntax highlighting on
// fenced code blocks.
func WithSyntaxHighlighting(enabled bool) Option {
	return func(c *config) { c.syntaxHighlighting = enabled }
}

// WithMermaid enables or disables Mermaid diagram support.
func WithMermaid(enabled bool) Option {
	return func(c *config) { c.mermaid = enabled }
}

// WithMermaidRenderMode sets the Mermaid render mode. The default is
// mermaid.RenderModeClient, which emits <pre class="mermaid"> blocks
// for the browser to render with mermaid.js. mermaid.RenderModeServer
// renders to inline <svg> at parse time (requires the mermaid CLI).
// Has no effect when WithMermaid(false) is set.
func WithMermaidRenderMode(mode mermaid.RenderMode) Option {
	return func(c *config) { c.mermaidMode = mode }
}

// WithMath enables or disables math expression support ($...$ and $$...$$).
func WithMath(enabled bool) Option {
	return func(c *config) { c.math = enabled }
}

// WithCallouts enables or disables GitHub-style callout/alert rendering
// (> [!NOTE], > [!TIP], > [!IMPORTANT], > [!WARNING], > [!CAUTION]).
func WithCallouts(enabled bool) Option {
	return func(c *config) { c.callouts = enabled }
}

// WithFootnotes enables or disables extended-syntax footnotes
// ([^1] references and [^1]: definitions). Definitions are collected
// into a <div class="footnotes"> endnote list rendered at the end of
// the output, regardless of where they appear in the source.
func WithFootnotes(enabled bool) Option {
	return func(c *config) { c.footnotes = enabled }
}

// New creates a Parser with the given options. By default, GFM extensions,
// syntax highlighting, Mermaid, math, callouts, and footnotes are all
// enabled.
func New(opts ...Option) *Parser {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}

	var extensions []goldmark.Extender
	if cfg.gfm {
		extensions = append(extensions, extension.GFM)
	}
	if cfg.syntaxHighlighting {
		extensions = append(extensions, highlighting.NewHighlighting(
			highlighting.WithStyle("github"),
			highlighting.WithFormatOptions(
				chromahtml.WithClasses(true),
			),
		))
	}
	if cfg.mermaid {
		extensions = append(extensions, &mermaid.Extender{
			RenderMode: cfg.mermaidMode,
		})
	}
	if cfg.math {
		extensions = append(extensions, mathjax.MathJax)
	}
	if cfg.callouts {
		extensions = append(extensions, alertcallouts.NewAlertCallouts(
			alertcallouts.UseGFMStrictIcons(),
		))
	}
	if cfg.footnotes {
		extensions = append(extensions, extension.Footnote)
	}

	md := goldmark.New(
		goldmark.WithExtensions(extensions...),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(
				util.Prioritized(&lineAnnotator{}, 0),
			),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)

	return &Parser{md: md}
}

// Render converts markdown bytes to HTML bytes.
func (p *Parser) Render(src []byte) ([]byte, error) {
	// Serialize Convert to work around a data race in
	// gm-alert-callouts@v0.8.0 (shared cases.Caser, not safe for
	// concurrent use). Remove when upstream ships a fix — see
	// INV-0003 and IMPL-0005.
	p.mu.Lock()
	defer p.mu.Unlock()

	var buf bytes.Buffer
	if err := p.md.Convert(src, &buf); err != nil {
		// coverage: goldmark.Convert only errors on impossible
		// conditions (nil writer, malformed AST from a broken
		// extender). The default extender set never triggers this.
		return nil, err
	}
	return buf.Bytes(), nil
}
