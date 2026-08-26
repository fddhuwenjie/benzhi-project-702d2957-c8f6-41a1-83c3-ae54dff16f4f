package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	"cleanroom-recovery-ledger/internal/application"
	"cleanroom-recovery-ledger/internal/domain"
	"cleanroom-recovery-ledger/internal/store"
	"cleanroom-recovery-ledger/internal/web"
)

type selfCheckResult struct {
	Case *domain.DeviationCase `json:"case"`
}

func runSelfCheck(cfg config) error {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.selfCheckTimeout)
	defer cancel()
	dir, err := os.MkdirTemp("", "cleanroom-recovery-selfcheck-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	repo, err := store.Open(dir)
	if err != nil {
		return err
	}
	server := &http.Server{Handler: web.New(application.New(repo)).Handler(), ReadHeaderTimeout: 2 * time.Second}
	listener, err := net.Listen("tcp", cfg.addr)
	if err != nil {
		return fmt.Errorf("自检监听失败: %w", err)
	}
	serveDone := make(chan error, 1)
	go func() { serveDone <- server.Serve(listener) }()
	base := "http://" + listener.Addr().String()
	client := &http.Client{Timeout: 3 * time.Second}
	flowErr := exerciseFlow(ctx, client, base)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer shutdownCancel()
	shutdownErr := server.Shutdown(shutdownCtx)
	<-serveDone
	if flowErr != nil {
		return flowErr
	}
	if shutdownErr != nil {
		return shutdownErr
	}
	fmt.Println("自检通过：真实 HTTP 全流程已完成，放行档案摘要验证成功")
	return nil
}

func exerciseFlow(ctx context.Context, client *http.Client, base string) error {
	now := time.Now().UTC().Truncate(time.Second)
	caseID := "SELF-CHECK-CASE"
	result, err := post(ctx, client, base+"/api/cases", map[string]any{"request_id": "req-create", "case_id": caseID, "room_code": "CR-A01", "excursion_kind": "microbial", "sample_point": "SP-07", "observed_value": 12, "limit_value": 5, "occurred_at": now.Add(-2 * time.Hour), "affected_window_start": now.Add(-3 * time.Hour), "affected_window_end": now.Add(-time.Hour), "actor_id": "monitor-01"})
	if err != nil {
		return err
	}
	rev := result.Case.Revision
	result, err = post(ctx, client, base+"/api/cases/"+caseID+"/investigation", map[string]any{"request_id": "req-investigation", "expected_revision": rev, "investigation_id": "INV-1", "investigator_id": "investigator-01", "personnel_findings": "人员进出记录已核对", "equipment_findings": "采样设备校准有效", "cleaning_findings": "发现清洁接触时间不足", "adjacent_sample_findings": "相邻点未检出异常", "root_cause_category": "cleaning", "root_cause_statement": "消毒剂接触时间不足导致偏离", "evidence_digests": []string{"sha256:investigation"}})
	if err != nil {
		return err
	}
	rev = result.Case.Revision
	result, err = post(ctx, client, base+"/api/cases/"+caseID+"/actions", map[string]any{"request_id": "req-action", "expected_revision": rev, "action_id": "CAPA-1", "description": "重新清洁并满足接触时间", "owner_id": "operator-02", "actor_id": "investigator-01"})
	if err != nil {
		return err
	}
	rev = result.Case.Revision
	result, err = post(ctx, client, base+"/api/cases/"+caseID+"/actions/CAPA-1/complete", map[string]any{"request_id": "req-action-done", "expected_revision": rev, "evidence_digest": "sha256:capa", "actor_id": "operator-02"})
	if err != nil {
		return err
	}
	rev = result.Case.Revision
	for i := 1; i <= 3; i++ {
		result, err = post(ctx, client, base+"/api/cases/"+caseID+"/retests", map[string]any{"request_id": fmt.Sprintf("req-retest-%d", i), "expected_revision": rev, "round_id": fmt.Sprintf("ROUND-%d", i), "sampled_at": now.Add(time.Duration(i) * time.Hour), "observed_value": float64(i), "limit_value": 5, "instrument_ref": "INST-01", "evidence_digest": fmt.Sprintf("sha256:retest-%d", i), "recorded_by": "monitor-01"})
		if err != nil {
			return err
		}
		rev = result.Case.Revision
	}
	result, err = post(ctx, client, base+"/api/cases/"+caseID+"/review", map[string]any{"request_id": "req-review", "expected_revision": rev, "reviewer_id": "qa-independent", "decision": "approve", "reason": ""})
	if err != nil {
		return err
	}
	if result.Case.Status != domain.StatusReleased || result.Case.Archive == nil {
		return fmt.Errorf("自检终态或档案不正确")
	}
	var verification struct {
		Valid bool `json:"valid"`
	}
	if err = getOrPost(ctx, client, "POST", base+"/api/cases/"+caseID+"/archive/verify", map[string]any{"request_id": "req-verify", "verifier_id": "qa-auditor"}, &struct {
		RequestID string `json:"request_id"`
		Result    *struct {
			Valid bool `json:"valid"`
		} `json:"result"`
	}{Result: &verification}); err != nil {
		return err
	}
	if !verification.Valid {
		return fmt.Errorf("档案摘要验证失败")
	}
	return nil
}

func post(ctx context.Context, client *http.Client, url string, body any) (selfCheckResult, error) {
	var out selfCheckResult
	err := getOrPost(ctx, client, "POST", url, body, &out)
	return out, err
}
func getOrPost(ctx context.Context, client *http.Client, method, url string, body, dst any) error {
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%s %s 返回 %d: %s", method, url, resp.StatusCode, string(data))
	}
	if err = json.Unmarshal(data, dst); err != nil {
		return fmt.Errorf("解析自检响应失败: %w", err)
	}
	return nil
}
