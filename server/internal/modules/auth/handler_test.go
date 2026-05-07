package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/config"
	jwtplatform "opspilot/server/internal/platform/jwt"
	"opspilot/server/internal/shared/apperror"
)

func TestLoginSuccessAndMemberUserManagementForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := jwtplatform.NewManager(config.JWTConfig{
		AccessSecret:  "test-access-secret",
		RefreshSecret: "test-refresh-secret",
		AccessTTL:     time.Hour,
		RefreshTTL:    24 * time.Hour,
	})
	service := &fakeAuthService{jwtManager: manager}
	handler := NewHandler(service)

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))

	loginBody := bytes.NewBufferString(`{"username":"member","password":"Member@123456"}`)
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", loginBody)
	loginReq.Header.Set("Content-Type", "application/json")
	loginResp := httptest.NewRecorder()
	router.ServeHTTP(loginResp, loginReq)

	if loginResp.Code != http.StatusOK {
		t.Fatalf("expected login 200, got %d: %s", loginResp.Code, loginResp.Body.String())
	}

	var loginEnvelope struct {
		Code int `json:"code"`
		Data struct {
			AccessToken string `json:"accessToken"`
		} `json:"data"`
	}
	if err := json.Unmarshal(loginResp.Body.Bytes(), &loginEnvelope); err != nil {
		t.Fatalf("decode login response: %v", err)
	}
	if loginEnvelope.Code != 0 || loginEnvelope.Data.AccessToken == "" {
		t.Fatalf("unexpected login response: %+v", loginEnvelope)
	}

	usersReq := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	usersReq.Header.Set("Authorization", "Bearer "+loginEnvelope.Data.AccessToken)
	usersResp := httptest.NewRecorder()
	router.ServeHTTP(usersResp, usersReq)

	if usersResp.Code != http.StatusForbidden {
		t.Fatalf("expected users 403, got %d: %s", usersResp.Code, usersResp.Body.String())
	}
	if service.listUsersCalled {
		t.Fatal("user management handler should not run when permission is denied")
	}
}

func TestAuthMiddlewareRejectsOpaqueDomainTokens(t *testing.T) {
	gin.SetMode(gin.TestMode)

	manager := jwtplatform.NewManager(config.JWTConfig{
		AccessSecret:  "test-access-secret",
		RefreshSecret: "test-refresh-secret",
		AccessTTL:     time.Hour,
		RefreshTTL:    24 * time.Hour,
	})
	handler := NewHandler(&fakeAuthService{jwtManager: manager})

	router := gin.New()
	handler.RegisterRoutes(router.Group("/api/v1"))

	for name, token := range map[string]string{
		"agent token":   "opagt_test_agent_token",
		"webhook token": "opswhk_test_webhook_token",
	} {
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/me", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d: %s", resp.Code, resp.Body.String())
			}
		})
	}
}

type fakeAuthService struct {
	jwtManager      jwtplatform.Manager
	listUsersCalled bool
}

func (f *fakeAuthService) JWTManager() jwtplatform.Manager {
	return f.jwtManager
}

func (f *fakeAuthService) Register(context.Context, RegisterInput) (AuthResult, *apperror.Error) {
	return AuthResult{}, nil
}

func (f *fakeAuthService) Login(context.Context, LoginInput) (AuthResult, *apperror.Error) {
	accessToken, err := f.jwtManager.GenerateAccessToken("user-member", "member", []string{"member"})
	if err != nil {
		return AuthResult{}, apperror.Wrap(http.StatusInternalServerError, 500001, "issue token failed", err)
	}
	refreshToken, err := f.jwtManager.GenerateRefreshToken("user-member", "member", []string{"member"})
	if err != nil {
		return AuthResult{}, apperror.Wrap(http.StatusInternalServerError, 500001, "issue token failed", err)
	}
	return AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int64(time.Hour.Seconds()),
		User: UserProfile{
			ID:       "user-member",
			Username: "member",
			Roles:    []string{"member"},
		},
	}, nil
}

func (f *fakeAuthService) Refresh(context.Context, RefreshInput) (AuthResult, *apperror.Error) {
	return AuthResult{}, nil
}

func (f *fakeAuthService) Logout(context.Context, LogoutInput) *apperror.Error {
	return nil
}

func (f *fakeAuthService) Me(context.Context, string) (UserProfile, *apperror.Error) {
	return UserProfile{}, nil
}

func (f *fakeAuthService) ListUsers(context.Context, string, string) ([]UserSummary, *apperror.Error) {
	f.listUsersCalled = true
	return []UserSummary{}, nil
}

func (f *fakeAuthService) CreateUser(context.Context, CreateUserInput) (UserSummary, *apperror.Error) {
	return UserSummary{}, nil
}

func (f *fakeAuthService) UpdateUserStatus(context.Context, UpdateUserStatusInput) *apperror.Error {
	return nil
}

func (f *fakeAuthService) UpdateUserRoles(context.Context, UpdateUserRolesInput) (UserSummary, *apperror.Error) {
	return UserSummary{}, nil
}

func (f *fakeAuthService) ListPermissions(context.Context) ([]PermissionSummary, *apperror.Error) {
	return []PermissionSummary{}, nil
}

func (f *fakeAuthService) ListRoles(context.Context) ([]RoleSummary, *apperror.Error) {
	return []RoleSummary{}, nil
}

func (f *fakeAuthService) CreateRole(context.Context, CreateRoleInput) (RoleSummary, *apperror.Error) {
	return RoleSummary{}, nil
}

func (f *fakeAuthService) UpdateRolePermissions(context.Context, UpdateRolePermissionsInput) (RoleSummary, *apperror.Error) {
	return RoleSummary{}, nil
}

func (f *fakeAuthService) HasPermission(_ context.Context, _ string, permissionCode string) (bool, error) {
	return permissionCode != "user.read", nil
}
