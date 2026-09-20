package web

import "embed"

// Static holds the built CSS, vendored HTMX, and SVGs. `static/css/app.css` is
// produced by Tailwind (`just generate`) before build — see PRD §8.
//
//go:embed static
var Static embed.FS
