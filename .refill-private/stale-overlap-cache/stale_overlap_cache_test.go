package stale_overlap_cache_test

import (
	"testing"
	"time"

	"cleanroom-recovery-ledger/internal/application"
	"cleanroom-recovery-ledger/internal/domain"
	"cleanroom-recovery-ledger/internal/store"
)

func TestRelatedCaseStatusRefreshesAfterRelease(t *testing.T) {
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := application.New(repo)
	base := time.Date(2026, 8, 27, 8, 0, 0, 0, time.UTC)

	first := createCase(t, service, "CASE-CACHE-A", "create-a", base, false)
	_ = first
	second := createCase(t, service, "CASE-CACHE-B", "create-b", base.Add(30*time.Minute), true)

	before, err := service.Get("CASE-CACHE-A")
	if err != nil {
		t.Fatal(err)
	}
	if got := relatedStatus(t, before, "CASE-CACHE-B"); got != domain.StatusInvestigating {
		t.Fatalf("初次关联状态应为 investigating，实际为 %s", got)
	}

	released := releaseCase(t, service, second, base)
	if released.Case.Status != domain.StatusReleased {
		t.Fatalf("关联案件未放行: %s", released.Case.Status)
	}

	after, err := service.Get("CASE-CACHE-A")
	if err != nil {
		t.Fatal(err)
	}
	if got := relatedStatus(t, after, "CASE-CACHE-B"); got != domain.StatusReleased {
		t.Fatalf("关联案件已放行后查询仍返回缓存状态: got %s, want %s", got, domain.StatusReleased)
	}
}

func createCase(t *testing.T, service *application.Service, caseID, requestID string, occurred time.Time, confirmOverlap bool) *application.OperationResult {
	t.Helper()
	result, err := service.Create(application.CreateCase{
		RequestID: requestID, CaseID: caseID, RoomCode: "CR-CACHE", ExcursionKind: domain.KindParticle,
		SamplePoint: "P-1", ObservedValue: 8, LimitValue: 5, OccurredAt: occurred,
		WindowStart: occurred, WindowEnd: occurred.Add(2 * time.Hour), ActorID: "monitor",
		OverlapConfirmed: confirmOverlap,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func releaseCase(t *testing.T, service *application.Service, result *application.OperationResult, base time.Time) *application.OperationResult {
	t.Helper()
	caseID := result.Case.CaseID
	var err error
	result, err = service.ConfirmInvestigation(caseID, application.ConfirmInvestigation{
		Envelope: application.Envelope{RequestID: "investigate-b", ExpectedRevision: result.Case.Revision},
		Input: domain.InvestigationInput{
			InvestigationID: "INV-B", InvestigatorID: "investigator", PersonnelFindings: "正常",
			EquipmentFindings: "正常", CleaningFindings: "清洁不足", AdjacentSampleFindings: "相邻点正常",
			RootCauseCategory: "cleaning", RootCauseStatement: "清洁接触不足", EvidenceDigests: []string{"investigation-evidence"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err = service.AddAction(caseID, application.AddAction{
		Envelope: application.Envelope{RequestID: "action-b", ExpectedRevision: result.Case.Revision},
		Input:    domain.AddActionInput{ActionID: "ACTION-B", Description: "重新清洁", OwnerID: "owner", ActorID: "investigator"},
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err = service.CompleteAction(caseID, application.CompleteAction{
		Envelope: application.Envelope{RequestID: "complete-b", ExpectedRevision: result.Case.Revision},
		Input:    domain.CompleteActionInput{ActionID: "ACTION-B", EvidenceDigest: "action-evidence", ActorID: "owner"},
	})
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 2; i++ {
		result, err = service.RecordRetest(caseID, application.RecordRetest{
			Envelope: application.Envelope{RequestID: "retest-b-" + string(rune('0'+i)), ExpectedRevision: result.Case.Revision},
			Input: domain.RetestInput{
				RoundID: "ROUND-B-" + string(rune('0'+i)), SampledAt: base.Add(time.Duration(i+4) * time.Hour),
				ObservedValue: 2, LimitValue: 5, InstrumentRef: "particle-counter", EvidenceDigest: "retest-evidence", RecordedBy: "monitor",
			},
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	result, err = service.Review(caseID, application.ReviewCase{
		Envelope: application.Envelope{RequestID: "review-b", ExpectedRevision: result.Case.Revision},
		Input:    domain.ReviewInput{ReviewerID: "qa-reviewer", Decision: "approve"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func relatedStatus(t *testing.T, c *domain.DeviationCase, caseID string) domain.Status {
	t.Helper()
	for _, related := range c.RelatedCases {
		if related.CaseID == caseID {
			return related.Status
		}
	}
	t.Fatalf("未找到关联案件 %s", caseID)
	return ""
}
