package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"

	"cleanroom-recovery-ledger/internal/archive"
)

type archiveAuditFile struct {
	Records []archive.VerificationRecord `json:"records"`
}

func (s *FileStore) archiveAuditPath(caseID string) string {
	return filepath.Join(s.root, "archive-audits", caseID+".json")
}

func (s *FileStore) ArchiveVerifications(caseID string) ([]archive.VerificationRecord, error) {
	if err := validateID(caseID); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(s.archiveAuditPath(caseID))
	if errors.Is(err, os.ErrNotExist) {
		return []archive.VerificationRecord{}, nil
	}
	if err != nil {
		return nil, err
	}
	var file archiveAuditFile
	if err = json.Unmarshal(b, &file); err != nil {
		return nil, err
	}
	out := append([]archive.VerificationRecord(nil), file.Records...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ExecutedAt.After(out[j].ExecutedAt) })
	return out, nil
}

func (s *FileStore) SaveArchiveVerification(caseID string, record archive.VerificationRecord) (archive.VerificationRecord, bool, error) {
	if !s.writable {
		return archive.VerificationRecord{}, false, errors.New("存储处于只读故障状态")
	}
	if err := validateID(caseID); err != nil {
		return archive.VerificationRecord{}, false, err
	}
	lock := s.caseLock(caseID)
	lock.Lock()
	defer lock.Unlock()
	file := archiveAuditFile{Records: []archive.VerificationRecord{}}
	b, err := os.ReadFile(s.archiveAuditPath(caseID))
	if err == nil {
		if err = json.Unmarshal(b, &file); err != nil {
			return archive.VerificationRecord{}, false, err
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return archive.VerificationRecord{}, false, err
	}
	for _, existing := range file.Records {
		if existing.RequestID == record.RequestID {
			return existing, true, nil
		}
	}
	file.Records = append(file.Records, record)
	encoded, err := json.Marshal(file)
	if err != nil {
		return archive.VerificationRecord{}, false, err
	}
	if err = atomicWrite(s.archiveAuditPath(caseID), encoded, 0640); err != nil {
		return archive.VerificationRecord{}, false, err
	}
	return record, false, nil
}
