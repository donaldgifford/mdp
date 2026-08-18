package parser_test

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"

	"go.abhg.dev/goldmark/mermaid"

	"github.com/donaldgifford/mdp/pkg/parser"
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

func TestRender_Footnote(t *testing.T) {
	t.Parallel()

	p := parser.New()
	html, err := p.Render([]byte("Text.[^1]\n\n[^1]: The note.\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(html)
	for _, want := range []string{
		`<sup id="fnref:1">`,
		`href="#fn:1"`,
		`class="footnote-ref"`,
		`<li id="fn:1"`,
		`class="footnotes"`,
		`class="footnote-backref"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got: %s", want, got)
		}
	}
}

func TestRender_FootnoteWithLink(t *testing.T) {
	t.Parallel()

	md := "Text.[^1]\n\n[^1]: See [the guide](https://example.com/guide) for details.\n"

	p := parser.New()
	html, err := p.Render([]byte(md))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(html)
	for _, want := range []string{
		`<a href="https://example.com/guide">the guide</a>`,
		`class="footnotes"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got: %s", want, got)
		}
	}
}

func TestRender_FootnoteMultiParagraph(t *testing.T) {
	t.Parallel()

	md := "Text.[^long]\n\n[^long]:\n    First paragraph.\n\n    Second paragraph.\n"

	p := parser.New()
	html, err := p.Render([]byte(md))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(html)
	for _, want := range []string{"First paragraph.", "Second paragraph."} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got: %s", want, got)
		}
	}

	// Both paragraphs belong to a single <li>.
	if n := strings.Count(got, "<li id="); n != 1 {
		t.Errorf("expected 1 footnote list item, got %d: %s", n, got)
	}

	// The backlink attaches to the last paragraph, not the first.
	backref := strings.Index(got, `class="footnote-backref"`)
	second := strings.Index(got, "Second paragraph.")
	if backref < second {
		t.Errorf("expected backlink after the last paragraph, got: %s", got)
	}
}

func TestRender_FootnoteRepeatedReference(t *testing.T) {
	t.Parallel()

	md := "First.[^a]\n\nSecond.[^a]\n\n[^a]: Shared note.\n"

	p := parser.New()
	html, err := p.Render([]byte(md))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(html)
	for _, want := range []string{`id="fnref:1"`, `id="fnref1:1"`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got: %s", want, got)
		}
	}

	// One definition, but a backlink per reference.
	if n := strings.Count(got, `class="footnote-backref"`); n != 2 {
		t.Errorf("expected 2 backlinks for 2 references, got %d: %s", n, got)
	}
	if n := strings.Count(got, "<li id="); n != 1 {
		t.Errorf("expected 1 footnote list item, got %d: %s", n, got)
	}
}

// TestRender_FootnoteNamedLabel verifies that named labels render as
// sequential integers numbered by order of first *reference*, not by
// the order the definitions appear in the source.
func TestRender_FootnoteNamedLabel(t *testing.T) {
	t.Parallel()

	md := "Alpha.[^zeta] Beta.[^alpha]\n\n" +
		"[^alpha]: Defined second in source.\n" +
		"[^zeta]: Defined first in source.\n"

	p := parser.New()
	html, err := p.Render([]byte(md))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(html)

	// [^zeta] is referenced first, so it becomes fn:1 even though its
	// definition comes second in the source.
	zeta := strings.Index(got, "Defined first in source.")
	alpha := strings.Index(got, "Defined second in source.")
	if zeta < 0 || alpha < 0 {
		t.Fatalf("expected both definitions in output, got: %s", got)
	}
	if zeta > alpha {
		t.Errorf("expected first-referenced footnote to render first, got: %s", got)
	}

	// Labels are replaced by integers in the rendered references.
	for _, want := range []string{`href="#fn:1"`, `href="#fn:2"`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in output, got: %s", want, got)
		}
	}
	if strings.Contains(got, "zeta") || strings.Contains(got, "alpha") {
		t.Errorf("expected labels replaced by integers, got: %s", got)
	}
}

