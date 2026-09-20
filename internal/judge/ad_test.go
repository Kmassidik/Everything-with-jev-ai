package judge

import (
	"context"
	"testing"

	"jevai/internal/jev"
)

func TestPreflight_Mock(t *testing.T) {
	rep, err := Preflight(context.Background(), jev.Mock{}, AdPreflight, Ad{
		Creative: "Guaranteed #1 results, lose 10kg in a week!",
		Landing:  "A gym membership signup page.",
		Platform: "meta",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Context != "meta" {
		t.Errorf("context=%q, want meta", rep.Context)
	}
	if len(rep.Findings) != len(AdPreflight.Checks) {
		t.Fatalf("findings=%d, want %d", len(rep.Findings), len(AdPreflight.Checks))
	}
	if !rep.Sample {
		t.Error("mock should mark sample")
	}
	for i := 1; i < len(rep.Findings); i++ {
		if rep.Findings[i-1].Prob < rep.Findings[i].Prob {
			t.Errorf("not sorted worst-first at %d", i)
		}
	}
}
