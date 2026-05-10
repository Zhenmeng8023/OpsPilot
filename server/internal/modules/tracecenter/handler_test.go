package tracecenter

import (
	"context"
	"net/http"
	"net/http/httptest"
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

type captureService struct {
	input SearchInput
}

func (s *captureService) Search(_ context.Context, input SearchInput) (SearchResult, *apperror.Error) {
	s.input = input
	return SearchResult{}, nil
}

func passThroughAuth(c *gin.Context) {
	c.Next()
}

func passThroughPermission(string) gin.HandlerFunc {
	return passThroughAuth
}

