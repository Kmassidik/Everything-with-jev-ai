package server

import (
	"net/http"

	"jevai/internal/web"
)

func (s *Server) adsPage(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requirePage(w, r)
	if !ok {
		return
	}
	_ = web.Ads(u).Render(r.Context(), w)
}
