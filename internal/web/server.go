package web

import (
	"net/http"

	"cleanroom-recovery-ledger/internal/application"
)

type Server struct {
	app *application.Service
	mux *http.ServeMux
}

func New(app *application.Service) *Server {
	s := &Server{app: app, mux: http.NewServeMux()}
	s.routes()
	return s
}
func (s *Server) Handler() http.Handler { return securityHeaders(s.mux) }
func (s *Server) routes() {
	s.mux.HandleFunc("GET /", s.HandleWorkbench)
	s.mux.HandleFunc("GET /assets/app.css", s.HandleCSS)
	s.mux.HandleFunc("GET /assets/app.js", s.HandleJS)
	s.mux.HandleFunc("GET /api/health", s.HandleHealth)
	s.mux.HandleFunc("GET /api/cases", s.HandleListCases)
	s.mux.HandleFunc("POST /api/cases", s.HandleCreateCase)
	s.mux.HandleFunc("GET /api/cases/overlaps", s.HandleCaseOverlaps)
	s.mux.HandleFunc("GET /api/cases/{case_id}", s.HandleGetCase)
	s.mux.HandleFunc("GET /api/cases/{case_id}/timeline", s.HandleTimeline)
	s.mux.HandleFunc("POST /api/cases/{case_id}/investigation", s.HandleInvestigation)
	s.mux.HandleFunc("POST /api/cases/{case_id}/actions", s.HandleAddAction)
	s.mux.HandleFunc("POST /api/cases/{case_id}/actions/{action_id}/complete", s.HandleCompleteAction)
	s.mux.HandleFunc("POST /api/cases/{case_id}/actions/{action_id}/revoke", s.HandleRevokeAction)
	s.mux.HandleFunc("POST /api/cases/{case_id}/retests", s.HandleRetest)
	s.mux.HandleFunc("GET /api/cases/{case_id}/retests/progress", s.HandleRetestProgress)
	s.mux.HandleFunc("GET /api/cases/{case_id}/review/preflight", s.HandleReviewPreflight)
	s.mux.HandleFunc("POST /api/cases/{case_id}/review", s.HandleReview)
	s.mux.HandleFunc("GET /api/cases/{case_id}/archive", s.HandleArchive)
	s.mux.HandleFunc("GET /api/cases/{case_id}/archive/summary", s.HandleArchiveSummary)
	s.mux.HandleFunc("POST /api/cases/{case_id}/archive/verify", s.HandleVerifyArchive)
	s.mux.HandleFunc("GET /api/cases/{case_id}/archive/verify", s.HandleArchiveVerificationHistory)
}
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; connect-src 'self'")
		next.ServeHTTP(w, r)
	})
}
