package eventappendatomicity_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"cleanroom-recovery-ledger/internal/application"
	"cleanroom-recovery-ledger/internal/domain"
	"cleanroom-recovery-ledger/internal/store"
)

func TestFailedEventAppendDoesNotPublishSnapshot(t *testing.T) {
	root := t.TempDir()
	repo, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	service := application.New(repo)
	base := time.Date(2026, 8, 27, 8, 0, 0, 0, time.UTC)
	created, err := service.Create(application.CreateCase{
		RequestID:     "create-event-atomicity",
		CaseID:        "CASE-EVENT-ATOMICITY",
		RoomCode:      "CR-ATOMIC",
		ExcursionKind: domain.KindParticle,
		SamplePoint:   "P-ATOMIC",
		ObservedValue: 8,
		LimitValue:    5,
		OccurredAt:    base,
		WindowStart:   base.Add(-time.Hour),
		WindowEnd:     base,
		ActorID:       "monitor",
	})
	if err != nil {
		t.Fatal(err)
	}

	eventPath := filepath.Join(root, "events", created.Case.CaseID+".frames")
	if err = os.Rename(eventPath, eventPath+".saved"); err != nil {
		t.Fatal(err)
	}
	if err = os.Mkdir(eventPath, 0750); err != nil {
		t.Fatal(err)
	}

	_, err = service.ConfirmInvestigation(created.Case.CaseID, application.ConfirmInvestigation{
		Envelope: application.Envelope{RequestID: "investigate-event-atomicity", ExpectedRevision: created.Case.Revision},
		Input: domain.InvestigationInput{
			InvestigationID:        "INV-ATOMIC",
			InvestigatorID:         "investigator",
			PersonnelFindings:      "人员活动正常",
			EquipmentFindings:      "设备运行正常",
			CleaningFindings:       "清洁接触不足",
			AdjacentSampleFindings: "相邻点正常",
			RootCauseCategory:      "cleaning",
			RootCauseStatement:     "清洁接触时间不足",
			EvidenceDigests:        []string{"evidence-investigation"},
		},
	})
	if err == nil {
		t.Fatal("事件流失效后命令应返回持久化错误")
	}

	after, loadErr := repo.Load(created.Case.CaseID)
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if after.Revision != created.Case.Revision || after.Status != domain.StatusInvestigating || after.Investigation != nil {
		t.Fatalf("TestFailedEventAppendDoesNotPublishSnapshot: 返回失败的命令仍发布了快照，revision=%d status=%s", after.Revision, after.Status)
	}
	if _, ok, replayErr := repo.Idempotent(created.Case.CaseID, "investigate-event-atomicity"); replayErr != nil || ok {
		t.Fatalf("TestFailedEventAppendDoesNotPublishSnapshot: 失败命令的幂等响应被发布，ok=%v err=%v", ok, replayErr)
	}
}
