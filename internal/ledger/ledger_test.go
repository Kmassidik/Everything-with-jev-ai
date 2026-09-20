package ledger

import (
	"context"
	"path/filepath"
	"testing"
)

// runLedgerContract exercises any Ledger implementation the same way.
func runLedgerContract(t *testing.T, l Ledger) {
	t.Helper()
	ctx := context.Background()

	for _, e := range []struct {
		user          string
		in, out, judg int
	}{
		{"u1", 100, 20, 3},
		{"u1", 50, 10, 1},
		{"u2", 5, 1, 1},
	} {
		if err := l.Record(ctx, e.user, e.in, e.out, e.judg); err != nil {
			t.Fatalf("record: %v", err)
		}
	}

	u1, err := l.Usage(ctx, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if u1.InputTokens != 150 || u1.OutputTokens != 30 || u1.Judgments != 4 {
		t.Errorf("u1 rollup wrong: %+v", u1)
	}
	if u1.Tokens() != 180 {
		t.Errorf("u1 tokens=%d, want 180", u1.Tokens())
	}

	if u2, _ := l.Usage(ctx, "u2"); u2.Judgments != 1 {
		t.Errorf("u2 not isolated: %+v", u2)
	}
	if empty, _ := l.Usage(ctx, "nobody"); empty.Tokens() != 0 {
		t.Errorf("unknown user should be zero, got %+v", empty)
	}
}

func TestMemory(t *testing.T) {
	l := NewMemory()
	defer l.Close()
	runLedgerContract(t, l)
}

func TestSQLite(t *testing.T) {
	l, err := OpenSQLite(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	runLedgerContract(t, l)
}
