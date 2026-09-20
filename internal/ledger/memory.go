package ledger

import (
	"context"
	"sync"
)

// Memory is an in-process Ledger — a fallback when no SQLite path is usable and the
// default in tests. Not durable across restarts.
type Memory struct {
	mu sync.Mutex
	m  map[string]Usage
}

func NewMemory() *Memory { return &Memory{m: make(map[string]Usage)} }

func (l *Memory) Record(_ context.Context, user string, in, out, judgments int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	u := l.m[user]
	u.InputTokens += in
	u.OutputTokens += out
	u.Judgments += judgments
	l.m[user] = u
	return nil
}

func (l *Memory) Usage(_ context.Context, user string) (Usage, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.m[user], nil
}

func (l *Memory) Close() error { return nil }
