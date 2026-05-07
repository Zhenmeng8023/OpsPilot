package alerts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/shared/apperror"
)

func TestListAlertsPassesStatusFilter(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureAlertService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughAlertAuth, passThroughAlertPermission)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts?status=silenced&severity=critical&ruleId=rule-9&hostId=host-4", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.alertsInput.Status != "silenced" || service.alertsInput.Severity != "critical" || service.alertsInput.RuleID != "rule-9" || service.alertsInput.HostID != "host-4" {
		t.Fatalf("unexpected alert filters: %#v", service.alertsInput)
	}
}

func TestListAlertHistoryPassesWindowParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureAlertService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughAlertAuth, passThroughAlertPermission)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/history?hours=72&bucketMinutes=30&severity=warning&ruleId=rule-3&hostId=host-2", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.historyInput.Hours != 72 || service.historyInput.BucketMinutes != 30 || service.historyInput.Severity != "warning" || service.historyInput.RuleID != "rule-3" || service.historyInput.HostID != "host-2" {
		t.Fatalf("unexpected history input: %#v", service.historyInput)
	}
}

func TestAcknowledgePassesID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureAlertService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughAlertAuth, passThroughAlertPermission)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/al-1/ack", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.ackAlertID != "al-1" {
		t.Fatalf("unexpected ack id: %q", service.ackAlertID)
	}
}

func TestListAlertEventsPassesID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureAlertService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughAlertAuth, passThroughAlertPermission)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/al-4/events?eventType=resolved", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.eventsAlertID != "al-4" {
		t.Fatalf("unexpected events id: %q", service.eventsAlertID)
	}
	if service.eventsEventType != "resolved" {
		t.Fatalf("unexpected event type filter: %q", service.eventsEventType)
	}
}

func TestUpdateRulePassesIDAndPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureAlertService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughAlertAuth, passThroughAlertPermission)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/alert-rules/rule-1", strings.NewReader(`{"name":"cpu hot","metricCode":"agent.cpu.logical","operator":">=","threshold":8,"durationSeconds":120,"cooldownSeconds":600,"severity":"critical"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.updateRuleID != "rule-1" || service.updateRuleInput.CooldownSeconds != 600 || service.updateRuleInput.Operator != ">=" {
		t.Fatalf("unexpected update rule call: id=%q input=%#v", service.updateRuleID, service.updateRuleInput)
	}
}

func TestPauseResumeDisableRulePassesStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureAlertService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughAlertAuth, passThroughAlertPermission)

	cases := []struct {
		path   string
		status string
	}{
		{"/api/v1/alert-rules/rule-2/pause", "paused"},
		{"/api/v1/alert-rules/rule-2/resume", "active"},
		{"/api/v1/alert-rules/rule-2/disable", "disabled"},
	}
	for _, tt := range cases {
		req := httptest.NewRequest(http.MethodPost, tt.path, nil)
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		if resp.Code != http.StatusOK {
			t.Fatalf("%s expected 200, got %d: %s", tt.path, resp.Code, resp.Body.String())
		}
		if service.ruleStatusID != "rule-2" || service.ruleStatus != tt.status {
			t.Fatalf("%s unexpected rule status call: id=%q status=%q", tt.path, service.ruleStatusID, service.ruleStatus)
		}
	}
}

func TestSilencePassesPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureAlertService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughAlertAuth, passThroughAlertPermission)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/al-2/silence", strings.NewReader(`{"durationSeconds":1800,"reason":"maintenance"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.silenceAlertID != "al-2" || service.silenceInput.DurationSeconds != 1800 || service.silenceInput.Reason != "maintenance" {
		t.Fatalf("unexpected silence payload: id=%q input=%#v", service.silenceAlertID, service.silenceInput)
	}
}

func TestUnsilencePassesID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureAlertService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughAlertAuth, passThroughAlertPermission)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/al-3/unsilence", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.unsilenceAlertID != "al-3" {
		t.Fatalf("unexpected unsilence id: %q", service.unsilenceAlertID)
	}
}

type captureAlertService struct {
	alertsInput      ListAlertsInput
	historyInput     AlertHistoryInput
	eventsAlertID    string
	eventsEventType  string
	ackAlertID       string
	silenceAlertID   string
	silenceInput     SilenceInput
	unsilenceAlertID string
	updateRuleID     string
	updateRuleInput  UpdateRuleInput
	ruleStatusID     string
	ruleStatus       string
}

func (s *captureAlertService) ListRules(context.Context) ([]AlertRuleSummary, *apperror.Error) {
	return nil, nil
}

func (s *captureAlertService) CreateRule(context.Context, CreateRuleInput) (AlertRuleSummary, *apperror.Error) {
	return AlertRuleSummary{}, nil
}

func (s *captureAlertService) UpdateRule(_ context.Context, id string, input UpdateRuleInput) (AlertRuleSummary, *apperror.Error) {
	s.updateRuleID = id
	s.updateRuleInput = input
	return AlertRuleSummary{ID: id}, nil
}

func (s *captureAlertService) UpdateRuleStatus(_ context.Context, id string, status string, _ AuditContext) *apperror.Error {
	s.ruleStatusID = id
	s.ruleStatus = status
	return nil
}

func (s *captureAlertService) ListAlerts(_ context.Context, input ListAlertsInput) ([]AlertSummary, *apperror.Error) {
	s.alertsInput = input
	return nil, nil
}

func (s *captureAlertService) ListAlertHistory(_ context.Context, input AlertHistoryInput) ([]AlertHistoryPoint, *apperror.Error) {
	s.historyInput = input
	return nil, nil
}

func (s *captureAlertService) ListAlertEvents(_ context.Context, id string, eventType string) ([]AlertEventSummary, *apperror.Error) {
	s.eventsAlertID = id
	s.eventsEventType = eventType
	return nil, nil
}

func (s *captureAlertService) Acknowledge(_ context.Context, id string, _ AuditContext) *apperror.Error {
	s.ackAlertID = id
	return nil
}

func (s *captureAlertService) Silence(_ context.Context, id string, input SilenceInput) *apperror.Error {
	s.silenceAlertID = id
	s.silenceInput = input
	return nil
}

func (s *captureAlertService) Unsilence(_ context.Context, id string, _ AuditContext) *apperror.Error {
	s.unsilenceAlertID = id
	return nil
}

func (s *captureAlertService) Resolve(context.Context, string, AuditContext) *apperror.Error {
	return nil
}

func passThroughAlertAuth(c *gin.Context) {
	c.Next()
}

func passThroughAlertPermission(string) gin.HandlerFunc {
	return passThroughAlertAuth
}
