package web

import (
	"fmt"

	"jevai/internal/judge"
)

// small view helpers used by the templ files (same package).

func pct(f float64) string { return fmt.Sprintf("%.0f%%", f*100) }

func riskClass(f float64) string {
	switch {
	case f >= 0.66:
		return "font-bold text-magenta"
	case f >= 0.4:
		return "font-bold text-ink"
	default:
		return "text-grass"
	}
}

func noulClass(p float64) string {
	if p >= 0.5 {
		return "text-3xl font-bold text-grass"
	}
	return "text-3xl font-bold text-muted"
}

func listingTitle(res judge.Result) string {
	if res.Listing.Title != "" {
		return res.Listing.Title
	}
	if res.Listing.SKU != "" {
		return res.Listing.SKU
	}
	return "(untitled)"
}

func reportMeta(r *judge.Report) string {
	return fmt.Sprintf("%d judged · %d tokens", r.Judgments, r.TokensIn+r.TokensOut)
}

func demoNote(live bool) string {
	if live {
		return "Live via Jev."
	}
	return "SAMPLE — set TYPESAFE_API_KEY for a live judgment."
}
