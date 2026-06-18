package parser_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/donaldgifford/mdp/pkg/parser"
)

func BenchmarkRender(b *testing.B) {
	sizes := []int{100, 1000, 5000, 10000}
	for _, size := range sizes {
		md := generateMarkdown(size)
		p := parser.New()

		b.Run(fmt.Sprintf("lines_%d", size), func(b *testing.B) {
			b.SetBytes(int64(len(md)))
			for b.Loop() {
				if _, err := p.Render(md); err != nil {
					b.Fatalf("render: %v", err)
				}
			}
		})
	}
}

// generateMarkdown creates a markdown document of approximately n lines.
func generateMarkdown(n int) []byte {
	var sb strings.Builder
	sb.WriteString("# Benchmark Document\n\n")

	for i := 0; i < n/10; i++ {
		fmt.Fprintf(&sb, "## Section %d\n\n", i)
		sb.WriteString("This is a paragraph with **bold**, *italic*, and `code`.\n\n")
		sb.WriteString("| Column A | Column B |\n| --- | --- |\n")
		fmt.Fprintf(&sb, "| value %d | data %d |\n\n", i, i)
		sb.WriteString("- Item one\n- Item two\n- Item three\n\n")
	}

	return []byte(sb.String())
}

// mixedSource exercises the same extension set real consumers hit:
// GFM table + fenced code block + callout. Used by the mutex-cost
// benchmarks added for INV-0003 / IMPL-0005.
var mixedSource = []byte(`# Bench

| col1 | col2 |
|------|------|
|   a  |   b  |
|   c  |   d  |

` + "```go\n" + `func main() {
    fmt.Println("hello")
}
` + "```\n" + `

> [!NOTE]
> This is a callout that exercises the gm-alert-callouts renderer.
`)

func BenchmarkRenderMixed(b *testing.B) {
	p := parser.New()
	b.SetBytes(int64(len(mixedSource)))
	for b.Loop() {
		if _, err := p.Render(mixedSource); err != nil {
			b.Fatalf("render: %v", err)
		}
	}
}

func BenchmarkRenderMixedParallel(b *testing.B) {
	p := parser.New()
	b.SetBytes(int64(len(mixedSource)))
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := p.Render(mixedSource); err != nil {
				b.Fatalf("render: %v", err)
			}
		}
	})
}
