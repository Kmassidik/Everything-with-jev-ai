package server

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"jevai/internal/judge"
	"jevai/internal/web"
)

// Bounds that keep one request memory- and cost-safe.
const (
	maxUploadBytes   = 5 << 20 // 5 MB
	maxRows          = 200     // caps Jev calls, memory, and spend per audit
	auditConcurrency = 6       // in-flight Jev requests
	auditTimeout     = 90 * time.Second
)

func (s *Server) listingPage(w http.ResponseWriter, r *http.Request) {
	_ = web.Listing().Render(r.Context(), w)
}

// listingAudit parses an uploaded CSV, judges every row, and returns a ranked
// HTMX report fragment.
func (s *Server) listingAudit(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		_ = web.ErrorBox("Upload too large or malformed (max 5 MB).").Render(r.Context(), w)
		return
	}
	defer func() {
		if r.MultipartForm != nil {
			_ = r.MultipartForm.RemoveAll() // clean any spilled temp files
		}
	}()

	file, _, err := r.FormFile("csv")
	if err != nil {
		_ = web.ErrorBox("Choose a CSV file to audit.").Render(r.Context(), w)
		return
	}
	defer file.Close()

	items, err := judge.ParseCSV(file, maxRows)
	if err != nil {
		_ = web.ErrorBox(err.Error()).Render(r.Context(), w)
		return
	}
	if s.gated(w, r) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), auditTimeout)
	defer cancel()

	report, err := judge.Audit(ctx, s.judge, judge.ListingHygiene, items, auditConcurrency)
	if err != nil {
		s.log.Warn("listing audit failed", "err", err)
		_ = web.ErrorBox("Audit failed: "+err.Error()).Render(r.Context(), w)
		return
	}

	if err := s.ledger.Record(r.Context(), s.user(r), report.TokensIn, report.TokensOut, report.Judgments); err != nil {
		s.log.Warn("ledger record failed", "err", err)
	}

	// persist so the result is a shareable artifact (permalink + CSV), not throwaway
	id, err := s.audits.Save(r.Context(), report)
	if err != nil {
		s.log.Warn("save audit failed", "err", err)
	}
	_ = web.ListingReport(report, id).Render(r.Context(), w)
}

// auditPage renders a saved audit at its permalink.
func (s *Server) auditPage(w http.ResponseWriter, r *http.Request) {
	report, at, err := s.audits.Get(r.Context(), r.PathValue("id"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		_ = web.AuditNotFound().Render(r.Context(), w)
		return
	}
	_ = web.AuditPage(report, r.PathValue("id"), at).Render(r.Context(), w)
}

// auditCSV streams a saved audit as a CSV of the flagged rows.
func (s *Server) auditCSV(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	report, _, err := s.audits.Get(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"listing-audit-"+id+".csv\"")

	cw := csv.NewWriter(w)
	defer cw.Flush()
	_ = cw.Write([]string{"row", "sku", "title", "category", "price", "risk_pct", "flagged_issues", "detail"})
	for _, res := range report.Results {
		if res.Err != "" {
			_ = cw.Write([]string{strconv.Itoa(res.Listing.Row), res.Listing.SKU, res.Listing.Title, res.Listing.Category, res.Listing.Price, "", "error: " + res.Err, ""})
			continue
		}
		var flagged, detail []string
		for _, f := range res.Findings {
			detail = append(detail, fmt.Sprintf("%s=%.0f%%", f.Key, f.Prob*100))
			if f.Flag {
				flagged = append(flagged, f.Label)
			}
		}
		_ = cw.Write([]string{
			strconv.Itoa(res.Listing.Row), res.Listing.SKU, res.Listing.Title,
			res.Listing.Category, res.Listing.Price,
			fmt.Sprintf("%.0f", res.Risk*100),
			strings.Join(flagged, "; "), strings.Join(detail, " "),
		})
	}
}
