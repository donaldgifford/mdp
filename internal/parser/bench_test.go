package parser_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/donaldgifford/mdp/internal/parser"
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
