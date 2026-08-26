package domain

import (
	"errors"
	"testing"
	"time"
)

func TestClosedIntervalOverlapUsesRoomAndPreservesReleasedCases(t *testing.T) {
	base := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	cases := []*DeviationCase{
		{CaseID: "CASE-ONE", RoomCode: "CR-01", ExcursionKind: KindMicrobial, Status: StatusReleased, AffectedWindowStart: base, AffectedWindowEnd: base.Add(2 * time.Hour)},
		{CaseID: "CASE-ROOM", RoomCode: "CR-02", ExcursionKind: KindParticle, Status: StatusCorrecting, AffectedWindowStart: base, AffectedWindowEnd: base.Add(2 * time.Hour)},
		{CaseID: "CASE-LATE", RoomCode: "CR-01", ExcursionKind: KindParticle, Status: StatusCorrecting, AffectedWindowStart: base.Add(2*time.Hour + time.Minute), AffectedWindowEnd: base.Add(3 * time.Hour)},
	}
	got := FindOverlaps("CR-01", base.Add(time.Hour), base.Add(2*time.Hour), cases, "")
	if len(got) != 1 || got[0].CaseID != "CASE-ONE" || got[0].Status != StatusReleased {
		t.Fatalf("交叠结果不正确: %#v", got)
	}
	if !got[0].OverlapStart.Equal(base.Add(time.Hour)) || !got[0].OverlapEnd.Equal(base.Add(2*time.Hour)) {
		t.Fatalf("交叠区间不正确: %#v", got[0])
	}
}

func TestRevokedActionRequiresCompletedReplacementAndPreservesEvidence(t *testing.T) {
	base := time.Date(2026, 8, 27, 8, 0, 0, 0, time.UTC)
	c := &DeviationCase{
		CaseID: "CASE-ACTION", Status: StatusCorrecting, ExcursionKind: KindParticle, LimitValue: 5,
		Investigation: &Investigation{CaseID: "CASE-ACTION"},
		Actions:       []CorrectiveAction{{ActionID: "ACTION-OLD", CaseID: "CASE-ACTION", Description: "旧措施", OwnerID: "owner", Status: ActionCompleted, EvidenceDigest: "evidence-old", CompletedAt: timePtr(base)}},
	}
	if err := c.RevokeAction(RevokeActionInput{ActionID: "ACTION-OLD", Reason: "复核判定无效", ActorID: "reviewer", Now: base.Add(time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if c.Actions[0].EvidenceDigest != "evidence-old" || c.Actions[0].CompletedAt == nil {
		t.Fatal("撤销不应覆盖原完成证据")
	}
	err := c.RecordRetest(RetestInput{RoundID: "ROUND-BLOCK", SampledAt: base.Add(time.Hour), ObservedValue: 1, LimitValue: 5, InstrumentRef: "I", EvidenceDigest: "E", RecordedBy: "M", Now: base.Add(2 * time.Minute)})
	if err == nil {
		t.Fatal("未替代的撤销措施应阻止复测")
	}
	if err = c.AddAction(AddActionInput{ActionID: "ACTION-NEW", Description: "替代措施", OwnerID: "owner-2", ActorID: "reviewer", ReplacedActionID: "ACTION-OLD", Now: base.Add(3 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if err = c.CompleteAction(CompleteActionInput{ActionID: "ACTION-OLD", EvidenceDigest: "again", ActorID: "owner", Now: base.Add(4 * time.Minute)}); err == nil {
		t.Fatal("已撤销措施不能再次完成")
	}
	if err = c.CompleteAction(CompleteActionInput{ActionID: "ACTION-NEW", EvidenceDigest: "evidence-new", ActorID: "owner-2", Now: base.Add(5 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	if reasons := c.actionBlockingReasons(); len(reasons) != 0 {
		t.Fatalf("完成替代措施后仍有阻断: %v", reasons)
	}
}

func TestRetestRejectsRelaxedThresholdWithoutMutation(t *testing.T) {
	base := time.Date(2026, 8, 27, 8, 0, 0, 0, time.UTC)
	c := &DeviationCase{
		CaseID: "CASE-THRESHOLD", Status: StatusCorrecting, ExcursionKind: KindMicrobial, LimitValue: 5, Revision: 4,
		Investigation: &Investigation{CaseID: "CASE-THRESHOLD"},
		Actions:       []CorrectiveAction{{ActionID: "ACTION-ONE", CaseID: "CASE-THRESHOLD", Status: ActionCompleted, EvidenceDigest: "e", CompletedAt: timePtr(base)}},
	}
	err := c.RecordRetest(RetestInput{RoundID: "ROUND-ONE", SampledAt: base.Add(time.Hour), ObservedValue: 5, LimitValue: 6, InstrumentRef: "I", EvidenceDigest: "E", RecordedBy: "M", Now: base.Add(time.Minute)})
	var domainErr *DomainError
	if !errors.As(err, &domainErr) || domainErr.Field != "limit_value" {
		t.Fatalf("期望限值字段错误，得到 %v", err)
	}
	if c.Revision != 4 || len(c.Retests) != 0 {
		t.Fatal("拒绝阈值后不应增加修订或轮次")
	}
}

func timePtr(v time.Time) *time.Time { return &v }
