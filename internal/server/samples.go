package server

import (
	"net/http"
	"strings"

	"jevai/internal/judge"
)

// "Run the sample" — judge built-in example data so a new user sees the product work
// in one click, no file needed.

func (s *Server) listingSample(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireHTMX(w, r)
	if !ok {
		return
	}
	items, _ := judge.ParseListingsCSV(strings.NewReader(judge.SampleListingsCSV), maxRows)
	s.runItems(w, r, u, judge.ListingHygiene, items)
}

func (s *Server) adsSample(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireHTMX(w, r)
	if !ok {
		return
	}
	items, _ := judge.ParseAdsCSV(strings.NewReader(judge.SampleAdsCSV), maxRows)
	s.runItems(w, r, u, judge.AdPreflight, items)
}

func (s *Server) claimsSample(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requireHTMX(w, r)
	if !ok {
		return
	}
	items, _ := judge.ParseClaimsCSV(strings.NewReader(judge.SampleClaimsCSV), maxRows)
	s.runItems(w, r, u, judge.ClaimScreening, items)
}

// sample CSV templates (public — just a format example to download and edit).

func serveCSV(w http.ResponseWriter, filename, content string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	_, _ = w.Write([]byte(content))
}

func (s *Server) listingSampleCSV(w http.ResponseWriter, r *http.Request) {
	serveCSV(w, "listing-sample.csv", judge.SampleListingsCSV)
}
func (s *Server) adsSampleCSV(w http.ResponseWriter, r *http.Request) {
	serveCSV(w, "ads-sample.csv", judge.SampleAdsCSV)
}
func (s *Server) claimsSampleCSV(w http.ResponseWriter, r *http.Request) {
	serveCSV(w, "claims-sample.csv", judge.SampleClaimsCSV)
}
