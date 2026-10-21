package parser_test

import (
	"strings"
	"testing"

	"github.com/donaldgifford/mdp/pkg/parser"
)

// TestRender_Math checks math parsing against GitHub's behavior for the
// same input (via the GitHub markdown API), except where noted.
func TestRender_Math(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		src  string
		want []string
		deny []string
	}{
		{
			name: "inline",
			src:  "Energy $E = mc^2$ here.",
			want: []string{`<span class="math inline" data-math="inline">E = mc^2</span>`},
		},
		{
			name: "markdown syntax inside math is left alone",
			src:  `$a_1 * b_2 * c_3$ and $\{0, 1\}$`,
			want: []string{
				`<span class="math inline" data-math="inline">a_1 * b_2 * c_3</span>`,
				`<span class="math inline" data-math="inline">\{0, 1\}</span>`,
			},
		},
		{
			name: "html is escaped",
			src:  "$a<b$",
			want: []string{`<span class="math inline" data-math="inline">a&lt;b</span>`},
		},
		{
			name: "currency is not math",
			src:  "costs $5 and $10 today",
			deny: []string{"math"},
		},
		{
			name: "opener followed by digit is math",
			src:  `$2\pi r$ ok`,
			want: []string{`<span class="math inline" data-math="inline">2\pi r</span> ok`},
		},
		{
			name: "opener followed by space is not math",
			src:  "a $ x$ b",
			deny: []string{"math"},
		},
		{
			name: "closer with space on both sides is not math",
			src:  "a $x $ b",
			deny: []string{"math"},
		},
		{
			name: "closer after trailing space before punctuation",
			src:  `and $\Vert a\Vert $. ok`,
			want: []string{`<span class="math inline" data-math="inline">\Vert a\Vert </span>.`},
		},
		{
			name: "closer after trailing space at end of line",
			src:  "see $x $\nnext line\n",
			want: []string{`<span class="math inline" data-math="inline">x </span>`},
		},
		{
			name: "closer after trailing space before a letter is not math",
			src:  "see $a $x end",
			deny: []string{"math"},
		},
		{
			name: "closer followed by digit is not math",
			src:  "$x$5 then",
			deny: []string{"math"},
		},
		{
			name: "closer followed by letter is not math",
			src:  "see $x$y end",
			deny: []string{"math"},
		},
		{
			// The first $ after the opener decides. Scanning on past the
			// rejected closer would pair "$x" with the "$" of "$y" and
			// render "x$5 and " as math while leaving "y$" as text.
			name: "rejected closer ends the expression",
			src:  "$x$5 and $y$ end",
			want: []string{`$x$5 and <span class="math inline" data-math="inline">y</span> end`},
		},
		{
			// GitHub leaves "$b$" as text here, apparently because its
			// opener follows another dollar. mdp has no such rule and
			// renders both; the input is contrived enough to accept.
			name: "adjacent expressions both render",
			src:  "see $a$$b$ end",
			want: []string{
				`<span class="math inline" data-math="inline">a</span>` +
					`<span class="math inline" data-math="inline">b</span> end`,
			},
		},
		{
			name: "rejected closer ends the expression after a digit opener",
			src:  "$5 and $y$ end",
			want: []string{`$5 and <span class="math inline" data-math="inline">y</span> end`},
		},
		{
			// GitHub treats "\$5 ... $" as math; mdp keeps \$ a literal
			// dollar as CommonMark specifies.
			name: "escaped dollar",
			src:  `price \$5 then $\$x$`,
			want: []string{`price $5 then <span class="math inline" data-math="inline">\$x</span>`},
		},
		{
			name: "unclosed dollar is text",
			src:  "lone $x here",
			deny: []string{"math"},
		},
		{
			name: "code span wins",
			src:  "use `$x$` literally",
			want: []string{"<code>$x$</code>"},
			deny: []string{"math"},
		},
		{
			name: "display math inside a paragraph",
			src:  "top $$x$$ inline",
			want: []string{`top <span class="math display" data-math="display">x</span> inline`},
		},
		{
			// Before the first-closer rule in Open, this was one block
			// holding "a$$ and $$b", which KaTeX shows as an error.
			name: "two display expressions on one line",
			src:  "$$a$$ and $$b$$\n",
			want: []string{
				`<span class="math display" data-math="display">a</span> and ` +
					`<span class="math display" data-math="display">b</span>`,
			},
			deny: []string{"<div"},
		},
		{
			name: "first closer decides and a trailing dollar stays text",
			src:  "$$x$$$\n",
			want: []string{`<span class="math display" data-math="display">x</span>$`},
			deny: []string{"<div"},
		},
		{
			name: "single-line display block",
			src:  "a\n\n$$x^2$$\n\nafter\n",
			want: []string{
				`<div class="math display" data-math="display" data-source-line="3">x^2</div>`,
				`<p data-source-line="5">after</p>`,
			},
		},
		{
			// GitHub breaks this block into a list; mdp keeps it math.
			name: "block lines that look like markdown stay math",
			src:  "$$\na\n+ b\n= c\n$$\n\nafter\n",
			want: []string{
				"<div class=\"math display\" data-math=\"display\" data-source-line=\"2\">a\n+ b\n= c\n</div>",
				`<p data-source-line="7">after</p>`,
			},
			deny: []string{"<li", "<h1"},
		},
		{
			name: "block in blockquote",
			src:  "> text\n>\n> $$\n> x_1 * y_2\n> $$\n>\n> after $y$\n",
			want: []string{
				"<div class=\"math display\" data-math=\"display\" data-source-line=\"4\">x_1 * y_2\n</div>",
				`<p data-source-line="7">after <span class="math inline" data-math="inline">y</span></p>`,
			},
			deny: []string{"&gt;", "> x"},
		},
		{
			name: "single-line block in blockquote",
			src:  "> $$x^2$$\n>\n> after\n",
			want: []string{
				`<div class="math display" data-math="display" data-source-line="1">x^2</div>`,
				`<p data-source-line="3">after</p>`,
			},
		},
		{
			name: "multi-line inline math in blockquote",
			src:  "> see $a +\n> b$ here\n",
			want: []string{"<span class=\"math inline\" data-math=\"inline\">a +\nb</span> here"},
		},
		{
			name: "block interrupts a paragraph",
			src:  "text\n$$\nx\n$$\ntail\n",
			want: []string{
				`<p data-source-line="1">text</p>`,
				"<div class=\"math display\" data-math=\"display\" data-source-line=\"3\">x\n</div>",
				`<p data-source-line="5">tail</p>`,
			},
		},
		{
			// GitHub renders all of this as text. mdp closes the
			// expression opened at the end of line 1 instead of treating
			// the bare "$$" on line 3 as a block opener, which would have
			// swallowed everything after it into one block.
			name: "bare dollar-dollar closing a paragraph expression is not a block",
			src:  "text $$\nE = mc^2\n$$\nafter\n\nmore\n",
			want: []string{
				"text <span class=\"math display\" data-math=\"display\">\nE = mc^2\n</span>\nafter</p>",
				`<p data-source-line="6">more</p>`,
			},
			deny: []string{"<div"},
		},
		{
			name: "block closed at end of content line",
			src:  "$$\na\nb $$\n",
			want: []string{"<div class=\"math display\" data-math=\"display\" data-source-line=\"2\">a\nb </div>"},
		},
		{
			// \$ is an escaped dollar, so "\$$" is not a closer; the
			// block ends at the real "$$" two lines later.
			name: "escaped dollar before a dollar does not close a block",
			src:  "$$\nx = \\$$\ny\n$$\nafter\n",
			want: []string{
				"<div class=\"math display\" data-math=\"display\" data-source-line=\"2\">x = \\$$\ny\n</div>",
				`<p data-source-line="5">after</p>`,
			},
		},
		{
			// Like an unclosed code fence, an unclosed block runs to the
			// end of its container.
			name: "unclosed block runs to end of container",
			src:  "$$\nx\n\nafter\n",
			want: []string{"<div class=\"math display\" data-math=\"display\" data-source-line=\"2\">x\n\nafter\n</div>"},
			deny: []string{"<p"},
		},
		{
			name: "block in list item",
			src:  "- item\n\n  $$\n  s\n  $$\n",
			want: []string{"<div class=\"math display\" data-math=\"display\" data-source-line=\"4\">s\n</div>"},
		},
		{
			// Four spaces of indentation make a code block elsewhere; in
			// a paragraph they are a lazy continuation, so the math is
			// inline display math rather than a block.
			name: "indented dollar-dollar inside a paragraph stays inline",
			src:  "para\n    $$\n    x\n    $$\n",
			want: []string{"<span class=\"math display\" data-math=\"display\">\nx\n</span>"},
			deny: []string{"<div", "<pre"},
		},
		{
			name: "dollar-dollar line with trailing text is not a block",
			src:  "$$5 and $$ not a block",
			want: []string{`<span class="math display" data-math="display">5 and </span> not a block`},
			deny: []string{"<div"},
		},
		{
			name: "triple dollar is text",
			src:  "a $$$ b",
			want: []string{"a $$$ b"},
		},
		{
			// Like GitHub, math inside a raw HTML block is not parsed.
			name: "math inside an html block is left alone",
			src:  "<div>\n$x$\n</div>\n",
			want: []string{"<div>\n$x$\n</div>"},
			deny: []string{"math"},
		},
	}

	p := parser.New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			out, err := p.Render([]byte(tt.src))
			if err != nil {
				t.Fatalf("Render: %v", err)
			}
			html := string(out)
			for _, w := range tt.want {
				if !strings.Contains(html, w) {
					t.Errorf("missing %q in:\n%s", w, html)
				}
			}
			for _, d := range tt.deny {
				if strings.Contains(html, d) {
					t.Errorf("unexpected %q in:\n%s", d, html)
				}
			}
		})
	}
}

func TestRender_MathDisabled(t *testing.T) {
	t.Parallel()

	p := parser.New(parser.WithMath(false))
	out, err := p.Render([]byte("$a_1 * b_2 * c$\n"))
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if strings.Contains(string(out), "math") {
		t.Errorf("expected no math markup with WithMath(false), got: %s", out)
	}
}
