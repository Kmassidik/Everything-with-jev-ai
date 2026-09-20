package server

import (
	"context"
	"net/http"
	"strings"
	"time"

	"jevai/internal/judge"
	"jevai/internal/web"
)

const (
	maxCopyBytes  = 1 << 20 // 1 MB of pasted copy is plenty
	screenTimeout = 30 * time.Second
)

func (s *Server) claimsPage(w http.ResponseWriter, r *http.Request) {
	_ = web.Claims().Render(r.Context(), w)
}

// claimsScreen runs the claim-screening pack over one piece of copy and returns the
// HTMX report fragment.
func (s *Server) claimsScreen(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxCopyBytes)
	if err := r.ParseForm(); err != nil {
		_ = web.ErrorBox("Copy too large or malformed (max 1 MB).").Render(r.Context(), w)
		return
	}
	copy := strings.TrimSpace(r.FormValue("copy"))
	if copy == "" {
		_ = web.ErrorBox("Paste some copy to screen.").Render(r.Context(), w)
		return
	}
	category := r.FormValue("category")
	if category == "" {
		category = "general"
	}

	ctx, cancel := context.WithTimeout(r.Context(), screenTimeout)
	defer cancel()

	rep, err := judge.Screen(ctx, s.judge, judge.ClaimScreening, judge.Claim{Copy: copy, Category: category})
	if err != nil {
		s.log.Warn("claim screen failed", "err", err)
		_ = web.ErrorBox("Screening failed: "+err.Error()).Render(r.Context(), w)
		return
	}

	if err := s.ledger.Record(r.Context(), s.user(r), rep.TokensIn, rep.TokensOut, 1); err != nil {
		s.log.Warn("ledger record failed", "err", err)
	}
	_ = web.ClaimReport(rep).Render(r.Context(), w)
}
