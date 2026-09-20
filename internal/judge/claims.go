package judge

import (
	"context"
	"sort"

	"jevai/internal/jev"
)

// Claim is one piece of marketing copy to screen, with its product category.
type Claim struct {
	Copy     string
	Category string
}

// ClaimFinding is one rule's verdict on the copy.
type ClaimFinding struct {
	Key   string
	Label string
	Prob  float64
	Flag  bool
}

// ClaimReport is the outcome of screening one piece of copy.
type ClaimReport struct {
	Category  string
	Findings  []ClaimFinding // sorted most-likely-problem first
	Flags     int
	TokensIn  int
	TokensOut int
	Sample    bool
}

// Screen runs the pack over one piece of copy in a single Jev request (many rules,
// one item — the fan-out pattern), and returns findings sorted worst-first.
func Screen(ctx context.Context, j jev.Judge, pack Pack, c Claim) (*ClaimReport, error) {
	state := map[string]string{"copy": c.Copy, "category": c.Category}
	resp, err := j.SystemOne(ctx, state, pack.questions())
	if err != nil {
		return nil, err
	}

	rep := &ClaimReport{
		Category:  c.Category,
		Sample:    !j.Live(),
		TokensIn:  resp.Usage.InputTokens,
		TokensOut: resp.Usage.OutputTokens,
		Findings:  make([]ClaimFinding, 0, len(pack.Checks)),
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
		rep.Findings = append(rep.Findings, ClaimFinding{Key: ch.Key, Label: ch.Label, Prob: p, Flag: flag})
	}
	sort.SliceStable(rep.Findings, func(a, b int) bool {
		return rep.Findings[a].Prob > rep.Findings[b].Prob
	})
	return rep, nil
}
