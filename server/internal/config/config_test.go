package config

import "testing"

func TestLoadUsesBuildInfoWhenVersionEnvIsBlank(t *testing.T) {
	oldVersion := BuildVersion
	oldCommit := BuildCommit
	oldBuildTime := BuildTime
	t.Cleanup(func() {
		BuildVersion = oldVersion
		BuildCommit = oldCommit
		BuildTime = oldBuildTime
	})

	BuildVersion = "0.7.0-build"
	BuildCommit = "abc123"
	BuildTime = "2026-05-07T00:00:00Z"
	t.Setenv("APP_ENV", "local")
	t.Setenv("APP_VERSION", "")
	t.Setenv("APP_COMMIT", "")
	t.Setenv("APP_BUILD_TIME", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.App.Version != "0.7.0-build" || cfg.App.Commit != "abc123" || cfg.App.BuildTime != "2026-05-07T00:00:00Z" {
		t.Fatalf("unexpected build info: %+v", cfg.App)
	}
}

func TestLoadVersionEnvOverridesBuildInfo(t *testing.T) {
	oldVersion := BuildVersion
	t.Cleanup(func() {
		BuildVersion = oldVersion
	})

	BuildVersion = "0.7.0-build"
	t.Setenv("APP_ENV", "local")
	t.Setenv("APP_VERSION", "0.7.0-env")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.App.Version != "0.7.0-env" {
		t.Fatalf("expected env version override, got %q", cfg.App.Version)
	}
}

func TestLoadProdDisablesPublicRegistrationByDefault(t *testing.T) {
	setProdBaselineEnv(t)
	t.Setenv("AUTH_PUBLIC_REGISTRATION_ENABLED", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Auth.PublicRegistrationEnabled {
		t.Fatal("expected public registration to be disabled by default in prod")
	}
}

func TestLoadProdAllowsExplicitPublicRegistration(t *testing.T) {
	setProdBaselineEnv(t)
	t.Setenv("AUTH_PUBLIC_REGISTRATION_ENABLED", "true")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if !cfg.Auth.PublicRegistrationEnabled {
		t.Fatal("expected explicit public registration override in prod")
	}
}

func TestLoadRejectsDefaultJWTSecretInProd(t *testing.T) {
	setProdBaselineEnv(t)
	t.Setenv("JWT_ACCESS_SECRET", defaultAccessSecret)

	if _, err := Load(); err == nil {
		t.Fatal("expected default JWT secret to be rejected in prod")
	}
}

func TestLoadRejectsShortJWTSecretInProd(t *testing.T) {
	setProdBaselineEnv(t)
	t.Setenv("JWT_ACCESS_SECRET", "short-secret")

	if _, err := Load(); err == nil {
		t.Fatal("expected short JWT secret to be rejected in prod")
	}
}

func TestLoadRejectsLocalAllowOriginInProd(t *testing.T) {
	setProdBaselineEnv(t)
	t.Setenv("HTTP_ALLOW_ORIGIN", "http://localhost:5173")

	if _, err := Load(); err == nil {
		t.Fatal("expected local HTTP_ALLOW_ORIGIN to be rejected in prod")
	}
}

func TestLoadRejectsDatabaseDSNWithoutParseTimeInProd(t *testing.T) {
	setProdBaselineEnv(t)
	t.Setenv("DATABASE_DSN", "opspilot:opspilot@tcp(127.0.0.1:3306)/opspilot")

	if _, err := Load(); err == nil {
		t.Fatal("expected DATABASE_DSN without parseTime=True to be rejected in prod")
	}
}

func setProdBaselineEnv(t *testing.T) {
	t.Setenv("APP_ENV", "prod")
	t.Setenv("JWT_ACCESS_SECRET", "12345678901234567890123456789012")
	t.Setenv("JWT_REFRESH_SECRET", "abcdefghijklmnopqrstuvwxyz123456")
	t.Setenv("SECRET_ENCRYPTION_KEY", "prod-secret-encryption-key-123456")
	t.Setenv("AGENT_BOOTSTRAP_SECRET", "prod-agent-bootstrap-secret-123456")
	t.Setenv("BOOTSTRAP_ADMIN_PASSWORD", "prod-admin-password-1234567890-secure")
	t.Setenv("HTTP_ALLOW_ORIGIN", "https://ops.example.com")
	t.Setenv("DATABASE_DSN", "opspilot:prod@tcp(db:3306)/opspilot?charset=utf8mb4&parseTime=True&loc=UTC")
}
