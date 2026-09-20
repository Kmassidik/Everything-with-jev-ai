package judge

import (
	"context"
	"strings"
	"testing"

	"jevai/internal/jev"
)

func TestParseCSV(t *testing.T) {
	in := "sku,title,description,category,price\nA1,Red Shoe,Nice red shoe,Shoes,10\nA2,,,,\nA3,Blue Hat,A warm hat,Hats,5\n"
	got, err := ParseCSV(strings.NewReader(in), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 non-blank rows, got %d", len(got))
	}
	if got[0].Title != "Red Shoe" || got[0].Category != "Shoes" {
		t.Errorf("bad parse: %+v", got[0])
	}
}

func TestParseCSV_MaxCap(t *testing.T) {
	var b strings.Builder
	b.WriteString("title\n")
	for i := 0; i < 50; i++ {
		b.WriteString("x\n")
	}
	got, err := ParseCSV(strings.NewReader(b.String()), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 10 {
		t.Fatalf("cap not applied: got %d", len(got))
	}
}

func TestParseCSV_NeedsColumn(t *testing.T) {
	if _, err := ParseCSV(strings.NewReader("foo,bar\n1,2\n"), 10); err == nil {
		t.Fatal("expected error for missing title/description column")
	}
}

func TestAudit_MockRanksAndMeters(t *testing.T) {
	items := []Listing{
		{Row: 2, Title: "A", Description: "aaa"},
		{Row: 3, Title: "B", Description: "bbb"},
		{Row: 4, Title: "C", Description: "ccc"},
	}
	rep, err := Audit(context.Background(), jev.Mock{}, ListingHygiene, items, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.Sample {
		t.Error("mock judge should mark report as sample")
	}
	if rep.Judgments != 3 {
		t.Errorf("judged=%d, want 3", rep.Judgments)
	}
	if rep.TokensIn == 0 {
		t.Error("tokens not metered")
	}
	if len(rep.Results) != 3 {
		t.Fatalf("results=%d, want 3", len(rep.Results))
	}
	for i := 1; i < len(rep.Results); i++ {
		if rep.Results[i-1].Risk < rep.Results[i].Risk {
			t.Errorf("not ranked worst-first at %d", i)
		}
	}
	for _, r := range rep.Results {
		if r.Risk < 0 || r.Risk > 1 {
			t.Errorf("risk out of range: %f", r.Risk)
		}
		if len(r.Findings) != len(ListingHygiene.Checks) {
			t.Errorf("findings=%d, want %d", len(r.Findings), len(ListingHygiene.Checks))
		}
	}
}
