package server

import (
	"net/http"

	"jevai/internal/judge"
	"jevai/internal/web"
)

func (s *Server) adsPage(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requirePage(w, r)
	if !ok {
		return
	}
	_ = web.Ads(u, samplePreview(judge.SampleAdsCSV, judge.ParseAdsCSV)).Render(r.Context(), w)
}
