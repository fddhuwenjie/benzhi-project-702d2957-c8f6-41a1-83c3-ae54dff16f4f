package web

import (
	"net/http"
	"time"

	"cleanroom-recovery-ledger/internal/domain"
)

func (s *Server) HandleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{"status": "ok"})
}
func (s *Server) HandleListCases(w http.ResponseWriter, r *http.Request) {
	cases, err := s.app.List()
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"cases": cases})
}
func (s *Server) HandleCreateCase(w http.ResponseWriter, r *http.Request) {
	var input createRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.app.Create(input.command())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, result)
}
func (s *Server) HandleCaseOverlaps(w http.ResponseWriter, r *http.Request) {
	start, err := time.Parse(time.RFC3339, r.URL.Query().Get("affected_window_start"))
	if err != nil {
		writeError(w, &domain.DomainError{Code: domain.CodeValidation, Field: "affected_window_start", Message: "affected_window_start 必须是 RFC3339 时间"})
		return
	}
	end, err := time.Parse(time.RFC3339, r.URL.Query().Get("affected_window_end"))
	if err != nil || end.Before(start) {
		writeError(w, &domain.DomainError{Code: domain.CodeValidation, Field: "affected_window_end", Message: "affected_window_end 必须是不早于开始时间的 RFC3339 时间"})
		return
	}
	overlaps, err := s.app.Overlaps(r.URL.Query().Get("room_code"), start, end, r.URL.Query().Get("exclude_case_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"overlaps": overlaps, "confirmation_required": len(overlaps) > 0})
}
func (s *Server) HandleGetCase(w http.ResponseWriter, r *http.Request) {
	c, err := s.app.Get(r.PathValue("case_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, c)
}
func (s *Server) HandleTimeline(w http.ResponseWriter, r *http.Request) {
	timeline, err := s.app.Timeline(r.PathValue("case_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"timeline": timeline})
}
