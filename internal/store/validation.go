package store

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"cleanroom-recovery-ledger/internal/archive"
	"cleanroom-recovery-ledger/internal/domain"
)

func validateAggregate(c *domain.DeviationCase) error {
	if c == nil {
		return errors.New("案件快照为空")
	}
	if err := validateID(c.CaseID); err != nil {
		return err
	}
	if strings.TrimSpace(c.RoomCode) == "" || strings.TrimSpace(c.SamplePoint) == "" {
		return errors.New("房间或采样点为空")
	}
	if c.ExcursionKind != domain.KindMicrobial && c.ExcursionKind != domain.KindParticle {
		return errors.New("超限类型无效")
	}
	if c.ObservedValue <= c.LimitValue || c.LimitValue < 0 || math.IsNaN(c.ObservedValue) || math.IsInf(c.ObservedValue, 0) || math.IsNaN(c.LimitValue) || math.IsInf(c.LimitValue, 0) {
		return errors.New("初始观测值没有形成有效超限")
	}
	if c.OccurredAt.IsZero() || c.CreatedAt.IsZero() {
		return errors.New("关键时间为空")
	}
	if c.AffectedWindowStart.IsZero() || c.AffectedWindowEnd.Before(c.AffectedWindowStart) {
		return errors.New("影响时间窗无效")
	}
	if c.Revision < 1 || int64(len(c.Timeline)) != c.Revision {
		return errors.New("修订号与时间线长度不一致")
	}
	if err := validateTimeline(c); err != nil {
		return err
	}
	for _, related := range c.RelatedCases {
		if err := validateID(related.CaseID); err != nil || related.CaseID == c.CaseID || related.OverlapStart.IsZero() || related.OverlapEnd.Before(related.OverlapStart) {
			return errors.New("关联案件范围无效")
		}
	}
	if err := validateInvestigation(c); err != nil {
		return err
	}
	if err := validateActions(c); err != nil {
		return err
	}
	if err := validateRetests(c); err != nil {
		return err
	}
	return validateState(c)
}

func validateTimeline(c *domain.DeviationCase) error {
	previous := time.Time{}
	for i, entry := range c.Timeline {
		if entry.Revision != int64(i+1) {
			return fmt.Errorf("时间线第 %d 项修订不连续", i+1)
		}
		if entry.At.IsZero() || strings.TrimSpace(entry.Type) == "" || strings.TrimSpace(entry.Summary) == "" {
			return fmt.Errorf("时间线第 %d 项内容不完整", i+1)
		}
		if !previous.IsZero() && entry.At.Before(previous) {
			return fmt.Errorf("时间线第 %d 项时间倒退", i+1)
		}
		previous = entry.At
	}
	if c.Timeline[0].Type != "case_created" {
		return errors.New("时间线首项不是案件创建事件")
	}
	return nil
}

func validateInvestigation(c *domain.DeviationCase) error {
	if c.Investigation == nil {
		if c.Status != domain.StatusInvestigating {
			return errors.New("非调查状态缺少调查记录")
		}
		return nil
	}
	i := c.Investigation
	if i.CaseID != c.CaseID || strings.TrimSpace(i.InvestigatorID) == "" {
		return errors.New("调查记录案件或调查员无效")
	}
	required := []string{i.PersonnelFindings, i.EquipmentFindings, i.CleaningFindings, i.AdjacentSampleFindings, i.RootCauseCategory, i.RootCauseStatement}
	for _, value := range required {
		if strings.TrimSpace(value) == "" {
			return errors.New("调查记录存在空的必填项")
		}
	}
	if len(i.EvidenceDigests) == 0 || i.ConfirmedAt.IsZero() {
		return errors.New("调查证据或确认时间缺失")
	}
	return nil
}

