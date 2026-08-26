package domain

import (
	"fmt"
	"strings"
	"time"
)

func Create(in CreateInput) (*DeviationCase, error) {
	for k, v := range map[string]string{"case_id": in.CaseID, "room_code": in.RoomCode, "sample_point": in.SamplePoint, "actor_id": in.ActorID} {
		if err := textRequired(k, v); err != nil {
			return nil, err
		}
	}
	if in.ExcursionKind != KindMicrobial && in.ExcursionKind != KindParticle {
		return nil, invalid("excursion_kind", "超限类型必须是 microbial 或 particle")
	}
	if !validNonNegativeFinite(in.LimitValue) || !validNonNegativeFinite(in.ObservedValue) || in.ObservedValue <= in.LimitValue {
		return nil, invalid("observed_value", "观测值必须大于非负限值")
	}
	if err := validateTime("occurred_at", in.OccurredAt); err != nil {
		return nil, err
	}
	if in.WindowStart.IsZero() || in.WindowEnd.IsZero() || in.WindowEnd.Before(in.WindowStart) {
		return nil, invalid("affected_window", "影响时间窗无效")
	}
	seenRelated := map[string]struct{}{}
	for _, related := range in.RelatedCases {
		if strings.TrimSpace(related.CaseID) == "" || related.CaseID == in.CaseID || related.OverlapStart.Before(in.WindowStart) || related.OverlapEnd.After(in.WindowEnd) || related.OverlapEnd.Before(related.OverlapStart) {
			return nil, invalid("related_cases", "关联案件交叠摘要无效")
		}
		if _, exists := seenRelated[related.CaseID]; exists {
			return nil, invalid("related_cases", "关联案件不能重复")
		}
		seenRelated[related.CaseID] = struct{}{}
	}
	now := in.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	related := append([]CaseOverlap(nil), in.RelatedCases...)
	c := &DeviationCase{CaseID: in.CaseID, RoomCode: strings.TrimSpace(in.RoomCode), ExcursionKind: in.ExcursionKind, SamplePoint: strings.TrimSpace(in.SamplePoint), ObservedValue: in.ObservedValue, LimitValue: in.LimitValue, OccurredAt: in.OccurredAt.UTC(), AffectedWindowStart: in.WindowStart.UTC(), AffectedWindowEnd: in.WindowEnd.UTC(), Status: StatusInvestigating, Revision: 1, CreatedAt: now, Actions: []CorrectiveAction{}, Retests: []RetestRound{}, RelatedCases: related, Timeline: []TimelineEntry{}}
	summary := "偏离已登记，受影响区域及作业时间窗已冻结"
	if len(related) > 0 {
		parts := make([]string, 0, len(related))
		for _, overlap := range related {
			parts = append(parts, fmt.Sprintf("%s（%s 至 %s）", overlap.CaseID, overlap.OverlapStart.Format(time.RFC3339), overlap.OverlapEnd.Format(time.RFC3339)))
		}
		summary += "；已确认关联范围：" + strings.Join(parts, "、")
	}
	c.addTimeline("case_created", in.ActorID, summary, now)
	return c, nil
}

func (c *DeviationCase) ConfirmInvestigation(in InvestigationInput) error {
	if c.Status != StatusInvestigating && c.Status != StatusCorrecting {
		return state("当前状态不能确认调查")
	}
	checks := map[string]string{"investigation_id": in.InvestigationID, "investigator_id": in.InvestigatorID, "personnel_findings": in.PersonnelFindings, "equipment_findings": in.EquipmentFindings, "cleaning_findings": in.CleaningFindings, "adjacent_sample_findings": in.AdjacentSampleFindings, "root_cause_category": in.RootCauseCategory, "root_cause_statement": in.RootCauseStatement}
	for k, v := range checks {
		if err := textRequired(k, v); err != nil {
			return err
		}
	}
	if len(in.EvidenceDigests) == 0 {
		return invalid("evidence_digests", "至少需要一个调查证据摘要")
	}
	now := normalizedNow(in.Now)
	c.Investigation = &Investigation{InvestigationID: in.InvestigationID, CaseID: c.CaseID, InvestigatorID: in.InvestigatorID, PersonnelFindings: in.PersonnelFindings, EquipmentFindings: in.EquipmentFindings, CleaningFindings: in.CleaningFindings, AdjacentSampleFindings: in.AdjacentSampleFindings, RootCauseCategory: in.RootCauseCategory, RootCauseStatement: in.RootCauseStatement, EvidenceDigests: append([]string(nil), in.EvidenceDigests...), ConfirmedAt: now}
	c.Status = StatusCorrecting
	c.bump("investigation_confirmed", in.InvestigatorID, "根因调查已确认", now)
	return nil
}

