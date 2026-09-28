package parser

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	gmparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// mathExtension parses $...$ (inline) and $$...$$ (display) math the way
// GitHub does: the contents are taken verbatim, so markdown syntax such
// as _ and * inside an expression is left alone. A $$ block that starts
// its own line is parsed as a block, like a fenced code block, so lines
// inside it that look like list items or setext underlines (+ x, = y)
// stay math. Each expression renders as an element with class
// "math inline" or "math display" holding the escaped TeX source without
// its delimiters; assets/preview.js renders exactly those elements with
// KaTeX.
type mathExtension struct{}

func (mathExtension) Extend(m goldmark.Markdown) {
	m.Parser().AddOptions(
		gmparser.WithBlockParsers(
			util.Prioritized(mathBlockParser{}, 650),
		),
		gmparser.WithInlineParsers(
			util.Prioritized(mathParser{}, 150),
		),
	)
	m.Renderer().AddOptions(renderer.WithNodeRenderers(
		util.Prioritized(mathRenderer{}, 500),
	))
}

var (
	kindMath      = ast.NewNodeKind("Math")
	kindMathBlock = ast.NewNodeKind("MathBlock")
)

// mathBlockNode is a display math block whose lines are raw TeX.
type mathBlockNode struct {
	ast.BaseBlock

	// singleLine marks a block opened and closed on one line ($$x$$),
	// which the parser closes before reading the next line.
	singleLine bool
}

func (*mathBlockNode) Kind() ast.NodeKind { return kindMathBlock }

func (*mathBlockNode) IsRaw() bool { return true }

func (n *mathBlockNode) Dump(source []byte, level int) {
	ast.DumpHelper(n, source, level, nil, nil)
}

type mathBlockParser struct{}

func (mathBlockParser) Trigger() []byte { return []byte{'$'} }

// Open starts a block on a line that is either "$$" alone or a complete
// "$$...$$". Any other line starting with $$ is left to the inline
// parser, so a stray "$$" in prose cannot swallow the rest of the
// document.
func (mathBlockParser) Open(_ ast.Node, reader text.Reader, pc gmparser.Context) (ast.Node, gmparser.State) {
	line, segment := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 || !bytes.HasPrefix(line[pos:], []byte("$$")) {
		return nil, gmparser.NoChildren
	}
	rest := util.TrimRightSpace(line[pos+2:])
	start := segment.Start - segment.Padding + pos + 2
	node := &mathBlockNode{}
	switch {
	case len(rest) == 0:
		reader.AdvanceToEOL()
		return node, gmparser.NoChildren
	case len(rest) > 2 && rest[0] != '$' && bytes.HasSuffix(rest, []byte("$$")):
		node.singleLine = true
		node.Lines().Append(text.NewSegment(start, start+len(rest)-2))
		reader.AdvanceToEOL()
		return node, gmparser.NoChildren
	default:
		return nil, gmparser.NoChildren
	}
}

// Continue adds each line to the block until one ends with $$. Text
// before the closing $$ on that line belongs to the expression.
func (mathBlockParser) Continue(n ast.Node, reader text.Reader, _ gmparser.Context) gmparser.State {
	if node, ok := n.(*mathBlockNode); ok && node.singleLine {
		return gmparser.Close
	}
	line, segment := reader.PeekLine()
	trimmed := util.TrimRightSpace(line)
	if bytes.HasSuffix(trimmed, []byte("$$")) {
		if content := len(trimmed) - 2; content > 0 {
			n.Lines().Append(segment.WithStop(segment.Start - segment.Padding + content))
		}
		reader.AdvanceToEOL()
		return gmparser.Close
	}
	n.Lines().Append(segment)
	reader.AdvanceToEOL()
	return gmparser.Continue | gmparser.NoChildren
}

func (mathBlockParser) Close(ast.Node, text.Reader, gmparser.Context) {}

func (mathBlockParser) CanInterruptParagraph() bool { return true }

func (mathBlockParser) CanAcceptIndentedLine() bool { return false }

// mathNode is an inline math expression. Its children are raw text
// segments, one per source line, like ast.CodeSpan.
type mathNode struct {
	ast.BaseInline

	display bool
}

