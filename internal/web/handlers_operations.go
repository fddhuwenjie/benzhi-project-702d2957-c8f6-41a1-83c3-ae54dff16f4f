package web

import (
	"net/http"

	"cleanroom-recovery-ledger/internal/application"
	"cleanroom-recovery-ledger/internal/domain"
)

func (s *Server) HandleInvestigation(w http.ResponseWriter, r *http.Request) {
	var input investigationRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.app.ConfirmInvestigation(r.PathValue("case_id"), input.command())
	respondOperation(w, result, err)
}
func (s *Server) HandleAddAction(w http.ResponseWriter, r *http.Request) {
	var input actionRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.app.AddAction(r.PathValue("case_id"), input.command())
	respondOperation(w, result, err)
}
func (s *Server) HandleCompleteAction(w http.ResponseWriter, r *http.Request) {
	var input completeActionRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	cmd := application.CompleteAction{Envelope: input.envelope(), Input: domain.CompleteActionInput{ActionID: r.PathValue("action_id"), EvidenceDigest: input.EvidenceDigest, ActorID: input.ActorID}}
	result, err := s.app.CompleteAction(r.PathValue("case_id"), cmd)
	respondOperation(w, result, err)
}
func (s *Server) HandleRevokeAction(w http.ResponseWriter, r *http.Request) {
	var input revokeActionRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	cmd := application.RevokeAction{Envelope: input.envelope(), Input: domain.RevokeActionInput{ActionID: r.PathValue("action_id"), Reason: input.Reason, ActorID: input.ActorID}}
	result, err := s.app.RevokeAction(r.PathValue("case_id"), cmd)
	respondOperation(w, result, err)
}
func (s *Server) HandleRetest(w http.ResponseWriter, r *http.Request) {
	var input retestRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.app.RecordRetest(r.PathValue("case_id"), input.command())
	respondOperation(w, result, err)
}
func (s *Server) HandleRetestProgress(w http.ResponseWriter, r *http.Request) {
	p, err := s.app.RetestProgress(r.PathValue("case_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, p)
}
func (s *Server) HandleReviewPreflight(w http.ResponseWriter, r *http.Request) {
	p, err := s.app.Preflight(r.PathValue("case_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, p)
}
func (s *Server) HandleReview(w http.ResponseWriter, r *http.Request) {
	var input reviewRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	result, err := s.app.Review(r.PathValue("case_id"), input.command())
	respondOperation(w, result, err)
}
func (s *Server) HandleArchive(w http.ResponseWriter, r *http.Request) {
	a, err := s.app.Archive(r.PathValue("case_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, a)
}
func (s *Server) HandleArchiveSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := s.app.ArchiveSummary(r.PathValue("case_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, summary)
}
func (s *Server) HandleVerifyArchive(w http.ResponseWriter, r *http.Request) {
	var input verifyArchiveRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, err)
		return
	}
	v, err := s.app.VerifyArchiveContext(r.Context(), r.PathValue("case_id"), application.VerifyArchive{RequestID: input.RequestID, VerifierID: input.VerifierID})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, v)
}
func (s *Server) HandleArchiveVerificationHistory(w http.ResponseWriter, r *http.Request) {
	v, err := s.app.ArchiveVerificationHistory(r.PathValue("case_id"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, v)
}
func respondOperation(w http.ResponseWriter, result *application.OperationResult, err error) {
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, 200, result)
}
