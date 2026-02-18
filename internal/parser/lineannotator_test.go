package parser_test

import (
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/donaldgifford/mdp/internal/parser"
)

func TestLineAnnotator_AddsDataSourceLine(t *testing.T) {
	t.Parallel()

	md := []byte(`# Heading

Paragraph text.

> Blockquote.

- List item one
- List item two

---

` + "```go\nfunc main() {}\n```\n" + `
| Col A | Col B |
| ----- | ----- |
| val1  | val2  |
`)

	p := parser.New()
	html, err := p.Render(md)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(html)

	// Every block-level element should have a data-source-line attribute.
	// Note: <hr> (thematic break) is not annotated because goldmark's AST
	// does not store text segments for thematic break nodes.
	checks := []struct {
		tag  string
		line int
	}{
		{"h1", 1},
		{"p", 3},
		{"blockquote", 5},
		{"li", 7},
	}

	re := regexp.MustCompile(`data-source-line="(\d+)"`)

	for _, tc := range checks {
		// Find the tag opening.
		idx := strings.Index(got, "<"+tc.tag)
		if idx < 0 {
			t.Errorf("expected <%s> tag in output", tc.tag)
			continue
		}

		// Extract data-source-line from this tag.
		tagEnd := strings.Index(got[idx:], ">")
		if tagEnd < 0 {
			t.Errorf("<%s> tag not closed", tc.tag)
			continue
		}

		tagStr := got[idx : idx+tagEnd+1]
		match := re.FindStringSubmatch(tagStr)
		if match == nil {
			t.Errorf("<%s> missing data-source-line attribute: %s", tc.tag, tagStr)
			continue
		}

		lineNum, _ := strconv.Atoi(match[1])
		if lineNum != tc.line {
			t.Errorf("<%s> data-source-line=%d, want %d", tc.tag, lineNum, tc.line)
		}
	}
}

func TestLineAnnotator_NoAnnotationOnInlineElements(t *testing.T) {
	t.Parallel()

	md := []byte("Hello **bold** and *italic* text.\n")

	p := parser.New()
	html, err := p.Render(md)
	if err != nil {
		t.Fatalf("render: %v", err)
	}

	got := string(html)

	// Inline elements should NOT have data-source-line.
	for _, tag := range []string{"<strong", "<em"} {
		idx := strings.Index(got, tag)
		if idx < 0 {
			continue
		}
		tagEnd := strings.Index(got[idx:], ">")
		tagStr := got[idx : idx+tagEnd+1]
		if strings.Contains(tagStr, "data-source-line") {
			t.Errorf("inline element %s should not have data-source-line: %s", tag, tagStr)
		}
	}

	// But the wrapping <p> should have it.
	if !strings.Contains(got, `<p data-source-line="1"`) {
		t.Errorf("expected <p data-source-line=\"1\"> in output, got: %s", got)
	}
}
