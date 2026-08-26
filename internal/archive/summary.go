package archive

import (
	"fmt"
	"time"

	"cleanroom-recovery-ledger/internal/domain"
)

type Summary struct {
	CaseID             string    `json:"case_id"`
	RoomCode           string    `json:"room_code"`
	ExcursionLabel     string    `json:"excursion_label"`
	ExcursionReading   string    `json:"excursion_reading"`
	RootCause          string    `json:"root_cause"`
	ActionCount        int       `json:"action_count"`
	PassingRetests     int       `json:"passing_retests"`
	ApprovedBy         string    `json:"approved_by"`
	ApprovedAt         time.Time `json:"approved_at"`
	SealedRevision     int64     `json:"sealed_revision"`
	CanonicalDigest    string    `json:"canonical_digest"`
	IntegrityValid     bool      `json:"integrity_valid"`
	VerificationChecks []Check   `json:"verification_checks"`
}

func Summarize(a *domain.ReleaseArchive) (Summary, error) {
	manifest, err := ReadManifest(a.Manifest)
	if err != nil {
		return Summary{}, err
	}
	verification := Verify(a)
	passing := 0
	for _, retest := range manifest.RetestRounds {
		if retest.Outcome == "pass" {
			passing++
		}
	}
	label := "粒子超限"
	if manifest.Case.ExcursionKind == string(domain.KindMicrobial) {
		label = "微生物超限"
	}
	return Summary{CaseID: manifest.Case.CaseID, RoomCode: manifest.Case.RoomCode, ExcursionLabel: label, ExcursionReading: fmt.Sprintf("观测值 %.4g / 限值 %.4g", manifest.Case.ObservedValue, manifest.Case.LimitValue), RootCause: manifest.Investigation.RootCauseStatement, ActionCount: len(manifest.CorrectiveActions), PassingRetests: passing, ApprovedBy: manifest.Approval.ApprovedBy, ApprovedAt: manifest.Approval.ApprovedAt, SealedRevision: manifest.Case.SealedRevision, CanonicalDigest: a.CanonicalDigest, IntegrityValid: verification.Valid, VerificationChecks: verification.Checks}, nil
}
