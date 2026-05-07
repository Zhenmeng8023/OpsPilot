package config

import (
	"bufio"
	"errors"
	"os"
	"regexp"
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
	Command   CommandPolicyConfig
	Schedule  ScheduleConfig
	Alert     AlertConfig
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
	APIBaseURL          string
	Token               string
	BootstrapSecret     string
	Hostname            string
	Workspace           string
	PollInterval        time.Duration
	HeartbeatInterval   time.Duration
	MaxConcurrentTasks  int
	WorkDir             string
	TokenFile           string
	RegistrationEnabled bool
	OfflineScanInterval time.Duration
}

type CommandPolicyConfig struct {
	AllowPatterns []string
	DenyPatterns  []string
}

type ScheduleConfig struct {
	ScanInterval time.Duration
}

type AlertConfig struct {
	ScanInterval time.Duration
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
			APIBaseURL:          getEnv("AGENT_API_BASE_URL", "http://localhost:8080"),
			Token:               getEnv("AGENT_TOKEN", ""),
			BootstrapSecret:     getEnv("AGENT_BOOTSTRAP_SECRET", "dev-agent-bootstrap-secret"),
			Hostname:            getEnv("AGENT_HOSTNAME", ""),
			Workspace:           getEnv("AGENT_WORKSPACE", "default"),
			PollInterval:        getEnvDurationSeconds("AGENT_POLL_INTERVAL_SECONDS", getEnvDuration("AGENT_POLL_INTERVAL", 5*time.Second)),
			HeartbeatInterval:   getEnvDurationSeconds("AGENT_HEARTBEAT_INTERVAL_SECONDS", getEnvDuration("AGENT_HEARTBEAT_INTERVAL", 30*time.Second)),
			MaxConcurrentTasks:  getEnvInt("AGENT_MAX_CONCURRENT_TASKS", 2),
			WorkDir:             getEnv("AGENT_WORK_DIR", ".tmp/agent-work"),
			TokenFile:           getEnv("AGENT_TOKEN_FILE", ".tmp/agent-token"),
			RegistrationEnabled: getEnvBool("AGENT_REGISTRATION_ENABLED", true),
			OfflineScanInterval: getEnvDurationSeconds("AGENT_OFFLINE_SCAN_INTERVAL_SECONDS", getEnvDuration("AGENT_OFFLINE_SCAN_INTERVAL", 45*time.Second)),
		},
		Command: CommandPolicyConfig{
			AllowPatterns: getEnvList("TASK_COMMAND_ALLOW_PATTERNS"),
			DenyPatterns:  getEnvList("TASK_COMMAND_DENY_PATTERNS"),
		},
		Schedule: ScheduleConfig{
			ScanInterval: getEnvDurationSeconds("SCHEDULE_SCAN_INTERVAL_SECONDS", 30*time.Second),
		},
		Alert: AlertConfig{
			ScanInterval: getEnvDurationSeconds("ALERT_SCAN_INTERVAL_SECONDS", 30*time.Second),
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
	if cfg.App.Env == "prod" && strings.TrimSpace(cfg.HTTP.AllowOrigin) == "*" {
		return Config{}, errors.New("HTTP_ALLOW_ORIGIN cannot be * in prod")
	}
	if cfg.Agent.MaxConcurrentTasks <= 0 {
		cfg.Agent.MaxConcurrentTasks = 1
	}
	if err := validateCommandPatterns(cfg.Command.AllowPatterns); err != nil {
		return Config{}, err
	}
	if err := validateCommandPatterns(cfg.Command.DenyPatterns); err != nil {
		return Config{}, err
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

func getEnvBool(key string, fallback bool) bool {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getEnvList(key string) []string {
	value := strings.TrimSpace(getEnv(key, ""))
	if value == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func validateCommandPatterns(patterns []string) error {
	for _, pattern := range patterns {
		if _, err := regexp.Compile(pattern); err != nil {
			return err
		}
	}
	return nil
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

func getEnvDurationSeconds(key string, fallback time.Duration) time.Duration {
	value, ok := os.LookupEnv(key)
	if !ok || strings.TrimSpace(value) == "" {
		return fallback
	}
	seconds, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return time.Duration(seconds) * time.Second
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
