package workflows

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/shared/apperror"
)

func TestRunHandlerPassesManualRunPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureWorkflowService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughWorkflowAuth, passThroughWorkflowPermission)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/wf-1/run", bytes.NewBufferString(`{"triggerType":"manual","input":"{\"environment\":\"prod\"}"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if service.runInput.ID != "wf-1" || service.runInput.TriggerType != "manual" {
		t.Fatalf("unexpected run input: %#v", service.runInput)
	}
	if service.runInput.Input != "{\"environment\":\"prod\"}" {
		t.Fatalf("unexpected workflow input payload: %q", service.runInput.Input)
	}
}

func TestRetryApproveAndRejectHandlersPassNodeContext(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureWorkflowService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughWorkflowAuth, passThroughWorkflowPermission)

	retryReq := httptest.NewRequest(http.MethodPost, "/api/v1/workflow-runs/run-1/retry", nil)
	retryResp := httptest.NewRecorder()
	router.ServeHTTP(retryResp, retryReq)
	if retryResp.Code != http.StatusOK {
		t.Fatalf("expected retry 200, got %d", retryResp.Code)
	}
	if service.retryInput.ID != "run-1" {
		t.Fatalf("unexpected retry input: %#v", service.retryInput)
	}

	approveReq := httptest.NewRequest(http.MethodPost, "/api/v1/workflow-runs/run-1/nodes/gate/approve", bytes.NewBufferString(`{"comment":"ship it"}`))
	approveReq.Header.Set("Content-Type", "application/json")
	approveResp := httptest.NewRecorder()
	router.ServeHTTP(approveResp, approveReq)
	if approveResp.Code != http.StatusOK {
		t.Fatalf("expected approve 200, got %d", approveResp.Code)
	}
	if service.approvalInput.RunID != "run-1" || service.approvalInput.NodeID != "gate" || service.approvalInput.Comment != "ship it" {
		t.Fatalf("unexpected approval input: %#v", service.approvalInput)
	}

	rejectReq := httptest.NewRequest(http.MethodPost, "/api/v1/workflow-runs/run-1/nodes/gate/reject", bytes.NewBufferString(`{"comment":"needs changes"}`))
	rejectReq.Header.Set("Content-Type", "application/json")
	rejectResp := httptest.NewRecorder()
	router.ServeHTTP(rejectResp, rejectReq)
	if rejectResp.Code != http.StatusOK {
		t.Fatalf("expected reject 200, got %d", rejectResp.Code)
	}
	if service.rejectInput.RunID != "run-1" || service.rejectInput.NodeID != "gate" || service.rejectInput.Comment != "needs changes" {
		t.Fatalf("unexpected reject input: %#v", service.rejectInput)
	}
}

type captureWorkflowService struct {
	runInput      RunInput
	retryInput    RetryInput
	approvalInput ApprovalInput
	rejectInput   ApprovalInput
}

func (s *captureWorkflowService) List(context.Context, ListInput) (DefinitionListResult, *apperror.Error) {
	return DefinitionListResult{}, nil
}

func (s *captureWorkflowService) Get(context.Context, string) (DefinitionDetail, *apperror.Error) {
	return DefinitionDetail{}, nil
}

func (s *captureWorkflowService) Create(context.Context, CreateInput) (DefinitionDetail, *apperror.Error) {
	return DefinitionDetail{}, nil
}

func (s *captureWorkflowService) Update(context.Context, UpdateInput) (DefinitionDetail, *apperror.Error) {
	return DefinitionDetail{}, nil
}

func (s *captureWorkflowService) Publish(context.Context, string, AuditContext) (DefinitionDetail, *apperror.Error) {
	return DefinitionDetail{}, nil
}

func (s *captureWorkflowService) Disable(context.Context, string, AuditContext) (DefinitionDetail, *apperror.Error) {
	return DefinitionDetail{}, nil
}

func (s *captureWorkflowService) Copy(context.Context, CopyInput) (DefinitionDetail, *apperror.Error) {
	return DefinitionDetail{}, nil
}

func (s *captureWorkflowService) ListVersions(context.Context, string) ([]VersionSummary, *apperror.Error) {
	return []VersionSummary{}, nil
}

func (s *captureWorkflowService) Run(_ context.Context, input RunInput) (RunDetail, *apperror.Error) {
	s.runInput = input
	return RunDetail{RunSummary: RunSummary{ID: "run-1"}}, nil
}

func (s *captureWorkflowService) ListRuns(context.Context, ListInput) (RunListResult, *apperror.Error) {
	return RunListResult{}, nil
}

func (s *captureWorkflowService) GetRun(context.Context, string) (RunDetail, *apperror.Error) {
	return RunDetail{}, nil
}

func (s *captureWorkflowService) CancelRun(context.Context, CancelInput) (RunDetail, *apperror.Error) {
	return RunDetail{}, nil
}

func (s *captureWorkflowService) RetryRun(_ context.Context, input RetryInput) (RunDetail, *apperror.Error) {
	s.retryInput = input
	return RunDetail{RunSummary: RunSummary{ID: "run-2"}}, nil
}

func (s *captureWorkflowService) ApproveNode(_ context.Context, input ApprovalInput) (RunDetail, *apperror.Error) {
	s.approvalInput = input
	return RunDetail{RunSummary: RunSummary{ID: input.RunID}}, nil
}

func (s *captureWorkflowService) RejectNode(_ context.Context, input ApprovalInput) (RunDetail, *apperror.Error) {
	s.rejectInput = input
	return RunDetail{RunSummary: RunSummary{ID: input.RunID}}, nil
}

func passThroughWorkflowAuth(c *gin.Context) {
	c.Next()
}

func passThroughWorkflowPermission(string) gin.HandlerFunc {
	return passThroughWorkflowAuth
}
