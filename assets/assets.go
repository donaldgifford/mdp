// Package assets embeds the client-side preview files.
package assets

import "embed"

// FS contains the embedded preview assets (HTML template, CSS, JS).
//
//go:embed preview.html preview.css preview.js
var FS embed.FS