// TestRender_FootnoteDisabled verifies WithFootnotes(false) emits no
// footnote markup.
//
// Note the second case: with the extension off, a definition whose
// body is a single word is a valid CommonMark *link reference
// definition* ([^1] is a legal link label), so goldmark turns the
// reference into an anchor. That is upstream CommonMark behavior, not
// mdp's — the invariant that matters here is only that no footnote
// markup is produced.
func TestRender_FootnoteDisabled(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		md          string
		wantLiteral bool
	}{
		{
			name:        "multi-word definition stays literal text",
			md:          "Text.[^1]\n\n[^1]: The note text.\n",
			wantLiteral: true,
		},
		{
			name:        "single-word definition is a link reference definition",
			md:          "Text.[^1]\n\n[^1]: Note.\n",
			wantLiteral: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			p := parser.New(parser.WithFootnotes(false))
			html, err := p.Render([]byte(tc.md))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			got := string(html)
			for _, unwanted := range []string{"footnotes", "footnote-ref", "footnote-backref"} {
				if strings.Contains(got, unwanted) {
					t.Errorf("expected no %q when footnotes disabled, got: %s", unwanted, got)
				}
			}
			if tc.wantLiteral && !strings.Contains(got, "[^1]") {
				t.Errorf("expected literal [^1] when footnotes disabled, got: %s", got)
			}
		})
	}
}

func TestRender_FootnoteUndefinedReference(t *testing.T) {
	t.Parallel()

	p := parser.New()
	html, err := p.Render([]byte("Text.[^missing]\n"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(html)
	if !strings.Contains(got, "[^missing]") {
		t.Errorf("expected undefined reference to stay literal, got: %s", got)
	}
	if strings.Contains(got, "footnotes") {
		t.Errorf("expected no footnotes section for an undefined reference, got: %s", got)
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

// TestParser_AllOptionsOff verifies the disable-side of each toggle.
// The default Parser enables every extension, so the only way to
// exercise the false branch of the With* setters is to call them
// explicitly here.
func TestParser_AllOptionsOff(t *testing.T) {
	t.Parallel()

	p := parser.New(
		parser.WithGFM(false),
		parser.WithSyntaxHighlighting(false),
		parser.WithMermaid(false),
		parser.WithMath(false),
		parser.WithCallouts(false),
		parser.WithFootnotes(false),
	)
	html, err := p.Render([]byte("# Plain"))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(string(html), "<h1") {
		t.Errorf("expected heading even with all extensions off, got: %s", html)
	}
}

const mermaidFixture = "```mermaid\ngraph TD\nA-->B\n```\n"

// TestParser_WithMermaidRenderMode_Client asserts the default client
// render mode emits a <pre class="mermaid"> placeholder for the
// browser to render with mermaid.js.
func TestParser_WithMermaidRenderMode_Client(t *testing.T) {
	t.Parallel()

	p := parser.New(parser.WithMermaidRenderMode(mermaid.RenderModeClient))
	html, err := p.Render([]byte(mermaidFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(html)
	if !strings.Contains(got, `<pre class="mermaid"`) {
		t.Errorf("client mode should emit <pre class=\"mermaid\">, got: %s", got)
	}
}

// TestParser_WithMermaidRenderMode_Server asserts the server render
// mode compiles mermaid blocks to inline <svg>. Requires the mmdc CLI
// to be installed; skipped otherwise so the suite stays green on
// machines that don't have it.
func TestParser_WithMermaidRenderMode_Server(t *testing.T) {
	t.Parallel()
	if _, err := exec.LookPath("mmdc"); err != nil {
		t.Skip("mmdc CLI not on $PATH; skipping server-side mermaid render test")
	}

	p := parser.New(parser.WithMermaidRenderMode(mermaid.RenderModeServer))
	html, err := p.Render([]byte(mermaidFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := string(html)
	if !strings.Contains(got, "<svg") {
		t.Errorf("server mode should emit <svg>, got: %s", got)
	}
	if strings.Contains(got, `<pre class="mermaid"`) {
		t.Errorf("server mode should not emit client-mode placeholder, got: %s", got)
	}
}

// TestParser_ConcurrentRender_NoRace guards the public-API contract
// that a single *Parser is safe to share across goroutines. The
// upstream gm-alert-callouts extension holds a shared cases.Caser
// that is not safe for concurrent use — see INV-0003. This test
// must pass under `-race`; failure means the local mitigation in
// Parser.Render has regressed.
func TestParser_ConcurrentRender_NoRace(t *testing.T) {
	t.Parallel()

	const (
		goroutines = 32
		iterations = 16
	)

	// Callout-heavy input: every iteration hits the gm-alert-callouts
	// renderer that owns the shared cases.Caser.
	src := []byte(strings.Join([]string{
		"> [!NOTE]\n> note body",
		"> [!TIP]\n> tip body",
		"> [!IMPORTANT]\n> important body",
		"> [!WARNING]\n> warning body",
		"> [!CAUTION]\n> caution body",
	}, "\n\n"))

	p := parser.New()

	var wg sync.WaitGroup
	errs := make(chan error, goroutines*iterations)

	for range goroutines {
		wg.Go(func() {
			for range iterations {
				if _, err := p.Render(src); err != nil {
					errs <- err
					return
				}
			}
		})
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent Render returned error: %v", err)
	}
}
