package store

import (
	"cleanroom-recovery-ledger/internal/domain"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

type pendingTransaction struct {
	CaseID   string          `json:"case_id"`
	Revision int64           `json:"revision"`
	State    json.RawMessage `json:"state"`
}

func (s *FileStore) transactionPath(caseID string) string {
	return filepath.Join(s.root, "transactions", caseID+".json")
}

func (s *FileStore) commit(c *domain.DeviationCase, encoded []byte) error {
	pending := pendingTransaction{CaseID: c.CaseID, Revision: c.Revision, State: append(json.RawMessage(nil), encoded...)}
	payload, err := json.Marshal(pending)
	if err != nil {
		return err
	}
	transactionPath := s.transactionPath(c.CaseID)
	if err = atomicWrite(transactionPath, payload, 0640); err != nil {
		return fmt.Errorf("写入事务标记失败: %w", err)
	}
	if err = s.appendLatestEvent(c); err != nil {
		return fmt.Errorf("追加事件失败: %w", err)
	}
	if err = atomicWrite(s.casePath(c.CaseID), encoded, 0640); err != nil {
		return fmt.Errorf("替换快照失败: %w", err)
	}
	if err = removeAndSync(transactionPath); err != nil {
		return fmt.Errorf("清理事务标记失败: %w", err)
	}
	return nil
}

func (s *FileStore) recoverTransactions() error {
	dir := filepath.Join(s.root, "transactions")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		payload, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		var pending pendingTransaction
		if err = json.Unmarshal(payload, &pending); err != nil {
			return fmt.Errorf("事务标记 %s 无效: %w", entry.Name(), err)
		}
		if err = validateID(pending.CaseID); err != nil {
			return err
		}
		if entry.Name() != pending.CaseID+".json" {
			return errors.New("事务标记文件名与案件编号不一致")
		}
		var sf stateFile
		if err = json.Unmarshal(pending.State, &sf); err != nil || sf.Case == nil {
			return fmt.Errorf("事务中的案件快照无效")
		}
		if sf.Case.CaseID != pending.CaseID || sf.Case.Revision != pending.Revision {
			return errors.New("事务修订与案件快照不一致")
		}
		events, _, frameErr := readFrames(s.eventPath(pending.CaseID))
		if frameErr != nil {
			return frameErr
		}
		lastRevision := int64(0)
		if len(events) > 0 {
			lastRevision = events[len(events)-1].Revision
		}
		if lastRevision == pending.Revision-1 {
			if err = s.appendLatestEvent(sf.Case); err != nil {
				return err
			}
		} else if lastRevision != pending.Revision {
			return fmt.Errorf("事务 %s 无法恢复：事件修订为 %d，目标为 %d", pending.CaseID, lastRevision, pending.Revision)
		}
		if err = atomicWrite(s.casePath(pending.CaseID), pending.State, 0640); err != nil {
			return err
		}
		if err = removeAndSync(path); err != nil {
			return err
		}
	}
	return nil
}

func removeAndSync(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
