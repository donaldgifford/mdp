package parser_test

import (
	"os"
	"strings"
	"testing"

	"github.com/donaldgifford/mdp/internal/parser"
)

func TestRender_Heading(t *testing.T) {
	t.Parallel()

	p := parser.New()
	html, err := p.Render([]byte("# Hello World"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(html)
	if !strings.Contains(got, "<h1") {
		t.Errorf("expected <h1> tag, got: %s", got)
	}
	if !strings.Contains(got, "Hello World") {
		t.Errorf("expected heading text, got: %s", got)
	}
}

func TestRender_GFMTable(t *testing.T) {
	t.Parallel()

	md := []byte(`| Name | Age |
| ---- | --- |
| Alice | 30 |
| Bob | 25 |`)

	p := parser.New()
	html, err := p.Render(md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(html)
	for _, want := range []string{"<table>", "<th>", "<td>", "Alice", "Bob"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got: %s", want, got)
		}
	}
}

func TestRender_GFMTaskList(t *testing.T) {
	t.Parallel()

	md := []byte("- [x] Done\n- [ ] Not done")

	p := parser.New()
	html, err := p.Render(md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(html)
	if !strings.Contains(got, `type="checkbox"`) {
		t.Errorf("expected checkbox input, got: %s", got)
	}
}

func TestRender_GFMStrikethrough(t *testing.T) {
	t.Parallel()

	p := parser.New()
	html, err := p.Render([]byte("~~deleted~~"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(html)
	if !strings.Contains(got, "<del>") {
		t.Errorf("expected <del> tag, got: %s", got)
	}
}

func TestRender_SyntaxHighlighting(t *testing.T) {
	t.Parallel()

	md := []byte("```go\nfunc main() {}\n```")

	p := parser.New()
	html, err := p.Render(md)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(html)
	// With CSS-class mode, chroma adds class attributes.
	if !strings.Contains(got, "chroma") {
		t.Errorf("expected chroma syntax classes, got: %s", got)
	}
}

func TestRender_WithoutGFM(t *testing.T) {
	t.Parallel()

	p := parser.New(parser.WithGFM(false))
	html, err := p.Render([]byte("~~not deleted~~"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(html)
	if strings.Contains(got, "<del>") {
		t.Errorf("expected no <del> tag with GFM disabled, got: %s", got)
	}
}

func TestRender_MarkdownFixture(t *testing.T) {
	t.Parallel()

	fixture, err := os.ReadFile("testdata/fixture.md")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	p := parser.New()
	html, err := p.Render(fixture)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(html)
	checks := []struct {
		name string
		want string
	}{
		{"heading", "<h1"},
		{"bold", "<strong>"},
		{"italic", "<em>"},
		{"link", `href="https://example.com"`},
		{"code block", "<pre"},
		{"table", "<table>"},
		{"task list", `type="checkbox"`},
		{"blockquote", "<blockquote>"},
		{"horizontal rule", "<hr"},
	}

	for _, tc := range checks {
		if !strings.Contains(got, tc.want) {
			t.Errorf("expected %s (%q) in output", tc.name, tc.want)
		}
	}
}
