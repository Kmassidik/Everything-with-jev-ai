// Package ledger meters Jev usage per user. Because one platform key can't be
// attributed by TypeSafe, this is our source of truth for per-user token usage and
// what the payment gate reads (PRD §6).
package ledger

import "context"

// Usage is a per-user rollup.
type Usage struct {
	InputTokens  int
	OutputTokens int
	Judgments    int
}

// Tokens is the billing unit (in + out).
func (u Usage) Tokens() int { return u.InputTokens + u.OutputTokens }

// Ledger records usage events and rolls them up. Implementations must be safe for
// concurrent use.
type Ledger interface {
	Record(ctx context.Context, user string, in, out, judgments int) error
	Usage(ctx context.Context, user string) (Usage, error)
	Close() error
}
