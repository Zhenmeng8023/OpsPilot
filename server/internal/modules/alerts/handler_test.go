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

func TestListAlertGroupsPassesFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureAlertService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughAlertAuth, passThroughAlertPermission)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/alert-groups?status=firing&severity=critical&ruleId=rule-7&hostGroupId=group-3", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.alertGroupsInput.Status != "firing" || service.alertGroupsInput.Severity != "critical" || service.alertGroupsInput.RuleID != "rule-7" || service.alertGroupsInput.HostGroupID != "group-3" {
		t.Fatalf("unexpected alert group filters: %#v", service.alertGroupsInput)
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

func TestListNoisyRulesPassesQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureAlertService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughAlertAuth, passThroughAlertPermission)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/noisy-rules?hours=48&limit=7", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.noisyInput.Hours != 48 || service.noisyInput.Limit != 7 {
		t.Fatalf("unexpected noisy input: %#v", service.noisyInput)
	}
}

func TestListNoiseTrendsPassesQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureAlertService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughAlertAuth, passThroughAlertPermission)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/alerts/noise-trends?hours=36", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.noiseTrendInput.Hours != 36 {
		t.Fatalf("unexpected trend input: %#v", service.noiseTrendInput)
	}
}

func TestSuppressionDryRunPassesPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureAlertService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughAlertAuth, passThroughAlertPermission)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/suppression-dry-run", strings.NewReader(`{"ruleId":"rule-1","hostId":"host-1","hostGroupId":"group-1","severity":"critical"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.suppressionDryRunInput.RuleID != "rule-1" || service.suppressionDryRunInput.HostID != "host-1" || service.suppressionDryRunInput.HostGroupID != "group-1" || service.suppressionDryRunInput.Severity != "critical" {
		t.Fatalf("unexpected suppression dry-run input: %#v", service.suppressionDryRunInput)
	}
}

func TestRoutingDryRunPassesPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureAlertService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughAlertAuth, passThroughAlertPermission)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/routing-dry-run", strings.NewReader(`{"ruleId":"rule-1","hostGroupId":"group-2","severity":"warning","channelId":"channel-7"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.routingDryRunInput.RuleID != "rule-1" || service.routingDryRunInput.HostGroupID != "group-2" || service.routingDryRunInput.Severity != "warning" || service.routingDryRunInput.ChannelID != "channel-7" {
		t.Fatalf("unexpected routing dry-run input: %#v", service.routingDryRunInput)
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

func TestCreateSuppressionRulePassesPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureAlertService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughAlertAuth, passThroughAlertPermission)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-suppression-rules", strings.NewReader(`{"name":"quiet cpu","ruleId":"rule-1","hostId":"host-1","hostGroupId":"group-1","severity":"critical","reason":"maintenance","status":"active"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.suppressionInput.Name != "quiet cpu" || service.suppressionInput.RuleID != "rule-1" || service.suppressionInput.HostID != "host-1" || service.suppressionInput.HostGroupID != "group-1" || service.suppressionInput.Severity != "critical" {
		t.Fatalf("unexpected suppression payload: %#v", service.suppressionInput)
	}
}

func TestCreateRoutingPolicyPassesPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureAlertService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughAlertAuth, passThroughAlertPermission)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/alert-routing-policies", strings.NewReader(`{"name":"critical email","hostGroupId":"group-2","severity":"critical","channelId":"chan-1","status":"active"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.routingInput.Name != "critical email" || service.routingInput.HostGroupID != "group-2" || service.routingInput.Severity != "critical" || service.routingInput.ChannelID != "chan-1" {
		t.Fatalf("unexpected routing payload: %#v", service.routingInput)
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
	alertGroupsInput       ListAlertGroupsInput
	alertsInput            ListAlertsInput
	historyInput           AlertHistoryInput
	noisyInput             NoisyRuleInput
	noiseTrendInput        NoiseTrendInput
	suppressionDryRunInput SuppressionDryRunInput
	routingDryRunInput     RoutingDryRunInput
	eventsAlertID          string
	eventsEventType        string
	ackAlertID             string
	silenceAlertID         string
	silenceInput           SilenceInput
	unsilenceAlertID       string
	updateRuleID           string
	updateRuleInput        UpdateRuleInput
	ruleStatusID           string
	ruleStatus             string
	suppressionInput       SuppressionRuleInput
	routingInput           RoutingPolicyInput
}

func (s *captureAlertService) ListRules(context.Context) ([]AlertRuleSummary, *apperror.Error) {
	return nil, nil
}

func (s *captureAlertService) ListAlertGroups(_ context.Context, input ListAlertGroupsInput) ([]AlertGroupSummary, *apperror.Error) {
	s.alertGroupsInput = input
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

func (s *captureAlertService) ListSuppressionRules(context.Context) ([]SuppressionRuleSummary, *apperror.Error) {
	return nil, nil
}

func (s *captureAlertService) CreateSuppressionRule(_ context.Context, input SuppressionRuleInput) (SuppressionRuleSummary, *apperror.Error) {
	s.suppressionInput = input
	return SuppressionRuleSummary{}, nil
}

func (s *captureAlertService) UpdateSuppressionRule(_ context.Context, _ string, input SuppressionRuleInput) (SuppressionRuleSummary, *apperror.Error) {
	s.suppressionInput = input
	return SuppressionRuleSummary{}, nil
}

func (s *captureAlertService) ListRoutingPolicies(context.Context) ([]RoutingPolicySummary, *apperror.Error) {
	return nil, nil
}

func (s *captureAlertService) CreateRoutingPolicy(_ context.Context, input RoutingPolicyInput) (RoutingPolicySummary, *apperror.Error) {
	s.routingInput = input
	return RoutingPolicySummary{}, nil
}

func (s *captureAlertService) UpdateRoutingPolicy(_ context.Context, _ string, input RoutingPolicyInput) (RoutingPolicySummary, *apperror.Error) {
	s.routingInput = input
	return RoutingPolicySummary{}, nil
}

func (s *captureAlertService) ListAlerts(_ context.Context, input ListAlertsInput) ([]AlertSummary, *apperror.Error) {
	s.alertsInput = input
	return nil, nil
}

func (s *captureAlertService) ListAlertHistory(_ context.Context, input AlertHistoryInput) ([]AlertHistoryPoint, *apperror.Error) {
	s.historyInput = input
	return nil, nil
}

func (s *captureAlertService) ListNoisyRules(_ context.Context, input NoisyRuleInput) ([]NoisyRuleSummary, *apperror.Error) {
	s.noisyInput = input
	return nil, nil
}

func (s *captureAlertService) SuppressionDryRun(_ context.Context, input SuppressionDryRunInput) (SuppressionDryRunResult, *apperror.Error) {
	s.suppressionDryRunInput = input
	return SuppressionDryRunResult{}, nil
}

func (s *captureAlertService) RoutingDryRun(_ context.Context, input RoutingDryRunInput) (RoutingDryRunResult, *apperror.Error) {
	s.routingDryRunInput = input
	return RoutingDryRunResult{}, nil
}

func (s *captureAlertService) ListNoiseTrends(_ context.Context, input NoiseTrendInput) ([]NoiseTrendPoint, *apperror.Error) {
	s.noiseTrendInput = input
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
