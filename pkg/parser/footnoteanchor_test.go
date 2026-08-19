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

	// fnItemStartRe matches the opening tag of one endnote list item.
	// Finding its matching close needs depth tracking, not a regex --
	// see footnoteItems.
	fnItemStartRe = regexp.MustCompile(`<li id="([^"]+)"[^>]*>`)
)

// footnoteItems maps each endnote list item's id to its full inner
// HTML.
//
// A non-greedy `(.*?)</li>` is wrong here. A definition may contain a
// nested list, and goldmark attaches the backlink after the inner
// </li> tags, so a non-greedy capture stops at the first close and
// truncates the backlink away -- reporting a broken round trip for
// markup that is perfectly correct. Depth has to be tracked.
func footnoteItems(html string) map[string]string {
	items := make(map[string]string)

	for rest := html; ; {
		loc := fnItemStartRe.FindStringSubmatchIndex(rest)
		if loc == nil {
			return items
		}

		id := rest[loc[2]:loc[3]]
		body := rest[loc[1]:]

		end := closingLIIndex(body)
		if end < 0 {
			// Unbalanced markup: record what is there rather than
			// silently dropping the item.
			items[id] = body
			return items
		}

		items[id] = body[:end]
		rest = body[end:]
	}
}

// closingLIIndex returns the offset of the </li> that closes the item
// whose body starts at s, or -1 if the markup is unbalanced.
func closingLIIndex(s string) int {
	depth, i := 0, 0

	for i < len(s) {
		open := strings.Index(s[i:], "<li")
		closing := strings.Index(s[i:], "</li>")
		if closing < 0 {
			return -1
		}

		if open >= 0 && open < closing {
			depth++
			i += open + len("<li")
			continue
		}

		if depth == 0 {
			return i + closing
		}
		depth--
		i += closing + len("</li>")
	}

	return -1
}

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
			// A nested list inside a definition puts the backlink
			// after the inner </li>, which a non-greedy capture
			// would truncate away.
			name: "definition containing a nested list",
			md:   "Text.[^a]\n\n[^a]: Note with a list:\n\n    - one\n    - two\n",
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

			items := footnoteItems(got)

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

// TestFootnoteTestRegexesDoNotUnderMatch keeps the tests above honest.
//
// They locate elements with regexes that assume goldmark's exact
// attribute order and spacing -- fnRefPairRe, for instance, requires
// <a> to follow <sup> with no whitespace between them. If goldmark
// ever reorders an attribute or adds one, such a regex stops matching
// some elements and the tests above quietly verify less than they
// claim, while still passing.
//
// That is not hypothetical: footnoteItems previously used a non-greedy
// `(.*?)</li>` that truncated the backlink out of any definition
// containing a nested list.
//
// Cross-checking each regex against a direct substring count turns
// that silent weakening into a failure.
func TestFootnoteTestRegexesDoNotUnderMatch(t *testing.T) {
	t.Parallel()

	for _, tc := range footnoteAnchorDocs(t) {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			html, err := parser.New().Render([]byte(tc.md))
			if err != nil {
				t.Fatalf("render: %v", err)
			}
			got := string(html)

			refs := strings.Count(got, `class="footnote-ref"`)
			backrefs := strings.Count(got, `class="footnote-backref"`)
			defs := strings.Count(got, `<li id="fn:`)

			if n := len(fnAnchorRe.FindAllStringSubmatch(got, -1)); n != refs+backrefs {
				t.Errorf("fnAnchorRe matched %d anchors, but the output has %d "+
					"(%d refs + %d backrefs); the anchor test is checking "+
					"fewer links than it appears to", n, refs+backrefs, refs, backrefs)
			}
			if n := len(fnRefPairRe.FindAllStringSubmatch(got, -1)); n != refs {
				t.Errorf("fnRefPairRe matched %d reference pairs, but the output "+
					"has %d footnote-refs; the round-trip test is skipping "+
					"references", n, refs)
			}
			if n := len(footnoteItems(got)); n != defs {
				t.Errorf("footnoteItems extracted %d definitions, but the output "+
					"has %d; the round-trip test is skipping definitions", n, defs)
			}
		})
	}
}
