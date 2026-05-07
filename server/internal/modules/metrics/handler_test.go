package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/modules/agents"
	"opspilot/server/internal/shared/apperror"
)

func TestListPassesFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureMetricService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughMetricAuth, passThroughMetricAuth, passThroughMetricPermission)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/hosts?hostId=host-1&agentId=agent-1&metricCode=agent.runtime.goroutines&limit=42", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.listInput.HostID != "host-1" || service.listInput.AgentID != "agent-1" || service.listInput.MetricCode != "agent.runtime.goroutines" || service.listInput.Limit != 42 {
		t.Fatalf("unexpected list input: %#v", service.listInput)
	}
}

func TestListTrendsPassesFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureMetricService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughMetricAuth, passThroughMetricAuth, passThroughMetricPermission)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/metrics/hosts/trends?hostId=host-9&agentId=agent-9&metricCode=agent.runtime.alloc_bytes&hours=6&limit=90", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.trendInput.HostID != "host-9" || service.trendInput.AgentID != "agent-9" || service.trendInput.MetricCode != "agent.runtime.alloc_bytes" || service.trendInput.Hours != 6 || service.trendInput.Limit != 90 {
		t.Fatalf("unexpected trend input: %#v", service.trendInput)
	}
}

type captureMetricService struct {
	listInput  ListInput
	trendInput TrendInput
}

func (s *captureMetricService) Upload(context.Context, agents.AgentIdentity, []MetricInput) *apperror.Error {
	return nil
}

func (s *captureMetricService) List(_ context.Context, input ListInput) ([]MetricSummary, *apperror.Error) {
	s.listInput = input
	return nil, nil
}

func (s *captureMetricService) ListTrends(_ context.Context, input TrendInput) ([]MetricTrendSeries, *apperror.Error) {
	s.trendInput = input
	return nil, nil
}

func passThroughMetricAuth(c *gin.Context) {
	c.Next()
}

func passThroughMetricPermission(string) gin.HandlerFunc {
	return passThroughMetricAuth
}
