package notifications

import (
	"database/sql"
	"encoding/json"
	"strings"

	"opspilot/server/internal/shared/security"
)

func (s *Service) encryptChannelConfig(value map[string]interface{}) (sql.NullString, error) {
	if len(value) == 0 {
		return sql.NullString{}, nil
	}
	cipher, err := security.NewFieldCipher(s.cfg.Security.SecretEncryptionKey)
	if err != nil {
		return sql.NullString{}, err
	}
	encrypted, err := security.EncryptJSONMap(cipher, value)
	if err != nil {
		return sql.NullString{}, err
	}
	return sql.NullString{String: encrypted, Valid: encrypted != ""}, nil
}

func (s *Service) channelConfigMap(raw sql.NullString) map[string]interface{} {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil
	}
	cipher, err := security.NewFieldCipher(s.cfg.Security.SecretEncryptionKey)
	if err != nil {
		return configMapFromRaw(raw)
	}
	value, err := security.DecryptJSONMap(cipher, raw.String)
	if err != nil {
		return configMapFromRaw(raw)
	}
	return value
}

func configMapFromRaw(raw sql.NullString) map[string]interface{} {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return nil
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal([]byte(raw.String), &cfg); err != nil {
		return nil
	}
	return cfg
}

func mergeChannelConfig(existing, patch map[string]interface{}, channelType string) map[string]interface{} {
	if channelType == "site" {
		return nil
	}
	merged := map[string]interface{}{}
	for key, value := range existing {
		merged[key] = value
	}
	for key, value := range patch {
		if strings.TrimSpace(key) == "" || value == nil {
			continue
		}
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) == "" {
				continue
			}
			merged[key] = typed
		default:
			merged[key] = typed
		}
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}
