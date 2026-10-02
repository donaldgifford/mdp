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
// stay math.
//
// Each expression renders as an element with class "math inline" or
// "math display" (the shape goldmark-mathjax used, kept for CSS) and a
// data-math="inline"|"display" attribute holding the escaped TeX source
// without its delimiters. data-math is the contract with
// assets/preview.js, which renders exactly those elements with KaTeX and
// leaves author HTML that merely uses class "math" alone.
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
	// coverage: Dump is goldmark's AST debugging aid; nothing in the
	// render path calls it.
	ast.DumpHelper(n, source, level, nil, nil)
}

type mathBlockParser struct{}

func (mathBlockParser) Trigger() []byte { return []byte{'$'} }

// Open starts a block on a line that is either "$$" alone or a complete
// "$$...$$" whose first closer is at the end of the line. Any other line
// starting with $$ is left to the inline parser, so "$$a$$ and $$b$$"
// is two display expressions rather than one block with "$$" inside.
//
// A bare "$$" that would interrupt a paragraph is also left alone when
// that paragraph already holds an unmatched "$$": the line is the
// closer of an expression opened mid-paragraph ("text $$" / "x" / "$$"),
// and claiming it as an opener would start a block that runs to the end
// of the container. An opened block does run until its closing "$$" or
// the end of the container, exactly like an unclosed code fence.
func (mathBlockParser) Open(_ ast.Node, reader text.Reader, pc gmparser.Context) (ast.Node, gmparser.State) {
	line, segment := reader.PeekLine()
	pos := pc.BlockOffset()
	if pos < 0 || !bytes.HasPrefix(line[pos:], []byte("$$")) {
		return nil, gmparser.NoChildren
	}
	if last := pc.LastOpenedBlock().Node; last != nil && ast.IsParagraph(last) &&
		hasOpenDisplayMath(last, reader.Source()) {
		return nil, gmparser.NoChildren
	}
	rest := util.TrimRightSpace(line[pos+2:])
	start := segment.Start - segment.Padding + pos + 2
	node := &mathBlockNode{}
	end, found := findCloser(rest, 2)
	switch {
	case len(rest) == 0:
		reader.AdvanceToEOL()
		return node, gmparser.NoChildren
	case len(rest) > 2 && rest[0] != '$' && found && end == len(rest)-2:
		node.singleLine = true
		node.Lines().Append(text.NewSegment(start, start+end))
		reader.AdvanceToEOL()
		return node, gmparser.NoChildren
	default:
		return nil, gmparser.NoChildren
	}
}

// hasOpenDisplayMath reports whether the paragraph's lines contain an
// odd number of unescaped "$$", meaning a display expression opened
// mid-paragraph is still waiting for its closer.
func hasOpenDisplayMath(paragraph ast.Node, source []byte) bool {
	count := 0
	lines := paragraph.Lines()
	for i := range lines.Len() {
		seg := lines.At(i)
		line := seg.Value(source)
		for off := 0; off < len(line); {
			end, found := findCloser(line[off:], 2)
			if !found {
				break
			}
			count++
			off += end + 2
		}
	}
	return count%2 == 1
}

// Continue adds each line to the block until one whose first unescaped
// "$$" sits at the end of the line. Text before that closer belongs to
// the expression; "a = \$$" does not close the block because \$ is an
// escaped dollar, matching the inline rule in findCloser.
func (mathBlockParser) Continue(n ast.Node, reader text.Reader, _ gmparser.Context) gmparser.State {
	if node, ok := n.(*mathBlockNode); ok && node.singleLine {
		return gmparser.Close
	}
	line, segment := reader.PeekLine()
	trimmed := util.TrimRightSpace(line)
	if end, found := findCloser(trimmed, 2); found && end == len(trimmed)-2 {
		if end > 0 {
			n.Lines().Append(segment.WithStop(segment.Start - segment.Padding + end))
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
	// coverage: Dump is goldmark's AST debugging aid; nothing in the
	// render path calls it.
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
// multi-line expressions.
//
// Following GitHub, inline math must not begin with whitespace, and the
// first unescaped $ after the opener decides the expression: if it
// cannot close (see isInlineMathCloser) the opener is plain text and
// scanning stops, so "$x$5 and $y$" leaves "$x$5 and " as text and
// renders only "$y$". Scanning past a rejected closer would pair the
// opener with the next expression's opening dollar instead.
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
			break
		}
		end, found := findCloser(line, opener)
		if found && (opener == 2 || isInlineMathCloser(line, end)) {
			segment = segment.WithStop(segment.Start + end)
			if !segment.IsEmpty() {
				node.AppendChild(node, ast.NewRawTextSegment(segment))
			}
			block.Advance(end + opener)
			return node
		}
		if found {
			break
		}
		node.AppendChild(node, ast.NewRawTextSegment(segment))
		block.AdvanceLine()
	}
	block.SetPosition(l, pos)
	return notMath
}

// findCloser returns the index in line of the first unescaped "$"
// (n == 1) or "$$" (n == 2). Backslash-escaped characters are skipped so
// TeX such as \$ never closes an expression. The first match decides:
// "$$x$$$" closes after x and leaves a literal "$", as on GitHub.
func findCloser(line []byte, n int) (int, bool) {
	for i := 0; i < len(line); i++ {
		switch line[i] {
		case '\\':
			i++
		case '$':
			if n == 1 || (i+1 < len(line) && line[i+1] == '$') {
				return i, true
			}
		}
	}
	return 0, false
}

// isInlineMathCloser reports whether the $ at line[i] can close inline
// math. As on GitHub, it may not be followed by a letter or digit
// ("$x$5" and "$x$y" stay text), and when preceded by whitespace it must
// be followed by punctuation or the end of the line: "$x $." and "$x $"
// close, "$x $ b" and "$a $x" do not. A line start counts as whitespace
// before; a line break counts as the end of the line.
func isInlineMathCloser(line []byte, i int) bool {
	next := byte('\n')
	if i+1 < len(line) {
		next = line[i+1]
	}
	if next == '\n' || next == '\r' {
		return true
	}
	if util.IsAlphaNumeric(next) {
		return false
	}
	spaceBefore := i == 0 || util.IsSpace(line[i-1])
	return !spaceBefore || !util.IsSpace(next)
}

type mathRenderer struct{}

func (mathRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(kindMath, renderMath)
	reg.Register(kindMathBlock, renderMathBlock)
}

func renderMathBlock(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	if _, err := w.WriteString(`<div class="math display" data-math="display"`); err != nil {
		// coverage: BufWriter wraps an in-memory buffer; its writes do
		// not fail.
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
		// coverage: BufWriter wraps an in-memory buffer; its writes do
		// not fail.
		return ast.WalkStop, err
	}
	return ast.WalkSkipChildren, nil
}

func renderMath(w util.BufWriter, source []byte, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	mode := "inline"
	if node, ok := n.(*mathNode); ok && node.display {
		mode = "display"
	}
	out := []byte(`<span class="math ` + mode + `" data-math="` + mode + `">`)
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		if t, ok := c.(*ast.Text); ok {
			out = append(out, util.EscapeHTML(t.Segment.Value(source))...)
		}
	}
	out = append(out, "</span>"...)
	if _, err := w.Write(out); err != nil {
		// coverage: BufWriter wraps an in-memory buffer; its writes do
		// not fail.
		return ast.WalkStop, err
	}
	return ast.WalkSkipChildren, nil
}
