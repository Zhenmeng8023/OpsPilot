package response

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/app/middleware"
	"opspilot/server/internal/shared/apperror"
)

func TestSuccessEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(middleware.TraceID())
	router.GET("/", func(c *gin.Context) {
		Success(c, gin.H{"ok": true})
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, req)

	var body Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != 0 || body.Message != "success" || body.TraceID == "" {
		t.Fatalf("unexpected envelope: %+v", body)
	}
}

func TestFailAppErrorIncludesDebugMessageInLocal(t *testing.T) {
	gin.SetMode(gin.TestMode)
	oldEnv := os.Getenv("APP_ENV")
	t.Cleanup(func() {
		_ = os.Setenv("APP_ENV", oldEnv)
	})
	_ = os.Setenv("APP_ENV", "local")

	router := gin.New()
	router.Use(middleware.TraceID())
	router.GET("/", func(c *gin.Context) {
		FailAppError(c, apperror.Wrap(http.StatusInternalServerError, 500001, "write heartbeat failed", errors.New("insert agent_heartbeats failed: foreign key violation")))
	})

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	router.ServeHTTP(w, req)

	var body Envelope
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != 500001 || body.DebugMessage == "" || body.TraceID == "" {
		t.Fatalf("unexpected error envelope: %+v", body)
	}
}
