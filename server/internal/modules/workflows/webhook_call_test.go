package workflows

import (
	"strings"
	"testing"
)

func TestBuildWebhookCallPlanRendersTemplates(t *testing.T) {
	node := Node{
		ID:             "callback",
		Type:           "webhook-call",
		TimeoutSeconds: 12,
		Config: map[string]interface{}{
			"url":         "https://example.com/hooks/${payload.service}",
			"method":      "post",
			"contentType": "application/json",
			"headers": map[string]interface{}{
				"X-Env": "${payload.environment}",
			},
			"bodyPath": "payload",
		},
	}
	input := map[string]interface{}{
		"payload": map[string]interface{}{
			"service":     "api",
			"environment": "prod",
		},
	}

	plan, err := buildWebhookCallPlan(node, input)
	if err != nil {
		t.Fatalf("buildWebhookCallPlan returned error: %v", err)
	}
	if plan.Method != "POST" || plan.URL != "https://example.com/hooks/api" {
		t.Fatalf("unexpected plan: %#v", plan)
	}
	if plan.Headers["X-Env"] != "prod" {
		t.Fatalf("expected rendered header, got %#v", plan.Headers)
	}
	if !strings.Contains(string(plan.Body), `"service":"api"`) {
		t.Fatalf("expected marshaled payload body, got %s", string(plan.Body))
	}
}

func TestWebhookCallSucceededHonorsExpectedStatus(t *testing.T) {
	node := Node{
		ID:   "callback",
		Type: "webhook-call",
		Config: map[string]interface{}{
			"expectedStatus": 202,
		},
	}
	if !webhookCallSucceeded(node, 202) {
		t.Fatal("expected configured status to succeed")
	}
	if webhookCallSucceeded(node, 200) {
		t.Fatal("expected unexpected status to fail")
	}
}
