package precorrection_sample_test

import (
	"testing"
	"time"

	"cleanroom-recovery-ledger/internal/domain"
)

func TestRetestSampleBeforeSupplementalCorrectionIsRejected(t *testing.T) {
	base := time.Date(2026, 8, 27, 8, 0, 0, 0, time.UTC)
	c, err := domain.Create(domain.CreateInput{CaseID: "CASE-SAMPLE-ORDER", RoomCode: "CR-3", ExcursionKind: domain.KindParticle, SamplePoint: "P-3", ObservedValue: 9, LimitValue: 5, OccurredAt: base, WindowStart: base, WindowEnd: base.Add(time.Hour), ActorID: "monitor", Now: base})
	must(t, err)
	must(t, c.ConfirmInvestigation(domain.InvestigationInput{InvestigationID: "INV", InvestigatorID: "investigator", PersonnelFindings: "正常", EquipmentFindings: "正常", CleaningFindings: "不足", AdjacentSampleFindings: "正常", RootCauseCategory: "cleaning", RootCauseStatement: "接触时间不足", EvidenceDigests: []string{"evidence"}, Now: base.Add(time.Minute)}))
	must(t, c.AddAction(domain.AddActionInput{ActionID: "ACTION-1", Description: "首次纠正", OwnerID: "owner", ActorID: "investigator", Now: base.Add(2 * time.Minute)}))
	must(t, c.CompleteAction(domain.CompleteActionInput{ActionID: "ACTION-1", EvidenceDigest: "action-one", ActorID: "owner", Now: base.Add(3 * time.Minute)}))
	must(t, c.RecordRetest(domain.RetestInput{RoundID: "ROUND-FAIL", SampledAt: base.Add(time.Hour), ObservedValue: 6, LimitValue: 5, InstrumentRef: "INST", EvidenceDigest: "failed", RecordedBy: "monitor", Now: base.Add(2 * time.Hour)}))
	must(t, c.AddAction(domain.AddActionInput{ActionID: "ACTION-2", Description: "补充纠正", OwnerID: "owner", ActorID: "investigator", Now: base.Add(2*time.Hour + 10*time.Minute)}))
	must(t, c.CompleteAction(domain.CompleteActionInput{ActionID: "ACTION-2", EvidenceDigest: "action-two", ActorID: "owner", Now: base.Add(3 * time.Hour)}))

	beforeRevision := c.Revision
	err = c.RecordRetest(domain.RetestInput{RoundID: "ROUND-PREMATURE", SampledAt: base.Add(2*time.Hour + 30*time.Minute), ObservedValue: 1, LimitValue: 5, InstrumentRef: "INST", EvidenceDigest: "premature", RecordedBy: "monitor", Now: base.Add(4 * time.Hour)})
	if err == nil || c.Revision != beforeRevision || len(c.Retests) != 1 {
		t.Fatalf("补充纠正完成前取得的样本被计入连续复测: err=%v revision=%d retests=%d", err, c.Revision, len(c.Retests))
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
