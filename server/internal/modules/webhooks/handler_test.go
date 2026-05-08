package webhooks

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/shared/apperror"
)

func TestListEventsPassesFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureListEventsService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughWebhookAuth, passThroughWebhookPermission)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhook-events?sourceId=src-1&status=rejected&deliveryId=del-1&receivedFrom=2026-05-01T00:00:00&receivedTo=2026-05-08T23:59:59&page=2&pageSize=25", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.input.SourceID != "src-1" || service.input.Status != "rejected" || service.input.DeliveryID != "del-1" {
		t.Fatalf("unexpected filters: %#v", service.input)
	}
	if service.input.ReceivedFrom != "2026-05-01T00:00:00" || service.input.ReceivedTo != "2026-05-08T23:59:59" {
		t.Fatalf("unexpected time filters: %#v", service.input)
	}
	if service.input.Page != 2 || service.input.PageSize != 25 {
		t.Fatalf("unexpected page args: %#v", service.input)
	}
}

func TestGetEventPassesID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureListEventsService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughWebhookAuth, passThroughWebhookPermission)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/webhooks/events/evt-1", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.eventID != "evt-1" {
		t.Fatalf("unexpected event id: %q", service.eventID)
	}
}

func TestSourceStatusActionsPassID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureListEventsService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughWebhookAuth, passThroughWebhookPermission)

	cases := []struct {
		name       string
		method     string
		path       string
		wantAction string
	}{
		{name: "pause", method: http.MethodPost, path: "/api/v1/webhooks/sources/src-1/pause", wantAction: "pause"},
		{name: "resume", method: http.MethodPost, path: "/api/v1/webhooks/sources/src-2/resume", wantAction: "resume"},
		{name: "disable", method: http.MethodPost, path: "/api/v1/webhooks/sources/src-3/disable", wantAction: "disable"},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			service.sourceAction = ""
			service.sourceID = ""

			req := httptest.NewRequest(item.method, item.path, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
			}
			if service.sourceAction != item.wantAction || service.sourceID == "" {
				t.Fatalf("unexpected source action capture: action=%q id=%q", service.sourceAction, service.sourceID)
			}
		})
	}
}

func TestRuleStatusActionsPassID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureListEventsService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughWebhookAuth, passThroughWebhookPermission)

	cases := []struct {
		name       string
		method     string
		path       string
		wantAction string
	}{
		{name: "pause", method: http.MethodPost, path: "/api/v1/webhooks/rules/rule-1/pause", wantAction: "pause"},
		{name: "resume", method: http.MethodPost, path: "/api/v1/webhooks/rules/rule-2/resume", wantAction: "resume"},
		{name: "disable", method: http.MethodPost, path: "/api/v1/webhooks/rules/rule-3/disable", wantAction: "disable"},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			service.ruleAction = ""
			service.ruleID = ""

			req := httptest.NewRequest(item.method, item.path, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusOK {
				t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
			}
			if service.ruleAction != item.wantAction || service.ruleID == "" {
				t.Fatalf("unexpected action capture: action=%q id=%q", service.ruleAction, service.ruleID)
			}
		})
	}
}

func TestUpdateRulePassesIDAndPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureListEventsService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughWebhookAuth, passThroughWebhookPermission)

	req := httptest.NewRequest(http.MethodPut, "/api/v1/webhooks/rules/rule-9", strings.NewReader(`{"name":"deploy main","eventType":"push","matcher":{"conditions":[{"type":"payload_equals","path":"ref","value":"refs/heads/main"}]}}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.updateRule.ID != "rule-9" || service.updateRule.Name != "deploy main" || service.updateRule.EventType != "push" {
		t.Fatalf("unexpected update input: %#v", service.updateRule)
	}
	if service.updateRule.Matcher == nil || len(service.updateRule.Matcher.Conditions) != 1 {
		t.Fatalf("matcher was not passed through: %#v", service.updateRule)
	}
}

type captureListEventsService struct {
	input        ListEventsInput
	eventID      string
	sourceID     string
	sourceAction string
	ruleID       string
	ruleAction   string
	updateRule   UpdateRuleInput
}

func (s *captureListEventsService) ListSources(_ context.Context) ([]SourceSummary, *apperror.Error) {
	return nil, nil
}

func (s *captureListEventsService) CreateSource(_ context.Context, _ CreateSourceInput) (SourceDetail, *apperror.Error) {
	return SourceDetail{}, nil
}

func (s *captureListEventsService) RotateSourceSecrets(_ context.Context, _ RotateSourceSecretsInput) (SourceDetail, *apperror.Error) {
	return SourceDetail{}, nil
}

func (s *captureListEventsService) PauseSource(_ context.Context, id string, _ AuditContext) *apperror.Error {
	s.sourceID = id
	s.sourceAction = "pause"
	return nil
}

func (s *captureListEventsService) ResumeSource(_ context.Context, id string, _ AuditContext) *apperror.Error {
	s.sourceID = id
	s.sourceAction = "resume"
	return nil
}

func (s *captureListEventsService) DisableSource(_ context.Context, id string, _ AuditContext) *apperror.Error {
	s.sourceID = id
	s.sourceAction = "disable"
	return nil
}

func (s *captureListEventsService) ListRules(_ context.Context) ([]RuleSummary, *apperror.Error) {
	return nil, nil
}

func (s *captureListEventsService) CreateRule(_ context.Context, _ CreateRuleInput) (RuleSummary, *apperror.Error) {
	return RuleSummary{}, nil
}

func (s *captureListEventsService) UpdateRule(_ context.Context, input UpdateRuleInput) (RuleSummary, *apperror.Error) {
	s.updateRule = input
	return RuleSummary{}, nil
}

func (s *captureListEventsService) PauseRule(_ context.Context, id string, _ AuditContext) *apperror.Error {
	s.ruleID = id
	s.ruleAction = "pause"
	return nil
}

func (s *captureListEventsService) ResumeRule(_ context.Context, id string, _ AuditContext) *apperror.Error {
	s.ruleID = id
	s.ruleAction = "resume"
	return nil
}

func (s *captureListEventsService) DisableRule(_ context.Context, id string, _ AuditContext) *apperror.Error {
	s.ruleID = id
	s.ruleAction = "disable"
	return nil
}

func (s *captureListEventsService) ListEvents(_ context.Context, input ListEventsInput) (EventListResult, *apperror.Error) {
	s.input = input
	return EventListResult{}, nil
}

func (s *captureListEventsService) GetEvent(_ context.Context, input EventDetailInput) (EventDetail, *apperror.Error) {
	s.eventID = input.EventID
	return EventDetail{}, nil
}

func (s *captureListEventsService) SimulateMatcher(context.Context, MatcherSimulationInput) (MatcherSimulationResult, *apperror.Error) {
	return MatcherSimulationResult{}, nil
}

func (s *captureListEventsService) ReplayEvent(context.Context, ReplayEventInput) (TriggerResult, *apperror.Error) {
	return TriggerResult{}, nil
}

func (s *captureListEventsService) Trigger(_ context.Context, _ TriggerInput) (TriggerResult, *apperror.Error) {
	return TriggerResult{}, nil
}
