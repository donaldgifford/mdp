// Package theme provides theme resolution for the mdp preview server.
package theme

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/donaldgifford/mdp/assets"
)

// Theme holds everything the server needs to render a page with the correct styling.
type Theme struct {
	// CSS is the complete theme stylesheet (prose + hljs tokens + mermaid vars).
	// Empty for "auto" — the base preview.css handles auto via media query.
	CSS string

	// HljsVendorCSS is the path to a vendored hljs sheet to inject via <link>.
	// Only set for github-light / github-dark. Empty for all other themes.
	HljsVendorCSS string

	// MermaidTheme is the string passed to mermaid.initialize().
	// "base" for named themes (uses CSS vars), "" for auto.
	MermaidTheme string

	// IsAuto skips server-side CSS injection and lets the browser's
	// prefers-color-scheme media query drive appearance.
	IsAuto bool
}

// NOTE: Theme files are embedded via the assets.FS from the assets package.

// builtinThemes maps theme names to their Theme configurations.
// The GitHub themes share a single CSS file with multiple [data-theme] blocks.
var builtinThemes = map[string]Theme{
	// Auto theme - uses CSS media queries
	"auto": {
		CSS:           "",
		HljsVendorCSS: "",
		MermaidTheme:  "",
		IsAuto:        true,
	},

	// GitHub theme family - all use shared github.css file
	"github-light": {
		CSS:           mustReadThemeCSS("github.css"),
		HljsVendorCSS: "/vendor/hljs/github.min.css",
		MermaidTheme:  "base",
		IsAuto:        false,
	},
	"github-dark": {
		CSS:           mustReadThemeCSS("github.css"),
		HljsVendorCSS: "/vendor/hljs/github-dark.min.css",
		MermaidTheme:  "base",
		IsAuto:        false,
	},
	"github-dimmed": {
		CSS:           mustReadThemeCSS("github.css"),
		HljsVendorCSS: "",
		MermaidTheme:  "base",
		IsAuto:        false,
	},

	// Tokyo Night theme family
	"tokyo-night": {
		CSS:           mustReadThemeCSS("tokyo-night.css"),
		HljsVendorCSS: "",
		MermaidTheme:  "base",
		IsAuto:        false,
	},
	"tokyo-night-moon": {
		CSS:           mustReadThemeCSS("tokyo-night-moon.css"),
		HljsVendorCSS: "",
		MermaidTheme:  "base",
		IsAuto:        false,
	},
	"tokyo-night-storm": {
		CSS:           mustReadThemeCSS("tokyo-night-storm.css"),
		HljsVendorCSS: "",
		MermaidTheme:  "base",
		IsAuto:        false,
	},
	"tokyo-night-day": {
		CSS:           mustReadThemeCSS("tokyo-night-day.css"),
		HljsVendorCSS: "",
		MermaidTheme:  "base",
		IsAuto:        false,
	},

	// Rosé Pine theme family
	"rose-pine": {
		CSS:           mustReadThemeCSS("rose-pine.css"),
		HljsVendorCSS: "",
		MermaidTheme:  "base",
		IsAuto:        false,
	},
	"rose-pine-moon": {
		CSS:           mustReadThemeCSS("rose-pine-moon.css"),
		HljsVendorCSS: "",
		MermaidTheme:  "base",
		IsAuto:        false,
	},
	"rose-pine-dawn": {
		CSS:           mustReadThemeCSS("rose-pine-dawn.css"),
		HljsVendorCSS: "",
		MermaidTheme:  "base",
		IsAuto:        false,
	},

	// donald — personal dark theme based on donald.dev palette
	"donald": {
		CSS:           mustReadThemeCSS("donald.css"),
		HljsVendorCSS: "",
		MermaidTheme:  "base",
		IsAuto:        false,
	},

	// Catppuccin theme family
	"catppuccin-latte": {
		CSS:           mustReadThemeCSS("catppuccin-latte.css"),
		HljsVendorCSS: "",
		MermaidTheme:  "base",
		IsAuto:        false,
	},
	"catppuccin-frappe": {
		CSS:           mustReadThemeCSS("catppuccin-frappe.css"),
		HljsVendorCSS: "",
		MermaidTheme:  "base",
		IsAuto:        false,
	},
	"catppuccin-macchiato": {
		CSS:           mustReadThemeCSS("catppuccin-macchiato.css"),
		HljsVendorCSS: "",
		MermaidTheme:  "base",
		IsAuto:        false,
	},
	"catppuccin-mocha": {
		CSS:           mustReadThemeCSS("catppuccin-mocha.css"),
		HljsVendorCSS: "",
		MermaidTheme:  "base",
		IsAuto:        false,
	},
}

// mustReadThemeCSS reads a theme CSS file from the embedded filesystem.
// It panics if the file cannot be read, which is acceptable at package init time
// since theme files are embedded assets that should always be present.
func mustReadThemeCSS(filename string) string {
	data, err := assets.FS.ReadFile("themes/" + filename)
	if err != nil {
		panic(fmt.Sprintf("failed to read embedded theme file %q: %v", filename, err))
	}
	return string(data)
}

// Resolve returns the Theme for the given name.
// name may be a built-in theme name, "auto", an empty string (treated as auto),
// or an absolute/relative path to a CSS file.
func Resolve(name string) (Theme, error) {
	// Empty string or explicit "auto" both resolve to auto theme
	if name == "" || name == "auto" {
		return builtinThemes["auto"], nil
	}

	// Check if it's a built-in theme name
	if theme, exists := builtinThemes[name]; exists {
		return theme, nil
	}

	// Check if it's a file path (starts with / or ./)
	if strings.HasPrefix(name, "/") || strings.HasPrefix(name, "./") {
		css, err := readThemeFile(name)
		if err != nil {
			return Theme{}, err
		}
		return Theme{
			CSS:           css,
			HljsVendorCSS: "",
			MermaidTheme:  "base",
			IsAuto:        false,
		}, nil
	}

	// Unknown theme name
	validNames := Names()
	return Theme{}, fmt.Errorf("unknown theme %q, valid themes: %s",
		name, strings.Join(validNames, ", "))
}

// readThemeFile reads a theme CSS file from disk and validates it's non-empty.
func readThemeFile(path string) (string, error) {
	// Convert relative paths to absolute
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolving theme file path %q: %w", path, err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return "", fmt.Errorf("reading theme file %q: %w", absPath, err)
	}

	css := strings.TrimSpace(string(data))
	if css == "" {
		return "", fmt.Errorf("theme file %q is empty", absPath)
	}

	return css, nil
}

// Names returns all valid built-in theme names in sorted order.
// The "auto" theme is not included since it's a special value.
func Names() []string {
	names := make([]string, 0, len(builtinThemes)-1) // -1 to exclude "auto"
	for name := range builtinThemes {
		if name != "auto" {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}
