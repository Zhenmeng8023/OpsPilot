package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/app/middleware"
	"opspilot/server/internal/config"
	"opspilot/server/internal/shared/response"
)

func NewRouter(cfg config.Config, log *slog.Logger) *gin.Engine {
	if cfg.App.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.TraceID())
	router.Use(middleware.CORS(cfg.HTTP.AllowOrigin))
	router.Use(middleware.AccessLog(log))

	router.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{
			"status":  "ok",
			"service": cfg.App.Name,
			"version": cfg.App.Version,
			"env":     cfg.App.Env,
			"time":    time.Now().Format(time.RFC3339),
		})
	})

	api := router.Group("/api/v1")
	api.GET("/ping", func(c *gin.Context) {
		response.Success(c, gin.H{"pong": true})
	})

	router.NoRoute(func(c *gin.Context) {
		response.Fail(c, http.StatusNotFound, 404001, "route not found")
	})

	return router
}
