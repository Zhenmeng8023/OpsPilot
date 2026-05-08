package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

const (
	defaultEncryptionVersion = "v1"
	encryptedPrefix          = "enc:"
)

type FieldCipher struct {
	key []byte
}

func NewFieldCipher(secret string) (*FieldCipher, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, errors.New("secret encryption key is required")
	}
	sum := sha256.Sum256([]byte(secret))
	return &FieldCipher{key: sum[:]}, nil
}

func (c *FieldCipher) EncryptString(plaintext string) (string, error) {
	if c == nil {
		return "", errors.New("field cipher is not configured")
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	payload := append(nonce, ciphertext...)
	return encryptedPrefix + defaultEncryptionVersion + ":" + base64.StdEncoding.EncodeToString(payload), nil
}

func (c *FieldCipher) DecryptString(ciphertext string) (string, error) {
	if c == nil {
		return "", errors.New("field cipher is not configured")
	}
	ciphertext = strings.TrimSpace(ciphertext)
	if ciphertext == "" {
		return "", nil
	}
	if !IsEncrypted(ciphertext) {
		return ciphertext, nil
	}
	version, payload, ok := strings.Cut(strings.TrimPrefix(ciphertext, encryptedPrefix), ":")
	if !ok || strings.TrimSpace(version) != defaultEncryptionVersion {
		return "", errors.New("unsupported encrypted value version")
	}
	decoded, err := base64.StdEncoding.DecodeString(payload)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(c.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(decoded) < gcm.NonceSize() {
		return "", errors.New("encrypted value is malformed")
	}
	nonce := decoded[:gcm.NonceSize()]
	sealed := decoded[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, sealed, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func IsEncrypted(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), encryptedPrefix)
}
