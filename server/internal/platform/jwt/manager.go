package jwt

import (
	"errors"
	"time"

	jwtv5 "github.com/golang-jwt/jwt/v5"

	"opspilot/server/internal/config"
)

type Manager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

type Claims struct {
	UserID   string   `json:"userId"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwtv5.RegisteredClaims
}

func NewManager(cfg config.JWTConfig) Manager {
	return Manager{
		accessSecret:  []byte(cfg.AccessSecret),
		refreshSecret: []byte(cfg.RefreshSecret),
		accessTTL:     cfg.AccessTTL,
		refreshTTL:    cfg.RefreshTTL,
	}
}

func (m Manager) GenerateAccessToken(userID, username string, roles []string) (string, error) {
	return m.generate(userID, username, roles, m.accessSecret, m.accessTTL)
}

func (m Manager) GenerateRefreshToken(userID, username string, roles []string) (string, error) {
	return m.generate(userID, username, roles, m.refreshSecret, m.refreshTTL)
}

func (m Manager) ParseAccessToken(token string) (*Claims, error) {
	return m.parse(token, m.accessSecret)
}

func (m Manager) parse(token string, secret []byte) (*Claims, error) {
	claims := &Claims{}
	parsed, err := jwtv5.ParseWithClaims(token, claims, func(token *jwtv5.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwtv5.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	if !parsed.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func (m Manager) generate(userID, username string, roles []string, secret []byte, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwtv5.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwtv5.NewNumericDate(now),
			ExpiresAt: jwtv5.NewNumericDate(now.Add(ttl)),
		},
	}

	return jwtv5.NewWithClaims(jwtv5.SigningMethodHS256, claims).SignedString(secret)
}
