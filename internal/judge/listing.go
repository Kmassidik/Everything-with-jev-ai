package judge

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"

	"jevai/internal/jev"
)

// Field is one named display/CSV column for an item.
type Field struct{ Key, Val string }

// Item is one thing to score (a listing row, an ad variant…). State is sent to Jev;
// Cols are the ordered columns shown in the report and CSV.
type Item struct {
	Row   int
	Title string
	State map[string]string
	Cols  []Field
}

// Finding is one check's result for one item.
type Finding struct {
	Key   string
	Label string
	Prob  float64
	Flag  bool
}

// Result is an item plus its findings and overall risk (0..1, higher = worse).
type Result struct {
	Item     Item
	Findings []Finding
	Risk     float64
	Err      string
}

// Report is the ranked outcome of a batch audit, with metering totals.
type Report struct {
	Pack      string
	Results   []Result
	Judgments int
	TokensIn  int
	TokensOut int
	Sample    bool
}

const flagThreshold = 0.5

// Audit runs the pack over every item with at most `concurrency` in-flight Jev
// requests, computes a weighted risk per item, and returns a worst-first report.
// A failed item is recorded (Err set) and does not abort the run.
func Audit(ctx context.Context, j jev.Judge, pack Pack, items []Item, concurrency int) (*Report, error) {
	if concurrency < 1 {
		concurrency = 1
	}
	questions := pack.questions()
	results := make([]Result, len(items)) // each goroutine writes its own index

	var (
		mu                  sync.Mutex
		tokensIn, tokensOut int
		judged              int
		wg                  sync.WaitGroup
	)
	sem := make(chan struct{}, concurrency)

	for i := range items {
		if ctx.Err() != nil {
			break
		}
		wg.Add(1)
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			defer func() { <-sem }()

			item := items[i]
			resp, err := j.SystemOne(ctx, item.State, questions)
			if err != nil {
				results[i] = Result{Item: item, Err: err.Error()}
				return
			}
			res := Result{Item: item, Findings: make([]Finding, 0, len(pack.Checks))}
			var num, den float64
			for _, c := range pack.Checks {
				p := resp.Noul(c.Key)
				contrib := p
				if !c.BadWhenTrue {
					contrib = 1 - p
				}
				num += c.Weight * contrib
				den += c.Weight
				res.Findings = append(res.Findings, Finding{Key: c.Key, Label: c.Label, Prob: p, Flag: contrib >= flagThreshold})
			}
			if den > 0 {
				res.Risk = num / den
			}
			results[i] = res

			mu.Lock()
			tokensIn += resp.Usage.InputTokens
			tokensOut += resp.Usage.OutputTokens
			judged++
			mu.Unlock()
		}(i)
	}
	wg.Wait()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	sort.SliceStable(results, func(a, b int) bool {
		aok, bok := results[a].Err == "", results[b].Err == ""
		if aok != bok {
			return aok
		}
		return results[a].Risk > results[b].Risk
	})

	return &Report{
		Pack: pack.Name, Results: results, Judgments: judged,
		TokensIn: tokensIn, TokensOut: tokensOut, Sample: !j.Live(),
	}, nil
}

// ParseListingsCSV reads listing items from a CSV with a header row. Recognised
// columns (case-insensitive): sku, title, description, category, price. Reads at
// most max data rows.
func ParseListingsCSV(r io.Reader, max int) ([]Item, error) {
	rows, idx, err := readCSV(r, max, []string{"sku", "title", "description", "category", "price"})
	if err != nil {
		return nil, err
	}
	if idx["title"] < 0 && idx["description"] < 0 {
		return nil, errors.New("csv needs at least a 'title' or 'description' column")
	}
	var out []Item
	for _, rec := range rows {
		get := colGetter(rec.fields, idx)
		title, sku := get("title"), get("sku")
		if title == "" && get("description") == "" {
			continue
		}
		display := title
		if display == "" {
			display = sku
		}
		out = append(out, Item{
			Row:   rec.row,
			Title: display,
			State: map[string]string{"title": title, "description": get("description"), "category": get("category"), "price": get("price")},
			Cols:  []Field{{"sku", sku}, {"title", title}, {"category", get("category")}, {"price", get("price")}},
		})
	}
	if len(out) == 0 {
		return nil, errors.New("no listings found in csv")
	}
	return out, nil
}

// ParseAdsCSV reads ad variants from a CSV. Recognised columns: creative (required),
// landing, platform.
func ParseAdsCSV(r io.Reader, max int) ([]Item, error) {
	rows, idx, err := readCSV(r, max, []string{"creative", "landing", "platform"})
	if err != nil {
		return nil, err
	}
	if idx["creative"] < 0 {
		return nil, errors.New("csv needs a 'creative' column")
	}
	var out []Item
	for _, rec := range rows {
		get := colGetter(rec.fields, idx)
		creative := get("creative")
		if creative == "" {
			continue
		}
		platform := get("platform")
		if platform == "" {
			platform = "general"
		}
		out = append(out, Item{
			Row:   rec.row,
			Title: truncate(creative, 60),
			State: map[string]string{"creative": creative, "landing": get("landing"), "platform": platform},
			Cols:  []Field{{"platform", platform}, {"creative", creative}, {"landing", get("landing")}},
		})
	}
	if len(out) == 0 {
		return nil, errors.New("no ad variants found in csv")
	}
	return out, nil
}

// ParseClaimsCSV reads copy to screen from a CSV. Recognised columns: copy
// (required), category.
func ParseClaimsCSV(r io.Reader, max int) ([]Item, error) {
	rows, idx, err := readCSV(r, max, []string{"copy", "category"})
	if err != nil {
		return nil, err
	}
	if idx["copy"] < 0 {
		return nil, errors.New("csv needs a 'copy' column")
	}
	var out []Item
	for _, rec := range rows {
		get := colGetter(rec.fields, idx)
		copy := get("copy")
		if copy == "" {
			continue
		}
		cat := get("category")
		if cat == "" {
			cat = "general"
		}
		out = append(out, Item{
			Row:   rec.row,
			Title: truncate(copy, 60),
			State: map[string]string{"copy": copy, "category": cat},
			Cols:  []Field{{"category", cat}, {"copy", copy}},
		})
	}
	if len(out) == 0 {
		return nil, errors.New("no copy found in csv")
	}
	return out, nil
}

// --- shared CSV helpers ---

type csvRow struct {
	row    int
	fields []string
}

func readCSV(r io.Reader, max int, want []string) ([]csvRow, map[string]int, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.TrimLeadingSpace = true
	header, err := cr.Read()
	if err != nil {
		return nil, nil, fmt.Errorf("read header: %w", err)
	}
	idx := map[string]int{}
	for _, w := range want {
		idx[w] = -1
	}
	for i, h := range header {
		key := strings.ToLower(strings.TrimSpace(h))
		if _, ok := idx[key]; ok {
			idx[key] = i
		}
	}
	var out []csvRow
	row := 1
	for len(out) < max {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("row %d: %w", row+1, err)
		}
		row++
		out = append(out, csvRow{row: row, fields: rec})
	}
	return out, idx, nil
}

func colGetter(rec []string, idx map[string]int) func(string) string {
	return func(k string) string {
		if i := idx[k]; i >= 0 && i < len(rec) {
			return strings.TrimSpace(rec[i])
		}
		return ""
	}
}

func truncate(s string, n int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
