package webhooks

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/shared/apperror"
)

func TestValidSignature(t *testing.T) {
	body := []byte(`{"event":"deploy"}`)
	secret := "test-signing-secret"
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !validSignature(secret, body, signature) {
		t.Fatalf("validSignature() rejected a correct signature")
	}
	if validSignature(secret, body, "sha256=bad") {
		t.Fatalf("validSignature() accepted an incorrect signature")
	}
	if validSignature(secret, body, "") {
		t.Fatalf("validSignature() accepted an empty signature")
	}
}

func TestParseWebhookTimestamp(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	unixValue := now.Format(time.RFC3339)

	parsedRFC3339, err := parseWebhookTimestamp(unixValue)
	if err != nil {
		t.Fatalf("parse RFC3339 timestamp: %v", err)
	}
	if !parsedRFC3339.Equal(now) {
		t.Fatalf("expected %s, got %s", now, parsedRFC3339)
	}

	parsedUnix, err := parseWebhookTimestamp("1735689600")
	if err != nil {
		t.Fatalf("parse unix timestamp: %v", err)
	}
	if parsedUnix.Unix() != 1735689600 {
		t.Fatalf("unexpected unix timestamp: %s", parsedUnix)
	}

	if _, err := parseWebhookTimestamp("bad-value"); err == nil {
		t.Fatal("expected invalid timestamp to fail")
	}
}

func TestTriggerHandlerPassesSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureWebhookService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughWebhookAuth, passThroughWebhookPermission)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/trigger/source-token", bytes.NewBufferString(`{"event":"deploy"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-OpsPilot-Signature", "sha256=test")
	req.Header.Set("X-OpsPilot-Timestamp", "1735689600")
	req.Header.Set("X-OpsPilot-Nonce", "nonce-1")
	req.Header.Set("X-Delivery-Id", "delivery-1")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.input.Token != "source-token" || service.input.Timestamp != "1735689600" || service.input.Nonce != "nonce-1" {
		t.Fatalf("unexpected trigger input: %#v", service.input)
	}
	if service.input.SignatureHeader != "X-OpsPilot-Signature" || service.input.DeliveryID != "delivery-1" {
		t.Fatalf("unexpected trigger headers: %#v", service.input)
	}
}

func TestMatchRule(t *testing.T) {
	payload := parsePayloadObject([]byte(`{"repository":{"name":"OpsPilot"},"ref":"refs/heads/main"}`))
	headers := map[string]string{
		"X-GitHub-Event": "push",
	}

	cases := []struct {
		name       string
		matcher    *Matcher
		wantOK     bool
		wantReason string
	}{
		{
			name: "header equals",
			matcher: &Matcher{Conditions: []MatcherCondition{
				{Type: "header_equals", Key: "X-GitHub-Event", Value: "push"},
			}},
			wantOK: true,
		},
		{
			name: "payload equals",
			matcher: &Matcher{Conditions: []MatcherCondition{
				{Type: "payload_equals", Path: "ref", Value: "refs/heads/main"},
			}},
			wantOK: true,
		},
		{
			name: "payload contains",
			matcher: &Matcher{Conditions: []MatcherCondition{
				{Type: "payload_contains", Path: "repository.name", Value: "Pilot"},
			}},
			wantOK: true,
		},
		{
			name: "event type equals",
			matcher: &Matcher{Conditions: []MatcherCondition{
				{Type: "event_type_equals", Value: "push"},
			}},
			wantOK: true,
		},
		{
			name: "ref equals",
			matcher: &Matcher{Conditions: []MatcherCondition{
				{Type: "ref_equals", Value: "refs/heads/main"},
			}},
			wantOK: true,
		},
		{
			name: "branch equals",
			matcher: &Matcher{Conditions: []MatcherCondition{
				{Type: "branch_equals", Value: "main"},
			}},
			wantOK: true,
		},
		{
			name: "header mismatch",
			matcher: &Matcher{Conditions: []MatcherCondition{
				{Type: "header_equals", Key: "X-GitHub-Event", Value: "release"},
			}},
			wantReason: "header_mismatch:X-GitHub-Event",
		},
		{
			name: "payload mismatch",
			matcher: &Matcher{Conditions: []MatcherCondition{
				{Type: "payload_equals", Path: "repository.name", Value: "Other"},
			}},
			wantReason: "payload_mismatch:repository.name",
		},
		{
			name: "event type mismatch",
			matcher: &Matcher{Conditions: []MatcherCondition{
				{Type: "event_type_equals", Value: "release"},
			}},
			wantReason: "event_type_condition_mismatch",
		},
		{
			name: "ref mismatch",
			matcher: &Matcher{Conditions: []MatcherCondition{
				{Type: "ref_equals", Value: "refs/heads/release"},
			}},
			wantReason: "ref_mismatch",
		},
		{
			name: "branch mismatch",
			matcher: &Matcher{Conditions: []MatcherCondition{
				{Type: "branch_equals", Value: "release"},
			}},
			wantReason: "branch_mismatch",
		},
	}

	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			ok, reason := matchRule(item.matcher, "push", headers, payload)
			if ok != item.wantOK {
				t.Fatalf("expected ok=%v, got %v (%s)", item.wantOK, ok, reason)
			}
			if reason != item.wantReason {
				t.Fatalf("expected reason %q, got %q", item.wantReason, reason)
			}
		})
	}
}

