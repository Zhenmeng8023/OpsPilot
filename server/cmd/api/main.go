package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"opspilot/server/internal/app"
	"opspilot/server/internal/config"
	"opspilot/server/internal/modules/agents"
	"opspilot/server/internal/modules/alerts"
	"opspilot/server/internal/modules/auth"
	"opspilot/server/internal/modules/schedules"
	"opspilot/server/internal/platform/db"
	"opspilot/server/internal/platform/logger"
	redisplatform "opspilot/server/internal/platform/redis"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		slog.Error("load config failed", "error", err)
		os.Exit(1)
	}

	log := logger.New(cfg.App.Env)

	startupCtx, startupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer startupCancel()

	dbHandle, err := db.Open(startupCtx, cfg.Database)
	if err != nil {
		log.Error("connect database failed", "error", err)
		os.Exit(1)
	}
	sqlDB, err := dbHandle.DB()
	if err != nil {
		log.Error("read database handle failed", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()
	if err := auth.Seed(startupCtx, dbHandle, cfg); err != nil {
		log.Error("bootstrap auth data failed", "error", err)
		os.Exit(1)
	}
	scannerCtx, scannerCancel := context.WithCancel(context.Background())
	defer scannerCancel()
	startOfflineScanner(scannerCtx, log, agents.NewService(dbHandle, cfg), cfg.Agent.OfflineScanInterval)
	schedules.StartScheduler(scannerCtx, log, schedules.NewService(dbHandle, cfg), cfg.Schedule.ScanInterval)
	alerts.StartScanner(scannerCtx, log, alerts.NewService(dbHandle, cfg), cfg.Alert.ScanInterval)

	redisClient := redisplatform.NewClient(cfg.Redis)
	if err := redisplatform.Ping(startupCtx, redisClient); err != nil {
		log.Error("connect redis failed", "error", err)
		os.Exit(1)
	}
	defer redisClient.Close()

	router := app.NewRouterWithDependencies(cfg, log, app.Dependencies{
		DB:    dbHandle,
		Redis: redisClient,
	})
	server := &http.Server{
		Addr:              cfg.HTTP.Addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("opspilot api starting", "addr", cfg.HTTP.Addr, "env", cfg.App.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("api server failed", "error", err)
			os.Exit(1)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	scannerCancel()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("api server shutdown failed", "error", err)
		os.Exit(1)
	}

	log.Info("opspilot api stopped")
}

func startOfflineScanner(ctx context.Context, log *slog.Logger, service *agents.Service, interval time.Duration) {
	if interval <= 0 {
		interval = 45 * time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				result, appErr := service.MarkOffline(ctx)
				if appErr != nil {
					log.Warn("agent offline scan failed", "error", appErr)
					continue
				}
				if result.OfflineAgents > 0 || result.OfflineHosts > 0 {
					log.Info("agent offline scan completed", "offline_agents", result.OfflineAgents, "offline_hosts", result.OfflineHosts)
				}
			}
		}
	}()
}
