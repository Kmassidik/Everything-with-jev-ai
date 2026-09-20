// Package server wires HTTP routes to the web surfaces, accounts, the Jev judge,
// and the ledger. Auth is invite-only: an admin mints invite links, users register
// a username + password, sessions are server-side cookies.
package server

import (
	"context"
	"encoding/json"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"jevai/internal/account"
	"jevai/internal/billing"
	"jevai/internal/jev"
	"jevai/internal/ledger"
	"jevai/internal/store"
	"jevai/internal/web"
)

const sessionCookie = "javai_session"

// Config is the engine's runtime configuration (secrets come from the environment).
type Config struct {
	Port        string
	JevModel    string
	JevKey      string
	TrialTokens int
	AdminToken  string // gates /admin (invite minting); empty disables the admin page
}

// Server holds the handler dependencies.
type Server struct {
	cfg      Config
	log      *slog.Logger
	judge    jev.Judge
	ledger   ledger.Ledger
	audits   *store.Audits
	accounts *account.Store
	plan     billing.Plan
	mux      *http.ServeMux
}

func New(cfg Config, log *slog.Logger, j jev.Judge, l ledger.Ledger, a *store.Audits, acc *account.Store) *Server {
	plan := billing.Free
	if cfg.TrialTokens >= 0 {
		plan.TrialTokens = cfg.TrialTokens
	}
	s := &Server{cfg: cfg, log: log, judge: j, ledger: l, audits: a, accounts: acc, plan: plan, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return s.mux }

func (s *Server) routes() {
	static, _ := fs.Sub(web.Static, "static")
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(static))))

	// public
	s.mux.HandleFunc("GET /{$}", s.landing)
	s.mux.HandleFunc("GET /health", s.health)
	s.mux.HandleFunc("POST /demo/judge", s.demoJudge) // landing teaser (anon)
	s.mux.HandleFunc("GET /login", s.loginPage)
	s.mux.HandleFunc("POST /login", s.loginSubmit)
	s.mux.HandleFunc("GET /join/{token}", s.joinPage)
	s.mux.HandleFunc("POST /join/{token}", s.joinSubmit)
	s.mux.HandleFunc("POST /logout", s.logout)

	// saved audits — shareable by unguessable id (public on purpose)
	s.mux.HandleFunc("GET /a/{id}", s.auditPage)
	s.mux.HandleFunc("GET /a/{id}/csv", s.auditCSV)

	// admin (token-gated)
	s.mux.HandleFunc("GET /admin", s.adminPage)
	s.mux.HandleFunc("POST /admin/invite", s.adminInvite)

	// authenticated app
	s.mux.HandleFunc("GET /app", s.dashboard)
	s.mux.HandleFunc("GET /profile", s.profilePage)
	s.mux.HandleFunc("POST /profile/name", s.profileName)
	s.mux.HandleFunc("POST /profile/password", s.profilePassword)
	s.mux.HandleFunc("GET /listing", s.listingPage)
	s.mux.HandleFunc("POST /listing/audit", s.listingAudit)
	s.mux.HandleFunc("GET /ads", s.adsPage)
	s.mux.HandleFunc("POST /ads/audit", s.adsAudit)
	s.mux.HandleFunc("GET /claims", s.claimsPage)
	s.mux.HandleFunc("POST /claims/screen", s.claimsScreen)
}

// ---- session helpers ----

func (s *Server) currentUser(r *http.Request) *account.User {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil
	}
	return s.accounts.UserBySession(r.Context(), c.Value)
}

func (s *Server) setSession(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/",
		HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode,
		Expires: time.Now().Add(30 * 24 * time.Hour),
	})
}

func (s *Server) clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}

// requirePage returns the user or redirects to /login (for full-page GETs).
func (s *Server) requirePage(w http.ResponseWriter, r *http.Request) (*account.User, bool) {
	u := s.currentUser(r)
	if u == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return nil, false
	}
	return u, true
}

// requireHTMX returns the user or asks HTMX to redirect to /login (for POST fragments).
func (s *Server) requireHTMX(w http.ResponseWriter, r *http.Request) (*account.User, bool) {
	u := s.currentUser(r)
	if u == nil {
		w.Header().Set("HX-Redirect", "/login")
		w.WriteHeader(http.StatusUnauthorized)
		return nil, false
	}
	return u, true
}

// ---- public ----

func (s *Server) landing(w http.ResponseWriter, r *http.Request) {
	_ = web.Landing(s.currentUser(r), s.judge.Live()).Render(r.Context(), w)
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok": true, "service": "jevai-engine", "model": s.cfg.JevModel, "jev_key": s.judge.Live(),
	})
}

