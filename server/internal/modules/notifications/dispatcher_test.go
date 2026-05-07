package notifications

import (
	"database/sql"
	"testing"
)

func TestDeliveryURL(t *testing.T) {
	tests := []struct {
		name string
		raw  sql.NullString
		want string
	}{
		{"url", sql.NullString{String: `{"url":"https://example.com/hook"}`, Valid: true}, "https://example.com/hook"},
		{"webhookUrl", sql.NullString{String: `{"webhookUrl":"https://example.com/webhook"}`, Valid: true}, "https://example.com/webhook"},
		{"invalid json", sql.NullString{String: `{`, Valid: true}, ""},
		{"null", sql.NullString{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deliveryURL(tt.raw); got != tt.want {
				t.Fatalf("deliveryURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestChannelPayload(t *testing.T) {
	row := pendingDelivery{
		NotificationUID: "n1",
		Title:           "CPU high",
		Content:         sql.NullString{String: "host01 value 90", Valid: true},
		Category:        "alert",
		Severity:        "critical",
	}
	row.ChannelType = "slack"
	if payload := channelPayload(row).(map[string]interface{}); payload["text"] == "" {
		t.Fatalf("slack payload text is empty")
	}
	row.ChannelType = "dingtalk"
	if payload := channelPayload(row).(map[string]interface{}); payload["msgtype"] != "text" {
		t.Fatalf("dingtalk payload should use text message")
	}
	row.ChannelType = "webhook"
	if payload := channelPayload(row).(map[string]interface{}); payload["id"] != "n1" {
		t.Fatalf("webhook payload id mismatch")
	}
}
