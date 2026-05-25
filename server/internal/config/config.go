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

var (
	BuildVersion string
	BuildCommit  string
	BuildTime    string
)

const (
	defaultAccessSecret        = "dev-access-secret-change-me"
	defaultRefreshSecret       = "dev-refresh-secret-change-me"
	defaultSecretEncryptionKey = "dev-secret-encryption-key-change-me"
	defaultAgentBootstrapToken = "dev-agent-bootstrap-secret"
	defaultAdminPassword       = "Admin@123456"
	minJWTSecretLength         = 32
)

type Config struct {
	App       AppConfig
	HTTP      HTTPConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	Security  SecurityConfig
	Auth      AuthConfig
	Agent     AgentConfig
	Command   CommandPolicyConfig
	Schedule  ScheduleConfig
	Alert     AlertConfig
	Metric    MetricConfig
	Audit     AuditConfig
	Trace     TraceConfig
	Notify    NotificationConfig
	Bootstrap BootstrapConfig
}

type AppConfig struct {
	Name      string
	Env       string
	Version   string
	Commit    string
	BuildTime string
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

type SecurityConfig struct {
	SecretEncryptionKey string
}

type AuthConfig struct {
	PublicRegistrationEnabled bool
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

type MetricConfig struct {
	DetailRetentionDays int
	RollupRetentionDays int
	RollupInterval      time.Duration
	RetentionInterval   time.Duration
}

type AuditConfig struct {
	RetentionDays int
}

type TraceConfig struct {
	RetentionDays int
}

type NotificationConfig struct {
	DispatchInterval time.Duration
	HTTPTimeout      time.Duration
	SMTPHost         string
	SMTPPort         int
	SMTPUsername     string
	SMTPPassword     string
	SMTPFrom         string
	SMTPSecurity     string
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
			Name:      getEnv("APP_NAME", "OpsPilot"),
			Env:       getEnv("APP_ENV", "local"),
			Version:   resolveVersion(),
			Commit:    envOrBuild("APP_COMMIT", BuildCommit),
			BuildTime: envOrBuild("APP_BUILD_TIME", BuildTime),
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
			AccessSecret:  getEnv("JWT_ACCESS_SECRET", defaultAccessSecret),
			RefreshSecret: getEnv("JWT_REFRESH_SECRET", defaultRefreshSecret),
			AccessTTL:     getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute),
			RefreshTTL:    getEnvDuration("JWT_REFRESH_TTL", 7*24*time.Hour),
		},
		Security: SecurityConfig{
			SecretEncryptionKey: getEnv("SECRET_ENCRYPTION_KEY", defaultSecretEncryptionKey),
		},
		Auth: AuthConfig{
			PublicRegistrationEnabled: getEnvBool("AUTH_PUBLIC_REGISTRATION_ENABLED", getEnv("APP_ENV", "local") != "prod"),
		},
		Agent: AgentConfig{
			APIBaseURL:          getEnv("AGENT_API_BASE_URL", "http://localhost:8080"),
			Token:               getEnv("AGENT_TOKEN", ""),
			BootstrapSecret:     getEnv("AGENT_BOOTSTRAP_SECRET", defaultAgentBootstrapToken),
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
		Metric: MetricConfig{
			DetailRetentionDays: getEnvInt("METRIC_DETAIL_RETENTION_DAYS", 7),
			RollupRetentionDays: getEnvInt("METRIC_ROLLUP_RETENTION_DAYS", 90),
			RollupInterval:      getEnvDurationSeconds("METRIC_ROLLUP_INTERVAL_SECONDS", 5*time.Minute),
			RetentionInterval:   getEnvDurationSeconds("METRIC_RETENTION_INTERVAL_SECONDS", 24*time.Hour),
		},
		Audit: AuditConfig{
			RetentionDays: getEnvInt("AUDIT_RETENTION_DAYS", 180),
		},
		Trace: TraceConfig{
			RetentionDays: getEnvInt("TRACE_RETENTION_DAYS", 30),
		},
		Notify: NotificationConfig{
			DispatchInterval: getEnvDurationSeconds("NOTIFICATION_DISPATCH_INTERVAL_SECONDS", 15*time.Second),
			HTTPTimeout:      getEnvDurationSeconds("NOTIFICATION_HTTP_TIMEOUT_SECONDS", 10*time.Second),
			SMTPHost:         getEnv("NOTIFICATION_SMTP_HOST", ""),
			SMTPPort:         getEnvInt("NOTIFICATION_SMTP_PORT", 587),
			SMTPUsername:     getEnv("NOTIFICATION_SMTP_USERNAME", ""),
			SMTPPassword:     getEnv("NOTIFICATION_SMTP_PASSWORD", ""),
			SMTPFrom:         getEnv("NOTIFICATION_SMTP_FROM", ""),
			SMTPSecurity:     strings.ToLower(strings.TrimSpace(getEnv("NOTIFICATION_SMTP_SECURITY", "starttls"))),
		},
		Bootstrap: BootstrapConfig{
			WorkspaceName: getEnv("BOOTSTRAP_WORKSPACE_NAME", "Default Workspace"),
			WorkspaceSlug: getEnv("BOOTSTRAP_WORKSPACE_SLUG", "default"),
			AdminUsername: getEnv("BOOTSTRAP_ADMIN_USERNAME", "admin"),
			AdminPassword: getEnv("BOOTSTRAP_ADMIN_PASSWORD", defaultAdminPassword),
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
	if cfg.App.Env == "prod" {
		if err := validateProdJWTSecret("JWT_ACCESS_SECRET", cfg.JWT.AccessSecret, defaultAccessSecret); err != nil {
			return Config{}, err
		}
		if err := validateProdJWTSecret("JWT_REFRESH_SECRET", cfg.JWT.RefreshSecret, defaultRefreshSecret); err != nil {
			return Config{}, err
		}
		if err := validateProdSecretEncryptionKey(cfg.Security.SecretEncryptionKey); err != nil {
			return Config{}, err
		}
		if err := validateProdJWTSecret("AGENT_BOOTSTRAP_SECRET", cfg.Agent.BootstrapSecret, defaultAgentBootstrapToken); err != nil {
			return Config{}, err
		}
		if err := validateProdJWTSecret("BOOTSTRAP_ADMIN_PASSWORD", cfg.Bootstrap.AdminPassword, defaultAdminPassword); err != nil {
			return Config{}, err
		}
		if err := validateProdAllowOrigin(cfg.HTTP.AllowOrigin); err != nil {
			return Config{}, err
		}
		if err := validateProdDatabaseDSN(cfg.Database.DSN); err != nil {
			return Config{}, err
		}
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
	if cfg.Notify.SMTPSecurity == "" {
		cfg.Notify.SMTPSecurity = "starttls"
	}
	if cfg.Notify.SMTPSecurity != "starttls" && cfg.Notify.SMTPSecurity != "tls" && cfg.Notify.SMTPSecurity != "plain" {
		return Config{}, errors.New("NOTIFICATION_SMTP_SECURITY must be one of starttls, tls or plain")
	}
	if cfg.Audit.RetentionDays <= 0 {
		cfg.Audit.RetentionDays = 180
	}
	if cfg.Trace.RetentionDays <= 0 {
		cfg.Trace.RetentionDays = 30
	}
	if cfg.Metric.DetailRetentionDays <= 0 {
		cfg.Metric.DetailRetentionDays = 7
	}
	if cfg.Metric.RollupRetentionDays <= 0 {
		cfg.Metric.RollupRetentionDays = 90
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func envOrBuild(key, buildValue string) string {
	if value, ok := os.LookupEnv(key); ok {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return strings.TrimSpace(buildValue)
}

func resolveVersion() string {
	if value := envOrBuild("APP_VERSION", BuildVersion); value != "" {
		return value
	}
	if value := readVersionFile(); value != "" {
		return value
	}
	return "dev"
}

func readVersionFile() string {
	for _, path := range []string{"VERSION", "../VERSION", "../../VERSION", "../../../VERSION"} {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if value := strings.TrimSpace(string(content)); value != "" {
			return value
		}
	}
	return ""
}

func validateProdJWTSecret(name, value, defaultValue string) error {
	value = strings.TrimSpace(value)
	switch {
	case value == "":
		return errors.New(name + " cannot be empty in prod")
	case value == defaultValue:
		return errors.New(name + " cannot use the default development value in prod")
	case len(value) < minJWTSecretLength:
		return errors.New(name + " must be at least 32 characters in prod")
	default:
		return nil
	}
}

func validateProdSecretEncryptionKey(value string) error {
	value = strings.TrimSpace(value)
	switch {
	case value == "":
		return errors.New("SECRET_ENCRYPTION_KEY cannot be empty in prod")
	case value == defaultSecretEncryptionKey:
		return errors.New("SECRET_ENCRYPTION_KEY cannot use the default development value in prod")
	case len(value) < minJWTSecretLength:
		return errors.New("SECRET_ENCRYPTION_KEY must be at least 32 characters in prod")
	default:
		return nil
	}
}

func validateProdAllowOrigin(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("HTTP_ALLOW_ORIGIN cannot be empty in prod")
	}
	origins := strings.Split(value, ",")
	for _, origin := range origins {
		item := strings.ToLower(strings.TrimSpace(origin))
		if item == "" {
			continue
		}
		if strings.Contains(item, "localhost") || strings.Contains(item, "127.0.0.1") || strings.Contains(item, "::1") {
			return errors.New("HTTP_ALLOW_ORIGIN cannot contain local loopback origins in prod")
		}
	}
	return nil
}

func validateProdDatabaseDSN(value string) error {
	dsn := strings.TrimSpace(value)
	if dsn == "" {
		return errors.New("DATABASE_DSN cannot be empty in prod")
	}
	if !strings.Contains(strings.ToLower(dsn), "parsetime=true") {
		return errors.New("DATABASE_DSN must include parseTime=True in prod")
	}
	return nil
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
