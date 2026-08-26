package domain

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

func FindOverlaps(roomCode string, start, end time.Time, cases []*DeviationCase, excludeCaseID string) []CaseOverlap {
	roomCode = strings.TrimSpace(roomCode)
	overlaps := make([]CaseOverlap, 0)
	if roomCode == "" || start.IsZero() || end.IsZero() || end.Before(start) {
		return overlaps
	}
	for _, existing := range cases {
		if existing == nil || existing.CaseID == excludeCaseID || existing.RoomCode != roomCode {
			continue
		}
		// Closed intervals overlap unless either interval ends before the other starts.
		if existing.AffectedWindowEnd.Before(start) || end.Before(existing.AffectedWindowStart) {
			continue
		}
		overlapStart := start
		if existing.AffectedWindowStart.After(overlapStart) {
			overlapStart = existing.AffectedWindowStart
		}
		overlapEnd := end
		if existing.AffectedWindowEnd.Before(overlapEnd) {
			overlapEnd = existing.AffectedWindowEnd
		}
		overlaps = append(overlaps, CaseOverlap{CaseID: existing.CaseID, Status: existing.Status, ExcursionKind: existing.ExcursionKind, OverlapStart: overlapStart.UTC(), OverlapEnd: overlapEnd.UTC()})
	}
	sort.Slice(overlaps, func(i, j int) bool {
		if overlaps[i].OverlapStart.Equal(overlaps[j].OverlapStart) {
			return overlaps[i].CaseID < overlaps[j].CaseID
		}
		return overlaps[i].OverlapStart.Before(overlaps[j].OverlapStart)
	})
	return overlaps
}

type RetestSegment struct {
	Number int           `json:"number"`
	Rounds []RetestRound `json:"rounds"`
}

type RetestProgress struct {
	ApplicableLimit                 float64         `json:"applicable_limit"`
	RequiredPassingRounds           int             `json:"required_passing_rounds"`
	ConsecutivePassing              int             `json:"consecutive_passing"`
	RemainingRounds                 int             `json:"remaining_rounds"`
	LatestFailureBoundary           *time.Time      `json:"latest_failure_boundary,omitempty"`
	SupplementalCorrectionCompleted bool            `json:"supplemental_correction_completed"`
	EligibleForReview               bool            `json:"eligible_for_review"`
	BlockingReasons                 []string        `json:"blocking_reasons"`
	Segments                        []RetestSegment `json:"segments"`
}

func RequiredPassingRounds(kind ExcursionKind) int {
	if kind == KindMicrobial {
		return 3
	}
	return 2
}

func (c *DeviationCase) ConsecutivePassing() int {
	n := 0
	for i := len(c.Retests) - 1; i >= 0; i-- {
		if c.Retests[i].Outcome != "pass" {
			break
		}
		n++
	}
	return n
}

func (c *DeviationCase) latestRemediationBoundary() (time.Time, bool) {
	for i := len(c.Timeline) - 1; i >= 0; i-- {
		if c.Timeline[i].Type == "retest_failed" || c.Timeline[i].Type == "review_rejected" {
			return c.Timeline[i].At, true
		}
	}
	return time.Time{}, false
}

func (c *DeviationCase) hasSupplementalCorrection() bool {
	boundary, needed := c.latestRemediationBoundary()
	if !needed {
		return true
	}
	for _, action := range c.Actions {
		if action.Status == ActionCompleted && action.CompletedAt != nil && action.CompletedAt.After(boundary) {
			return true
		}
	}
	return false
}

func (c *DeviationCase) actionBlockingReasons() []string {
	replacements := make(map[string]CorrectiveAction)
	for _, action := range c.Actions {
		if action.ReplacedActionID != "" {
			replacements[action.ReplacedActionID] = action
		}
	}
	reasons := []string{}
	for _, action := range c.Actions {
		switch action.Status {
		case ActionOpen:
			reasons = append(reasons, fmt.Sprintf("纠正措施 %s 尚未完成", action.ActionID))
		case ActionRevoked:
			if !replacementResolved(action.ActionID, replacements, map[string]bool{}) {
				reasons = append(reasons, fmt.Sprintf("已撤销措施 %s 尚无已完成且有证据的替代措施", action.ActionID))
			}
		case ActionCompleted:
			if strings.TrimSpace(action.EvidenceDigest) == "" {
				reasons = append(reasons, fmt.Sprintf("纠正措施 %s 缺少完成证据", action.ActionID))
			}
		}
	}
	return reasons
}

