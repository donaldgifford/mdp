package assets_test

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/donaldgifford/mdp/assets"
)

var (
	// footnoteRuleRe matches a rule whose selector mentions footnotes.
	// Anchoring on the selector line keeps the @media block near the
	// top of preview.css from confusing a flat block scan.
	footnoteRuleRe = regexp.MustCompile(`(?m)^([^{}\n]*footnote[^{}\n]*)\{([^{}]*)\}`)

	// customPropRe matches a var() reference to a custom property.
	customPropRe = regexp.MustCompile(`var\((--[a-zA-Z0-9-]+)\)`)
)

// TestFootnoteCSSPropertiesDefinedByEveryTheme asserts that every
// custom property the footnote rules consume is defined by every
// built-in theme.
//
// The footnote styling deliberately reuses existing theme properties
// rather than adding its own, which is why DESIGN-0003 Decision 4 left
// all 13 theme files untouched. That decision only holds while the
// properties actually exist everywhere: a theme missing
// --color-border-muted renders the endnote hairline invisible, and a
// theme missing --color-fg-muted renders the note text at the default
// colour. Both fail silently -- the page still loads, it just looks
// wrong on that one theme, which is exactly the kind of defect a
// per-theme visual sweep is expensive to catch and easy to miss.
//
// Deriving the property list from preview.css rather than hardcoding
// it means adding a new var() to a footnote rule is covered
// automatically.
func TestFootnoteCSSPropertiesDefinedByEveryTheme(t *testing.T) {
	t.Parallel()

	previewCSS, err := assets.FS.ReadFile("preview.css")
	if err != nil {
		t.Fatalf("reading preview.css: %v", err)
	}

	rules := footnoteRuleRe.FindAllStringSubmatch(string(previewCSS), -1)
	if len(rules) == 0 {
		t.Fatal("found no footnote rules in preview.css; the selector " +
			"scan is broken and this test would pass vacuously")
	}

	required := make(map[string][]string)
	for _, rule := range rules {
		selector := strings.TrimSpace(rule[1])
		for _, ref := range customPropRe.FindAllStringSubmatch(rule[2], -1) {
			required[ref[1]] = append(required[ref[1]], selector)
		}
	}
	if len(required) == 0 {
		t.Fatalf("footnote rules reference no custom properties; expected "+
			"the styling to reuse theme properties (scanned %d rules)",
			len(rules))
	}

	themeFiles, err := fs.Glob(assets.FS, "themes/*.css")
	if err != nil {
		t.Fatalf("globbing themes: %v", err)
	}
	if len(themeFiles) == 0 {
		t.Fatal("found no theme files; this test would pass vacuously")
	}

	for _, path := range themeFiles {
		t.Run(strings.TrimSuffix(strings.TrimPrefix(path, "themes/"), ".css"), func(t *testing.T) {
			t.Parallel()

			css, err := assets.FS.ReadFile(path)
			if err != nil {
				t.Fatalf("reading %s: %v", path, err)
			}
			body := string(css)

			for prop, selectors := range required {
				if !strings.Contains(body, prop+":") {
					t.Errorf("%s does not define %s, used by footnote rule(s) %s; "+
						"footnotes will render incorrectly on this theme",
						path, prop, strings.Join(selectors, ", "))
				}
			}
		})
	}
}
