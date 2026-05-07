package tasks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/config"
	authmodule "opspilot/server/internal/modules/auth"
	jwtplatform "opspilot/server/internal/platform/jwt"
	"opspilot/server/internal/shared/apperror"
)

func TestAgentRouteRejectsUserJWTWithoutAgentIdentity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := jwtplatform.NewManager(config.JWTConfig{
		AccessSecret:  "test-access-secret",
		RefreshSecret: "test-refresh-secret",
		AccessTTL:     time.Hour,
		RefreshTTL:    24 * time.Hour,
	})
	token, err := manager.GenerateAccessToken("user-member", "member", []string{"member"})
	if err != nil {
		t.Fatalf("generate access token: %v", err)
	}

	router := gin.New()
	NewHandler(&agentAuthTaskService{}).RegisterAgentRoutes(router.Group("/api/v1"), authmodule.AuthMiddleware(manager))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/agent/tasks/poll", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", resp.Code, resp.Body.String())
	}
}

type agentAuthTaskService struct{}

func (s *agentAuthTaskService) List(context.Context, ListTasksInput) (TaskListResult, *apperror.Error) {
	return TaskListResult{}, nil
}

func (s *agentAuthTaskService) Get(context.Context, string) (TaskDetail, *apperror.Error) {
	return TaskDetail{}, nil
}

func (s *agentAuthTaskService) Create(context.Context, CreateTaskInput) (TaskDetail, *apperror.Error) {
	return TaskDetail{}, nil
}

func (s *agentAuthTaskService) Cancel(context.Context, string, AuditContext) *apperror.Error {
	return nil
}

func (s *agentAuthTaskService) Targets(context.Context, string) ([]TaskTargetSummary, *apperror.Error) {
	return nil, nil
}

func (s *agentAuthTaskService) Poll(context.Context, AgentIdentity, int) ([]AgentTask, *apperror.Error) {
	return nil, nil
}

func (s *agentAuthTaskService) Claim(context.Context, AgentIdentity, string) (AgentTask, *apperror.Error) {
	return AgentTask{}, nil
}

func (s *agentAuthTaskService) TargetState(context.Context, AgentIdentity, string) (TargetState, *apperror.Error) {
	return TargetState{}, nil
}

func (s *agentAuthTaskService) UploadLog(context.Context, AgentIdentity, LogInput) *apperror.Error {
	return nil
}

func (s *agentAuthTaskService) ReportResult(context.Context, AgentIdentity, ResultInput) *apperror.Error {
	return nil
}

func (s *agentAuthTaskService) Logs(context.Context, LogQuery) ([]TaskLogEntry, *apperror.Error) {
	return nil, nil
}
