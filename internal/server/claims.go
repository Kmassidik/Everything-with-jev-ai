package server

import (
	"net/http"

	"jevai/internal/judge"
	"jevai/internal/web"
)

func (s *Server) claimsPage(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requirePage(w, r)
	if !ok {
		return
	}
	_ = web.Claims(u).Render(r.Context(), w)
}

func (s *Server) claimsAudit(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireHTMX(w, r)
	if !ok {
		return
	}
	s.runBatch(w, r, u, judge.ClaimScreening, judge.ParseClaimsCSV)
}
