package application

import (
	"time"

	"cleanroom-recovery-ledger/internal/archive"
	"cleanroom-recovery-ledger/internal/domain"
)

type Envelope struct {
	RequestID        string `json:"request_id"`
	ExpectedRevision int64  `json:"expected_revision"`
}
type CreateCase struct {
	RequestID, CaseID, RoomCode, SamplePoint, ActorID string
	ExcursionKind                                     domain.ExcursionKind
	ObservedValue, LimitValue                         float64
	OccurredAt, WindowStart, WindowEnd                time.Time
	OverlapConfirmed                                  bool
}
type ConfirmInvestigation struct {
	Envelope
	Input domain.InvestigationInput
}
type AddAction struct {
	Envelope
	Input domain.AddActionInput
}
type CompleteAction struct {
	Envelope
	Input domain.CompleteActionInput
}
type RevokeAction struct {
	Envelope
	Input domain.RevokeActionInput
}
type RecordRetest struct {
	Envelope
	Input domain.RetestInput
}
type ReviewCase struct {
	Envelope
	Input domain.ReviewInput
}
type OperationResult struct {
	Case           *domain.DeviationCase `json:"case"`
	RetestProgress domain.RetestProgress `json:"retest_progress"`
	Replayed       bool                  `json:"replayed"`
}
type VerifyArchive struct {
	RequestID, VerifierID string
}
type ArchiveVerificationView struct {
	Latest  *archive.VerificationRecord  `json:"latest,omitempty"`
	History []archive.VerificationRecord `json:"history"`
}
type ReviewPreflight struct {
	Eligible              bool     `json:"eligible"`
	RequiredPassingRounds int      `json:"required_passing_rounds"`
	ConsecutivePassing    int      `json:"consecutive_passing"`
	EvidenceComplete      bool     `json:"evidence_complete"`
	BlockingReasons       []string `json:"blocking_reasons"`
	Participants          []string `json:"participants"`
}
