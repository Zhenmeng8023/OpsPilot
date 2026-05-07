package auth

import (
	"context"
	"net/http"
	"testing"

	"opspilot/server/internal/config"
)

func TestRegisterForbiddenWhenPublicRegistrationDisabledInProd(t *testing.T) {
	service := &Service{
		cfg: config.Config{
			App: config.AppConfig{
				Env: "prod",
			},
			Auth: config.AuthConfig{
				PublicRegistrationEnabled: false,
			},
		},
	}

	_, appErr := service.Register(context.Background(), RegisterInput{
		Username: "member",
		Password: "Member@123456",
	})
	if appErr == nil {
		t.Fatal("expected registration to be forbidden in prod")
	}
	if appErr.HTTPStatus != http.StatusForbidden || appErr.Code != 403003 {
		t.Fatalf("unexpected error: %+v", appErr)
	}
}
