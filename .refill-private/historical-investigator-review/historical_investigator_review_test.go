package historical_investigator_review_test

import (
	"errors"
	"testing"
	"time"

	"cleanroom-recovery-ledger/internal/domain"
)

func TestHistoricalInvestigatorCannotApproveAfterReconfirmation(t *testing.T) {
	base := time.Date(2026, 8, 27, 8, 0, 0, 0, time.UTC)
	c, err := domain.Create(domain.CreateInput{CaseID: "CASE-HISTORY-REVIEW", RoomCode: "CR-2", ExcursionKind: domain.KindParticle, SamplePoint: "P-2", ObservedValue: 9, LimitValue: 5, OccurredAt: base, WindowStart: base, WindowEnd: base.Add(time.Hour), ActorID: "monitor", Now: base})
	must(t, err)
	must(t, c.ConfirmInvestigation(investigation("INV-OLD", "investigator-old", base.Add(10*time.Minute))))
	must(t, c.AddAction(domain.AddActionInput{ActionID: "ACTION-1", Description: "首次纠正", OwnerID: "owner", ActorID: "investigator-old", Now: base.Add(20 * time.Minute)}))
	must(t, c.CompleteAction(domain.CompleteActionInput{ActionID: "ACTION-1", EvidenceDigest: "action-one", ActorID: "owner", Now: base.Add(30 * time.Minute)}))
	must(t, c.RecordRetest(domain.RetestInput{RoundID: "ROUND-FAIL", SampledAt: base.Add(time.Hour), ObservedValue: 6, LimitValue: 5, InstrumentRef: "INST", EvidenceDigest: "failed", RecordedBy: "monitor", Now: base.Add(time.Hour + 10*time.Minute)}))

	must(t, c.ConfirmInvestigation(investigation("INV-NEW", "investigator-new", base.Add(time.Hour+20*time.Minute))))
	must(t, c.AddAction(domain.AddActionInput{ActionID: "ACTION-2", Description: "补充纠正", OwnerID: "owner", ActorID: "investigator-new", Now: base.Add(time.Hour + 30*time.Minute)}))
	must(t, c.CompleteAction(domain.CompleteActionInput{ActionID: "ACTION-2", EvidenceDigest: "action-two", ActorID: "owner", Now: base.Add(time.Hour + 40*time.Minute)}))
	for i := 1; i <= 2; i++ {
		must(t, c.RecordRetest(domain.RetestInput{RoundID: "ROUND-PASS-" + string(rune('0'+i)), SampledAt: base.Add(time.Duration(i+1) * time.Hour), ObservedValue: 1, LimitValue: 5, InstrumentRef: "INST", EvidenceDigest: "passing", RecordedBy: "monitor", Now: base.Add(time.Duration(i+1)*time.Hour + 10*time.Minute)}))
	}

	err = c.ReviewDecision(domain.ReviewInput{ReviewerID: "investigator-old", Decision: "approve", Now: base.Add(3*time.Hour + 20*time.Minute)})
	var de *domain.DomainError
	if !errors.As(err, &de) || de.Code != domain.CodeForbidden {
		t.Fatalf("历史调查员在记录被覆盖后仍获准批准: err=%v status=%s", err, c.Status)
	}
}

func investigation(id, investigator string, now time.Time) domain.InvestigationInput {
	return domain.InvestigationInput{InvestigationID: id, InvestigatorID: investigator, PersonnelFindings: "正常", EquipmentFindings: "正常", CleaningFindings: "不足", AdjacentSampleFindings: "正常", RootCauseCategory: "cleaning", RootCauseStatement: "接触时间不足", EvidenceDigests: []string{"evidence"}, Now: now}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
