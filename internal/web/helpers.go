package web

import (
	"fmt"

	"jevai/internal/billing"
	"jevai/internal/judge"
	"jevai/internal/ledger"
	"jevai/internal/store"
)

func usageChip(_ ledger.Usage, gate billing.Status) string {
	return fmt.Sprintf("%d of %d free tokens used", gate.Used, gate.Allowance)
}

func auditMetaLine(a store.AuditMeta) string {
	return fmt.Sprintf("%d items · %d flagged · %s", a.Items, a.Flagged, a.CreatedAt.UTC().Format("2 Jan 15:04"))
}

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

func itemTitle(res judge.Result) string {
	if res.Item.Title != "" {
		return res.Item.Title
	}
	return "(untitled)"
}

func reportMeta(r *judge.Report) string {
	return fmt.Sprintf("%d scored · %d tokens", r.Judgments, r.TokensIn+r.TokensOut)
}

// packTitle names a saved report by its pack.
func packTitle(pack string) string {
	switch pack {
	case "ad-preflight":
		return "Ad pre-flight"
	case "listing-hygiene":
		return "Listing audit"
	case "claim-screening":
		return "Claim screening"
	default:
		return "Audit"
	}
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
