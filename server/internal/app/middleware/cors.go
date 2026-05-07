package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORS(allowOrigin string) gin.HandlerFunc {
	allowedOrigins := parseAllowOrigins(allowOrigin)

	return func(c *gin.Context) {
		if origin := allowedOrigin(c.GetHeader("Origin"), allowedOrigins); origin != "" {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Access-Control-Allow-Credentials", "true")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Trace-Id")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			c.Header("Vary", "Origin")
		}

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

func parseAllowOrigins(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = "http://localhost:5173"
	}
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if origin := strings.TrimSpace(part); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}

func allowedOrigin(requestOrigin string, allowed []string) string {
	requestOrigin = strings.TrimSpace(requestOrigin)
	for _, origin := range allowed {
		switch {
		case origin == "*":
			return "*"
		case requestOrigin != "" && strings.EqualFold(origin, requestOrigin):
			return requestOrigin
		}
	}
	return ""
}
