package domain

import "time"

type CreateInput struct {
	CaseID, RoomCode, SamplePoint, ActorID  string
	ExcursionKind                           ExcursionKind
	ObservedValue, LimitValue               float64
	OccurredAt, WindowStart, WindowEnd, Now time.Time
	RelatedCases                            []CaseOverlap
}
type InvestigationInput struct {
	InvestigationID, InvestigatorID                               string
	PersonnelFindings, EquipmentFindings, CleaningFindings        string
	AdjacentSampleFindings, RootCauseCategory, RootCauseStatement string
	EvidenceDigests                                               []string
	Now                                                           time.Time
}
type AddActionInput struct {
	ActionID, Description, OwnerID, ActorID, ReplacedActionID string
	Now                                                       time.Time
}
type CompleteActionInput struct {
	ActionID, EvidenceDigest, ActorID string
	Now                               time.Time
}
type RevokeActionInput struct {
	ActionID, Reason, ActorID string
	Now                       time.Time
}
type RetestInput struct {
	RoundID, InstrumentRef, EvidenceDigest, RecordedBy string
	SampledAt                                          time.Time
	ObservedValue, LimitValue                          float64
	Now                                                time.Time
}
type ReviewInput struct {
	ReviewerID, Decision, Reason string
	Now                          time.Time
}
