package parser

import (
	"fmt"

	"github.com/yuin/goldmark/ast"
	gmparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// lineAnnotator is a goldmark AST transformer that adds
// data-source-line attributes to block-level elements.
type lineAnnotator struct{}

// Transform walks the AST and annotates block-level elements with their
// source line number via data-source-line attributes.
func (*lineAnnotator) Transform(doc *ast.Document, reader text.Reader, _ gmparser.Context) {
	source := reader.Source()
	lineStarts := buildLineIndex(source)

	//nolint:errcheck,gosec // Walk callback never returns an error.
	ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if node.Type() != ast.TypeBlock {
			return ast.WalkContinue, nil
		}
		if node.Kind() == ast.KindDocument {
			return ast.WalkContinue, nil
		}

		seg := firstSegment(node)
		if seg.Start < 0 {
			return ast.WalkContinue, nil
		}

		line := offsetToLine(lineStarts, seg.Start)
		node.SetAttributeString("data-source-line", fmt.Sprintf("%d", line))

		return ast.WalkContinue, nil
	})
}

// firstSegment returns the first text segment of a node.
func firstSegment(node ast.Node) text.Segment {
	if node.Lines().Len() > 0 {
		return node.Lines().At(0)
	}
	// For container nodes (e.g. list items, blockquotes), try first child.
	if node.HasChildren() {
		child := node.FirstChild()
		if child != nil && child.Lines().Len() > 0 {
			return child.Lines().At(0)
		}
	}
	return text.Segment{Start: -1}
}

// buildLineIndex returns the byte offset of each line start.
func buildLineIndex(source []byte) []int {
	starts := []int{0}
	for i, b := range source {
		if b == '\n' && i+1 < len(source) {
			starts = append(starts, i+1)
		}
	}
	return starts
}

// offsetToLine converts a byte offset to a 1-based line number.
func offsetToLine(lineStarts []int, offset int) int {
	lo, hi := 0, len(lineStarts)-1
	for lo <= hi {
		mid := (lo + hi) / 2
		if lineStarts[mid] <= offset {
			lo = mid + 1
		} else {
			hi = mid - 1
		}
	}
	return lo // 1-based: lineStarts[0]=0 → line 1.
}
