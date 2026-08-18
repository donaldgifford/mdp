package parser_test

import (
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/donaldgifford/mdp/pkg/parser"
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

// footnotesMarker opens the endnote list goldmark appends when the
// footnote extension is active.
const footnotesMarker = `<div class="footnotes"`

var sourceLineRe = regexp.MustCompile(`data-source-line="(\d+)"`)

// sourceLines returns every data-source-line value in html, in
// document order.
func sourceLines(t *testing.T, html string) []int {
	t.Helper()

	matches := sourceLineRe.FindAllStringSubmatch(html, -1)
	lines := make([]int, 0, len(matches))
	for _, m := range matches {
		n, err := strconv.Atoi(m[1])
		if err != nil {
			t.Fatalf("unparsable data-source-line %q: %v", m[1], err)
		}
		lines = append(lines, n)
	}
	return lines
}

// TestLineAnnotator_FootnoteOrdering pins the two shapes that make
// data-source-line non-monotonic in document order. Both are the
// reason assets/preview.js must exclude the .footnotes subtree before
// scanning for a scroll target.
func TestLineAnnotator_FootnoteOrdering(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		md   string
		want []int
	}{
		{
			// Definition on line 5 renders after body content from
			// lines 7 and 9.
			name: "mid-document definition",
			md: "# Title\n\n" + // 1
				"Intro with a note.[^a]\n\n" + // 3
				"[^a]: The note text.\n\n" + // 5
				"## Section\n\n" + // 7
				"Body paragraph.\n", // 9
			want: []int{1, 3, 7, 9, 5, 5},
		},
		{
			// Both definitions sit at the end of the file, but [^zeta]
			// is referenced first so it becomes fn:1 and renders
			// before [^alpha] -- descending source lines 4 then 3.
			name: "reference order differs from definition order",
			md: "Alpha.[^zeta] Beta.[^alpha]\n\n" + // 1
				"[^alpha]: Defined second in source.\n" + // 3
				"[^zeta]: Defined first in source.\n", // 4
			want: []int{1, 4, 4, 3, 3},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			html, err := parser.New().Render([]byte(tc.md))
			if err != nil {
				t.Fatalf("render: %v", err)
			}

			got := sourceLines(t, string(html))
			if len(got) != len(tc.want) {
				t.Fatalf("got %v (len %d), want %v (len %d)\n%s",
					got, len(got), tc.want, len(tc.want), html)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("document-order lines = %v, want %v\n%s", got, tc.want, html)
				}
			}
		})
	}
}

// TestLineAnnotator_NonFootnoteOrderIsMonotonic asserts the invariant
// findScrollTarget in assets/preview.js depends on: outside the
// .footnotes subtree, data-source-line values are non-decreasing in
// document order. If this breaks, scroll sync silently targets the
// wrong element.
func TestLineAnnotator_NonFootnoteOrderIsMonotonic(t *testing.T) {
	t.Parallel()

	// The shared fixture carries a deliberately mid-document
	// definition, so it exercises the invariant on realistic input and
	// cannot drift out of sync with it unnoticed.
	fixture, err := os.ReadFile("testdata/fixture.md")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	tests := []struct {
		name string
		md   string
	}{
		{
			name: "mid-document definition",
			md: "# Title\n\nIntro.[^a]\n\n[^a]: Note text.\n\n" +
				"## Section\n\nBody paragraph.\n",
		},
		{
			name: "reference order differs from definition order",
			md: "Alpha.[^zeta] Beta.[^alpha]\n\n" +
				"[^alpha]: Defined second in source.\n" +
				"[^zeta]: Defined first in source.\n",
		},
		{
			name: "definitions at end of file",
			md: "# Title\n\nIntro.[^a]\n\n## Section\n\nBody.[^b]\n\n" +
				"[^a]: First note.\n[^b]: Second note.\n",
		},
		{
			name: "no footnotes at all",
			md:   "# Title\n\nParagraph.\n\n## Section\n\nAnother paragraph.\n",
		},
		{
			name: "testdata/fixture.md",
			md:   string(fixture),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			html, err := parser.New().Render([]byte(tc.md))
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			got := string(html)

			// Splitting on the marker is only valid while goldmark
			// emits exactly one endnote list, as the final block.
			if n := strings.Count(got, footnotesMarker); n > 1 {
				t.Fatalf("expected at most one %s, got %d -- the "+
					"split below no longer isolates the body: %s",
					footnotesMarker, n, got)
			}

			body, _, _ := strings.Cut(got, footnotesMarker)
			lines := sourceLines(t, body)
			if len(lines) == 0 {
				t.Fatalf("expected annotated body content, got: %s", got)
			}
			for i := 1; i < len(lines); i++ {
				if lines[i-1] > lines[i] {
					t.Errorf("body data-source-line values %v decrease at index %d "+
						"(%d > %d); findScrollTarget's break would select the wrong "+
						"element:\n%s", lines, i, lines[i-1], lines[i], got)
				}
			}
		})
	}
}

// TestLineAnnotator_FootnoteListUnannotated verifies the endnote
// wrapper itself carries no data-source-line. FootnoteList has no
// source segment of its own, so lineAnnotator intentionally skips it
// (see the seg.Start < 0 guard in lineannotator.go).
func TestLineAnnotator_FootnoteListUnannotated(t *testing.T) {
	t.Parallel()

	html, err := parser.New().Render([]byte("Text.[^a]\n\n[^a]: The note.\n"))
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	got := string(html)

	idx := strings.Index(got, footnotesMarker)
	if idx < 0 {
		t.Fatalf("expected a footnotes section, got: %s", got)
	}
	tagEnd := strings.Index(got[idx:], ">")
	if tagEnd < 0 {
		t.Fatalf("footnotes div not closed: %s", got)
	}

	tag := got[idx : idx+tagEnd+1]
	if strings.Contains(tag, "data-source-line") {
		t.Errorf("footnotes wrapper should not be annotated: %s", tag)
	}

	// The definition item inside it still is.
	if !strings.Contains(got, `<li id="fn:1" data-source-line="3">`) {
		t.Errorf("expected the footnote item annotated with its "+
			"definition line, got: %s", got)
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
