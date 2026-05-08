package schedules

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/shared/apperror"
)

func TestCreateHandlerPassesWorkflowTarget(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureScheduleService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughScheduleAuth, passThroughSchedulePermission)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/schedules", bytes.NewBufferString(`{
		"name":"workflow schedule",
		"targetType":"workflow",
		"workflowId":"wf-1",
		"cronExpr":"*/5 * * * *",
		"timezone":"Asia/Shanghai",
		"misfirePolicy":"fire_once"
	}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	if service.createInput.TargetType != "workflow" || service.createInput.WorkflowID != "wf-1" {
		t.Fatalf("unexpected create input: %#v", service.createInput)
	}
}

type captureScheduleService struct {
	createInput CreateInput
}

func (s *captureScheduleService) List(context.Context, ListInput) (ScheduleListResult, *apperror.Error) {
	return ScheduleListResult{}, nil
}

func (s *captureScheduleService) Create(_ context.Context, input CreateInput) (ScheduleSummary, *apperror.Error) {
	s.createInput = input
	return ScheduleSummary{ID: "sch-1"}, nil
}

func (s *captureScheduleService) Preview(context.Context, PreviewInput) (PreviewResult, *apperror.Error) {
	return PreviewResult{}, nil
}

func (s *captureScheduleService) ListTriggers(context.Context, string, int) ([]TriggerSummary, *apperror.Error) {
	return nil, nil
}

func (s *captureScheduleService) Pause(context.Context, string, AuditContext) *apperror.Error {
	return nil
}

func (s *captureScheduleService) Resume(context.Context, string, AuditContext) *apperror.Error {
	return nil
}

func (s *captureScheduleService) Disable(context.Context, string, AuditContext) *apperror.Error {
	return nil
}

func passThroughScheduleAuth(c *gin.Context) {
	c.Next()
}

func passThroughSchedulePermission(string) gin.HandlerFunc {
	return passThroughScheduleAuth
}
