package rejection_resets_streak_test

import (
	"testing"
	"time"

	"cleanroom-recovery-ledger/internal/domain"
)

func TestReviewRejectionStartsNewPassingStreak(t *testing.T) {
	base := time.Date(2026, 8, 27, 8, 0, 0, 0, time.UTC)
	c, err := domain.Create(domain.CreateInput{CaseID: "CASE-REJECT-STREAK", RoomCode: "CR-1", ExcursionKind: domain.KindParticle, SamplePoint: "P-1", ObservedValue: 8, LimitValue: 5, OccurredAt: base, WindowStart: base, WindowEnd: base.Add(time.Hour), ActorID: "monitor", Now: base})
	must(t, err)
	must(t, c.ConfirmInvestigation(domain.InvestigationInput{InvestigationID: "INV-1", InvestigatorID: "investigator", PersonnelFindings: "正常", EquipmentFindings: "正常", CleaningFindings: "不足", AdjacentSampleFindings: "正常", RootCauseCategory: "cleaning", RootCauseStatement: "接触时间不足", EvidenceDigests: []string{"evidence-investigation"}, Now: base.Add(10 * time.Minute)}))
	must(t, c.AddAction(domain.AddActionInput{ActionID: "ACTION-1", Description: "重新清洁", OwnerID: "owner", ActorID: "investigator", Now: base.Add(20 * time.Minute)}))
	must(t, c.CompleteAction(domain.CompleteActionInput{ActionID: "ACTION-1", EvidenceDigest: "evidence-action", ActorID: "owner", Now: base.Add(30 * time.Minute)}))
	for i := 1; i <= 2; i++ {
		must(t, c.RecordRetest(domain.RetestInput{RoundID: "ROUND-" + string(rune('0'+i)), SampledAt: base.Add(time.Duration(i) * time.Hour), ObservedValue: 1, LimitValue: 5, InstrumentRef: "INST", EvidenceDigest: "evidence-retest", RecordedBy: "monitor", Now: base.Add(time.Duration(i)*time.Hour + 10*time.Minute)}))
	}
	must(t, c.ReviewDecision(domain.ReviewInput{ReviewerID: "qa-one", Decision: "reject", Reason: "需要补充纠正", Now: base.Add(3 * time.Hour)}))
	must(t, c.AddAction(domain.AddActionInput{ActionID: "ACTION-2", Description: "补充纠正", OwnerID: "owner", ActorID: "investigator", Now: base.Add(3*time.Hour + 10*time.Minute)}))
	must(t, c.CompleteAction(domain.CompleteActionInput{ActionID: "ACTION-2", EvidenceDigest: "evidence-supplemental", ActorID: "owner", Now: base.Add(3*time.Hour + 20*time.Minute)}))
	must(t, c.RecordRetest(domain.RetestInput{RoundID: "ROUND-AFTER-REJECT", SampledAt: base.Add(4 * time.Hour), ObservedValue: 1, LimitValue: 5, InstrumentRef: "INST", EvidenceDigest: "evidence-new", RecordedBy: "monitor", Now: base.Add(4*time.Hour + 10*time.Minute)}))

	if c.Status == domain.StatusReview || c.ConsecutivePassing() != 1 {
		t.Fatalf("驳回后的首轮合格错误复用了驳回前轮次: status=%s consecutive=%d", c.Status, c.ConsecutivePassing())
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
