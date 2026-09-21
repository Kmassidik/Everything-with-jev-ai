package server

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"jevai/internal/account"
	"jevai/internal/judge"
	"jevai/internal/web"
)

// Bounds that keep one request memory- and cost-safe.
const (
	maxUploadBytes   = 5 << 20 // 5 MB
	maxRows          = 200
	auditConcurrency = 6
	auditTimeout     = 120 * time.Second
)

func (s *Server) listingPage(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requirePage(w, r)
	if !ok {
		return
	}
	_ = web.Listing(u, samplePreview(judge.SampleListingsCSV, judge.ParseListingsCSV)).Render(r.Context(), w)
}

// samplePreview parses the first few rows of a built-in sample CSV so a product page
// can show the actual input as a table before the user runs it.
func samplePreview(csvStr string, parse func(io.Reader, int) ([]judge.Item, error)) []judge.Item {
	items, _ := parse(strings.NewReader(csvStr), samplePreviewRows)
	return items
}

const samplePreviewRows = 5

func (s *Server) listingAudit(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireHTMX(w, r)
	if !ok {
		return
	}
	s.runBatch(w, r, u, judge.ListingHygiene, judge.ParseListingsCSV)
}

func (s *Server) adsAudit(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireHTMX(w, r)
	if !ok {
		return
	}
	s.runBatch(w, r, u, judge.AdPreflight, judge.ParseAdsCSV)
}

// runBatch handles a CSV upload → parse → judge → save → HTMX report, for any pack.
func (s *Server) runBatch(w http.ResponseWriter, r *http.Request, u *account.User, pack judge.Pack,
	parse func(io.Reader, int) ([]judge.Item, error)) {

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		_ = web.ErrorBox("Upload too large or malformed (max 5 MB).").Render(r.Context(), w)
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll()
		}
	}()

	file, _, err := r.FormFile("csv")
	if err != nil {
		_ = web.ErrorBox("Choose a CSV file to run.").Render(r.Context(), w)
		return
	}
	defer file.Close()

	items, err := parse(file, maxRows)
	if err != nil {
		_ = web.ErrorBox(err.Error()).Render(r.Context(), w)
		return
	}
	s.runItems(w, r, u, pack, items)
}

// runItems judges parsed items, meters, saves, and renders — shared by uploads and
// the "Run the sample" button.
func (s *Server) runItems(w http.ResponseWriter, r *http.Request, u *account.User, pack judge.Pack, items []judge.Item) {
	if s.gated(w, r, u.Username) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), auditTimeout)
	defer cancel()

	report, err := judge.Audit(ctx, s.judge, pack, items, auditConcurrency)
	if err != nil {
		s.log.Warn("audit failed", "pack", pack.Name, "err", err)
		_ = web.ErrorBox("Run failed: "+err.Error()).Render(r.Context(), w)
		return
	}
	if err := s.ledger.Record(r.Context(), u.Username, report.TokensIn, report.TokensOut, report.Judgments); err != nil {
		s.log.Warn("ledger record failed", "err", err)
	}
	id, err := s.audits.Save(r.Context(), u.ID, report)
	if err != nil {
		s.log.Warn("save audit failed", "err", err)
	}
	_ = web.BatchReport(report, id).Render(r.Context(), w)
}

// auditPage renders a saved audit at its (public, shareable) permalink.
func (s *Server) auditPage(w http.ResponseWriter, r *http.Request) {
	report, at, err := s.audits.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = web.AuditNotFound().Render(r.Context(), w)
		return
	}
	_ = web.AuditPage(s.currentUser(r), report, r.PathValue("id"), at).Render(r.Context(), w)
}

// auditCSV streams a saved audit as CSV (columns from the item fields + scores).
func (s *Server) auditCSV(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	report, _, err := s.audits.Get(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+report.Pack+"-"+id+".csv\"")

	cw := csv.NewWriter(w)
	defer cw.Flush()

	// header: row + item columns + risk + flags + detail
	var colKeys []string
	for _, res := range report.Results {
		for _, c := range res.Item.Cols {
			colKeys = append(colKeys, c.Key)
		}
		break
	}
	header := append([]string{"row"}, colKeys...)
	header = append(header, "risk_pct", "flagged_issues", "detail")
	_ = cw.Write(header)

	for _, res := range report.Results {
		rec := []string{strconv.Itoa(res.Item.Row)}
		for _, c := range res.Item.Cols {
			rec = append(rec, c.Val)
		}
		if res.Err != "" {
			rec = append(rec, "", "error: "+res.Err, "")
			_ = cw.Write(rec)
			continue
		}
		var flagged, detail []string
		for _, f := range res.Findings {
			detail = append(detail, fmt.Sprintf("%s=%.0f%%", f.Key, f.Prob*100))
			if f.Flag {
				flagged = append(flagged, f.Label)
			}
		}
		rec = append(rec, fmt.Sprintf("%.0f", res.Risk*100), strings.Join(flagged, "; "), strings.Join(detail, " "))
		_ = cw.Write(rec)
	}
}