func TestMatchRuleMultipleConditions(t *testing.T) {
	payload := parsePayloadObject([]byte(`{"repository":{"name":"OpsPilot"},"ref":"refs/heads/main"}`))
	headers := map[string]string{
		"X-GitHub-Event": "push",
	}

	ok, reason := matchRule(&Matcher{Conditions: []MatcherCondition{
		{Type: "header_equals", Key: "X-GitHub-Event", Value: "push"},
		{Type: "event_type_equals", Value: "push"},
		{Type: "payload_equals", Path: "ref", Value: "refs/heads/main"},
		{Type: "payload_contains", Path: "repository.name", Value: "Pilot"},
		{Type: "ref_equals", Value: "refs/heads/main"},
		{Type: "branch_equals", Value: "main"},
	}}, "push", headers, payload)
	if !ok || reason != "" {
		t.Fatalf("expected all matcher conditions to pass, got ok=%v reason=%q", ok, reason)
	}

	ok, reason = matchRule(&Matcher{Conditions: []MatcherCondition{
		{Type: "header_equals", Key: "X-GitHub-Event", Value: "push"},
		{Type: "payload_equals", Path: "ref", Value: "refs/heads/release"},
	}}, "push", headers, payload)
	if ok {
		t.Fatal("expected matcher to fail when one condition fails")
	}
	if reason != "payload_mismatch:ref" {
		t.Fatalf("unexpected mismatch reason: %q", reason)
	}
}

func TestNormalizeMatcherRejectsInvalidCondition(t *testing.T) {
	_, appErr := normalizeMatcher(&Matcher{Conditions: []MatcherCondition{{Type: "payload_equals", Value: "main"}}})
	if appErr == nil {
		t.Fatal("expected invalid matcher to fail")
	}
	if appErr.Code != 400505 {
		t.Fatalf("unexpected app error: %+v", appErr)
	}
}

func TestNormalizeMatcherSupportsEventAndRefConditions(t *testing.T) {
	matcher, appErr := normalizeMatcher(&Matcher{Conditions: []MatcherCondition{
		{Type: "event_type_equals", Key: "ignored", Path: "ignored", Value: "push"},
		{Type: "ref_equals", Key: "ignored", Path: "ignored", Value: "refs/heads/main"},
		{Type: "branch_equals", Key: "ignored", Path: "ignored", Value: "main"},
	}})
	if appErr != nil {
		t.Fatalf("normalizeMatcher returned error: %+v", appErr)
	}
	if len(matcher.Conditions) != 3 {
		t.Fatalf("expected 3 conditions, got %d", len(matcher.Conditions))
	}
	for _, condition := range matcher.Conditions {
		if condition.Key != "" || condition.Path != "" {
			t.Fatalf("expected key/path to be cleared, got %+v", condition)
		}
	}
}

type captureWebhookService struct {
	input TriggerInput
}

func (s *captureWebhookService) ListSources(context.Context) ([]SourceSummary, *apperror.Error) {
	return nil, nil
}

func (s *captureWebhookService) CreateSource(context.Context, CreateSourceInput) (SourceDetail, *apperror.Error) {
	return SourceDetail{}, nil
}

func (s *captureWebhookService) PauseSource(context.Context, string, AuditContext) *apperror.Error {
	return nil
}

func (s *captureWebhookService) ResumeSource(context.Context, string, AuditContext) *apperror.Error {
	return nil
}

func (s *captureWebhookService) DisableSource(context.Context, string, AuditContext) *apperror.Error {
	return nil
}

func (s *captureWebhookService) ListRules(context.Context) ([]RuleSummary, *apperror.Error) {
	return nil, nil
}

func (s *captureWebhookService) CreateRule(context.Context, CreateRuleInput) (RuleSummary, *apperror.Error) {
	return RuleSummary{}, nil
}

func (s *captureWebhookService) UpdateRule(context.Context, UpdateRuleInput) (RuleSummary, *apperror.Error) {
	return RuleSummary{}, nil
}

func (s *captureWebhookService) PauseRule(context.Context, string, AuditContext) *apperror.Error {
	return nil
}

func (s *captureWebhookService) ResumeRule(context.Context, string, AuditContext) *apperror.Error {
	return nil
}

func (s *captureWebhookService) DisableRule(context.Context, string, AuditContext) *apperror.Error {
	return nil
}

func (s *captureWebhookService) ListEvents(context.Context, ListEventsInput) (EventListResult, *apperror.Error) {
	return EventListResult{}, nil
}

func (s *captureWebhookService) GetEvent(context.Context, EventDetailInput) (EventDetail, *apperror.Error) {
	return EventDetail{}, nil
}

func (s *captureWebhookService) Trigger(_ context.Context, input TriggerInput) (TriggerResult, *apperror.Error) {
	s.input = input
	return TriggerResult{EventID: "evt-1", Status: "received"}, nil
}

func passThroughWebhookAuth(c *gin.Context) {
	c.Next()
}

func passThroughWebhookPermission(string) gin.HandlerFunc {
	return passThroughWebhookAuth
}
