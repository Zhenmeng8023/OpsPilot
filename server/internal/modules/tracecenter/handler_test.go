package tracecenter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/shared/apperror"
)

func TestSearchPassesQueryKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughAuth, passThroughPermission)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/trace-center?traceId=trace-1&taskRunId=task-1&workflowRunId=wf-1&webhookEventId=web-1", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.input.TraceID != "trace-1" || service.input.TaskRunID != "task-1" || service.input.WorkflowRunID != "wf-1" || service.input.WebhookEventID != "web-1" {
		t.Fatalf("unexpected input: %#v", service.input)
	}
}

func TestRunRetentionPassesBody(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughAuth, passThroughPermission)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/traces/retention/run", strings.NewReader(`{"days":14,"dryRun":true,"traceId":"trc_1"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.retentionInput.Days != 14 || !service.retentionInput.DryRun || service.retentionInput.TraceID != "trc_1" {
		t.Fatalf("unexpected retention input: %#v", service.retentionInput)
	}
}

type captureService struct {
	input          SearchInput
	retentionInput RetentionInput
}

func (s *captureService) Search(_ context.Context, input SearchInput) (SearchResult, *apperror.Error) {
	s.input = input
	return SearchResult{}, nil
}

func (s *captureService) RunRetention(_ context.Context, input RetentionInput) (RetentionResult, *apperror.Error) {
	s.retentionInput = input
	return RetentionResult{}, nil
}

func passThroughAuth(c *gin.Context) {
	c.Next()
}

func passThroughPermission(string) gin.HandlerFunc {
	return passThroughAuth
}
