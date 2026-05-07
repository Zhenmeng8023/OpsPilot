package tasks

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/shared/apperror"
)

func TestCreateTaskApprovalGateAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name        string
		scriptID    string
		script      scriptRecord
		wantHTTP    int
		wantCode    int
		wantMessage string
	}{
		{
			name:     "pending approval",
			scriptID: "script-pending",
			script: scriptRecord{
				ApprovalRequired:     true,
				LatestApprovalStatus: nullString("pending"),
			},
			wantHTTP:    http.StatusConflict,
			wantCode:    409306,
			wantMessage: "script approval is pending",
		},
		{
			name:     "rejected approval",
			scriptID: "script-rejected",
			script: scriptRecord{
				ApprovalRequired:     true,
				LatestApprovalStatus: nullString("rejected"),
			},
			wantHTTP:    http.StatusBadRequest,
			wantCode:    400309,
			wantMessage: "script latest revision is not approved",
		},
		{
			name:     "missing approval record",
			scriptID: "script-no-approval-record",
			script: scriptRecord{
				ApprovalRequired: true,
			},
			wantHTTP:    http.StatusBadRequest,
			wantCode:    400309,
			wantMessage: "script requires approval before execution",
		},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			service := &approvalGateCreateService{
				scripts: map[string]scriptRecord{
					item.scriptID: item.script,
				},
			}
			router := gin.New()
			NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughMiddleware, passThroughPermission)

			body := bytes.NewBufferString(`{
				"name": "approval gate task",
				"scriptId": "` + item.scriptID + `",
				"timeoutSeconds": 30,
				"targetAgentIds": ["agent-1"]
			}`)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", body)
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			if resp.Code != item.wantHTTP {
				t.Fatalf("expected HTTP %d, got %d: %s", item.wantHTTP, resp.Code, resp.Body.String())
			}
			var envelope struct {
				Code    int             `json:"code"`
				Message string          `json:"message"`
				Data    json.RawMessage `json:"data"`
			}
			if err := json.Unmarshal(resp.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if envelope.Code != item.wantCode {
				t.Fatalf("expected code %d, got %d", item.wantCode, envelope.Code)
			}
			if envelope.Message != item.wantMessage {
				t.Fatalf("expected message %q, got %q", item.wantMessage, envelope.Message)
			}
			if string(envelope.Data) != "null" {
				t.Fatalf("expected null data for failed response, got %s", string(envelope.Data))
			}
		})
	}
}

type approvalGateCreateService struct {
	scripts map[string]scriptRecord
}

func (f *approvalGateCreateService) List(context.Context, ListTasksInput) (TaskListResult, *apperror.Error) {
	return TaskListResult{}, nil
}

func (f *approvalGateCreateService) Get(context.Context, string) (TaskDetail, *apperror.Error) {
	return TaskDetail{}, nil
}

func (f *approvalGateCreateService) Create(_ context.Context, input CreateTaskInput) (TaskDetail, *apperror.Error) {
	script, ok := f.scripts[input.ScriptID]
	if !ok {
		return TaskDetail{}, apperror.New(http.StatusNotFound, 404201, "script not found")
	}
	if err := validateScriptApproval(script); err != nil {
		appErr, ok := err.(*apperror.Error)
		if !ok {
			return TaskDetail{}, apperror.Wrap(http.StatusInternalServerError, 500301, "create task failed", err)
		}
		return TaskDetail{}, appErr
	}
	return TaskDetail{TaskSummary: TaskSummary{ID: "task-1", Status: string(StatusQueued)}}, nil
}

func (f *approvalGateCreateService) Cancel(context.Context, string, AuditContext) *apperror.Error {
	return nil
}

func (f *approvalGateCreateService) Targets(context.Context, string) ([]TaskTargetSummary, *apperror.Error) {
	return nil, nil
}

func (f *approvalGateCreateService) Poll(context.Context, AgentIdentity, int) ([]AgentTask, *apperror.Error) {
	return nil, nil
}

func (f *approvalGateCreateService) Claim(context.Context, AgentIdentity, string) (AgentTask, *apperror.Error) {
	return AgentTask{}, nil
}

func (f *approvalGateCreateService) TargetState(context.Context, AgentIdentity, string) (TargetState, *apperror.Error) {
	return TargetState{}, nil
}

func (f *approvalGateCreateService) UploadLog(context.Context, AgentIdentity, LogInput) *apperror.Error {
	return nil
}

func (f *approvalGateCreateService) ReportResult(context.Context, AgentIdentity, ResultInput) *apperror.Error {
	return nil
}

func (f *approvalGateCreateService) Logs(context.Context, LogQuery) ([]TaskLogEntry, *apperror.Error) {
	return nil, nil
}

func passThroughMiddleware(c *gin.Context) {
	c.Next()
}

func passThroughPermission(string) gin.HandlerFunc {
	return passThroughMiddleware
}
