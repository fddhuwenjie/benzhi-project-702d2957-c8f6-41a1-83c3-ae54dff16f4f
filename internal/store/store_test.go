package store

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"cleanroom-recovery-ledger/internal/domain"
)

func TestFileStorePersistsIdempotencyAndDetectsTampering(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	c, err := domain.Create(domain.CreateInput{CaseID: "CASE-STORE", RoomCode: "CR-1", ExcursionKind: domain.KindParticle, SamplePoint: "P-1", ObservedValue: 8, LimitValue: 5, OccurredAt: now, WindowStart: now.Add(-time.Hour), WindowEnd: now, ActorID: "monitor", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	response := json.RawMessage(`{"case":{"case_id":"CASE-STORE"},"replayed":false}`)
	if err = s.Save(c, 0, "request-1", response); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := reopened.Idempotent(c.CaseID, "request-1")
	if err != nil || !ok || string(got) != string(response) {
		t.Fatalf("幂等响应未恢复: %s, %v, %v", got, ok, err)
	}
	path := reopened.eventPath(c.CaseID)
	frames, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	frames[len(frames)-1] ^= 0x01
	if err = os.WriteFile(path, frames, 0640); err != nil {
		t.Fatal(err)
	}
	if _, err = Open(dir); err == nil {
		t.Fatal("篡改事件帧后启动应失败")
	}
}

func TestStaleRevisionIsRejected(t *testing.T) {
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	c, err := domain.Create(domain.CreateInput{CaseID: "CASE-REV", RoomCode: "CR-1", ExcursionKind: domain.KindParticle, SamplePoint: "P-1", ObservedValue: 8, LimitValue: 5, OccurredAt: now, WindowStart: now.Add(-time.Hour), WindowEnd: now, ActorID: "monitor", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	if err = s.Save(c, 0, "r1", json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}
	if err = s.Save(c, 0, "r2", json.RawMessage(`{}`)); err == nil {
		t.Fatal("陈旧修订写入应失败")
	}
}