func replacementResolved(actionID string, replacements map[string]CorrectiveAction, visiting map[string]bool) bool {
	if visiting[actionID] {
		return false
	}
	visiting[actionID] = true
	replacement, ok := replacements[actionID]
	if !ok {
		return false
	}
	if replacement.Status == ActionCompleted && strings.TrimSpace(replacement.EvidenceDigest) != "" {
		return true
	}
	if replacement.Status == ActionRevoked {
		return replacementResolved(replacement.ActionID, replacements, visiting)
	}
	return false
}

func (c *DeviationCase) RetestProgress() RetestProgress {
	required := RequiredPassingRounds(c.ExcursionKind)
	passing := c.ConsecutivePassing()
	remaining := required - passing
	if remaining < 0 {
		remaining = 0
	}
	p := RetestProgress{ApplicableLimit: c.LimitValue, RequiredPassingRounds: required, ConsecutivePassing: passing, RemainingRounds: remaining, SupplementalCorrectionCompleted: c.hasSupplementalCorrection(), BlockingReasons: []string{}, Segments: []RetestSegment{}}
	if c.Investigation == nil {
		p.BlockingReasons = append(p.BlockingReasons, "调查尚未确认")
	}
	if len(c.Actions) == 0 {
		p.BlockingReasons = append(p.BlockingReasons, "尚未制定纠正措施")
	}
	segment := RetestSegment{Number: 1, Rounds: []RetestRound{}}
	for _, round := range c.Retests {
		segment.Rounds = append(segment.Rounds, round)
		if round.Outcome == "fail" {
			p.Segments = append(p.Segments, segment)
			segment = RetestSegment{Number: segment.Number + 1, Rounds: []RetestRound{}}
		}
	}
	lastWasFailure := len(c.Retests) > 0 && c.Retests[len(c.Retests)-1].Outcome == "fail"
	if len(segment.Rounds) > 0 || len(p.Segments) == 0 || lastWasFailure {
		p.Segments = append(p.Segments, segment)
	}
	if boundary, ok := c.latestRemediationBoundary(); ok {
		b := boundary.UTC()
		p.LatestFailureBoundary = &b
		if !p.SupplementalCorrectionCompleted {
			p.BlockingReasons = append(p.BlockingReasons, "最近失败边界之后尚未完成补充纠正措施")
		}
	}
	p.BlockingReasons = append(p.BlockingReasons, c.actionBlockingReasons()...)
	if passing < required {
		p.BlockingReasons = append(p.BlockingReasons, fmt.Sprintf("连续合格复测还差 %d 轮", remaining))
	}
	p.EligibleForReview = c.Investigation != nil && len(c.Actions) > 0 && passing >= required && len(c.actionBlockingReasons()) == 0 && p.SupplementalCorrectionCompleted
	return p
}

func (c *DeviationCase) EvidenceComplete() bool {
	if c.Investigation == nil || len(c.Investigation.EvidenceDigests) == 0 {
		return false
	}
	if strings.TrimSpace(c.Investigation.RootCauseStatement) == "" {
		return false
	}
	if len(c.Actions) == 0 {
		return false
	}
	if len(c.actionBlockingReasons()) > 0 {
		return false
	}
	return c.ConsecutivePassing() >= RequiredPassingRounds(c.ExcursionKind)
}

func validNonNegativeFinite(v float64) bool { return v >= 0 && !math.IsNaN(v) && !math.IsInf(v, 0) }

func validateTime(label string, t time.Time) error {
	if t.IsZero() {
		return invalid(label, label+" 不能为空")
	}
	return nil
}

func textRequired(label, value string) error {
	if strings.TrimSpace(value) == "" {
		return invalid(label, label+" 不能为空")
	}
	return nil
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
