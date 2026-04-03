package parser_test

import (
	"fmt"
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
	for _, want := range []string{"<table", "<th", "<td", "Alice", "Bob"} {
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

func TestRender_GitHubCallout(t *testing.T) {
	t.Parallel()

	tests := []struct {
		alertType string
		wantClass string
	}{
		{"NOTE", "callout-note"},
		{"TIP", "callout-tip"},
		{"IMPORTANT", "callout-important"},
		{"WARNING", "callout-warning"},
		{"CAUTION", "callout-caution"},
	}

	p := parser.New()
	for _, tc := range tests {
		t.Run(tc.alertType, func(t *testing.T) {
			t.Parallel()

			md := fmt.Sprintf("> [!%s]\n> Callout body text", tc.alertType)
			html, err := p.Render([]byte(md))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := string(html)
			if !strings.Contains(got, tc.wantClass) {
				t.Errorf("expected class %q in output, got: %s", tc.wantClass, got)
			}
			if !strings.Contains(got, "callout-title-text") {
				t.Errorf("expected callout-title-text in output, got: %s", got)
			}
			if !strings.Contains(got, "Callout body text") {
				t.Errorf("expected body text in output, got: %s", got)
			}
		})
	}
}

func TestRender_CalloutPreservesBlockquote(t *testing.T) {
	t.Parallel()

	p := parser.New()
	html, err := p.Render([]byte("> This is a plain blockquote"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(html)
	if !strings.Contains(got, "<blockquote") {
		t.Errorf("expected <blockquote> for plain quote, got: %s", got)
	}
	if strings.Contains(got, "callout") {
		t.Errorf("plain blockquote should not contain callout class, got: %s", got)
	}
}

func TestRender_CalloutWithNestedContent(t *testing.T) {
	t.Parallel()

	md := "> [!NOTE]\n> Text with `inline code` and:\n>\n> - list item one\n> - list item two"

	p := parser.New()
	html, err := p.Render([]byte(md))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(html)
	for _, want := range []string{"callout-note", "<code>inline code</code>", "<li"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got: %s", want, got)
		}
	}
}

func TestRender_CalloutDisabled(t *testing.T) {
	t.Parallel()

	p := parser.New(parser.WithCallouts(false))
	html, err := p.Render([]byte("> [!NOTE]\n> Should be a plain blockquote"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(html)
	if !strings.Contains(got, "<blockquote") {
		t.Errorf("expected <blockquote> when callouts disabled, got: %s", got)
	}
	if strings.Contains(got, "callout") {
		t.Errorf("callout class should not appear when disabled, got: %s", got)
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
		{"table", "<table"},
		{"task list", `type="checkbox"`},
		{"blockquote", "<blockquote"},
		{"horizontal rule", "<hr"},
	}

	for _, tc := range checks {
		if !strings.Contains(got, tc.want) {
			t.Errorf("expected %s (%q) in output", tc.name, tc.want)
		}
	}
}
