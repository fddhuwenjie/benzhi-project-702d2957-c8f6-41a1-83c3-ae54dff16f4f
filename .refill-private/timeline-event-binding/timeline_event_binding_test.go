package timeline_event_binding_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cleanroom-recovery-ledger/internal/application"
	"cleanroom-recovery-ledger/internal/domain"
	"cleanroom-recovery-ledger/internal/store"
)

type persistedState struct {
	Case        *domain.DeviationCase      `json:"case"`
	Idempotency map[string]json.RawMessage `json:"idempotency"`
}

func TestTimelineActorTamperingMustBreakEventChainValidation(t *testing.T) {
	root := t.TempDir()
	repo, err := store.Open(root)
	must(t, err)
	service := application.New(repo)
	base := time.Date(2026, 8, 27, 8, 0, 0, 0, time.UTC)
	_, err = service.Create(application.CreateCase{RequestID: "create", CaseID: "CASE-TIMELINE-BINDING", RoomCode: "CR-5", ExcursionKind: domain.KindParticle, SamplePoint: "P-5", ObservedValue: 9, LimitValue: 5, OccurredAt: base, WindowStart: base, WindowEnd: base.Add(time.Hour), ActorID: "monitor-original"})
	must(t, err)

	path := filepath.Join(root, "cases", "CASE-TIMELINE-BINDING.json")
	payload, err := os.ReadFile(path)
	must(t, err)
	var state persistedState
	must(t, json.Unmarshal(payload, &state))
	state.Case.Timeline[0].ActorID = "attacker-substituted"
	payload, err = json.Marshal(state)
	must(t, err)
	must(t, os.WriteFile(path, payload, 0640))

	if _, err = store.Open(root); err == nil {
		t.Fatal("时间线参与者被篡改后事件链校验仍然成功")
	}
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}
