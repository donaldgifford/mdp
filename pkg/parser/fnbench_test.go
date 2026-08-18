package parser_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/donaldgifford/mdp/pkg/parser"
)

// BenchmarkFootnoteOverhead measures what the footnote extension costs
// when it is enabled but unused. Footnotes default to on, so every
// caller pays this -- including the overwhelming majority of documents
// that contain no footnotes at all. The on/off pair over identical
// input isolates the extension from the rest of the pipeline.
//
// Measured at 1000 lines on an M5 Max, -benchtime=500x -count=12:
// identical allocations (13558), and means of 985634 ns/op enabled vs
// 986190 ns/op disabled -- a 0.06% difference against a within-variant
// spread of 3.8-4.4%, so the delta sits roughly 60x below the noise
// floor. goldmark's footnote block parser only engages on a "[^" at
// the start of a line, so an unused extension does effectively
// nothing. This is the evidence for defaulting the option to on.
//
// Short runs mislead here: at -benchtime=50x the first sub-benchmark
// absorbs warm-up and shows a spurious 15% gap. Use a large benchtime
// and -count when re-measuring.
func BenchmarkFootnoteOverhead(b *testing.B) {
	md := generateMarkdown(1000)

	benchmarks := []struct {
		name string
		p    *parser.Parser
	}{
		{"on_default", parser.New()},
		{"off", parser.New(parser.WithFootnotes(false))},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			b.SetBytes(int64(len(md)))
			b.ReportAllocs()

			for b.Loop() {
				if _, err := bm.p.Render(md); err != nil {
					b.Fatalf("render: %v", err)
				}
			}
		})
	}
}

// BenchmarkFootnoteRender measures rendering a document that actually
// uses footnotes, scaling the reference count so the cost of the
// extension's collect-and-relocate pass is visible rather than lost in
// the surrounding prose.
func BenchmarkFootnoteRender(b *testing.B) {
	p := parser.New()

	for _, count := range []int{10, 100, 500} {
		md := generateFootnoteMarkdown(count)

		b.Run(fmt.Sprintf("refs_%d", count), func(b *testing.B) {
			b.SetBytes(int64(len(md)))
			b.ReportAllocs()

			for b.Loop() {
				if _, err := p.Render(md); err != nil {
					b.Fatalf("render: %v", err)
				}
			}
		})
	}
}

// generateFootnoteMarkdown builds a document with n referenced
// footnotes, all defined at the end so the shape matches how footnotes
// are conventionally written.
func generateFootnoteMarkdown(n int) []byte {
	var sb strings.Builder

	sb.WriteString("# Footnote Benchmark\n\n")
	for i := range n {
		fmt.Fprintf(&sb, "Paragraph %d with a reference.[^%d]\n\n", i, i)
	}
	for i := range n {
		fmt.Fprintf(&sb, "[^%d]: Definition %d with a [link](https://example.com/%d).\n", i, i, i)
	}

	return []byte(sb.String())
}
