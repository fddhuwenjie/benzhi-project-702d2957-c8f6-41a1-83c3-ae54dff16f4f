package domain

import "time"

type Status string

const (
	StatusInvestigating Status = "investigating"
	StatusCorrecting    Status = "correcting"
	StatusRetesting     Status = "retesting"
	StatusReview        Status = "pending_review"
	StatusReleased      Status = "released"
)

type ExcursionKind string

const (
	KindMicrobial ExcursionKind = "microbial"
	KindParticle  ExcursionKind = "particle"
)

type DeviationCase struct {
	CaseID              string             `json:"case_id"`
	RoomCode            string             `json:"room_code"`
	ExcursionKind       ExcursionKind      `json:"excursion_kind"`
	SamplePoint         string             `json:"sample_point"`
	ObservedValue       float64            `json:"observed_value"`
	LimitValue          float64            `json:"limit_value"`
	OccurredAt          time.Time          `json:"occurred_at"`
	AffectedWindowStart time.Time          `json:"affected_window_start"`
	AffectedWindowEnd   time.Time          `json:"affected_window_end"`
	Status              Status             `json:"status"`
	Revision            int64              `json:"revision"`
	CreatedAt           time.Time          `json:"created_at"`
	FrozenAt            *time.Time         `json:"frozen_at,omitempty"`
	Investigation       *Investigation     `json:"investigation,omitempty"`
	Actions             []CorrectiveAction `json:"corrective_actions"`
	Retests             []RetestRound      `json:"retests"`
	Review              *Review            `json:"review,omitempty"`
	Archive             *ReleaseArchive    `json:"archive,omitempty"`
	RelatedCases        []CaseOverlap      `json:"related_cases"`
	Timeline            []TimelineEntry    `json:"timeline"`
}

// CaseOverlap freezes the relationship that was reviewed when a case was
// created. Later release of either case does not erase this historical scope.
type CaseOverlap struct {
	CaseID        string        `json:"case_id"`
	Status        Status        `json:"status"`
	ExcursionKind ExcursionKind `json:"excursion_kind"`
	OverlapStart  time.Time     `json:"overlap_start"`
	OverlapEnd    time.Time     `json:"overlap_end"`
}

type Investigation struct {
	InvestigationID        string    `json:"investigation_id"`
	CaseID                 string    `json:"case_id"`
	InvestigatorID         string    `json:"investigator_id"`
	PersonnelFindings      string    `json:"personnel_findings"`
	EquipmentFindings      string    `json:"equipment_findings"`
	CleaningFindings       string    `json:"cleaning_findings"`
	AdjacentSampleFindings string    `json:"adjacent_sample_findings"`
	RootCauseCategory      string    `json:"root_cause_category"`
	RootCauseStatement     string    `json:"root_cause_statement"`
	EvidenceDigests        []string  `json:"evidence_digests"`
	ConfirmedAt            time.Time `json:"confirmed_at"`
}

type ActionStatus string

const (
	ActionOpen      ActionStatus = "open"
	ActionCompleted ActionStatus = "completed"
	ActionRevoked   ActionStatus = "revoked"
)

type CorrectiveAction struct {
	ActionID         string       `json:"action_id"`
	CaseID           string       `json:"case_id"`
	Description      string       `json:"description"`
	OwnerID          string       `json:"owner_id"`
	Status           ActionStatus `json:"status"`
	EvidenceDigest   string       `json:"evidence_digest,omitempty"`
	CompletedAt      *time.Time   `json:"completed_at,omitempty"`
	RevokedAt        *time.Time   `json:"revoked_at,omitempty"`
	RevocationReason string       `json:"revocation_reason,omitempty"`
	RevokedBy        string       `json:"revoked_by,omitempty"`
	ReplacedActionID string       `json:"replaced_action_id,omitempty"`
}

type RetestRound struct {
	RoundID        string    `json:"round_id"`
	CaseID         string    `json:"case_id"`
	Sequence       int       `json:"sequence"`
	SampledAt      time.Time `json:"sampled_at"`
	ObservedValue  float64   `json:"observed_value"`
	LimitValue     float64   `json:"limit_value"`
	InstrumentRef  string    `json:"instrument_ref"`
	EvidenceDigest string    `json:"evidence_digest"`
	Outcome        string    `json:"outcome"`
	RecordedBy     string    `json:"recorded_by"`
}

type Review struct {
	ReviewerID string    `json:"reviewer_id"`
	Decision   string    `json:"decision"`
	Reason     string    `json:"reason,omitempty"`
	ReviewedAt time.Time `json:"reviewed_at"`
}

type ReleaseArchive struct {
	ArchiveID          string    `json:"archive_id"`
	CaseID             string    `json:"case_id"`
	ApprovedBy         string    `json:"approved_by"`
	ApprovedAt         time.Time `json:"approved_at"`
	ManifestVersion    string    `json:"manifest_version"`
	CanonicalDigest    string    `json:"canonical_digest"`
	VerificationStatus string    `json:"verification_status"`
	SealedRevision     int64     `json:"sealed_revision"`
	Manifest           []byte    `json:"manifest"`
}

type TimelineEntry struct {
	Revision int64     `json:"revision"`
	Type     string    `json:"type"`
	At       time.Time `json:"at"`
	ActorID  string    `json:"actor_id"`
	Summary  string    `json:"summary"`
}