func (s *Server) demoJudge(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	state := r.FormValue("state")
	question := strings.TrimSpace(r.FormValue("question"))
	if question == "" {
		question = "Does this convey urgency?"
	}
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()
	resp, err := s.judge.SystemOne(ctx, state, map[string]jev.Question{"q": {Type: "noul", Instructions: question}})
	if err != nil {
		s.log.Warn("demo judge failed", "err", err)
		_ = web.DemoResult(question, 0, false, true).Render(r.Context(), w)
		return
	}
	_ = s.ledger.Record(r.Context(), "anon", resp.Usage.InputTokens, resp.Usage.OutputTokens, 1)
	_ = web.DemoResult(question, resp.Noul("q"), s.judge.Live(), false).Render(r.Context(), w)
}

// ---- auth ----

func (s *Server) loginPage(w http.ResponseWriter, r *http.Request) {
	if s.currentUser(r) != nil {
		http.Redirect(w, r, "/app", http.StatusFound)
		return
	}
	_ = web.Login(false).Render(r.Context(), w)
}

func (s *Server) loginSubmit(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	u, sess, err := s.accounts.Login(r.Context(), r.FormValue("username"), r.FormValue("password"))
	if err != nil {
		_ = web.Login(true).Render(r.Context(), w)
		return
	}
	s.setSession(w, sess)
	s.log.Info("login", "user", u.Username)
	http.Redirect(w, r, "/app", http.StatusFound)
}

func (s *Server) joinPage(w http.ResponseWriter, r *http.Request) {
	tok := r.PathValue("token")
	if !s.accounts.InviteValid(r.Context(), tok) {
		w.WriteHeader(http.StatusNotFound)
		_ = web.JoinInvalid().Render(r.Context(), w)
		return
	}
	_ = web.Join(tok, "").Render(r.Context(), w)
}

func (s *Server) joinSubmit(w http.ResponseWriter, r *http.Request) {
	tok := r.PathValue("token")
	_ = r.ParseForm()
	u, sess, err := s.accounts.Register(r.Context(), tok, r.FormValue("username"), r.FormValue("password"))
	if err != nil {
		if err == account.ErrBadInvite {
			w.WriteHeader(http.StatusNotFound)
			_ = web.JoinInvalid().Render(r.Context(), w)
			return
		}
		_ = web.Join(tok, err.Error()).Render(r.Context(), w)
		return
	}
	s.setSession(w, sess)
	s.log.Info("register", "user", u.Username)
	http.Redirect(w, r, "/app", http.StatusFound)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		s.accounts.DeleteSession(r.Context(), c.Value)
	}
	s.clearSession(w)
	http.Redirect(w, r, "/", http.StatusFound)
}

// ---- admin ----

func (s *Server) adminOK(r *http.Request) bool {
	return s.cfg.AdminToken != "" && subtleEqual(r.URL.Query().Get("token"), s.cfg.AdminToken)
}

func (s *Server) adminPage(w http.ResponseWriter, r *http.Request) {
	if !s.adminOK(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	invites, _ := s.accounts.ListInvites(r.Context())
	_ = web.Admin(s.cfg.AdminToken, invites).Render(r.Context(), w)
}

func (s *Server) adminInvite(w http.ResponseWriter, r *http.Request) {
	if !s.adminOK(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	_ = r.ParseForm()
	if _, err := s.accounts.CreateInvite(r.Context(), r.FormValue("note")); err != nil {
		s.log.Warn("create invite failed", "err", err)
	}
	http.Redirect(w, r, "/admin?token="+s.cfg.AdminToken, http.StatusFound)
}

// ---- dashboard & profile ----

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requirePage(w, r)
	if !ok {
		return
	}
	recent, _ := s.audits.ListByUser(r.Context(), u.ID, 20)
	usage, _ := s.ledger.Usage(r.Context(), u.Username)
	_ = web.Dashboard(u, recent, usage, billing.Evaluate(usage.Tokens(), s.plan)).Render(r.Context(), w)
}

func (s *Server) profilePage(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requirePage(w, r)
	if !ok {
		return
	}
	_ = web.Profile(u, "").Render(r.Context(), w)
}

func (s *Server) profileName(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requirePage(w, r)
	if !ok {
		return
	}
	_ = r.ParseForm()
	if err := s.accounts.SetDisplayName(r.Context(), u.ID, r.FormValue("display_name")); err != nil {
		s.log.Warn("set name failed", "err", err)
	}
	http.Redirect(w, r, "/profile", http.StatusFound)
}

func (s *Server) profilePassword(w http.ResponseWriter, r *http.Request) {
	u, ok := s.requirePage(w, r)
	if !ok {
		return
	}
	_ = r.ParseForm()
	msg := "Password updated."
	if err := s.accounts.SetPassword(r.Context(), u.ID, r.FormValue("password")); err != nil {
		msg = err.Error()
	}
	_ = web.Profile(u, msg).Render(r.Context(), w)
}

// gated reports whether the caller is over their trial (writes the fragment if so).
func (s *Server) gated(w http.ResponseWriter, r *http.Request, user string) bool {
	u, err := s.ledger.Usage(r.Context(), user)
	if err != nil {
		return false
	}
	st := billing.Evaluate(u.Tokens(), s.plan)
	if st.Blocked {
		_ = web.GateBlocked(st).Render(r.Context(), w)
		return true
	}
	return false
}

func subtleEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
