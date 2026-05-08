package notifications

import (
	"database/sql"
	"testing"
)

func TestMaskEmailAddress(t *testing.T) {
	got := maskEmailAddress("ops-alerts@example.com")
	if got != "op******ts@example.com" {
		t.Fatalf("maskEmailAddress() = %q", got)
	}
}

func TestMaskWebhookTarget(t *testing.T) {
	got := maskWebhookTarget("https://hooks.example.com/notify/token?id=1")
	if got != "https://hooks.example.com/..." {
		t.Fatalf("maskWebhookTarget() = %q", got)
	}
}

func TestMaskChannelTarget(t *testing.T) {
	tests := []struct {
		name        string
		channelType string
		raw         sql.NullString
		want        string
	}{
		{
			name:        "site",
			channelType: "site",
			want:        "in-app",
		},
		{
			name:        "email",
			channelType: "email",
			raw:         sql.NullString{String: `{"email":"ops@example.com"}`, Valid: true},
			want:        "o*s@example.com",
		},
		{
			name:        "webhook",
			channelType: "webhook",
			raw:         sql.NullString{String: `{"url":"https://hooks.example.com/a/b"}`, Valid: true},
			want:        "https://hooks.example.com/...",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maskChannelTarget(tt.channelType, configMapFromRaw(tt.raw)); got != tt.want {
				t.Fatalf("maskChannelTarget() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRenderNotificationTemplateText(t *testing.T) {
	values := notificationTemplateValues("CPU high", "node-a is above threshold", "alert", "email", "critical", "alert", 42)
	got := renderNotificationTemplateText("[{{severity}}] {{title}} #{{resourceId}}", values)
	if got != "[critical] CPU high #42" {
		t.Fatalf("renderNotificationTemplateText() = %q", got)
	}
}

func TestNormalizeTemplateInput(t *testing.T) {
	got, appErr := normalizeTemplateInput("Alert", "alert", "", "{{title}}", "{{content}}", "")
	if appErr != nil {
		t.Fatalf("expected template input to be valid: %v", appErr)
	}
	if got.ChannelType != "any" || got.Status != "active" {
		t.Fatalf("unexpected defaults: %#v", got)
	}
}
