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
			if got := maskChannelTarget(tt.channelType, tt.raw); got != tt.want {
				t.Fatalf("maskChannelTarget() = %q, want %q", got, tt.want)
			}
		})
	}
}
