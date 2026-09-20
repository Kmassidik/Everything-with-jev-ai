// Package server wires HTTP routes to the web surfaces and the judgment engine.
package server

import (
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"

	"jevai/internal/web"
)

// Config is the engine's runtime configuration (secrets come from the environment).
type Config struct {
	Port     string
	JevModel string
	JevKey   string // TypeSafe/Jev API key; empty = scaffold mode (no live judgments)
}

// Server holds dependencies for the HTTP handlers.
type Server struct {
	cfg Config
	log *slog.Logger
	mux *http.ServeMux
}

// New builds the router.
func New(cfg Config, log *slog.Logger) *Server {
	s := &Server{cfg: cfg, log: log, mux: http.NewServeMux()}
	s.routes()
	return s
}

// Handler returns the root http.Handler.
func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	static, _ := fs.Sub(web.Static, "static")
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))
	s.mux.HandleFunc("GET /{$}", s.landing)
	s.mux.HandleFunc("GET /health", s.health)
	s.mux.HandleFunc("POST /demo/judge", s.demoJudge)
}

func (s *Server) hasKey() bool { return s.cfg.JevKey != "" }

func (s *Server) landing(w http.ResponseWriter, r *http.Request) {
	_ = web.Landing(s.hasKey()).Render(r.Context(), w)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"service": "jevai-engine",
		"model":   s.cfg.JevModel,
		"jev_key": s.hasKey(),
	})
}

// demoJudge is the HTMX target for the landing "try it" form. It returns an HTML
// fragment. The live Jev call is wired in the build phase (see docs/PRD.md).
func (s *Server) demoJudge(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	question := r.FormValue("question")
	_ = web.DemoResult(question, s.hasKey()).Render(r.Context(), w)
}
