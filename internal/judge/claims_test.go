package judge

import (
	"context"
	"testing"

	"jevai/internal/jev"
)

func TestScreen_Mock(t *testing.T) {
	rep, err := Screen(context.Background(), jev.Mock{}, ClaimScreening, Claim{
		Copy:     "Cures diabetes in 3 days, 100% guaranteed, BPOM registered.",
		Category: "health",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Sample {
		t.Error("mock should mark report as sample")
	}
	if rep.TokensIn == 0 {
		t.Error("tokens not metered")
	}
	if len(rep.Findings) != len(ClaimScreening.Checks) {
		t.Fatalf("findings=%d, want %d", len(rep.Findings), len(ClaimScreening.Checks))
	}
	// sorted worst-first, probabilities in range, Flags count consistent
	flags := 0
	for i, f := range rep.Findings {
		if f.Prob < 0 || f.Prob > 1 {
			t.Errorf("prob out of range: %f", f.Prob)
		}
		if f.Label == "" {
			t.Errorf("finding %d missing label", i)
		}
		if i > 0 && rep.Findings[i-1].Prob < f.Prob {
			t.Errorf("not sorted worst-first at %d", i)
		}
		if f.Flag {
			flags++
		}
	}
	if flags != rep.Flags {
		t.Errorf("Flags=%d but counted %d", rep.Flags, flags)
	}
}
