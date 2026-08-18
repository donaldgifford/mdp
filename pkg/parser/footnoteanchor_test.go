package parser_test

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/donaldgifford/mdp/pkg/parser"
)

var (
	// anyIDRe matches every id attribute in the rendered output.
	anyIDRe = regexp.MustCompile(`\sid="([^"]+)"`)

	// fnAnchorRe matches both directions of footnote navigation: a
	// .footnote-ref pointing at a definition, and a .footnote-backref
	// pointing back at the reference.
	fnAnchorRe = regexp.MustCompile(`<a href="#([^"]+)" class="footnote-(?:ref|backref)"`)

	// fnRefPairRe captures a reference's own id alongside the
	// definition it targets, which is what makes the round trip
	// checkable.
	fnRefPairRe = regexp.MustCompile(`<sup id="([^"]+)"><a href="#([^"]+)" class="footnote-ref"`)

	// fnItemRe matches one <li> of the endnote list, non-greedily so
	// consecutive items do not merge.
	fnItemRe = regexp.MustCompile(`(?s)<li id="([^"]+)"[^>]*>(.*?)</li>`)
)

// footnoteAnchorDocs are the shapes whose id/href wiring is least
// obvious, so they are shared by the anchor tests below.
func footnoteAnchorDocs(t *testing.T) []struct{ name, md string } {
	t.Helper()

	fixture, err := os.ReadFile("testdata/fixture.md")
	if err != nil {
		t.Fatalf("reading fixture: %v", err)
	}

	return []struct{ name, md string }{
		{
			name: "single reference",
			md:   "Text.[^a]\n\n[^a]: The note.\n",
		},
		{
			// The hard case: goldmark emits fnref:1 and fnref1:1 for
			// the two references, and fn:1 carries a backref to each.
			name: "repeated reference to one definition",
			md:   "See[^a] and again[^a] and other[^b].\n\n[^a]: First.\n[^b]: Second.\n",
		},
		{
			name: "named labels numbered by first reference",
			md:   "Alpha.[^zeta] Beta.[^alpha]\n\n[^alpha]: Second.\n[^zeta]: First.\n",
		},
		{
			name: "definition placed mid-document",
			md:   "# T\n\nIntro.[^a]\n\n[^a]: Note.\n\n## S\n\nBody.\n",
		},
		{
			name: "testdata/fixture.md",
			md:   string(fixture),
		},
	}
}

// TestRender_FootnoteAnchorsResolve asserts every footnote link points
// at an id that actually exists, and that no id is duplicated. A
// dangling href or an ambiguous duplicate makes the click a no-op in
// the browser, which no rendering assertion would catch.
func TestRender_FootnoteAnchorsResolve(t *testing.T) {
	t.Parallel()

	for _, tc := range footnoteAnchorDocs(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			html, err := parser.New().Render([]byte(tc.md))
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			got := string(html)

			ids := make(map[string]int)
			for _, m := range anyIDRe.FindAllStringSubmatch(got, -1) {
				ids[m[1]]++
			}
			for id, n := range ids {
				if n > 1 {
					t.Errorf("id %q appears %d times; anchor navigation is "+
						"ambiguous with duplicate ids:\n%s", id, n, got)
				}
			}

			anchors := fnAnchorRe.FindAllStringSubmatch(got, -1)
			if len(anchors) == 0 {
				t.Fatalf("expected footnote anchors, got none:\n%s", got)
			}
			for _, m := range anchors {
				if _, ok := ids[m[1]]; !ok {
					t.Errorf("footnote link targets #%s but no element "+
						"carries that id; the click would do nothing:\n%s",
						m[1], got)
				}
			}
		})
	}
}

// TestRender_FootnoteBacklinksRoundTrip asserts navigation is
// bidirectional: following a reference to its definition and then the
// definition's backlink returns to the same reference. With repeated
// references this is a one-to-many mapping, and getting it wrong
// strands the reader on the wrong reference rather than failing
// visibly.
func TestRender_FootnoteBacklinksRoundTrip(t *testing.T) {
	t.Parallel()

	for _, tc := range footnoteAnchorDocs(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			html, err := parser.New().Render([]byte(tc.md))
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			got := string(html)

			items := make(map[string]string)
			for _, m := range fnItemRe.FindAllStringSubmatch(got, -1) {
				items[m[1]] = m[2]
			}

			pairs := fnRefPairRe.FindAllStringSubmatch(got, -1)
			if len(pairs) == 0 {
				t.Fatalf("expected footnote references, got none:\n%s", got)
			}

			for _, m := range pairs {
				refID, defID := m[1], m[2]

				body, ok := items[defID]
				if !ok {
					t.Errorf("reference %q targets definition #%s, which has "+
						"no <li>:\n%s", refID, defID, got)
					continue
				}
				if !strings.Contains(body, `href="#`+refID+`"`) {
					t.Errorf("definition #%s has no backlink to reference "+
						"#%s, so the return trip is broken:\n%s",
						defID, refID, body)
				}
			}
		})
	}
}
