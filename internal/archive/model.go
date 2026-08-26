package archive

import "time"

type Manifest struct {
	ManifestVersion   string              `json:"manifest_version"`
	Case              CaseRecord          `json:"case"`
	Investigation     InvestigationRecord `json:"investigation"`
	CorrectiveActions []ActionRecord      `json:"corrective_actions"`
	RetestRounds      []RetestRecord      `json:"retest_rounds"`
	Approval          ApprovalRecord      `json:"approval"`
}
type CaseRecord struct {
	CaseID              string    `json:"case_id"`
	RoomCode            string    `json:"room_code"`
	ExcursionKind       string    `json:"excursion_kind"`
	SamplePoint         string    `json:"sample_point"`
	ObservedValue       float64   `json:"observed_value"`
	LimitValue          float64   `json:"limit_value"`
	OccurredAt          time.Time `json:"occurred_at"`
	AffectedWindowStart time.Time `json:"affected_window_start"`
	AffectedWindowEnd   time.Time `json:"affected_window_end"`
	SealedRevision      int64     `json:"sealed_revision"`
}
type InvestigationRecord struct {
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
type ActionRecord struct {
	ActionID         string     `json:"action_id"`
	Description      string     `json:"description"`
	OwnerID          string     `json:"owner_id"`
	Status           string     `json:"status"`
	EvidenceDigest   string     `json:"evidence_digest"`
	CompletedAt      time.Time  `json:"completed_at"`
	RevokedAt        *time.Time `json:"revoked_at,omitempty"`
	RevocationReason string     `json:"revocation_reason,omitempty"`
	RevokedBy        string     `json:"revoked_by,omitempty"`
	ReplacedActionID string     `json:"replaced_action_id,omitempty"`
}
type RetestRecord struct {
	RoundID        string    `json:"round_id"`
	Sequence       int       `json:"sequence"`
	SampledAt      time.Time `json:"sampled_at"`
	ObservedValue  float64   `json:"observed_value"`
	LimitValue     float64   `json:"limit_value"`
	InstrumentRef  string    `json:"instrument_ref"`
	EvidenceDigest string    `json:"evidence_digest"`
	Outcome        string    `json:"outcome"`
	RecordedBy     string    `json:"recorded_by"`
}
type ApprovalRecord struct {
	ApprovedBy string    `json:"approved_by"`
	ApprovedAt time.Time `json:"approved_at"`
}
type Check struct {
	Name     string `json:"name"`
	OK       bool   `json:"ok"`
	Detail   string `json:"detail"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Location string `json:"location"`
}
type Verification struct {
	Valid            bool     `json:"valid"`
	ExpectedDigest   string   `json:"expected_digest"`
	ActualDigest     string   `json:"actual_digest"`
	FailureLocations []string `json:"failure_locations"`
	Checks           []Check  `json:"checks"`
}

type VerificationRecord struct {
	RequestID  string       `json:"request_id"`
	VerifierID string       `json:"verifier_id"`
	ExecutedAt time.Time    `json:"executed_at"`
	Result     Verification `json:"result"`
}
