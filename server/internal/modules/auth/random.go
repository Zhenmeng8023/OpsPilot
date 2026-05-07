package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

const crockfordBase32 = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

func newUID() (string, error) {
	bytes := make([]byte, 26)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	out := make([]byte, 26)
	for i, value := range bytes {
		out[i] = crockfordBase32[int(value)%len(crockfordBase32)]
	}
	return string(out), nil
}

func newTokenFamily() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(bytes[0:4]),
		hex.EncodeToString(bytes[4:6]),
		hex.EncodeToString(bytes[6:8]),
		hex.EncodeToString(bytes[8:10]),
		hex.EncodeToString(bytes[10:16]),
	), nil
}
