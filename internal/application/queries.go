package application

import (
	"strings"
	"time"

	"cleanroom-recovery-ledger/internal/archive"
	"cleanroom-recovery-ledger/internal/domain"
)

func (s *Service) Get(caseID string) (*domain.DeviationCase, error) {
	c, err := s.repo.Load(caseID)
	if err != nil {
		return nil, err
	}
	if cached, ok := s.overlapCache.Load(caseID); ok {
		c.RelatedCases = append([]domain.CaseOverlap(nil), cached.([]domain.CaseOverlap)...)
		return c, nil
	}
	all, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	c.RelatedCases = domain.FindOverlaps(c.RoomCode, c.AffectedWindowStart, c.AffectedWindowEnd, all, c.CaseID)
	s.overlapCache.Store(caseID, append([]domain.CaseOverlap(nil), c.RelatedCases...))
	return c, nil
}
func (s *Service) List() ([]*domain.DeviationCase, error) {
	cases, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	for _, c := range cases {
		c.RelatedCases = domain.FindOverlaps(c.RoomCode, c.AffectedWindowStart, c.AffectedWindowEnd, cases, c.CaseID)
	}
	return cases, nil
}
func (s *Service) Overlaps(roomCode string, start, end time.Time, excludeCaseID string) ([]domain.CaseOverlap, error) {
	cases, err := s.repo.List()
	if err != nil {
		return nil, err
	}
	return domain.FindOverlaps(strings.TrimSpace(roomCode), start, end, cases, excludeCaseID), nil
}
func (s *Service) Timeline(caseID string) ([]domain.TimelineEntry, error) {
	c, err := s.repo.Load(caseID)
	if err != nil {
		return nil, err
	}
	return append([]domain.TimelineEntry(nil), c.Timeline...), nil
}
func (s *Service) Preflight(caseID string) (ReviewPreflight, error) {
	c, err := s.repo.Load(caseID)
	if err != nil {
		return ReviewPreflight{}, err
	}
	progress := c.RetestProgress()
	p := ReviewPreflight{RequiredPassingRounds: progress.RequiredPassingRounds, ConsecutivePassing: progress.ConsecutivePassing, EvidenceComplete: c.EvidenceComplete(), BlockingReasons: []string{}, Participants: []string{}}
	if c.Investigation != nil {
		p.Participants = append(p.Participants, c.Investigation.InvestigatorID)
	}
	for _, a := range c.Actions {
		p.Participants = append(p.Participants, a.OwnerID)
	}
	p.BlockingReasons = append(p.BlockingReasons, progress.BlockingReasons...)
	if c.Status != domain.StatusReview {
		p.BlockingReasons = append(p.BlockingReasons, "案件状态不是待复核")
	}
	p.Eligible = len(p.BlockingReasons) == 0
	return p, nil
}
func (s *Service) Archive(caseID string) (*domain.ReleaseArchive, error) {
	c, err := s.repo.Load(caseID)
	if err != nil {
		return nil, err
	}
	if c.Archive == nil {
		return nil, &domain.DomainError{Code: domain.CodeNotFound, Message: "案件尚无放行档案"}
	}
	return c.Archive, nil
}
func (s *Service) ArchiveSummary(caseID string) (archive.Summary, error) {
	a, err := s.Archive(caseID)
	if err != nil {
		return archive.Summary{}, err
	}
	return archive.Summarize(a)
}

func (s *Service) RetestProgress(caseID string) (domain.RetestProgress, error) {
	c, err := s.repo.Load(caseID)
	if err != nil {
		return domain.RetestProgress{}, err
	}
	return c.RetestProgress(), nil
}

func (s *Service) ArchiveVerificationHistory(caseID string) (ArchiveVerificationView, error) {
	c, err := s.repo.Load(caseID)
	if err != nil {
		return ArchiveVerificationView{}, err
	}
	if c.Status != domain.StatusReleased || c.Archive == nil {
		return ArchiveVerificationView{}, &domain.DomainError{Code: domain.CodeState, Message: "案件未放行或没有放行档案"}
	}
	records, err := s.repo.ArchiveVerifications(caseID)
	if err != nil {
		return ArchiveVerificationView{}, err
	}
	view := ArchiveVerificationView{History: records}
	if len(records) > 0 {
		view.Latest = &records[0]
	}
	return view, nil
}
