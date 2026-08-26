package application

import (
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"cleanroom-recovery-ledger/internal/archive"
	"cleanroom-recovery-ledger/internal/domain"
	"cleanroom-recovery-ledger/internal/store"
)

type Service struct {
	repo         Repository
	coordinator  *Coordinator
	now          func() time.Time
	summaryMu    sync.RWMutex
	summaryCache map[string]archive.Summary
}

func New(repo Repository) *Service {
	return &Service{repo: repo, coordinator: NewCoordinator(64), now: func() time.Time { return time.Now().UTC() }, summaryCache: map[string]archive.Summary{}}
}

func validateRequestID(v string) error {
	if strings.TrimSpace(v) == "" {
		return &domain.DomainError{Code: domain.CodeValidation, Field: "request_id", Message: "request_id 不能为空"}
	}
	if len(v) > 120 {
		return &domain.DomainError{Code: domain.CodeValidation, Field: "request_id", Message: "request_id 过长"}
	}
	return nil
}

func (s *Service) replay(caseID, requestID string) (*OperationResult, bool, error) {
	raw, ok, err := s.repo.Idempotent(caseID, requestID)
	if err != nil || !ok {
		return nil, ok, err
	}
	var out OperationResult
	if err = json.Unmarshal(raw, &out); err != nil {
		return nil, false, err
	}
	return &out, true, nil
}

func encodeResult(c *domain.DeviationCase) (json.RawMessage, error) {
	return json.Marshal(newOperationResult(c))
}

func newOperationResult(c *domain.DeviationCase) *OperationResult {
	return &OperationResult{Case: c, RetestProgress: c.RetestProgress()}
}

func (s *Service) mutate(caseID string, env Envelope, fn func(*domain.DeviationCase) error) (*OperationResult, error) {
	if err := validateRequestID(env.RequestID); err != nil {
		return nil, err
	}
	unlock := s.coordinator.lock(caseID)
	defer unlock()
	if result, ok, err := s.replay(caseID, env.RequestID); err != nil || ok {
		return result, err
	}
	c, err := s.repo.Load(caseID)
	if err != nil {
		return nil, err
	}
	if c.Revision != env.ExpectedRevision {
		return nil, domain.Conflict(env.ExpectedRevision, c.Revision)
	}
	if c.Status == domain.StatusReleased {
		return nil, &domain.DomainError{Code: domain.CodeState, Message: "案件已冻结，不能修改"}
	}
	if err = fn(c); err != nil {
		return nil, err
	}
	raw, err := encodeResult(c)
	if err != nil {
		return nil, err
	}
	if err = s.repo.Save(c, env.ExpectedRevision, env.RequestID, raw); err != nil {
		return nil, err
	}
	return newOperationResult(c), nil
}

func notFound(err error) bool { return errors.Is(err, store.ErrNotFound) }
