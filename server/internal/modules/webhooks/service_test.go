package webhooks

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestValidSignature(t *testing.T) {
	body := []byte(`{"event":"deploy"}`)
	token := "test-token"
	mac := hmac.New(sha256.New, []byte(token))
	_, _ = mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !validSignature(token, body, signature) {
		t.Fatalf("validSignature() rejected a correct signature")
	}
	if validSignature(token, body, "sha256=bad") {
		t.Fatalf("validSignature() accepted an incorrect signature")
	}
	if validSignature(token, body, "") {
		t.Fatalf("validSignature() accepted an empty signature")
	}
}
