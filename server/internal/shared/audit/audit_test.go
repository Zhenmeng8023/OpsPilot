package audit

import (
	"strings"
	"testing"
)

func TestJSONNullRedactsSensitiveFields(t *testing.T) {
	got := jsonNull(map[string]interface{}{
		"name":          "prod webhook",
		"signingSecret": "secret-value",
		"smtpPassword":  "mail-password",
		"config": map[string]interface{}{
			"accessToken": "access-token",
		},
	})
	if !got.Valid {
		t.Fatal("expected json value")
	}
	for _, leaked := range []string{"secret-value", "mail-password", "access-token"} {
		if strings.Contains(got.String, leaked) {
			t.Fatalf("audit json leaked %q: %s", leaked, got.String)
		}
	}
	if strings.Count(got.String, "***") < 3 {
		t.Fatalf("expected redaction markers, got %s", got.String)
	}
}
