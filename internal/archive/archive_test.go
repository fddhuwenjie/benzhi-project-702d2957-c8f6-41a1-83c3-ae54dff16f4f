package archive

import (
	"testing"
	"time"

	"cleanroom-recovery-ledger/internal/domain"
)

func TestGenerateIsDeterministicAndVerificationDetectsMutation(t *testing.T) {
	now := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	c := &domain.DeviationCase{CaseID: "CASE-ARCHIVE", RoomCode: "CR-1", ExcursionKind: domain.KindParticle, SamplePoint: "P-1", ObservedValue: 9, LimitValue: 5, OccurredAt: now, AffectedWindowStart: now.Add(-time.Hour), AffectedWindowEnd: now, Status: domain.StatusReleased, Revision: 8, CreatedAt: now, FrozenAt: &now, Investigation: &domain.Investigation{CaseID: "CASE-ARCHIVE", InvestigatorID: "i", PersonnelFindings: "p", EquipmentFindings: "e", CleaningFindings: "c", AdjacentSampleFindings: "a", RootCauseCategory: "cleaning", RootCauseStatement: "root", EvidenceDigests: []string{"z", "a"}, ConfirmedAt: now}, Actions: []domain.CorrectiveAction{{ActionID: "B", CaseID: "CASE-ARCHIVE", Description: "b", OwnerID: "o", Status: domain.ActionCompleted, EvidenceDigest: "eb", CompletedAt: &now}, {ActionID: "A", CaseID: "CASE-ARCHIVE", Description: "a", OwnerID: "o", Status: domain.ActionCompleted, EvidenceDigest: "ea", CompletedAt: &now}}, Retests: []domain.RetestRound{{RoundID: "R1", CaseID: "CASE-ARCHIVE", Sequence: 1, SampledAt: now.Add(time.Hour), ObservedValue: 1, LimitValue: 5, InstrumentRef: "i", EvidenceDigest: "e1", Outcome: "pass", RecordedBy: "m"}, {RoundID: "R2", CaseID: "CASE-ARCHIVE", Sequence: 2, SampledAt: now.Add(2 * time.Hour), ObservedValue: 1, LimitValue: 5, InstrumentRef: "i", EvidenceDigest: "e2", Outcome: "pass", RecordedBy: "m"}}, Review: &domain.Review{ReviewerID: "qa", Decision: "approve", ReviewedAt: now}, Timeline: []domain.TimelineEntry{{Type: "retest_passed"}, {Type: "retest_passed"}}}
	a, err := Generate(c)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Generate(c)
	if err != nil {
		t.Fatal(err)
	}
	if a.CanonicalDigest != b.CanonicalDigest || string(a.Manifest) != string(b.Manifest) {
		t.Fatal("相同快照未生成确定性档案")
	}
	if !Verify(a).Valid {
		t.Fatal("原始档案应验证通过")
	}
	a.Manifest[len(a.Manifest)-2] ^= 1
	if Verify(a).Valid {
		t.Fatal("变更档案内容后验证应失败")
	}
}
