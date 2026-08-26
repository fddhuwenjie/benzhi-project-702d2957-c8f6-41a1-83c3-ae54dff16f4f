package domain

import "fmt"

type ErrorCode string

const (
	CodeValidation ErrorCode = "validation_failed"
	CodeConflict   ErrorCode = "revision_conflict"
	CodeState      ErrorCode = "invalid_state"
	CodeForbidden  ErrorCode = "reviewer_not_independent"
	CodeNotFound   ErrorCode = "not_found"
)

type DomainError struct {
	Code           ErrorCode
	Field, Message string
}

func (e *DomainError) Error() string { return e.Message }
func invalid(field, message string) error {
	return &DomainError{Code: CodeValidation, Field: field, Message: message}
}
func state(message string) error { return &DomainError{Code: CodeState, Message: message} }
func Conflict(expected, actual int64) error {
	return &DomainError{Code: CodeConflict, Field: "expected_revision", Message: fmt.Sprintf("修订冲突：期望 %d，当前 %d", expected, actual)}
}
