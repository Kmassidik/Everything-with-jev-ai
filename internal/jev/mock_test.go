package jev

import (
	"context"
	"testing"
)

func TestMock(t *testing.T) {
	m := Mock{}
	if m.Live() {
		t.Error("mock must not report as live")
	}

	qs := map[string]Question{
		"a": {Type: "noul", Instructions: "x"},
		"b": {Type: "noul", Instructions: "y"},
	}
	r1, err := m.SystemOne(context.Background(), "state", qs)
	if err != nil {
		t.Fatal(err)
	}
	if len(r1.Answers) != 2 {
		t.Fatalf("answers=%d, want 2", len(r1.Answers))
	}

	// deterministic
	r2, _ := m.SystemOne(context.Background(), "state", qs)
	if r1.Noul("a") != r2.Noul("a") {
		t.Error("mock is not deterministic")
	}
	// in range, and distinct keys generally differ
	if p := r1.Noul("a"); p < 0 || p >= 1 {
		t.Errorf("prob out of range: %f", p)
	}
	if r1.Noul("missing") != 0 {
		t.Error("absent key should read 0")
	}
}
