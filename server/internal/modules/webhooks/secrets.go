package webhooks

import (
	"strings"

	"opspilot/server/internal/shared/security"
)

func (s *Service) encryptSigningSecret(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	cipher, err := security.NewFieldCipher(s.cfg.Security.SecretEncryptionKey)
	if err != nil {
		return "", err
	}
	return cipher.EncryptString(value)
}

func (s *Service) decryptSigningSecret(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	cipher, err := security.NewFieldCipher(s.cfg.Security.SecretEncryptionKey)
	if err != nil {
		return value
	}
	plaintext, err := cipher.DecryptString(value)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(plaintext)
}
