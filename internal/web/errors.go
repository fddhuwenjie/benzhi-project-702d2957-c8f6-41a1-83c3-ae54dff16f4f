package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"cleanroom-recovery-ledger/internal/domain"
	"cleanroom-recovery-ledger/internal/store"
)

type problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
	Field  string `json:"field,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func writeError(w http.ResponseWriter, err error) {
	p := problem{Type: "about:blank", Title: "请求失败", Status: http.StatusInternalServerError, Detail: "服务无法完成请求"}
	var de *domain.DomainError
	if errors.As(err, &de) {
		p.Type = "urn:cleanroom:error:" + string(de.Code)
		p.Detail = de.Message
		p.Field = de.Field
		switch de.Code {
		case domain.CodeValidation:
			p.Status = 400
		case domain.CodeConflict:
			p.Status = 409
		case domain.CodeState:
			p.Status = 422
		case domain.CodeForbidden:
			p.Status = 403
		case domain.CodeNotFound:
			p.Status = 404
		}
	} else if errors.Is(err, store.ErrNotFound) {
		p.Status = 404
		p.Detail = "案件不存在"
	}
	writeJSON(w, p.Status, p)
}
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return &domain.DomainError{Code: domain.CodeValidation, Message: "JSON 请求体无效：" + err.Error()}
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		return &domain.DomainError{Code: domain.CodeValidation, Message: "JSON 请求体只能包含一个对象"}
	}
	return nil
}
