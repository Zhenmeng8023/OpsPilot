package app

import (
	"context"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"opspilot/server/internal/app/middleware"
	"opspilot/server/internal/config"
	"opspilot/server/internal/modules/agents"
	"opspilot/server/internal/modules/alerts"
	"opspilot/server/internal/modules/auth"
	"opspilot/server/internal/modules/metrics"
	"opspilot/server/internal/modules/notifications"
	"opspilot/server/internal/modules/schedules"
	"opspilot/server/internal/modules/scripts"
	"opspilot/server/internal/modules/tasks"
	"opspilot/server/internal/modules/webhooks"
	"opspilot/server/internal/shared/response"
)

type Dependencies struct {
	DB    *gorm.DB
	Redis *redis.Client
}

func NewRouter(cfg config.Config, log *slog.Logger) *gin.Engine {
	return NewRouterWithDependencies(cfg, log, Dependencies{})
}

func NewRouterWithDependencies(cfg config.Config, log *slog.Logger, deps Dependencies) *gin.Engine {
	if cfg.App.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.TraceID())
	router.Use(middleware.CORS(cfg.HTTP.AllowOrigin))
	router.Use(middleware.AccessLog(log))

	router.GET("/health", func(c *gin.Context) {
		httpStatus, data := healthData(c.Request.Context(), cfg, deps)
		if httpStatus != http.StatusOK {
			response.Fail(c, httpStatus, 503001, "service dependencies unavailable")
			return
		}
		response.Success(c, data)
	})

	api := router.Group("/api/v1")
	api.GET("/version", func(c *gin.Context) {
		response.Success(c, versionData(cfg))
	})
	api.GET("/ping", func(c *gin.Context) {
		response.Success(c, gin.H{"pong": true})
	})
	if deps.DB != nil {
		authService := auth.NewService(deps.DB, cfg)
		authHandler := auth.NewHandler(authService)
		authHandler.RegisterRoutes(api)

		userAuth := auth.AuthMiddleware(authService.JWTManager())
		agentHandler := agents.NewHandler(agents.NewService(deps.DB, cfg))
		agentHandler.RegisterRoutes(api, auth.AuthMiddleware(authService.JWTManager()), authHandler.RequirePermission)

		scripts.NewHandler(scripts.NewService(deps.DB, cfg)).RegisterRoutes(api, userAuth, authHandler.RequirePermission)
		schedules.NewHandler(schedules.NewService(deps.DB, cfg)).RegisterRoutes(api, userAuth, authHandler.RequirePermission)
		webhooks.NewHandler(webhooks.NewService(deps.DB, cfg)).RegisterRoutes(api, userAuth, authHandler.RequirePermission)
		taskHandler := tasks.NewHandler(tasks.NewService(deps.DB, cfg))
		taskHandler.RegisterRoutes(api, userAuth, authHandler.RequirePermission)
		taskHandler.RegisterAgentRoutes(api, agentHandler.AgentAuthMiddleware())
		metrics.NewHandler(metrics.NewService(deps.DB, cfg)).RegisterRoutes(api, userAuth, agentHandler.AgentAuthMiddleware(), authHandler.RequirePermission)
		alerts.NewHandler(alerts.NewService(deps.DB, cfg)).RegisterRoutes(api, userAuth, authHandler.RequirePermission)
		notifications.NewHandler(notifications.NewService(deps.DB, cfg)).RegisterRoutes(api, userAuth, authHandler.RequirePermission)
	}

	router.NoRoute(func(c *gin.Context) {
		response.Fail(c, http.StatusNotFound, 404001, "route not found")
	})

	return router
}

func versionData(cfg config.Config) gin.H {
	return gin.H{
		"service":   cfg.App.Name,
		"version":   cfg.App.Version,
		"commit":    cfg.App.Commit,
		"buildTime": cfg.App.BuildTime,
		"goVersion": runtime.Version(),
		"env":       cfg.App.Env,
	}
}

func healthData(parent context.Context, cfg config.Config, deps Dependencies) (int, gin.H) {
	ctx, cancel := context.WithTimeout(parent, time.Second)
	defer cancel()

	dependencyStatus := gin.H{}
	httpStatus := http.StatusOK

	if deps.DB == nil {
		dependencyStatus["database"] = "not_configured"
	} else if err := pingDB(ctx, deps.DB); err != nil {
		dependencyStatus["database"] = "unavailable"
		httpStatus = http.StatusServiceUnavailable
	} else {
		dependencyStatus["database"] = "ok"
	}

	if deps.Redis == nil {
		dependencyStatus["redis"] = "not_configured"
	} else if err := deps.Redis.Ping(ctx).Err(); err != nil {
		dependencyStatus["redis"] = "unavailable"
		httpStatus = http.StatusServiceUnavailable
	} else {
		dependencyStatus["redis"] = "ok"
	}

	status := "ok"
	if httpStatus != http.StatusOK {
		status = "degraded"
	}

	return httpStatus, gin.H{
		"status":  status,
		"service": cfg.App.Name,
		"version": cfg.App.Version,
		"env":     cfg.App.Env,
		"time":    time.Now().Format(time.RFC3339),
		"checks":  dependencyStatus,
	}
}

func pingDB(ctx context.Context, db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}
