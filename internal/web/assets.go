package web

import (
	"crypto/sha1"
	"embed"
	"encoding/hex"
)

// Static holds the built CSS, vendored HTMX, and SVGs. `static/css/app.css` is
// produced by Tailwind (`just generate`) before build — see PRD §8.
//
//go:embed static
var Static embed.FS

// CSSVersion is a short content hash of the built stylesheet. It's appended to the
// stylesheet URL (?v=…) so a new build always busts Cloudflare + browser caches.
var CSSVersion = cssHash()

func cssHash() string {
	b, err := Static.ReadFile("static/css/app.css")
	if err != nil {
		return "dev"
	}
	sum := sha1.Sum(b)
	return hex.EncodeToString(sum[:])[:10]
}
