package application

import (
	"encoding/json"

	"cleanroom-recovery-ledger/internal/archive"
	"cleanroom-recovery-ledger/internal/domain"
)

type Repository interface {
	Load(caseID string) (*domain.DeviationCase, error)
	List() ([]*domain.DeviationCase, error)
	Idempotent(caseID, requestID string) (json.RawMessage, bool, error)
	Save(c *domain.DeviationCase, expected int64, requestID string, response json.RawMessage) error
	SaveArchiveVerification(caseID string, record archive.VerificationRecord) (archive.VerificationRecord, bool, error)
	ArchiveVerifications(caseID string) ([]archive.VerificationRecord, error)
}