func (c *DeviationCase) AddAction(in AddActionInput) error {
	if c.Status != StatusCorrecting {
		return state("当前状态不能新增纠正措施")
	}
	if c.Investigation == nil {
		return state("必须先确认调查")
	}
	for _, a := range c.Actions {
		if a.ActionID == in.ActionID {
			return invalid("action_id", "纠正措施编号已存在")
		}
	}
	if err := textRequired("action_id", in.ActionID); err != nil {
		return err
	}
	if err := textRequired("description", in.Description); err != nil {
		return err
	}
	if err := textRequired("owner_id", in.OwnerID); err != nil {
		return err
	}
	if in.ReplacedActionID != "" {
		if in.ReplacedActionID == in.ActionID {
			return invalid("replaced_action_id", "替代措施不能引用自身")
		}
		var replaced *CorrectiveAction
		for i := range c.Actions {
			if c.Actions[i].ReplacedActionID == in.ReplacedActionID {
				return invalid("replaced_action_id", "该撤销措施已有替代措施")
			}
			if c.Actions[i].ActionID == in.ReplacedActionID {
				replaced = &c.Actions[i]
			}
		}
		if replaced == nil {
			return invalid("replaced_action_id", "被替代措施不属于本案")
		}
		if replaced.Status != ActionRevoked {
			return invalid("replaced_action_id", "只能替代本案已撤销措施")
		}
	}
	now := normalizedNow(in.Now)
	c.Actions = append(c.Actions, CorrectiveAction{ActionID: in.ActionID, CaseID: c.CaseID, Description: in.Description, OwnerID: in.OwnerID, Status: ActionOpen, ReplacedActionID: in.ReplacedActionID})
	summary := "新增纠正措施：" + in.Description
	if in.ReplacedActionID != "" {
		summary += "；替代已撤销措施 " + in.ReplacedActionID
	}
	c.bump("action_added", in.ActorID, summary, now)
	return nil
}

func (c *DeviationCase) RevokeAction(in RevokeActionInput) error {
	if c.Status != StatusCorrecting {
		return state("当前状态不能撤销纠正措施")
	}
	if err := textRequired("revocation_reason", in.Reason); err != nil {
		return err
	}
	if err := textRequired("actor_id", in.ActorID); err != nil {
		return err
	}
	for i := range c.Actions {
		if c.Actions[i].ActionID != in.ActionID {
			continue
		}
		if c.Actions[i].Status == ActionRevoked {
			return state("纠正措施已经撤销")
		}
		now := normalizedNow(in.Now)
		c.Actions[i].Status = ActionRevoked
		c.Actions[i].RevokedAt = &now
		c.Actions[i].RevocationReason = strings.TrimSpace(in.Reason)
		c.Actions[i].RevokedBy = strings.TrimSpace(in.ActorID)
		c.bump("action_revoked", in.ActorID, "纠正措施 "+in.ActionID+" 已撤销："+strings.TrimSpace(in.Reason), now)
		return nil
	}
	return invalid("action_id", "纠正措施不存在")
}

func (c *DeviationCase) CompleteAction(in CompleteActionInput) error {
	if c.Status != StatusCorrecting {
		return state("当前状态不能完成纠正措施")
	}
	if err := textRequired("evidence_digest", in.EvidenceDigest); err != nil {
		return err
	}
	for i := range c.Actions {
		if c.Actions[i].ActionID == in.ActionID {
			if c.Actions[i].Status == ActionRevoked {
				return state("已撤销措施不能完成")
			}
			if c.Actions[i].Status == ActionCompleted {
				return state("已完成措施不能重复登记完成")
			}
			now := normalizedNow(in.Now)
			c.Actions[i].Status = ActionCompleted
			c.Actions[i].EvidenceDigest = in.EvidenceDigest
			c.Actions[i].CompletedAt = &now
			c.bump("action_completed", in.ActorID, "纠正措施已完成："+in.ActionID, now)
			return nil
		}
	}
	return invalid("action_id", "纠正措施不存在")
}

