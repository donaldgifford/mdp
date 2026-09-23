package assets_test

import (
	"io/fs"
	"maps"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/donaldgifford/mdp/assets"
)

var (
	// fontFaceRe matches one @font-face block and captures its body.
	fontFaceRe = regexp.MustCompile(`@font-face\s*\{([^{}]*)\}`)

	// fontFamilyRe captures the family name declared in a block.
	fontFamilyRe = regexp.MustCompile(`font-family:\s*"([^"]+)"`)

	// vendorFontURLRe captures a /vendor/fonts/ URL referenced by a block.
	vendorFontURLRe = regexp.MustCompile(`url\("?/vendor/fonts/([^")]+)"?\)`)
)

// TestPreviewCSSDeclaresVendoredFonts asserts that preview.css declares
// the two diagram font families and that every file it points at is
// actually embedded.
//
// A missing or misnamed woff2 does not fail loudly: the browser logs a
// 404 and silently falls back to the next font in the stack, so the
// diagrams keep rendering in the wrong face. Checking the references
// against assets.FS turns that into a test failure.
func TestPreviewCSSDeclaresVendoredFonts(t *testing.T) {
	t.Parallel()

	css, err := fs.ReadFile(assets.FS, "preview.css")
	if err != nil {
		t.Fatalf("reading preview.css: %v", err)
	}

	blocks := fontFaceRe.FindAllStringSubmatch(string(css), -1)
	if len(blocks) == 0 {
		t.Fatal("preview.css declares no @font-face rules")
	}

	families := make(map[string]bool)
	for _, b := range blocks {
		body := b[1]

		fam := fontFamilyRe.FindStringSubmatch(body)
		if fam == nil {
			t.Errorf("@font-face block has no quoted font-family:\n%s", body)
			continue
		}
		families[fam[1]] = true

		urls := vendorFontURLRe.FindAllStringSubmatch(body, -1)
		if len(urls) == 0 {
			t.Errorf("@font-face for %q references no /vendor/fonts/ file", fam[1])
		}
		for _, u := range urls {
			path := "vendor/fonts/" + u[1]
			if _, err := fs.Stat(assets.FS, path); err != nil {
				t.Errorf("@font-face for %q references %s, which is not embedded: %v",
					fam[1], path, err)
			}
		}
	}

	for _, want := range []string{"Inter", "JetBrains Mono"} {
		if !families[want] {
			t.Errorf("preview.css has no @font-face for %q; declared families: %s",
				want, strings.Join(slices.Sorted(maps.Keys(families)), ", "))
		}
	}
}
