package server

import (
	"context"
	"net/http"
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
	_ = web.ListingReport(report).Render(r.Context(), w)
}
