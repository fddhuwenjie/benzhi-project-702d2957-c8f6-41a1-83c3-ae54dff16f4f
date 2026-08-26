package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cleanroom-recovery-ledger/internal/application"
	"cleanroom-recovery-ledger/internal/store"
)

func TestWorkbenchAndProblemDetails(t *testing.T) {
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	handler := New(application.New(repo)).Handler()
	page := httptest.NewRecorder()
	handler.ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != 200 || !strings.Contains(page.Body.String(), "环境偏离恢复放行台账") {
		t.Fatalf("工作台响应无效: %d", page.Code)
	}
	bad := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/cases", strings.NewReader(`{"request_id":"r"}`))
	req.Header.Set("Content-Type", "application/json")
	handler.ServeHTTP(bad, req)
	if bad.Code != 400 || !strings.Contains(bad.Body.String(), "validation_failed") {
		t.Fatalf("问题详情响应无效: %d %s", bad.Code, bad.Body.String())
	}
}
