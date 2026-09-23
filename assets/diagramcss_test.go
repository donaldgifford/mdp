package assets_test

import (
	"io/fs"
	"regexp"
	"strings"
	"testing"

	"github.com/donaldgifford/mdp/assets"
)

var (
	// themeBlockRe matches one [data-theme="name"] { ... } block.
	themeBlockRe = regexp.MustCompile(`\[data-theme="([^"]+)"\]\s*\{([^{}]*)\}`)

	// mermaidPropRe captures every --mermaid-* declaration and its value.
	mermaidPropRe = regexp.MustCompile(`(--mermaid-[A-Za-z0-9-]+):\s*([^;]+);`)

	// hexColorRe is the only value form preview.js can hand to Mermaid,
	// which does colour arithmetic and cannot evaluate var() or
	// color-mix().
	hexColorRe = regexp.MustCompile(`^#[0-9a-f]{6}$`)

	// diagramSlots are the seven palette slots readPalette() reads
	// (DESIGN-0004, beautiful-mermaid's colour model).
	diagramSlots = []string{
		"--mermaid-bg", "--mermaid-fg", "--mermaid-line", "--mermaid-accent",
		"--mermaid-muted", "--mermaid-surface", "--mermaid-border",
	}
)

// TestDiagramPaletteDefinedByEveryTheme asserts that every built-in theme
// defines the seven diagram palette slots as lowercase six-digit hex, and
// that none of the legacy --mermaid-<variableName> properties survive.
//
// readPalette() derives any missing slot from the prose colours, so a
// theme that forgets one still renders -- just not with its intended
// palette. That silent fallback is exactly why the contract is checked
// here rather than trusted to a visual sweep. A leftover legacy property
// is dead CSS that misleads the next theme author into editing it.
//
// Theme blocks are discovered from the embedded files, so a new theme is
// covered without editing this test.
func TestDiagramPaletteDefinedByEveryTheme(t *testing.T) {
	t.Parallel()

	themeFiles, err := fs.Glob(assets.FS, "themes/*.css")
	if err != nil {
		t.Fatalf("globbing themes: %v", err)
	}
	if len(themeFiles) == 0 {
		t.Fatal("found no theme files; this test would pass vacuously")
	}

	allowed := make(map[string]bool, len(diagramSlots))
	for _, slot := range diagramSlots {
		allowed[slot] = true
	}

	themes := 0
	for _, path := range themeFiles {
		css, err := assets.FS.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}

		for _, block := range themeBlockRe.FindAllStringSubmatch(string(css), -1) {
			name, body := block[1], block[2]
			if !strings.Contains(body, "--color-fg-default") {
				continue
			}
			themes++

			t.Run(name, func(t *testing.T) {
				t.Parallel()

				got := make(map[string]string)
				for _, m := range mermaidPropRe.FindAllStringSubmatch(body, -1) {
					prop, value := m[1], strings.TrimSpace(m[2])
					if !allowed[prop] {
						t.Errorf("%s (%s) still defines legacy property %s; only "+
							"the seven palette slots are read", name, path, prop)
						continue
					}
					got[prop] = value
				}

				for _, slot := range diagramSlots {
					value, ok := got[slot]
					if !ok {
						t.Errorf("%s (%s) does not define %s", name, path, slot)
						continue
					}
					if !hexColorRe.MatchString(value) {
						t.Errorf("%s (%s) sets %s to %q; want lowercase #rrggbb",
							name, path, slot, value)
					}
				}
			})
		}
	}

	if themes == 0 {
		t.Fatal("found no theme blocks defining --color-fg-default; the block " +
			"scan is broken and this test would pass vacuously")
	}
}
