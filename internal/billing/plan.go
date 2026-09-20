// Package billing turns metered usage into a gate decision. It is pure — no I/O —
// so the server reads the ledger and asks billing whether to allow a judgment.
// The trial is spent in total tokens (PRD §6); exact numbers are placeholders (§9).
package billing

// Plan is a billing tier.
type Plan struct {
	ID          string
	Name        string
	TrialTokens int  // free allowance before the gate trips (ignored when Paid)
	Paid        bool // a paid plan never trips the trial gate
}

// Free is the default trial plan. TrialTokens is overridden from config at startup.
var Free = Plan{ID: "free", Name: "Free trial", TrialTokens: 1_000_000}

// Pro is an unlimited paid plan (placeholder until pricing is set).
var Pro = Plan{ID: "pro", Name: "Pro", Paid: true}

// Status is the gate decision for a user.
type Status struct {
	Plan      string
	Used      int
	Allowance int
	Remaining int
	OverTrial bool
	Blocked   bool // refuse new judgments
}

// Evaluate decides the gate from a user's total tokens used and their plan.
// Paid plans never block; on the free trial, used >= allowance blocks (exactly-at
// counts as spent).
func Evaluate(usedTokens int, p Plan) Status {
	if p.Paid {
		return Status{Plan: p.ID, Used: usedTokens}
	}
	allowance := p.TrialTokens
	if allowance < 0 {
		allowance = 0
	}
	remaining := allowance - usedTokens
	if remaining < 0 {
		remaining = 0
	}
	over := usedTokens >= allowance
	return Status{
		Plan:      p.ID,
		Used:      usedTokens,
		Allowance: allowance,
		Remaining: remaining,
		OverTrial: over,
		Blocked:   over,
	}
}
