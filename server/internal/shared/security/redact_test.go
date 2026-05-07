package security

import (
	"strings"
	"testing"
)

func TestRedactSensitiveValues(t *testing.T) {
	input := `Authorization: Bearer opagt_secret token=abc password=hunter2 {"secret":"value"}`
	out := Redact(input)
	for _, leaked := range []string{"opagt_secret", "abc", "hunter2", `"value"`} {
		if strings.Contains(out, leaked) {
			t.Fatalf("redacted output leaked %q: %s", leaked, out)
		}
	}
	if !strings.Contains(out, "***") {
		t.Fatalf("expected redaction marker, got %s", out)
	}
}

func TestRedactCamelCaseSensitiveKeys(t *testing.T) {
	input := `{"signingSecret":"sig-value","smtpPassword":"mail-pass","accessToken":"jwt-value","authorization":"Bearer raw"} signingSecret=plain-secret`
	out := Redact(input)
	for _, leaked := range []string{"sig-value", "mail-pass", "jwt-value", "Bearer raw", "plain-secret"} {
		if strings.Contains(out, leaked) {
			t.Fatalf("redacted output leaked %q: %s", leaked, out)
		}
	}
	if strings.Count(out, "***") < 5 {
		t.Fatalf("expected redaction markers, got %s", out)
	}
}
