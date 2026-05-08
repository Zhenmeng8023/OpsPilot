package notifications

import (
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"opspilot/server/internal/config"
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
			if got := deliveryURL(configMapFromRaw(tt.raw)); got != tt.want {
				t.Fatalf("deliveryURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDeliveryEmail(t *testing.T) {
	tests := []struct {
		name string
		raw  sql.NullString
		want string
	}{
		{"email", sql.NullString{String: `{"email":"ops@example.com"}`, Valid: true}, "ops@example.com"},
		{"to", sql.NullString{String: `{"to":"to@example.com"}`, Valid: true}, "to@example.com"},
		{"invalid json", sql.NullString{String: `{`, Valid: true}, ""},
		{"null", sql.NullString{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deliveryEmail(configMapFromRaw(tt.raw)); got != tt.want {
				t.Fatalf("deliveryEmail() = %q, want %q", got, tt.want)
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

func TestBuildEmailMessage(t *testing.T) {
	row := pendingDelivery{
		Title:   "CPU high",
		Content: sql.NullString{String: "host01 value 90", Valid: true},
	}
	message := buildEmailMessage("from@example.com", "to@example.com", row)
	if !strings.Contains(message, "Subject:") || !strings.Contains(message, "host01 value 90") {
		t.Fatalf("unexpected email message: %q", message)
	}
}

func TestDeliverySigningSecret(t *testing.T) {
	tests := []struct {
		name string
		raw  sql.NullString
		want string
	}{
		{"camel", sql.NullString{String: `{"signingSecret":"secret-1"}`, Valid: true}, "secret-1"},
		{"snake", sql.NullString{String: `{"signing_secret":"secret-2"}`, Valid: true}, "secret-2"},
		{"plain", sql.NullString{String: `{"secret":"secret-3"}`, Valid: true}, "secret-3"},
		{"missing", sql.NullString{String: `{"url":"https://example.com"}`, Valid: true}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := deliverySigningSecret(configMapFromRaw(tt.raw)); got != tt.want {
				t.Fatalf("deliverySigningSecret() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestBuildDeliverySignature(t *testing.T) {
	got := buildDeliverySignature("secret", "1735689600", []byte(`{"ok":true}`))
	if got != "sha256=b8e38a3d9c4cdbbb56bfae9dbe0af50812c06558d576166bfb7e8f2317e3b96f" {
		t.Fatalf("buildDeliverySignature() = %q", got)
	}
}

func TestPostDeliveryAddsSigningHeaders(t *testing.T) {
	var gotDeliveryID, gotNotificationID, gotTimestamp, gotSignature string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		gotDeliveryID = r.Header.Get("X-OpsPilot-Delivery-Id")
		gotNotificationID = r.Header.Get("X-OpsPilot-Notification-Id")
		gotTimestamp = r.Header.Get("X-OpsPilot-Timestamp")
		gotSignature = r.Header.Get("X-OpsPilot-Signature")
		if gotTimestamp == "" || gotSignature == "" {
			t.Fatalf("expected signing headers to be present")
		}
		expected := buildDeliverySignature("secret-1", gotTimestamp, body)
		if gotSignature != expected {
			t.Fatalf("signature mismatch: got %q want %q", gotSignature, expected)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	service := &Service{
		cfg: config.Config{
			Notify: config.NotificationConfig{HTTPTimeout: time.Second},
		},
	}
	row := pendingDelivery{
		ID:               42,
		ChannelType:      "webhook",
		Config:           sql.NullString{String: `{"url":"` + server.URL + `","signingSecret":"secret-1"}`, Valid: true},
		NotificationUID:  "ntf-1",
		Title:            "Test",
		Content:          sql.NullString{String: "payload", Valid: true},
		Category:         "system",
		Severity:         "info",
		NotificationTime: "2026-05-07 12:00:00",
	}
	if err := service.postDelivery(t.Context(), row); err != nil {
		t.Fatalf("postDelivery() error = %v", err)
	}
	if gotDeliveryID != "42" || gotNotificationID != "ntf-1" {
		t.Fatalf("unexpected delivery headers: delivery=%q notification=%q", gotDeliveryID, gotNotificationID)
	}
}
