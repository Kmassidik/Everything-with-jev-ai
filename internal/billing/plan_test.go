package billing

import "testing"

func TestEvaluate_FreeTrial(t *testing.T) {
	p := Plan{ID: "free", TrialTokens: 100}

	under := Evaluate(40, p)
	if under.Blocked || under.OverTrial {
		t.Errorf("40/100 should be allowed: %+v", under)
	}
	if under.Remaining != 60 {
		t.Errorf("remaining=%d, want 60", under.Remaining)
	}

	at := Evaluate(100, p)
	if !at.Blocked || !at.OverTrial {
		t.Errorf("exactly-at allowance should block: %+v", at)
	}
	if at.Remaining != 0 {
		t.Errorf("remaining=%d, want 0", at.Remaining)
	}

	over := Evaluate(250, p)
	if !over.Blocked || over.Remaining != 0 {
		t.Errorf("over allowance should block with 0 remaining: %+v", over)
	}
}

func TestEvaluate_Paid(t *testing.T) {
	st := Evaluate(10_000_000, Pro)
	if st.Blocked {
		t.Errorf("paid plan must never block: %+v", st)
	}
}
