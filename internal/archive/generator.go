package archive

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"cleanroom-recovery-ledger/internal/domain"
)

func Generate(c *domain.DeviationCase) (*domain.ReleaseArchive, error) {
	if c.Status != domain.StatusReleased || c.Review == nil || c.Review.Decision != "approve" {
		return nil, errors.New("只有已批准案件可生成档案")
	}
	if !c.EvidenceComplete() {
		return nil, errors.New("案件证据不完整")
	}
	if c.FrozenAt == nil {
		return nil, errors.New("案件未冻结")
	}
	m := buildManifest(c)
	content, err := canonicalJSON(m)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(content)
	digest := hex.EncodeToString(sum[:])
	return &domain.ReleaseArchive{ArchiveID: "archive-" + c.CaseID, CaseID: c.CaseID, ApprovedBy: c.Review.ReviewerID, ApprovedAt: c.Review.ReviewedAt, ManifestVersion: "1.0", CanonicalDigest: digest, VerificationStatus: "verified", SealedRevision: c.Revision, Manifest: content}, nil
}

func buildManifest(c *domain.DeviationCase) Manifest {
	evidence := append([]string(nil), c.Investigation.EvidenceDigests...)
	sort.Strings(evidence)
	actions := make([]ActionRecord, 0, len(c.Actions))
	for _, a := range c.Actions {
		completed := time.Time{}
		if a.CompletedAt != nil {
			completed = *a.CompletedAt
		}
		var revoked *time.Time
		if a.RevokedAt != nil {
			value := a.RevokedAt.UTC()
			revoked = &value
		}
		actions = append(actions, ActionRecord{ActionID: a.ActionID, Description: a.Description, OwnerID: a.OwnerID, Status: string(a.Status), EvidenceDigest: a.EvidenceDigest, CompletedAt: completed.UTC(), RevokedAt: revoked, RevocationReason: a.RevocationReason, RevokedBy: a.RevokedBy, ReplacedActionID: a.ReplacedActionID})
	}
	sort.Slice(actions, func(i, j int) bool { return actions[i].ActionID < actions[j].ActionID })
	retests := make([]RetestRecord, 0, len(c.Retests))
	for _, r := range c.Retests {
		retests = append(retests, RetestRecord{RoundID: r.RoundID, Sequence: r.Sequence, SampledAt: r.SampledAt.UTC(), ObservedValue: r.ObservedValue, LimitValue: r.LimitValue, InstrumentRef: r.InstrumentRef, EvidenceDigest: r.EvidenceDigest, Outcome: r.Outcome, RecordedBy: r.RecordedBy})
	}
	sort.Slice(retests, func(i, j int) bool { return retests[i].Sequence < retests[j].Sequence })
	return Manifest{ManifestVersion: "1.0", Case: CaseRecord{CaseID: c.CaseID, RoomCode: c.RoomCode, ExcursionKind: string(c.ExcursionKind), SamplePoint: c.SamplePoint, ObservedValue: c.ObservedValue, LimitValue: c.LimitValue, OccurredAt: c.OccurredAt.UTC(), AffectedWindowStart: c.AffectedWindowStart.UTC(), AffectedWindowEnd: c.AffectedWindowEnd.UTC(), SealedRevision: c.Revision}, Investigation: InvestigationRecord{InvestigatorID: c.Investigation.InvestigatorID, PersonnelFindings: c.Investigation.PersonnelFindings, EquipmentFindings: c.Investigation.EquipmentFindings, CleaningFindings: c.Investigation.CleaningFindings, AdjacentSampleFindings: c.Investigation.AdjacentSampleFindings, RootCauseCategory: c.Investigation.RootCauseCategory, RootCauseStatement: c.Investigation.RootCauseStatement, EvidenceDigests: evidence, ConfirmedAt: c.Investigation.ConfirmedAt.UTC()}, CorrectiveActions: actions, RetestRounds: retests, Approval: ApprovalRecord{ApprovedBy: c.Review.ReviewerID, ApprovedAt: c.Review.ReviewedAt.UTC()}}
}

func canonicalJSON(v any) ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(b.Bytes(), []byte("\n")), nil
}

func ReadManifest(data []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return m, fmt.Errorf("档案 JSON 无效: %w", err)
	}
	return m, nil
}
