package uid

import "crypto/rand"

const alphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

func New() (string, error) {
	bytes := make([]byte, 26)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	out := make([]byte, 26)
	for i, value := range bytes {
		out[i] = alphabet[int(value)%len(alphabet)]
	}
	return string(out), nil
}
