package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"

	"cleanroom-recovery-ledger/internal/domain"
)

var ErrNotFound = errors.New("案件不存在")

type Event struct {
	CaseID         string `json:"case_id"`
	Revision       int64  `json:"revision"`
	Type           string `json:"type"`
	At             string `json:"at"`
	PreviousDigest string `json:"previous_digest"`
	Digest         string `json:"digest"`
}

type stateFile struct {
	Case        *domain.DeviationCase      `json:"case"`
	Idempotency map[string]json.RawMessage `json:"idempotency"`
}

type FileStore struct {
	root         string
	mu           sync.RWMutex
	caseLocks    map[string]*sync.Mutex
	auditReaders map[string]*os.File
	writable     bool
}

func Open(root string) (*FileStore, error) {
	if root == "" {
		return nil, errors.New("存储目录不能为空")
	}
	for _, d := range []string{root, filepath.Join(root, "cases"), filepath.Join(root, "events"), filepath.Join(root, "transactions"), filepath.Join(root, "archive-audits")} {
		if err := os.MkdirAll(d, 0750); err != nil {
			return nil, err
		}
	}
	s := &FileStore{root: root, caseLocks: map[string]*sync.Mutex{}, auditReaders: map[string]*os.File{}, writable: true}
	if err := s.recoverTransactions(); err != nil {
		s.writable = false
		return nil, fmt.Errorf("存储事务恢复失败: %w", err)
	}
	if err := s.scanAll(); err != nil {
		s.writable = false
		return nil, fmt.Errorf("存储完整性检查失败: %w", err)
	}
	return s, nil
}

func (s *FileStore) caseLock(id string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	m := s.caseLocks[id]
	if m == nil {
		m = &sync.Mutex{}
		s.caseLocks[id] = m
	}
	return m
}

func (s *FileStore) Load(id string) (*domain.DeviationCase, error) {
	b, err := os.ReadFile(s.casePath(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var sf stateFile
	if err = json.Unmarshal(b, &sf); err != nil {
		return nil, err
	}
	return domain.Clone(sf.Case)
}

func (s *FileStore) Idempotent(id, requestID string) (json.RawMessage, bool, error) {
	if requestID == "" {
		return nil, false, nil
	}
	b, err := os.ReadFile(s.casePath(id))
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	var sf stateFile
	if err = json.Unmarshal(b, &sf); err != nil {
		return nil, false, err
	}
	v, ok := sf.Idempotency[requestID]
	return append(json.RawMessage(nil), v...), ok, nil
}

func (s *FileStore) Save(c *domain.DeviationCase, expected int64, requestID string, response json.RawMessage) error {
	if !s.writable {
		return errors.New("存储处于只读故障状态")
	}
	if err := validateID(c.CaseID); err != nil {
		return err
	}
	l := s.caseLock(c.CaseID)
	l.Lock()
	defer l.Unlock()
	old := stateFile{Idempotency: map[string]json.RawMessage{}}
	b, err := os.ReadFile(s.casePath(c.CaseID))
	if err == nil {
		if err = json.Unmarshal(b, &old); err != nil {
			return err
		}
		if old.Case.Revision != expected {
			return domain.Conflict(expected, old.Case.Revision)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	} else if expected != 0 {
		return domain.Conflict(expected, 0)
	}
	if old.Idempotency == nil {
		old.Idempotency = map[string]json.RawMessage{}
	}
	if requestID != "" {
		old.Idempotency[requestID] = append(json.RawMessage(nil), response...)
	}
	old.Case = c
	encoded, err := json.Marshal(old)
	if err != nil {
		return err
	}
	return s.commit(c, encoded)
}

func (s *FileStore) List() ([]*domain.DeviationCase, error) {
	entries, err := os.ReadDir(filepath.Join(s.root, "cases"))
	if err != nil {
		return nil, err
	}
	out := make([]*domain.DeviationCase, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := entry.Name()[:len(entry.Name())-5]
		c, e := s.Load(id)
		if e != nil {
			return nil, e
		}
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (s *FileStore) casePath(id string) string  { return filepath.Join(s.root, "cases", id+".json") }
func (s *FileStore) eventPath(id string) string { return filepath.Join(s.root, "events", id+".frames") }
