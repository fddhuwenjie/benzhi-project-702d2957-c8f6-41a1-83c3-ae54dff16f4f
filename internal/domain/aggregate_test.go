package domain

import (
	"testing"
	"time"
)

func TestRecoveryGatesFailureAndIndependentReview(t *testing.T) {
	base := time.Date(2026, 8, 27, 8, 0, 0, 0, time.UTC)
	c, err := Create(CreateInput{CaseID: "CASE-001", RoomCode: "CR-1", ExcursionKind: KindParticle, SamplePoint: "P-1", ObservedValue: 20, LimitValue: 10, OccurredAt: base, WindowStart: base.Add(-time.Hour), WindowEnd: base.Add(time.Hour), ActorID: "monitor", Now: base})
	if err != nil {
		t.Fatal(err)
	}
	err = c.ConfirmInvestigation(InvestigationInput{InvestigationID: "INV-1", InvestigatorID: "investigator", PersonnelFindings: "正常", EquipmentFindings: "正常", CleaningFindings: "清洁不足", AdjacentSampleFindings: "相邻正常", RootCauseCategory: "cleaning", RootCauseStatement: "接触时间不足", EvidenceDigests: []string{"digest-investigation"}, Now: base.Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if err = c.AddAction(AddActionInput{ActionID: "A-1", Description: "重新清洁", OwnerID: "operator", ActorID: "investigator", Now: base.Add(2 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err = c.CompleteAction(CompleteActionInput{ActionID: "A-1", EvidenceDigest: "digest-action", ActorID: "operator", Now: base.Add(3 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err = c.RecordRetest(RetestInput{RoundID: "R-FAIL", SampledAt: base.Add(2 * time.Hour), ObservedValue: 11, LimitValue: 10, InstrumentRef: "I-1", EvidenceDigest: "digest-fail", RecordedBy: "monitor", Now: base.Add(4 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	err = c.RecordRetest(RetestInput{RoundID: "R-BLOCKED", SampledAt: base.Add(3 * time.Hour), ObservedValue: 1, LimitValue: 10, InstrumentRef: "I-1", EvidenceDigest: "digest", RecordedBy: "monitor", Now: base.Add(5 * time.Minute)})
	if err == nil {
		t.Fatal("失败复测后未补充纠正不应允许继续复测")
	}
	if err = c.AddAction(AddActionInput{ActionID: "A-2", Description: "延长消毒接触时间", OwnerID: "operator", ActorID: "investigator", Now: base.Add(6 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err = c.CompleteAction(CompleteActionInput{ActionID: "A-2", EvidenceDigest: "digest-action-2", ActorID: "operator", Now: base.Add(7 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	for index := 1; index <= 2; index++ {
		err = c.RecordRetest(RetestInput{RoundID: "R-PASS-" + string(rune('0'+index)), SampledAt: base.Add(time.Duration(3+index) * time.Hour), ObservedValue: 2, LimitValue: 10, InstrumentRef: "I-1", EvidenceDigest: "digest-pass", RecordedBy: "monitor", Now: base.Add(time.Duration(7+index) * time.Minute)})
		if err != nil {
			t.Fatal(err)
		}
	}
	if c.Status != StatusReview {
		t.Fatalf("状态 = %s，期望 %s", c.Status, StatusReview)
	}
	if err = c.ReviewDecision(ReviewInput{ReviewerID: "operator", Decision: "approve", Now: base.Add(20 * time.Minute)}); err == nil {
		t.Fatal("纠正参与者不应能批准")
	}
	if err = c.ReviewDecision(ReviewInput{ReviewerID: "qa-independent", Decision: "approve", Now: base.Add(21 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if c.Status != StatusReleased || c.FrozenAt == nil {
		t.Fatal("批准后应冻结并放行")
	}
}

func TestRejectRequiresReasonAndResetsPassingBoundary(t *testing.T) {
	c := &DeviationCase{Status: StatusReview, ExcursionKind: KindParticle, Investigation: &Investigation{InvestigatorID: "i", RootCauseStatement: "root", EvidenceDigests: []string{"e"}}, Actions: []CorrectiveAction{{Status: ActionCompleted, EvidenceDigest: "e"}}, Timeline: []TimelineEntry{{Type: "retest_passed"}, {Type: "retest_passed"}}}
	if err := c.ReviewDecision(ReviewInput{ReviewerID: "qa", Decision: "reject"}); err == nil {
		t.Fatal("驳回缺少理由应失败")
	}
}
