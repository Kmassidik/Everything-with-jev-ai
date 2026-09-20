package web

import (
	"fmt"

	"jevai/internal/billing"
	"jevai/internal/judge"
)

// pct formats a 0..1 probability as a whole percent.
func pct(f float64) string { return fmt.Sprintf("%.0f%%", f*100) }

// probTier buckets a risk/probability into safe / watch / flag.
func probTier(f float64) string {
	switch {
	case f >= 0.66:
		return "flag"
	case f >= 0.4:
		return "watch"
	default:
		return "safe"
	}
}

func tierBg(tier string) string {
	switch tier {
	case "flag":
		return "bg-flag"
	case "watch":
		return "bg-watch"
	default:
		return "bg-safe"
	}
}

func tierText(tier string) string {
	switch tier {
	case "flag":
		return "text-flag"
	case "watch":
		return "text-watch"
	default:
		return "text-safe"
	}
}

// riskClass colors a risk figure by tier.
func riskClass(f float64) string { return "font-semibold " + tierText(probTier(f)) }

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

func itemMeta(r *judge.ItemReport) string {
	return fmt.Sprintf("%d flagged of %d · %d tokens", r.Flags, len(r.Findings), r.TokensIn+r.TokensOut)
}

func demoNote(live bool) string {
	if live {
		return "Live via Jev."
	}
	return "SAMPLE — set TYPESAFE_API_KEY for a live judgment."
}

func gateUsage(st billing.Status) string {
	return fmt.Sprintf("Used %d of %d free tokens.", st.Used, st.Allowance)
}

// flagText emphasises a flagged finding's label.
func flagText(flag bool) string {
	if flag {
		return "font-semibold text-ink"
	}
	return "text-muted"
}
