package config

import (
	"bufio"
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App       AppConfig
	HTTP      HTTPConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	Agent     AgentConfig
	Bootstrap BootstrapConfig
}

type AppConfig struct {
	Name    string
	Env     string
	Version string
}

type HTTPConfig struct {
	Addr        string
	AllowOrigin string
}

type DatabaseConfig struct {
	Driver string
	DSN    string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type AgentConfig struct {
	APIBaseURL        string
	BootstrapSecret   string
	HeartbeatInterval time.Duration
	TokenFile         string
}

type BootstrapConfig struct {
	WorkspaceName string
	WorkspaceSlug string
	AdminUsername string
	AdminPassword string
	AdminEmail    string
}

func Load() (Config, error) {
	loadDotEnv(".env")
	loadDotEnv("../.env")

	cfg := Config{
		App: AppConfig{
			Name:    getEnv("APP_NAME", "OpsPilot"),
			Env:     getEnv("APP_ENV", "local"),
			Version: getEnv("APP_VERSION", "dev"),
		},
		HTTP: HTTPConfig{
			Addr:        getEnv("HTTP_ADDR", ":8080"),
			AllowOrigin: getEnv("HTTP_ALLOW_ORIGIN", "http://localhost:5173"),
		},
		Database: DatabaseConfig{
			Driver: getEnv("DATABASE_DRIVER", "mysql"),
			DSN:    getEnv("DATABASE_DSN", ""),
		},
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvInt("REDIS_DB", 0),
		},
		JWT: JWTConfig{
			AccessSecret:  getEnv("JWT_ACCESS_SECRET", "dev-access-secret-change-me"),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", "dev-refresh-secret-change-me"),
			AccessTTL:     getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL:    getEnvDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		Agent: AgentConfig{
			APIBaseURL:        getEnv("AGENT_API_BASE_URL", "http://localhost:8080"),
			BootstrapSecret:   getEnv("AGENT_BOOTSTRAP_SECRET", "dev-agent-bootstrap-secret"),
			HeartbeatInterval: getEnvDuration("AGENT_HEARTBEAT_INTERVAL", 30*time.Second),
			TokenFile:         getEnv("AGENT_TOKEN_FILE", ".tmp/agent-token"),
		},
		Bootstrap: BootstrapConfig{
			WorkspaceName: getEnv("BOOTSTRAP_WORKSPACE_NAME", "Default Workspace"),
			WorkspaceSlug: getEnv("BOOTSTRAP_WORKSPACE_SLUG", "default"),
			AdminUsername: getEnv("BOOTSTRAP_ADMIN_USERNAME", "admin"),
			AdminPassword: getEnv("BOOTSTRAP_ADMIN_PASSWORD", "Admin@123456"),
			AdminEmail:    getEnv("BOOTSTRAP_ADMIN_EMAIL", "admin@opspilot.local"),
		},
	}

	if cfg.App.Env == "" {
		return Config{}, errors.New("APP_ENV cannot be empty")
	}
	if cfg.HTTP.Addr == "" {
		return Config{}, errors.New("HTTP_ADDR cannot be empty")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, value)
		}
	}
}
