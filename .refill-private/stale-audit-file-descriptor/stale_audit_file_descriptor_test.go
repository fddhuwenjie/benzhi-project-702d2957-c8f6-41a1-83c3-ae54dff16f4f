package staleauditfiledescriptor_test

import (
	"testing"
	"time"

	"cleanroom-recovery-ledger/internal/archive"
	"cleanroom-recovery-ledger/internal/store"
)

func TestAuditHistoryRefreshesAfterAtomicReplacement(t *testing.T) {
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatalf("open store: %v", err)
	}

	caseID := "CASE-AUDIT-FD"
	base := time.Date(2026, 8, 27, 9, 0, 0, 0, time.UTC)
	first := archive.VerificationRecord{
		RequestID:  "verify-first",
		VerifierID: "qa-first",
		ExecutedAt: base,
	}
	if _, _, err = repo.SaveArchiveVerification(caseID, first); err != nil {
		t.Fatalf("save first verification: %v", err)
	}
	initial, err := repo.ArchiveVerifications(caseID)
	if err != nil {
		t.Fatalf("read initial history: %v", err)
	}
	if len(initial) != 1 {
		t.Fatalf("initial history length = %d, want 1", len(initial))
	}

	second := archive.VerificationRecord{
		RequestID:  "verify-second",
		VerifierID: "qa-second",
		ExecutedAt: base.Add(time.Minute),
	}
	if _, replayed, err := repo.SaveArchiveVerification(caseID, second); err != nil {
		t.Fatalf("save second verification: %v", err)
	} else if replayed {
		t.Fatal("second unique verification was treated as a replay")
	}

	refreshed, err := repo.ArchiveVerifications(caseID)
	if err != nil {
		t.Fatalf("read refreshed history: %v", err)
	}
	if len(refreshed) != 2 {
		t.Fatalf("TestAuditHistoryRefreshesAfterAtomicReplacement: history length = %d, want 2 after atomic replacement", len(refreshed))
	}
	if refreshed[0].RequestID != second.RequestID {
		t.Fatalf("latest request_id = %q, want %q", refreshed[0].RequestID, second.RequestID)
	}
}
