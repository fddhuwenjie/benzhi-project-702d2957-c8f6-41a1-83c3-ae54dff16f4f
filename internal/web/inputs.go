package web

import (
	"time"

	"cleanroom-recovery-ledger/internal/application"
	"cleanroom-recovery-ledger/internal/domain"
)

type createRequest struct {
	RequestID        string               `json:"request_id"`
	CaseID           string               `json:"case_id"`
	RoomCode         string               `json:"room_code"`
	ExcursionKind    domain.ExcursionKind `json:"excursion_kind"`
	SamplePoint      string               `json:"sample_point"`
	ObservedValue    float64              `json:"observed_value"`
	LimitValue       float64              `json:"limit_value"`
	OccurredAt       time.Time            `json:"occurred_at"`
	WindowStart      time.Time            `json:"affected_window_start"`
	WindowEnd        time.Time            `json:"affected_window_end"`
	ActorID          string               `json:"actor_id"`
	OverlapConfirmed bool                 `json:"overlap_confirmed"`
}

func (v createRequest) command() application.CreateCase {
	return application.CreateCase{RequestID: v.RequestID, CaseID: v.CaseID, RoomCode: v.RoomCode, ExcursionKind: v.ExcursionKind, SamplePoint: v.SamplePoint, ObservedValue: v.ObservedValue, LimitValue: v.LimitValue, OccurredAt: v.OccurredAt, WindowStart: v.WindowStart, WindowEnd: v.WindowEnd, ActorID: v.ActorID, OverlapConfirmed: v.OverlapConfirmed}
}

type envelopeRequest struct {
	RequestID        string `json:"request_id"`
	ExpectedRevision int64  `json:"expected_revision"`
}

func (v envelopeRequest) envelope() application.Envelope {
	return application.Envelope{RequestID: v.RequestID, ExpectedRevision: v.ExpectedRevision}
}

type investigationRequest struct {
	envelopeRequest
	InvestigationID        string   `json:"investigation_id"`
	InvestigatorID         string   `json:"investigator_id"`
	PersonnelFindings      string   `json:"personnel_findings"`
	EquipmentFindings      string   `json:"equipment_findings"`
	CleaningFindings       string   `json:"cleaning_findings"`
	AdjacentSampleFindings string   `json:"adjacent_sample_findings"`
	RootCauseCategory      string   `json:"root_cause_category"`
	RootCauseStatement     string   `json:"root_cause_statement"`
	EvidenceDigests        []string `json:"evidence_digests"`
}

func (v investigationRequest) command() application.ConfirmInvestigation {
	return application.ConfirmInvestigation{Envelope: v.envelope(), Input: domain.InvestigationInput{InvestigationID: v.InvestigationID, InvestigatorID: v.InvestigatorID, PersonnelFindings: v.PersonnelFindings, EquipmentFindings: v.EquipmentFindings, CleaningFindings: v.CleaningFindings, AdjacentSampleFindings: v.AdjacentSampleFindings, RootCauseCategory: v.RootCauseCategory, RootCauseStatement: v.RootCauseStatement, EvidenceDigests: v.EvidenceDigests}}
}

type actionRequest struct {
	envelopeRequest
	ActionID         string `json:"action_id"`
	Description      string `json:"description"`
	OwnerID          string `json:"owner_id"`
	ActorID          string `json:"actor_id"`
	ReplacedActionID string `json:"replaced_action_id"`
}

func (v actionRequest) command() application.AddAction {
	return application.AddAction{Envelope: v.envelope(), Input: domain.AddActionInput{ActionID: v.ActionID, Description: v.Description, OwnerID: v.OwnerID, ActorID: v.ActorID, ReplacedActionID: v.ReplacedActionID}}
}

type completeActionRequest struct {
	envelopeRequest
	EvidenceDigest string `json:"evidence_digest"`
	ActorID        string `json:"actor_id"`
}
type revokeActionRequest struct {
	envelopeRequest
	Reason  string `json:"reason"`
	ActorID string `json:"actor_id"`
}
type verifyArchiveRequest struct {
	RequestID  string `json:"request_id"`
	VerifierID string `json:"verifier_id"`
}
type retestRequest struct {
	envelopeRequest
	RoundID        string    `json:"round_id"`
	SampledAt      time.Time `json:"sampled_at"`
	ObservedValue  float64   `json:"observed_value"`
	LimitValue     float64   `json:"limit_value"`
	InstrumentRef  string    `json:"instrument_ref"`
	EvidenceDigest string    `json:"evidence_digest"`
	RecordedBy     string    `json:"recorded_by"`
}

func (v retestRequest) command() application.RecordRetest {
	return application.RecordRetest{Envelope: v.envelope(), Input: domain.RetestInput{RoundID: v.RoundID, SampledAt: v.SampledAt, ObservedValue: v.ObservedValue, LimitValue: v.LimitValue, InstrumentRef: v.InstrumentRef, EvidenceDigest: v.EvidenceDigest, RecordedBy: v.RecordedBy}}
}

type reviewRequest struct {
	envelopeRequest
	ReviewerID string `json:"reviewer_id"`
	Decision   string `json:"decision"`
	Reason     string `json:"reason"`
}

func (v reviewRequest) command() application.ReviewCase {
	return application.ReviewCase{Envelope: v.envelope(), Input: domain.ReviewInput{ReviewerID: v.ReviewerID, Decision: v.Decision, Reason: v.Reason}}
}
