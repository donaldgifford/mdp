// Package theme provides a registry of named themes plus loaders for
// custom CSS files. Resolve maps a name (built-in theme, "auto", a
// file path, or "") to a Theme struct containing the CSS, an optional
// vendored hljs sheet path, and a Mermaid theme identifier.
//
// # Binary bloat note
//
// Importing pkg/theme pulls in the embedded vendor assets (highlight.js
// stylesheets, theme CSS) via the mdp assets package. The footprint is
// modest — a few hundred KB — but worth knowing for consumers that
// want to ship the smallest possible binary. Callers that only need
// the parser can import pkg/parser alone.
//
// # Usage
//
// Resolve returns the Theme for any built-in name, "auto", or a CSS
// file path. The auto theme has empty CSS — the consumer is expected
// to skip server-side injection and let the browser's
// prefers-color-scheme media query drive appearance.
//
//	t, err := theme.Resolve("github-light")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("hljs sheet: %s\n", t.HljsVendorCSS)
//	fmt.Printf("mermaid theme: %s\n", t.MermaidTheme)
//	fmt.Printf("is auto: %v\n", t.IsAuto())
//
// Names returns all valid built-in theme names in sorted order, useful
// for CLI flag validation or theme-picker UIs.
package theme
