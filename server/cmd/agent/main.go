package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"opspilot/server/internal/config"
	"opspilot/server/internal/platform/logger"
)

type apiClient struct {
	baseURL string
	http    *http.Client
}

type envelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
	TraceID string `json:"traceId"`
}

type apiError struct {
	status  int
	message string
}

type agentPayload struct {
	BootstrapSecret string `json:"bootstrapSecret,omitempty"`
	Name            string `json:"name,omitempty"`
	Hostname        string `json:"hostname,omitempty"`
	IP              string `json:"ip,omitempty"`
	OS              string `json:"os,omitempty"`
	OSType          string `json:"osType,omitempty"`
	OSName          string `json:"osName,omitempty"`
	Arch            string `json:"arch,omitempty"`
	Version         string `json:"version,omitempty"`
	Status          string `json:"status,omitempty"`
}

type registerData struct {
	Token       string `json:"token"`
	TokenPrefix string `json:"tokenPrefix"`
}

func (e apiError) Error() string {
	return e.message
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config failed", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.App.Env)
	client := apiClient{
		baseURL: strings.TrimRight(cfg.Agent.APIBaseURL, "/"),
		http: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	token, err := readToken(cfg.Agent.TokenFile)
	if err != nil {
		log.Error("read agent token failed", "error", err)
		os.Exit(1)
	}
	if token == "" {
		token, err = register(ctx, client, cfg)
		if err != nil {
			log.Error("register agent failed", "error", err)
			os.Exit(1)
		}
		if err := writeToken(cfg.Agent.TokenFile, token); err != nil {
			log.Error("save agent token failed", "error", err)
			os.Exit(1)
		}
		log.Info("agent registered", "token_file", cfg.Agent.TokenFile)
	}

	if err := heartbeat(ctx, client, cfg, token); err != nil {
		var apiErr apiError
		if errors.As(err, &apiErr) && apiErr.status == http.StatusUnauthorized {
			token, err = register(ctx, client, cfg)
			if err == nil {
				err = writeToken(cfg.Agent.TokenFile, token)
			}
		}
		if err != nil {
			log.Error("initial heartbeat failed", "error", err)
			os.Exit(1)
		}
	}

	interval := cfg.Agent.HeartbeatInterval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Info("opspilot agent running", "api_base_url", cfg.Agent.APIBaseURL, "heartbeat_interval", interval.String())
	for {
		select {
		case <-ctx.Done():
			log.Info("opspilot agent stopped")
			return
		case <-ticker.C:
			if err := heartbeat(ctx, client, cfg, token); err != nil {
				log.Warn("heartbeat failed", "error", err)
			}
		}
	}
}

func register(ctx context.Context, client apiClient, cfg config.Config) (string, error) {
	payload := collectHostInfo(cfg)
	payload.BootstrapSecret = cfg.Agent.BootstrapSecret

	data, err := postJSON[registerData](ctx, client, "/api/v1/agents/register", "", payload)
	if err != nil {
		return "", err
	}
	if data.Token == "" {
		return "", errors.New("register response did not include token")
	}
	return data.Token, nil
}

func heartbeat(ctx context.Context, client apiClient, cfg config.Config, token string) error {
	payload := collectHostInfo(cfg)
	payload.Status = "online"
	_, err := postJSON[json.RawMessage](ctx, client, "/api/v1/agents/heartbeat", token, payload)
	return err
}

func postJSON[T any](ctx context.Context, client apiClient, path, token string, payload interface{}) (T, error) {
	var zero T
	body, err := json.Marshal(payload)
	if err != nil {
		return zero, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, client.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return zero, err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := client.http.Do(req)
	if err != nil {
		return zero, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return zero, err
	}
	var wrapped envelope[T]
	if err := json.Unmarshal(respBody, &wrapped); err != nil {
		return zero, fmt.Errorf("decode api response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || wrapped.Code != 0 {
		message := wrapped.Message
		if message == "" {
			message = resp.Status
		}
		return zero, apiError{status: resp.StatusCode, message: message}
	}
	return wrapped.Data, nil
}

func collectHostInfo(cfg config.Config) agentPayload {
	hostname, _ := os.Hostname()
	return agentPayload{
		Name:     hostname,
		Hostname: hostname,
		IP:       localIP(),
		OS:       runtime.GOOS,
		OSType:   runtime.GOOS,
		OSName:   runtime.GOOS,
		Arch:     runtime.GOARCH,
		Version:  cfg.App.Version,
	}
}

func localIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer conn.Close()

	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return ""
	}
	return addr.IP.String()
}

func readToken(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(bytes)), nil
}

func writeToken(path, token string) error {
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(token+"\n"), 0o600)
}
