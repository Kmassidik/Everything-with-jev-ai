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

// Listing is one row of the uploaded catalogue.
type Listing struct {
	Row         int
	SKU         string
	Title       string
	Description string
	Category    string
	Price       string
}

func (l Listing) state() map[string]string {
	return map[string]string{
		"title":       l.Title,
		"description": l.Description,
		"category":    l.Category,
		"price":       l.Price,
	}
}

// ParseCSV reads listings from a CSV with a header row. Recognised columns
// (case-insensitive): sku, title, description, category, price. Reads at most
// max data rows, so a huge upload can never blow up memory.
func ParseCSV(r io.Reader, max int) ([]Listing, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1 // tolerate ragged rows
	cr.TrimLeadingSpace = true

	header, err := cr.Read()
	if err != nil {
		return nil, fmt.Errorf("read header: %w", err)
	}
	idx := columnIndex(header)
	if idx["title"] < 0 && idx["description"] < 0 {
		return nil, errors.New("csv needs at least a 'title' or 'description' column")
	}

	out := make([]Listing, 0, min(max, 64))
	row := 1
	for len(out) < max {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", row+1, err)
		}
		row++
		get := func(k string) string {
			if i := idx[k]; i >= 0 && i < len(rec) {
				return strings.TrimSpace(rec[i])
			}
			return ""
		}
		l := Listing{Row: row, SKU: get("sku"), Title: get("title"), Description: get("description"), Category: get("category"), Price: get("price")}
		if l.Title == "" && l.Description == "" {
			continue // skip blank lines
		}
		out = append(out, l)
	}
	if len(out) == 0 {
		return nil, errors.New("no listings found in csv")
	}
	return out, nil
}

func columnIndex(header []string) map[string]int {
	idx := map[string]int{"sku": -1, "title": -1, "description": -1, "category": -1, "price": -1}
	for i, h := range header {
		if _, ok := idx[strings.ToLower(strings.TrimSpace(h))]; ok {
			idx[strings.ToLower(strings.TrimSpace(h))] = i
		}
	}
	return idx
}

// Finding is one check's result for one listing.
type Finding struct {
	Key  string
	Prob float64
	Flag bool
}

// Result is a listing plus its findings and overall risk (0..1, higher = worse).
type Result struct {
	Listing  Listing
	Findings []Finding
	Risk     float64
	Err      string
}

// Report is the ranked outcome of an audit, with metering totals.
type Report struct {
	Pack      string
	Results   []Result
	Judgments int
	TokensIn  int
	TokensOut int
	Sample    bool // produced by the mock judge (no live key)
}

const flagThreshold = 0.5

// Audit runs the pack over every listing with at most `concurrency` in-flight Jev
// requests, computes a weighted risk per listing, and returns a worst-first report.
// A failed row is recorded (Err set) and does not abort the whole audit.
func Audit(ctx context.Context, j jev.Judge, pack Pack, items []Listing, concurrency int) (*Report, error) {
	if concurrency < 1 {
		concurrency = 1
	}
	questions := pack.questions()
	results := make([]Result, len(items)) // each goroutine writes its own index — no shared slot

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
			resp, err := j.SystemOne(ctx, item.state(), questions)
			if err != nil {
				results[i] = Result{Listing: item, Err: err.Error()}
				return
			}

			res := Result{Listing: item, Findings: make([]Finding, 0, len(pack.Checks))}
			var num, den float64
			for _, c := range pack.Checks {
				p := resp.Noul(c.Key)
				contrib := p
				if !c.BadWhenTrue {
					contrib = 1 - p
				}
				num += c.Weight * contrib
				den += c.Weight
				res.Findings = append(res.Findings, Finding{Key: c.Key, Prob: p, Flag: contrib >= flagThreshold})
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

	// worst-first; errored rows sink to the bottom, stable within a tier
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
