package application

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"cleanroom-recovery-ledger/internal/domain"
	"cleanroom-recovery-ledger/internal/store"
)

func TestCreateOverlapConfirmationAndIdempotentRelatedResult(t *testing.T) {
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := New(repo)
	base := time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC)
	first := CreateCase{RequestID: "request-first", CaseID: "CASE-FIRST", RoomCode: "CR-01", ExcursionKind: domain.KindMicrobial, SamplePoint: "P-1", ObservedValue: 9, LimitValue: 5, OccurredAt: base, WindowStart: base, WindowEnd: base.Add(2 * time.Hour), ActorID: "monitor"}
	if _, err = service.Create(first); err != nil {
		t.Fatal(err)
	}
	second := CreateCase{RequestID: "request-second", CaseID: "CASE-SECOND", RoomCode: "CR-01", ExcursionKind: domain.KindParticle, SamplePoint: "P-2", ObservedValue: 8, LimitValue: 5, OccurredAt: base.Add(time.Hour), WindowStart: base.Add(time.Hour), WindowEnd: base.Add(3 * time.Hour), ActorID: "monitor"}
	if _, err = service.Create(second); err == nil {
		t.Fatal("未确认交叠范围不应创建")
	}
	var domainErr *domain.DomainError
	if !errors.As(err, &domainErr) || domainErr.Field != "overlap_confirmed" {
		t.Fatalf("未返回可处理的确认错误: %v", err)
	}
	if _, err = repo.Load(second.CaseID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("拒绝创建后产生了快照: %v", err)
	}
	second.OverlapConfirmed = true
	result, err := service.Create(second)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Case.RelatedCases) != 1 || result.Case.RelatedCases[0].CaseID != first.CaseID {
		t.Fatalf("关联结果不正确: %#v", result.Case.RelatedCases)
	}
	second.OverlapConfirmed = false
	replayed, err := service.Create(second)
	if err != nil {
		t.Fatal(err)
	}
	if len(replayed.Case.RelatedCases) != 1 || replayed.Case.RelatedCases[0] != result.Case.RelatedCases[0] {
		t.Fatalf("相同 request_id 未返回首次关联结果: %#v", replayed.Case.RelatedCases)
	}
}

func TestArchiveVerificationAuditIsIdempotentAndDoesNotReviseCase(t *testing.T) {
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := New(repo)
	base := time.Now().UTC().Add(-4 * time.Hour)
	created, err := service.Create(CreateCase{RequestID: "create", CaseID: "CASE-VERIFY", RoomCode: "CR-03", ExcursionKind: domain.KindParticle, SamplePoint: "P-3", ObservedValue: 8, LimitValue: 5, OccurredAt: base, WindowStart: base, WindowEnd: base.Add(time.Hour), ActorID: "monitor"})
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.ConfirmInvestigation("CASE-VERIFY", ConfirmInvestigation{Envelope: Envelope{RequestID: "investigate", ExpectedRevision: created.Case.Revision}, Input: domain.InvestigationInput{InvestigationID: "INV", InvestigatorID: "investigator", PersonnelFindings: "正常", EquipmentFindings: "正常", CleaningFindings: "清洁不足", AdjacentSampleFindings: "正常", RootCauseCategory: "cleaning", RootCauseStatement: "接触不足", EvidenceDigests: []string{"e"}}})
	if err != nil {
		t.Fatal(err)
	}
	result, err = service.AddAction("CASE-VERIFY", AddAction{Envelope: Envelope{RequestID: "action", ExpectedRevision: result.Case.Revision}, Input: domain.AddActionInput{ActionID: "ACTION", Description: "重新清洁", OwnerID: "owner", ActorID: "investigator"}})
	if err != nil {
		t.Fatal(err)
	}
	result, err = service.CompleteAction("CASE-VERIFY", CompleteAction{Envelope: Envelope{RequestID: "complete", ExpectedRevision: result.Case.Revision}, Input: domain.CompleteActionInput{ActionID: "ACTION", EvidenceDigest: "action-evidence", ActorID: "owner"}})
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 2; i++ {
		result, err = service.RecordRetest("CASE-VERIFY", RecordRetest{Envelope: Envelope{RequestID: "retest-" + string(rune('0'+i)), ExpectedRevision: result.Case.Revision}, Input: domain.RetestInput{RoundID: "ROUND-" + string(rune('0'+i)), SampledAt: base.Add(time.Duration(i+1) * time.Hour), ObservedValue: 2, LimitValue: 5, InstrumentRef: "I", EvidenceDigest: "R", RecordedBy: "monitor"}})
		if err != nil {
			t.Fatal(err)
		}
	}
	result, err = service.Review("CASE-VERIFY", ReviewCase{Envelope: Envelope{RequestID: "review", ExpectedRevision: result.Case.Revision}, Input: domain.ReviewInput{ReviewerID: "qa", Decision: "approve"}})
	if err != nil {
		t.Fatal(err)
	}
	revision, sealed, digest := result.Case.Revision, result.Case.Archive.SealedRevision, result.Case.Archive.CanonicalDigest
	first, err := service.VerifyArchive("CASE-VERIFY", VerifyArchive{RequestID: "verify-one", VerifierID: "auditor"})
	if err != nil || !first.Result.Valid {
		t.Fatalf("首次校验失败: %#v %v", first, err)
	}
	replayed, err := service.VerifyArchive("CASE-VERIFY", VerifyArchive{RequestID: "verify-one", VerifierID: "other"})
	if err != nil || !reflect.DeepEqual(replayed, first) {
		t.Fatalf("重复校验未返回首次记录: %#v %v", replayed, err)
	}
	if _, err = service.VerifyArchive("CASE-VERIFY", VerifyArchive{RequestID: "verify-two", VerifierID: "auditor"}); err != nil {
		t.Fatal(err)
	}
	view, err := service.ArchiveVerificationHistory("CASE-VERIFY")
	if err != nil || len(view.History) != 2 {
		t.Fatalf("校验历史不正确: %#v %v", view, err)
	}
	after, err := repo.Load("CASE-VERIFY")
	if err != nil {
		t.Fatal(err)
	}
	if after.Revision != revision || after.Archive.SealedRevision != sealed || after.Archive.CanonicalDigest != digest {
		t.Fatal("档案校验改变了案件封存字段")
	}
}
