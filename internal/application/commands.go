package application

import (
	"context"
	"strings"

	"cleanroom-recovery-ledger/internal/archive"
	"cleanroom-recovery-ledger/internal/domain"
)

func (s *Service) Create(in CreateCase) (*OperationResult, error) {
	if err := validateRequestID(in.RequestID); err != nil {
		return nil, err
	}
	unlock := s.coordinator.lock("room:" + strings.TrimSpace(in.RoomCode))
	defer unlock()
	if result, ok, err := s.replay(in.CaseID, in.RequestID); err != nil || ok {
		return result, err
	}
	cases, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	overlaps := domain.FindOverlaps(strings.TrimSpace(in.RoomCode), in.WindowStart, in.WindowEnd, cases, in.CaseID)
	if len(overlaps) > 0 && !in.OverlapConfirmed {
		return nil, &domain.DomainError{Code: domain.CodeState, Field: "overlap_confirmed", Message: "发现同房间交叠案件，必须确认已查看关联范围后再创建"}
	}
	now := s.now()
	c, err := domain.Create(domain.CreateInput{CaseID: in.CaseID, RoomCode: in.RoomCode, ExcursionKind: in.ExcursionKind, SamplePoint: in.SamplePoint, ObservedValue: in.ObservedValue, LimitValue: in.LimitValue, OccurredAt: in.OccurredAt, WindowStart: in.WindowStart, WindowEnd: in.WindowEnd, ActorID: in.ActorID, Now: now, RelatedCases: overlaps})
	if err != nil {
		return nil, err
	}
	raw, err := encodeResult(c)
	if err != nil {
		return nil, err
	}
	if err = s.repo.Save(c, 0, in.RequestID, raw); err != nil {
		return nil, err
	}
	return newOperationResult(c), nil
}
func (s *Service) ConfirmInvestigation(caseID string, cmd ConfirmInvestigation) (*OperationResult, error) {
	cmd.Input.Now = s.now()
	return s.mutate(caseID, cmd.Envelope, func(c *domain.DeviationCase) error { return c.ConfirmInvestigation(cmd.Input) })
}
func (s *Service) AddAction(caseID string, cmd AddAction) (*OperationResult, error) {
	cmd.Input.Now = s.now()
	return s.mutate(caseID, cmd.Envelope, func(c *domain.DeviationCase) error { return c.AddAction(cmd.Input) })
}
func (s *Service) CompleteAction(caseID string, cmd CompleteAction) (*OperationResult, error) {
	cmd.Input.Now = s.now()
	return s.mutate(caseID, cmd.Envelope, func(c *domain.DeviationCase) error { return c.CompleteAction(cmd.Input) })
}
func (s *Service) RevokeAction(caseID string, cmd RevokeAction) (*OperationResult, error) {
	cmd.Input.Now = s.now()
	return s.mutate(caseID, cmd.Envelope, func(c *domain.DeviationCase) error { return c.RevokeAction(cmd.Input) })
}
func (s *Service) RecordRetest(caseID string, cmd RecordRetest) (*OperationResult, error) {
	cmd.Input.Now = s.now()
	return s.mutate(caseID, cmd.Envelope, func(c *domain.DeviationCase) error { return c.RecordRetest(cmd.Input) })
}
func (s *Service) Review(caseID string, cmd ReviewCase) (*OperationResult, error) {
	cmd.Input.Now = s.now()
	if err := validateRequestID(cmd.RequestID); err != nil {
		return nil, err
	}
	unlock := s.coordinator.lock(caseID)
	defer unlock()
	if result, ok, err := s.replay(caseID, cmd.RequestID); err != nil || ok {
		return result, err
	}
	c, err := s.repo.Load(caseID)
	if err != nil {
		return nil, err
	}
	if c.Revision != cmd.ExpectedRevision {
		return nil, domain.Conflict(cmd.ExpectedRevision, c.Revision)
	}
	if err = c.ReviewDecision(cmd.Input); err != nil {
		return nil, err
	}
	if cmd.Input.Decision == "approve" {
		a, genErr := archive.Generate(c)
		if genErr != nil {
			return nil, genErr
		}
		if err = c.AttachArchive(a); err != nil {
			return nil, err
		}
	}
	raw, err := encodeResult(c)
	if err != nil {
		return nil, err
	}
	if err = s.repo.Save(c, cmd.ExpectedRevision, cmd.RequestID, raw); err != nil {
		return nil, err
	}
	return newOperationResult(c), nil
}

func (s *Service) VerifyArchive(caseID string, cmd VerifyArchive) (archive.VerificationRecord, error) {
	return s.VerifyArchiveContext(context.Background(), caseID, cmd)
}

// VerifyArchiveContext 将请求生命周期带入档案校验，审计仓储负责持久化结果。
func (s *Service) VerifyArchiveContext(ctx context.Context, caseID string, cmd VerifyArchive) (archive.VerificationRecord, error) {
	if err := ctx.Err(); err != nil {
		return archive.VerificationRecord{}, err
	}
	if err := validateRequestID(cmd.RequestID); err != nil {
		return archive.VerificationRecord{}, err
	}
	if strings.TrimSpace(cmd.VerifierID) == "" {
		return archive.VerificationRecord{}, &domain.DomainError{Code: domain.CodeValidation, Field: "verifier_id", Message: "verifier_id 不能为空"}
	}
	unlock := s.coordinator.lock(caseID)
	defer unlock()
	c, err := s.repo.Load(caseID)
	if err != nil {
		return archive.VerificationRecord{}, err
	}
	if c.Status != domain.StatusReleased || c.Archive == nil {
		return archive.VerificationRecord{}, &domain.DomainError{Code: domain.CodeState, Message: "案件未放行或没有放行档案，不能校验"}
	}
	record := archive.VerificationRecord{RequestID: cmd.RequestID, VerifierID: strings.TrimSpace(cmd.VerifierID), ExecutedAt: s.now(), Result: archive.VerifyCase(c)}
	stored, _, err := s.repo.SaveArchiveVerification(caseID, record)
	return stored, err
}
