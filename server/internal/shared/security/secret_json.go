package security

import (
	"encoding/json"
	"strings"
)

type SecretEnvelope struct {
	Encrypted string `json:"_encrypted"`
}

func EncryptJSON(cipher *FieldCipher, value interface{}) (string, error) {
	if value == nil {
		return "", nil
	}
	if envelope, ok := value.(SecretEnvelope); ok {
		return marshalJSON(envelope)
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	encrypted, err := cipher.EncryptString(string(bytes))
	if err != nil {
		return "", err
	}
	return marshalJSON(SecretEnvelope{Encrypted: encrypted})
}

func DecryptJSON(cipher *FieldCipher, raw string, target interface{}) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var envelope SecretEnvelope
	if err := json.Unmarshal([]byte(raw), &envelope); err == nil && strings.TrimSpace(envelope.Encrypted) != "" {
		plaintext, err := cipher.DecryptString(envelope.Encrypted)
		if err != nil {
			return err
		}
		return json.Unmarshal([]byte(plaintext), target)
	}
	return json.Unmarshal([]byte(raw), target)
}

func IsEncryptedJSON(raw string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	var envelope SecretEnvelope
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		return false
	}
	return strings.TrimSpace(envelope.Encrypted) != ""
}

func EncryptJSONMap(cipher *FieldCipher, value map[string]interface{}) (string, error) {
	if len(value) == 0 {
		return "", nil
	}
	return EncryptJSON(cipher, value)
}

func DecryptJSONMap(cipher *FieldCipher, raw string) (map[string]interface{}, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var target map[string]interface{}
	if err := DecryptJSON(cipher, raw, &target); err != nil {
		return nil, err
	}
	return target, nil
}

func MustMarshalJSON(value interface{}) string {
	text, err := marshalJSON(value)
	if err != nil {
		return ""
	}
	return text
}

func marshalJSON(value interface{}) (string, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
