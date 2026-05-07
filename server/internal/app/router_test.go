package app

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"opspilot/server/internal/config"
)

func TestHealth(t *testing.T) {
	router := NewRouter(config.Config{
		App: config.AppConfig{
			Name:    "OpsPilot",
			Env:     "test",
			Version: "test",
		},
		HTTP: config.HTTPConfig{
			AllowOrigin: "*",
		},
	}, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body struct {
		Code    int             `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
		TraceID string          `json:"traceId"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != 0 || body.Message != "success" || body.TraceID == "" {
		t.Fatalf("unexpected response: %+v", body)
	}
}

func TestVersion(t *testing.T) {
	router := NewRouter(config.Config{
		App: config.AppConfig{
			Name:      "OpsPilot",
			Env:       "test",
			Version:   "0.7.0-test",
			Commit:    "abc123",
			BuildTime: "2026-05-07T00:00:00Z",
		},
		HTTP: config.HTTPConfig{
			AllowOrigin: "*",
		},
	}, slog.Default())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/version", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var body struct {
		Code int `json:"code"`
		Data struct {
			Service   string `json:"service"`
			Version   string `json:"version"`
			Commit    string `json:"commit"`
			BuildTime string `json:"buildTime"`
			GoVersion string `json:"goVersion"`
			Env       string `json:"env"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Code != 0 || body.Data.Version != "0.7.0-test" || body.Data.Commit != "abc123" || body.Data.GoVersion == "" {
		t.Fatalf("unexpected version response: %+v", body.Data)
	}
}