func (*mathNode) Kind() ast.NodeKind { return kindMath }

func (n *mathNode) Dump(source []byte, level int) {
	display := "false"
	if n.display {
		display = "true"
	}
	ast.DumpHelper(n, source, level, map[string]string{"Display": display}, nil)
}

type mathParser struct{}

func (mathParser) Trigger() []byte { return []byte{'$'} }

// Parse reads an expression opened by $ or $$ and closed by the same
// number of dollar signs, possibly on a later line of the same
// paragraph. Reading line by line through the block reader, as the
// code span parser does, keeps container prefixes such as "> " out of
// multi-line expressions. Following GitHub, inline math must not begin
// with whitespace and has restrictions on its closer (see
// isInlineMathCloser), so "$5 and $10" stays text.
func (mathParser) Parse(_ ast.Node, block text.Reader, _ gmparser.Context) ast.Node {
	line, startSegment := block.PeekLine()
	opener := 0
	for opener < len(line) && line[opener] == '$' {
		opener++
	}
	notMath := ast.NewTextSegment(startSegment.WithStop(startSegment.Start + opener))
	if opener > 2 || (opener == 1 && (len(line) < 2 || util.IsSpace(line[1]))) {
		block.Advance(opener)
		return notMath
	}

	block.Advance(opener)
	l, pos := block.Position()
	node := &mathNode{display: opener == 2}
	for {
		line, segment := block.PeekLine()
		if line == nil {
			block.SetPosition(l, pos)
			return notMath
		}
		if end, ok := findMathCloser(line, opener); ok {
			segment = segment.WithStop(segment.Start + end)
			if !segment.IsEmpty() {
				node.AppendChild(node, ast.NewRawTextSegment(segment))
			}
			block.Advance(end + opener)
			return node
		}
		node.AppendChild(node, ast.NewRawTextSegment(segment))
		block.AdvanceLine()
	}
}

// findMathCloser returns the index in line of a run of exactly n dollar
// signs that closes an expression. Backslash-escaped characters are
// skipped so TeX such as \$ does not close it.
func findMathCloser(line []byte, n int) (int, bool) {
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '\\':
			i++
		case '$':
			j := i
			for j < len(line) && line[j] == '$' {
				j++
			}
			if j-i == n && (n == 2 || isInlineMathCloser(line, i, j)) {
				return i, true
			}
			i = j - 1
		}
	}
	return 0, false
}

// isInlineMathCloser reports whether the $ at line[start:end] can close
// inline math. As on GitHub, it cannot be followed by a digit, and it
// cannot have whitespace on both sides: "$x $." closes, "$x $ b" does
// not. Line boundaries count as whitespace.
func isInlineMathCloser(line []byte, start, end int) bool {
	spaceBefore := start == 0 || util.IsSpace(line[start-1])
	spaceAfter := end >= len(line) || util.IsSpace(line[end])
	if spaceBefore && spaceAfter {
		return false
	}
	return end >= len(line) || !isDigit(line[end])
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

type mathRenderer struct{}

func (mathRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindMath, renderMath)
	reg.Register(kindMathBlock, renderMathBlock)
}

func renderMathBlock(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	if _, err := w.WriteString(`<div class="math display"`); err != nil {
		return ast.WalkStop, err
	}
	html.RenderAttributes(w, n, html.GlobalAttributeFilter)
	out := []byte{'>'}
	lines := n.Lines()
	for i := range lines.Len() {
		seg := lines.At(i)
		out = append(out, util.EscapeHTML(seg.Value(source))...)
	}
	out = append(out, "</div>\n"...)
	if _, err := w.Write(out); err != nil {
		return ast.WalkStop, err
	}
	return ast.WalkSkipChildren, nil
}

func renderMath(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	class := "math inline"
	if node, ok := n.(*mathNode); ok && node.display {
		class = "math display"
	}
	out := []byte(`<span class="` + class + `">`)
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			out = append(out, util.EscapeHTML(t.Segment.Value(source))...)
		}
	}
	out = append(out, "</span>"...)
	if _, err := w.Write(out); err != nil {
		return ast.WalkStop, err
	}
	return ast.WalkSkipChildren, nil
}
