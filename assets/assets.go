// Package assets embeds the client-side preview files.
package assets

import "embed"

// FS contains the embedded preview assets (HTML template, CSS, JS, vendor libs).
//
//go:embed preview.html preview.css preview.js vendor
var FS embed.FS
