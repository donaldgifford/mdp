package theme

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/donaldgifford/mdp/assets"
)

func TestResolve_Auto(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"explicit auto", "auto"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme, err := Resolve(tt.input)
			if err != nil {
				t.Fatalf("Resolve(%q) error = %v, want nil", tt.input, err)
			}
			if !theme.IsAuto {
				t.Errorf("Resolve(%q).IsAuto = %v, want true", tt.input, theme.IsAuto)
			}
			if theme.CSS != "" {
				t.Errorf("Resolve(%q).CSS = %q, want empty", tt.input, theme.CSS)
			}
			if theme.HljsVendorCSS != "" {
				t.Errorf("Resolve(%q).HljsVendorCSS = %q, want empty", tt.input, theme.HljsVendorCSS)
			}
			if theme.MermaidTheme != "" {
				t.Errorf("Resolve(%q).MermaidTheme = %q, want empty", tt.input, theme.MermaidTheme)
			}
		})
	}
}

func TestResolve_BuiltinNames(t *testing.T) {
	names := Names()
	if len(names) != 15 {
		t.Fatalf("Names() returned %d themes, want 15", len(names))
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			theme, err := Resolve(name)
			if err != nil {
				t.Fatalf("Resolve(%q) error = %v, want nil", name, err)
			}
			if theme.IsAuto {
				t.Errorf("Resolve(%q).IsAuto = true, want false for named theme", name)
			}
			if theme.CSS == "" {
				t.Errorf("Resolve(%q).CSS is empty, want non-empty for named theme", name)
			}
			if theme.MermaidTheme != "base" {
				t.Errorf("Resolve(%q).MermaidTheme = %q, want base", name, theme.MermaidTheme)
			}
		})
	}
}

func TestResolve_UnknownName(t *testing.T) {
	_, err := Resolve("nonexistent-theme")
	if err == nil {
		t.Fatal("Resolve(\"nonexistent-theme\") error = nil, want error")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "unknown theme") {
		t.Errorf("error message %q should contain 'unknown theme'", errMsg)
	}

	// Should list valid names
	names := Names()
	for _, name := range names[:3] { // Check first few names are listed
		if !strings.Contains(errMsg, name) {
			t.Errorf("error message %q should contain theme name %q", errMsg, name)
		}
	}
}

func TestResolve_FilePath(t *testing.T) {
	// Create a temporary CSS file
	tmpDir := t.TempDir()
	cssFile := filepath.Join(tmpDir, "test-theme.css")
	testCSS := `/* test theme */
[data-theme="test"] {
  --color-fg-default: #000;
}`

	if err := os.WriteFile(cssFile, []byte(testCSS), 0o644); err != nil {
		t.Fatalf("failed to create test CSS file: %v", err)
	}

	// Test absolute path
	theme, err := Resolve(cssFile)
	if err != nil {
		t.Fatalf("Resolve(%q) error = %v, want nil", cssFile, err)
	}
	if theme.IsAuto {
		t.Errorf("Resolve(%q).IsAuto = true, want false for file theme", cssFile)
	}
	if theme.CSS != testCSS {
		t.Errorf("Resolve(%q).CSS = %q, want %q", cssFile, theme.CSS, testCSS)
	}
	if theme.HljsVendorCSS != "" {
		t.Errorf("Resolve(%q).HljsVendorCSS = %q, want empty for file theme", cssFile, theme.HljsVendorCSS)
	}
	if theme.MermaidTheme != "base" {
		t.Errorf("Resolve(%q).MermaidTheme = %q, want base", cssFile, theme.MermaidTheme)
	}

	// Test relative path
	relPath := "./test-theme.css"
	oldWd, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldWd) }()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("failed to change to temp dir: %v", err)
	}

	theme, err = Resolve(relPath)
	if err != nil {
		t.Fatalf("Resolve(%q) error = %v, want nil", relPath, err)
	}
	if theme.CSS != testCSS {
		t.Errorf("Resolve(%q).CSS = %q, want %q", relPath, theme.CSS, testCSS)
	}
}

func TestResolve_FileNotFound(t *testing.T) {
	_, err := Resolve("/nonexistent/path/theme.css")
	if err == nil {
		t.Fatal("Resolve(\"/nonexistent/path/theme.css\") error = nil, want error")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "reading theme file") {
		t.Errorf("error message %q should mention 'reading theme file'", errMsg)
	}
}

func TestResolve_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	emptyFile := filepath.Join(tmpDir, "empty.css")
	if err := os.WriteFile(emptyFile, []byte("   \n\t\n   "), 0o644); err != nil {
		t.Fatalf("failed to create empty CSS file: %v", err)
	}

	_, err := Resolve(emptyFile)
	if err == nil {
		t.Fatal("Resolve(empty file) error = nil, want error")
	}

	errMsg := err.Error()
	if !strings.Contains(errMsg, "is empty") {
		t.Errorf("error message %q should mention file is empty", errMsg)
	}
}

