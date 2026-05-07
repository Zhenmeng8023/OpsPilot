package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSAllowsMatchingOriginFromWhitelist(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS("http://localhost:5173,https://ops.example.com"))
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "https://ops.example.com")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if got := resp.Header().Get("Access-Control-Allow-Origin"); got != "https://ops.example.com" {
		t.Fatalf("expected matching allow origin, got %q", got)
	}
}

func TestCORSDoesNotAllowMismatchedOrigin(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORS("http://localhost:5173,https://ops.example.com"))
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if got := resp.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("expected no allow origin header for mismatched origin, got %q", got)
	}
}
