// Package parser provides markdown-to-HTML conversion using goldmark.
package parser

import (
	"bytes"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// Parser converts markdown to HTML using goldmark.
type Parser struct {
	md goldmark.Markdown
}

// Option configures a Parser.
type Option func(*config)

type config struct {
	gfm                bool
	syntaxHighlighting bool
}

func defaultConfig() config {
	return config{
		gfm:                true,
		syntaxHighlighting: true,
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

// New creates a Parser with the given options. By default, GFM extensions
// and syntax highlighting are enabled.
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

	md := goldmark.New(
		goldmark.WithExtensions(extensions...),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithUnsafe(),
		),
	)

	return &Parser{md: md}
}

// Render converts markdown bytes to HTML bytes.
func (p *Parser) Render(src []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := p.md.Convert(src, &buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
