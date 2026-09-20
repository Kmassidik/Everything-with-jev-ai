package judge

import (
	"context"
	"sort"

	"jevai/internal/jev"
)

// ItemReport is the result of running a pack over a single item (a piece of copy,
// an ad creative…). Surfaces that judge many rules against one item share it.
type ItemReport struct {
	Context   string // free label: a category, a platform, etc.
	Findings  []Finding
	Flags     int
	TokensIn  int
	TokensOut int
	Sample    bool
}

// runPack runs the pack over one state in a single Jev request (many rules, one
// item — the fan-out pattern) and returns a worst-first report.
func runPack(ctx context.Context, j jev.Judge, pack Pack, ctxLabel string, state map[string]string) (*ItemReport, error) {
	resp, err := j.SystemOne(ctx, state, pack.questions())
	if err != nil {
		return nil, err
	}
	rep := &ItemReport{
		Context:   ctxLabel,
		Sample:    !j.Live(),
		TokensIn:  resp.Usage.InputTokens,
		TokensOut: resp.Usage.OutputTokens,
		Findings:  make([]Finding, 0, len(pack.Checks)),
	}
	for _, ch := range pack.Checks {
		p := resp.Noul(ch.Key)
		contrib := p
		if !ch.BadWhenTrue {
			contrib = 1 - p
		}
		flag := contrib >= flagThreshold
		if flag {
			rep.Flags++
		}
		rep.Findings = append(rep.Findings, Finding{Key: ch.Key, Label: ch.Label, Prob: p, Flag: flag})
	}
	sort.SliceStable(rep.Findings, func(a, b int) bool {
		return rep.Findings[a].Prob > rep.Findings[b].Prob
	})
	return rep, nil
}

// Claim is one piece of marketing copy to screen, with its product category.
type Claim struct {
	Copy     string
	Category string
}

// Screen runs the claim-screening pack over one piece of copy.
func Screen(ctx context.Context, j jev.Judge, pack Pack, c Claim) (*ItemReport, error) {
	return runPack(ctx, j, pack, c.Category, map[string]string{"copy": c.Copy, "category": c.Category})
}