func (c *DeviationCase) RecordRetest(in RetestInput) error {
	if c.Status != StatusCorrecting && c.Status != StatusRetesting {
		return state("当前状态不能登记复测")
	}
	if c.Investigation == nil || len(c.Actions) == 0 {
		return state("调查和纠正措施尚未齐备")
	}
	if blockers := c.actionBlockingReasons(); len(blockers) > 0 {
		return state(strings.Join(blockers, "；"))
	}
	if !c.hasSupplementalCorrection() {
		return state("失败复测或驳回后必须新增并完成补充纠正措施")
	}
	for _, v := range []struct{ k, v string }{{"round_id", in.RoundID}, {"instrument_ref", in.InstrumentRef}, {"evidence_digest", in.EvidenceDigest}, {"recorded_by", in.RecordedBy}} {
		if err := textRequired(v.k, v.v); err != nil {
			return err
		}
	}
	for _, existing := range c.Retests {
		if existing.RoundID == in.RoundID {
			return invalid("round_id", "复测轮次编号已存在")
		}
	}
	if in.SampledAt.IsZero() {
		return invalid("sampled_at", "采样时间不能为空")
	}
	if !validNonNegativeFinite(in.ObservedValue) {
		return invalid("observed_value", "观测值必须是非负有限数值")
	}
	if !validNonNegativeFinite(in.LimitValue) {
		return invalid("limit_value", "限值必须是非负有限数值")
	}
	if in.LimitValue != c.LimitValue {
		return invalid("limit_value", fmt.Sprintf("复测限值必须与案件适用限值 %g 一致", c.LimitValue))
	}
	if len(c.Retests) > 0 && !in.SampledAt.After(c.Retests[len(c.Retests)-1].SampledAt) {
		return invalid("sampled_at", "复测必须按采样时间递增")
	}
	outcome := "pass"
	if in.ObservedValue > in.LimitValue {
		outcome = "fail"
	}
	now := normalizedNow(in.Now)
	c.Retests = append(c.Retests, RetestRound{RoundID: in.RoundID, CaseID: c.CaseID, Sequence: len(c.Retests) + 1, SampledAt: in.SampledAt.UTC(), ObservedValue: in.ObservedValue, LimitValue: in.LimitValue, InstrumentRef: in.InstrumentRef, EvidenceDigest: in.EvidenceDigest, Outcome: outcome, RecordedBy: in.RecordedBy})
	if outcome == "fail" {
		c.Status = StatusCorrecting
		c.bump("retest_failed", in.RecordedBy, "复测不合格，连续计数清零并要求补充纠正", now)
		return nil
	}
	c.Status = StatusRetesting
	passing := c.ConsecutivePassing()
	summary := fmt.Sprintf("复测合格，当前连续 %d 轮", passing)
	if passing >= RequiredPassingRounds(c.ExcursionKind) {
		c.Status = StatusReview
		summary += "，已达到复核门禁"
	}
	c.bump("retest_passed", in.RecordedBy, summary, now)
	return nil
}

func (c *DeviationCase) ReviewDecision(in ReviewInput) error {
	if c.Status != StatusReview {
		return state("案件尚未达到复核门禁")
	}
	if err := textRequired("reviewer_id", in.ReviewerID); err != nil {
		return err
	}
	if in.Decision != "approve" && in.Decision != "reject" {
		return invalid("decision", "决定必须是 approve 或 reject")
	}
	participants := []string{}
	if c.Investigation != nil {
		participants = append(participants, c.Investigation.InvestigatorID)
	}
	for _, a := range c.Actions {
		participants = append(participants, a.OwnerID)
	}
	if contains(participants, in.ReviewerID) {
		return &DomainError{Code: CodeForbidden, Field: "reviewer_id", Message: "复核员不得参与调查或纠正"}
	}
	if in.Decision == "reject" && strings.TrimSpace(in.Reason) == "" {
		return invalid("reason", "驳回必须填写理由")
	}
	if in.Decision == "approve" && !c.EvidenceComplete() {
		return state("证据不完整，不能批准")
	}
	now := normalizedNow(in.Now)
	c.Review = &Review{ReviewerID: in.ReviewerID, Decision: in.Decision, Reason: in.Reason, ReviewedAt: now}
	if in.Decision == "reject" {
		c.Status = StatusCorrecting
		c.bump("review_rejected", in.ReviewerID, "复核驳回："+in.Reason, now)
	} else {
		c.Status = StatusReleased
		c.FrozenAt = &now
		c.bump("review_approved", in.ReviewerID, "独立复核批准，案件已冻结", now)
	}
	return nil
}

func (c *DeviationCase) AttachArchive(a *ReleaseArchive) error {
	if c.Status != StatusReleased {
		return state("案件未批准")
	}
	if a.SealedRevision != c.Revision {
		return state("档案封存修订不一致")
	}
	c.Archive = a
	return nil
}
func (c *DeviationCase) bump(t, actor, summary string, at time.Time) {
	c.Revision++
	c.addTimeline(t, actor, summary, at)
}
func (c *DeviationCase) addTimeline(t, actor, summary string, at time.Time) {
	c.Timeline = append(c.Timeline, TimelineEntry{Revision: c.Revision, Type: t, At: at, ActorID: actor, Summary: summary})
}
func normalizedNow(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now().UTC()
	}
	return t.UTC()
}
