package main

import (
	"log/slog"
	"os"

	"opspilot/server/internal/config"
	"opspilot/server/internal/platform/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config failed", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.App.Env)
	log.Info(
		"opspilot agent scaffold ready",
		"api_base_url", cfg.Agent.APIBaseURL,
		"heartbeat_interval", cfg.Agent.HeartbeatInterval.String(),
	)
}
