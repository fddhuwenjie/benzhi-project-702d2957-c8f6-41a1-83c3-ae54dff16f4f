package store

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func readFrames(path string) ([]Event, int64, error) {
	f, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	events := []Event{}
	var offset int64
	prev := ""
	for {
		var head [4]byte
		n, err := io.ReadFull(f, head[:])
		if err == io.EOF {
			return events, offset, nil
		}
		if err != nil {
			return nil, offset, fmt.Errorf("偏移 %d 帧头截断: %w", offset, err)
		}
		offset += int64(n)
		size := binary.BigEndian.Uint32(head[:])
		if size == 0 || size > maxFrameSize {
			return nil, offset, fmt.Errorf("偏移 %d 帧长度无效", offset-4)
		}
		payload := make([]byte, size)
		n, err = io.ReadFull(f, payload)
		if err != nil {
			return nil, offset, fmt.Errorf("偏移 %d 帧内容截断: %w", offset, err)
		}
		offset += int64(n)
		var e Event
		if err = json.Unmarshal(payload, &e); err != nil {
			return nil, offset, err
		}
		if e.Revision != int64(len(events)+1) || e.PreviousDigest != prev || eventDigest(e) != e.Digest {
			return nil, offset, fmt.Errorf("事件 %d 摘要链无效", e.Revision)
		}
		events = append(events, e)
		prev = e.Digest
	}
}

func (s *FileStore) scanAll() error {
	entries, err := os.ReadDir(filepath.Join(s.root, "cases"))
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := entry.Name()[:len(entry.Name())-5]
		if err = validateID(id); err != nil {
			return err
		}
		c, err := s.Load(id)
		if err != nil {
			return err
		}
		if c.CaseID != id {
			return fmt.Errorf("快照编号与文件名不一致")
		}
		if err = validateAggregate(c); err != nil {
			return fmt.Errorf("案件 %s 聚合不变量无效: %w", id, err)
		}
		events, _, err := readFrames(s.eventPath(id))
		if err != nil {
			return err
		}
		if err = timelineMatches(c, events); err != nil {
			return err
		}
	}
	eventEntries, err := os.ReadDir(filepath.Join(s.root, "events"))
	if err != nil {
		return err
	}
	for _, entry := range eventEntries {
		if filepath.Ext(entry.Name()) != ".frames" {
			continue
		}
		id := entry.Name()[:len(entry.Name())-7]
		if _, err = os.Stat(s.casePath(id)); errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("发现没有快照的事件流 %s", id)
		}
	}
	auditEntries, err := os.ReadDir(filepath.Join(s.root, "archive-audits"))
	if err != nil {
		return err
	}
	for _, entry := range auditEntries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		id := entry.Name()[:len(entry.Name())-5]
		if err = validateID(id); err != nil {
			return err
		}
		if _, statErr := os.Stat(s.casePath(id)); errors.Is(statErr, os.ErrNotExist) {
			return fmt.Errorf("发现没有案件快照的档案校验记录 %s", id)
		}
		b, readErr := os.ReadFile(filepath.Join(s.root, "archive-audits", entry.Name()))
		if readErr != nil {
			return readErr
		}
		var audit archiveAuditFile
		if err = json.Unmarshal(b, &audit); err != nil {
			return err
		}
		seenRequests := map[string]struct{}{}
		for _, record := range audit.Records {
			if record.RequestID == "" || record.VerifierID == "" || record.ExecutedAt.IsZero() || len(record.Result.Checks) == 0 {
				return fmt.Errorf("案件 %s 的档案校验记录不完整", id)
			}
			if _, exists := seenRequests[record.RequestID]; exists {
				return fmt.Errorf("案件 %s 的档案校验 request_id 重复", id)
			}
			seenRequests[record.RequestID] = struct{}{}
		}
	}
	return nil
}