func validateActions(c *domain.DeviationCase) error {
	seen := make(map[string]struct{}, len(c.Actions))
	known := make(map[string]domain.CorrectiveAction, len(c.Actions))
	replaced := make(map[string]struct{})
	for _, action := range c.Actions {
		if action.CaseID != c.CaseID || strings.TrimSpace(action.ActionID) == "" {
			return errors.New("纠正措施案件或编号无效")
		}
		if _, exists := seen[action.ActionID]; exists {
			return fmt.Errorf("纠正措施编号 %s 重复", action.ActionID)
		}
		seen[action.ActionID] = struct{}{}
		switch action.Status {
		case domain.ActionOpen:
			if action.CompletedAt != nil || action.RevokedAt != nil {
				return errors.New("开放措施不应有终结时间")
			}
		case domain.ActionCompleted:
			if action.CompletedAt == nil || strings.TrimSpace(action.EvidenceDigest) == "" {
				return errors.New("完成措施缺少时间或证据")
			}
		case domain.ActionRevoked:
			if action.RevokedAt == nil || strings.TrimSpace(action.RevocationReason) == "" || strings.TrimSpace(action.RevokedBy) == "" {
				return errors.New("撤销措施缺少撤销时间")
			}
		default:
			return errors.New("纠正措施状态无效")
		}
		if action.ReplacedActionID != "" {
			if action.ReplacedActionID == action.ActionID {
				return errors.New("替代措施引用自身")
			}
			original, ok := known[action.ReplacedActionID]
			if !ok || original.Status != domain.ActionRevoked {
				return errors.New("替代措施必须引用本案较早的已撤销措施")
			}
			if _, ok := replaced[action.ReplacedActionID]; ok {
				return errors.New("同一撤销措施被重复替代")
			}
			replaced[action.ReplacedActionID] = struct{}{}
		}
		known[action.ActionID] = action
	}
	return nil
}

func validateRetests(c *domain.DeviationCase) error {
	seen := make(map[string]struct{}, len(c.Retests))
	previous := time.Time{}
	for index, retest := range c.Retests {
		if retest.CaseID != c.CaseID || retest.Sequence != index+1 {
			return errors.New("复测案件或序号无效")
		}
		if _, exists := seen[retest.RoundID]; exists {
			return fmt.Errorf("复测轮次编号 %s 重复", retest.RoundID)
		}
		seen[retest.RoundID] = struct{}{}
		if retest.SampledAt.IsZero() || (!previous.IsZero() && !retest.SampledAt.After(previous)) {
			return errors.New("复测采样时间没有严格递增")
		}
		previous = retest.SampledAt
		expected := "pass"
		if retest.ObservedValue > retest.LimitValue {
			expected = "fail"
		}
		if retest.LimitValue != c.LimitValue || retest.ObservedValue < 0 || math.IsNaN(retest.ObservedValue) || math.IsInf(retest.ObservedValue, 0) || math.IsNaN(retest.LimitValue) || math.IsInf(retest.LimitValue, 0) || retest.Outcome != expected {
			return errors.New("复测阈值结论无效")
		}
		if strings.TrimSpace(retest.InstrumentRef) == "" || strings.TrimSpace(retest.EvidenceDigest) == "" {
			return errors.New("复测仪器或证据缺失")
		}
	}
	return nil
}

func validateState(c *domain.DeviationCase) error {
	switch c.Status {
	case domain.StatusInvestigating:
		if c.Investigation != nil {
			return errors.New("调查状态不应已有确认调查")
		}
	case domain.StatusCorrecting:
		if c.Investigation == nil {
			return errors.New("纠正状态缺少调查")
		}
	case domain.StatusRetesting:
		if c.Investigation == nil || len(c.Actions) == 0 || len(c.Retests) == 0 {
			return errors.New("复测状态门禁不完整")
		}
	case domain.StatusReview:
		if !c.EvidenceComplete() {
			return errors.New("待复核案件证据不完整")
		}
	case domain.StatusReleased:
		if c.FrozenAt == nil || c.Review == nil || c.Review.Decision != "approve" || c.Archive == nil {
			return errors.New("已放行案件缺少复核、冻结时间或档案")
		}
		if c.Archive.SealedRevision != c.Revision || !archive.VerifyCase(c).Valid {
			return errors.New("已放行档案修订或摘要验证失败")
		}
	default:
		return errors.New("案件状态无效")
	}
	return nil
}
