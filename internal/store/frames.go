package store

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"cleanroom-recovery-ledger/internal/domain"
)

const maxFrameSize = 1 << 20

func (s *FileStore) appendLatestEvent(c *domain.DeviationCase) error {
	if len(c.Timeline) == 0 {
		return fmt.Errorf("时间线为空")
	}
	last := c.Timeline[len(c.Timeline)-1]
	prev, _, err := readFrames(s.eventPath(c.CaseID))
	if err != nil {
		return err
	}
	prevDigest := ""
	if len(prev) > 0 {
		if prev[len(prev)-1].Revision >= c.Revision {
			return nil
		}
		if prev[len(prev)-1].Revision != c.Revision-1 {
			return fmt.Errorf("事件修订不连续")
		}
		prevDigest = prev[len(prev)-1].Digest
	} else if c.Revision != 1 {
		return fmt.Errorf("首个事件修订必须为 1")
	}
	e := Event{CaseID: c.CaseID, Revision: c.Revision, Type: last.Type, At: last.At.UTC().Format("2006-01-02T15:04:05.000000000Z"), ActorID: last.ActorID, Summary: last.Summary, PreviousDigest: prevDigest}
	e.Digest = eventDigest(e)
	payload, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if len(payload) > maxFrameSize {
		return fmt.Errorf("事件帧过大")
	}
	f, err := os.OpenFile(s.eventPath(c.CaseID), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		return err
	}
	defer f.Close()
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(payload)))
	if _, err = f.Write(header[:]); err != nil {
		return err
	}
	if _, err = f.Write(payload); err != nil {
		return err
	}
	return f.Sync()
}

func eventDigest(e Event) string {
	h := sha256.New()
	io.WriteString(h, e.CaseID)
	io.WriteString(h, "\x00")
	io.WriteString(h, fmt.Sprint(e.Revision))
	io.WriteString(h, "\x00")
	io.WriteString(h, e.Type)
	io.WriteString(h, "\x00")
	io.WriteString(h, e.At)
	io.WriteString(h, "\x00")
	io.WriteString(h, e.ActorID)
	io.WriteString(h, "\x00")
	io.WriteString(h, e.Summary)
	io.WriteString(h, "\x00")
	io.WriteString(h, e.PreviousDigest)
	return hex.EncodeToString(h.Sum(nil))
}

func timelineMatches(c *domain.DeviationCase, events []Event) error {
	if int64(len(events)) != c.Revision {
		return fmt.Errorf("案件 %s 的事件数与修订不一致", c.CaseID)
	}
	if len(c.Timeline) != len(events) {
		return fmt.Errorf("案件 %s 的时间线与事件数不一致", c.CaseID)
	}
	for i, e := range events {
		entry := c.Timeline[i]
		if e.Revision != int64(i+1) || e.Type != entry.Type || e.At != entry.At.UTC().Format("2006-01-02T15:04:05.000000000Z") || e.ActorID != entry.ActorID || e.Summary != entry.Summary {
			return fmt.Errorf("案件 %s 的事件与时间线不一致", c.CaseID)
		}
	}
	return nil
}
