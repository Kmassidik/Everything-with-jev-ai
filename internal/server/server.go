// Package server wires HTTP routes to the web surfaces, the Jev judge, and the ledger.
package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"jevai/internal/billing"
	"jevai/internal/jev"
	"jevai/internal/ledger"
	"jevai/internal/store"
	"jevai/internal/web"
)

// Config is the engine's runtime configuration (secrets come from the environment).
type Config struct {
	Port        string
	JevModel    string
	JevKey      string
	TrialTokens int // free-trial token allowance
}

// Server holds the handler dependencies.
type Server struct {
	cfg    Config
	log    *slog.Logger
	judge  jev.Judge
	ledger ledger.Ledger
	audits *store.Audits
	plan   billing.Plan
	mux    *http.ServeMux
}

// New builds the router with its dependencies injected.
func New(cfg Config, log *slog.Logger, j jev.Judge, l ledger.Ledger, a *store.Audits) *Server {
	plan := billing.Free
	if cfg.TrialTokens >= 0 {
		plan.TrialTokens = cfg.TrialTokens
	}
	s := &Server{cfg: cfg, log: log, judge: j, ledger: l, audits: a, plan: plan, mux: http.NewServeMux()}
	s.routes()
	return s
}

// gated reports whether the caller is over their trial and, if so, writes the
// "trial spent" fragment. Fails open on a ledger error (a DB blip must not block use).
func (s *Server) gated(w http.ResponseWriter, r *http.Request) bool {
	u, err := s.ledger.Usage(r.Context(), s.user(r))
	if err != nil {
		s.log.Warn("gate: usage read failed", "err", err)
		return false
	}
	st := billing.Evaluate(u.Tokens(), s.plan)
	if st.Blocked {
		_ = web.GateBlocked(st).Render(r.Context(), w)
		return true
	}
	return false
}

// Handler returns the root http.Handler.
func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	static, _ := fs.Sub(web.Static, "static")
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	s.mux.HandleFunc("GET /{$}", s.landing)
	s.mux.HandleFunc("GET /health", s.health)
	s.mux.HandleFunc("POST /demo/judge", s.demoJudge)
	s.mux.HandleFunc("GET /listing", s.listingPage)
	s.mux.HandleFunc("POST /listing/audit", s.listingAudit)
	s.mux.HandleFunc("GET /a/{id}", s.auditPage)
	s.mux.HandleFunc("GET /a/{id}/csv", s.auditCSV)
	s.mux.HandleFunc("GET /claims", s.claimsPage)
	s.mux.HandleFunc("POST /claims/screen", s.claimsScreen)
	s.mux.HandleFunc("GET /ads", s.adsPage)
	s.mux.HandleFunc("POST /ads/preflight", s.adsPreflight)
}

// user identifies the caller for metering. Real auth arrives with the payment gate;
// for now every request meters to one demo account.
func (s *Server) user(*http.Request) string { return "demo" }

func (s *Server) landing(w http.ResponseWriter, r *http.Request) {
	_ = web.Landing(s.judge.Live()).Render(r.Context(), w)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"service": "jevai-engine",
		"model":   s.cfg.JevModel,
		"jev_key": s.judge.Live(),
	})
}

// demoJudge is the landing "try it" HTMX target: one live noul via the judge.
func (s *Server) demoJudge(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	state := r.FormValue("state")
	question := strings.TrimSpace(r.FormValue("question"))
	if question == "" {
		question = "Does this convey urgency?"
	}
	if s.gated(w, r) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	resp, err := s.judge.SystemOne(ctx, state, map[string]jev.Question{
		"q": {Type: "noul", Instructions: question},
	})
	if err != nil {
		s.log.Warn("demo judge failed", "err", err)
		_ = web.DemoResult(question, 0, false, true).Render(r.Context(), w)
		return
	}
	_ = s.ledger.Record(r.Context(), s.user(r), resp.Usage.InputTokens, resp.Usage.OutputTokens, 1)
	_ = web.DemoResult(question, resp.Noul("q"), s.judge.Live(), false).Render(r.Context(), w)
}
