package audits

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/shared/apperror"
)

func TestListPassesFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughAuth, passThroughPermission)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit-logs?action=task.create&actorType=user&result=success&resourceType=task_run&traceId=trace-1&keyword=deploy&createdFrom=2026-05-01&createdTo=2026-05-08&page=2&pageSize=50", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	got := service.input
	if got.Action != "task.create" || got.ActorType != "user" || got.Result != "success" || got.ResourceType != "task_run" || got.TraceID != "trace-1" || got.Keyword != "deploy" || got.CreatedFrom != "2026-05-01" || got.CreatedTo != "2026-05-08" || got.Page != 2 || got.PageSize != 50 {
		t.Fatalf("unexpected input: %#v", got)
	}
}

type captureService struct {
	input ListInput
}

func (s *captureService) List(_ context.Context, input ListInput) (ListResult, *apperror.Error) {
	s.input = input
	return ListResult{}, nil
}

func (s *captureService) Export(context.Context, ListInput, string, AuditContext) (ExportResult, *apperror.Error) {
	return ExportResult{}, nil
}

func (s *captureService) RunRetention(context.Context, RetentionInput) (RetentionResult, *apperror.Error) {
	return RetentionResult{}, nil
}

func passThroughAuth(c *gin.Context) {
	c.Next()
}

func passThroughPermission(string) gin.HandlerFunc {
	return passThroughAuth
}
