// Package ledger meters Jev usage per user. Because one platform key can't be
// attributed by TypeSafe, this is our source of truth for per-user token usage,
// and what the payment gate reads (PRD §6).
//
// In-memory for the scaffold; swap to SQLite (modernc.org/sqlite) in build step 2.
package ledger

import "sync"

// Usage is a rollup for one user.
type Usage struct {
	InputTokens  int
	OutputTokens int
	Judgments    int
}

// Tokens is the billing unit (in + out).
func (u Usage) Tokens() int { return u.InputTokens + u.OutputTokens }

// Ledger records and rolls up usage. Safe for concurrent use.
type Ledger struct {
	mu sync.Mutex
	m  map[string]Usage
}

func New() *Ledger { return &Ledger{m: make(map[string]Usage)} }

// Record adds one Jev response's usage to a user's rollup.
func (l *Ledger) Record(user string, in, out, judgments int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	u := l.m[user]
	u.InputTokens += in
	u.OutputTokens += out
	u.Judgments += judgments
	l.m[user] = u
}

// Usage returns a user's rollup.
func (l *Ledger) Usage(user string) Usage {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.m[user]
}
