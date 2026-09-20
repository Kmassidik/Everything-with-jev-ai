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
	maxAdBytes    = 1 << 20 // 1 MB of pasted creative + landing copy
	preflightWait = 30 * time.Second
)

func (s *Server) adsPage(w http.ResponseWriter, r *http.Request) {
	_ = web.Ads().Render(r.Context(), w)
}

// adsPreflight screens one ad creative (against its landing copy when supplied) and
// returns the HTMX report fragment.
func (s *Server) adsPreflight(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, maxAdBytes)
	if err := r.ParseForm(); err != nil {
		_ = web.ErrorBox("Input too large or malformed (max 1 MB).").Render(r.Context(), w)
		return
	}
	creative := strings.TrimSpace(r.FormValue("creative"))
	if creative == "" {
		_ = web.ErrorBox("Paste an ad creative to pre-flight.").Render(r.Context(), w)
		return
	}
	platform := r.FormValue("platform")
	if platform == "" {
		platform = "general"
	}

	ctx, cancel := context.WithTimeout(r.Context(), preflightWait)
	defer cancel()

	rep, err := judge.Preflight(ctx, s.judge, judge.AdPreflight, judge.Ad{
		Creative: creative,
		Landing:  strings.TrimSpace(r.FormValue("landing")),
		Platform: platform,
	})
	if err != nil {
		s.log.Warn("ad preflight failed", "err", err)
		_ = web.ErrorBox("Pre-flight failed: "+err.Error()).Render(r.Context(), w)
		return
	}

	if err := s.ledger.Record(r.Context(), s.user(r), rep.TokensIn, rep.TokensOut, 1); err != nil {
		s.log.Warn("ledger record failed", "err", err)
	}
	_ = web.ItemReport(rep, "platform").Render(r.Context(), w)
}