func TestNames(t *testing.T) {
	names := Names()

	// Should have exactly 15 named themes (excluding "auto")
	if len(names) != 15 {
		t.Errorf("Names() returned %d themes, want 15", len(names))
	}

	// Should be sorted
	if !slices.IsSorted(names) {
		t.Errorf("Names() returned unsorted slice: %v", names)
	}

	// Should not include "auto"
	for _, name := range names {
		if name == "auto" {
			t.Errorf("Names() should not include 'auto', got: %v", names)
		}
	}

	// Should include expected theme families
	expectedFamilies := []string{
		"github-light", "github-dark", "github-dimmed",
		"tokyo-night", "tokyo-night-moon", "tokyo-night-storm", "tokyo-night-day",
		"rose-pine", "rose-pine-moon", "rose-pine-dawn",
		"catppuccin-latte", "catppuccin-frappe", "catppuccin-macchiato", "catppuccin-mocha",
		"donald",
	}

	for _, expected := range expectedFamilies {
		if !slices.Contains(names, expected) {
			t.Errorf("Names() should include %q, got: %v", expected, names)
		}
	}
}

func TestGithubVariantsHaveCorrectVendorCSS(t *testing.T) {
	tests := []struct {
		name           string
		expectedVendor string
	}{
		{"github-light", "/vendor/hljs/github.min.css"},
		{"github-dark", "/vendor/hljs/github-dark.min.css"},
		{"github-dimmed", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme, err := Resolve(tt.name)
			if err != nil {
				t.Fatalf("Resolve(%q) error = %v, want nil", tt.name, err)
			}
			if theme.HljsVendorCSS != tt.expectedVendor {
				t.Errorf("Resolve(%q).HljsVendorCSS = %q, want %q",
					tt.name, theme.HljsVendorCSS, tt.expectedVendor)
			}
		})
	}

	// All other themes should have empty HljsVendorCSS
	names := Names()
	githubThemes := []string{"github-light", "github-dark", "github-dimmed"}
	for _, name := range names {
		if slices.Contains(githubThemes, name) {
			continue // Skip github themes, tested above
		}
		t.Run(name+"_no_vendor", func(t *testing.T) {
			theme, err := Resolve(name)
			if err != nil {
				t.Fatalf("Resolve(%q) error = %v, want nil", name, err)
			}
			if theme.HljsVendorCSS != "" {
				t.Errorf("Resolve(%q).HljsVendorCSS = %q, want empty for non-GitHub theme",
					name, theme.HljsVendorCSS)
			}
		})
	}
}

func TestAllBuiltins(t *testing.T) {
	// Table-driven test for all built-in themes
	names := Names()
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			theme, err := Resolve(name)
			if err != nil {
				t.Fatalf("Resolve(%q) failed: %v", name, err)
			}

			// All named themes should have non-empty CSS
			if theme.CSS == "" {
				t.Errorf("theme %q has empty CSS", name)
			}

			// All named themes should use "base" for Mermaid
			if theme.MermaidTheme != "base" {
				t.Errorf("theme %q has MermaidTheme = %q, want 'base'", name, theme.MermaidTheme)
			}

			// All named themes should not be auto
			if theme.IsAuto {
				t.Errorf("theme %q has IsAuto = true, want false", name)
			}
		})
	}
}

// Verify that the expected theme files can be read at test time.
func TestEmbeddedThemeFilesExist(t *testing.T) {
	expectedFiles := []string{
		"github.css",
		"tokyo-night.css", "tokyo-night-moon.css", "tokyo-night-storm.css", "tokyo-night-day.css",
		"rose-pine.css", "rose-pine-moon.css", "rose-pine-dawn.css",
		"catppuccin-latte.css", "catppuccin-frappe.css", "catppuccin-macchiato.css", "catppuccin-mocha.css",
		"donald.css",
	}

	for _, filename := range expectedFiles {
		t.Run(filename, func(t *testing.T) {
			data, err := assets.FS.ReadFile("themes/" + filename)
			if err != nil {
				t.Fatalf("failed to read embedded theme file %q: %v", filename, err)
			}
			if len(data) == 0 {
				t.Errorf("embedded theme file %q is empty", filename)
			}
			// Basic sanity check that it looks like CSS
			css := string(data)
			if !strings.Contains(css, "[data-theme=") {
				t.Errorf("theme file %q doesn't contain expected [data-theme= selector", filename)
			}
		})
	}
}
