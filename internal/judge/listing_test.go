package judge

import (
	"context"
	"strings"
	"testing"

	"jevai/internal/jev"
)

func TestParseListingsCSV(t *testing.T) {
	in := "sku,title,description,category,price\nA1,Red Shoe,Nice red shoe,Shoes,10\nA2,,,,\nA3,Blue Hat,A warm hat,Hats,5\n"
	got, err := ParseListingsCSV(strings.NewReader(in), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 non-blank rows, got %d", len(got))
	}
	if got[0].Title != "Red Shoe" || got[0].State["category"] != "Shoes" {
		t.Errorf("bad parse: %+v", got[0])
	}
}

func TestParseListingsCSV_NeedsColumn(t *testing.T) {
	if _, err := ParseListingsCSV(strings.NewReader("foo,bar\n1,2\n"), 10); err == nil {
		t.Fatal("expected error for missing title/description column")
	}
}

func TestParseAdsCSV(t *testing.T) {
	in := "creative,landing,platform\nBuy now 50% off!,shop page,meta\n,,\nMiracle cure guaranteed,,google\n"
	got, err := ParseAdsCSV(strings.NewReader(in), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 ad variants, got %d", len(got))
	}
	if got[0].State["platform"] != "meta" {
		t.Errorf("bad parse: %+v", got[0])
	}
	if got[1].State["platform"] != "google" {
		t.Errorf("platform default/parse wrong: %+v", got[1])
	}
}

func TestAudit_MockRanksAndMeters(t *testing.T) {
	items := []Item{
		{Row: 2, Title: "A", State: map[string]string{"title": "A", "description": "aaa"}},
		{Row: 3, Title: "B", State: map[string]string{"title": "B", "description": "bbb"}},
		{Row: 4, Title: "C", State: map[string]string{"title": "C", "description": "ccc"}},
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

func TestAudit_AdsPack(t *testing.T) {
	items, err := ParseAdsCSV(strings.NewReader("creative,landing,platform\nGuaranteed #1 results lose 10kg,gym page,meta\n"), 10)
	if err != nil {
		t.Fatal(err)
	}
	rep, err := Audit(context.Background(), jev.Mock{}, AdPreflight, items, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Results) != 1 || len(rep.Results[0].Findings) != len(AdPreflight.Checks) {
		t.Fatalf("unexpected ad report: %+v", rep)
	}
}
