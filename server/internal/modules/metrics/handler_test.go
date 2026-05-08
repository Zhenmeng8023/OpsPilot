package metrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestCreateDashboardPassesPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureMetricService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughMetricAuth, passThroughMetricAuth, passThroughMetricPermission)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/metrics/dashboards", strings.NewReader(`{"name":"CPU","metricCode":"agent.os.cpu.percent","rangeHours":72,"pointLimit":180,"granularity":"5m"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.dashboardInput.Name != "CPU" || service.dashboardInput.MetricCode != "agent.os.cpu.percent" || service.dashboardInput.RangeHours != 72 || service.dashboardInput.PointLimit != 180 || service.dashboardInput.Granularity != "5m" {
		t.Fatalf("unexpected dashboard input: %#v", service.dashboardInput)
	}
}

type captureMetricService struct {
	listInput      ListInput
	trendInput     TrendInput
	dashboardInput DashboardInput
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

func (s *captureMetricService) ListDashboards(context.Context) ([]DashboardSummary, *apperror.Error) {
	return nil, nil
}

func (s *captureMetricService) CreateDashboard(_ context.Context, input DashboardInput) (DashboardSummary, *apperror.Error) {
	s.dashboardInput = input
	return DashboardSummary{}, nil
}

func (s *captureMetricService) UpdateDashboard(_ context.Context, _ string, input DashboardInput) (DashboardSummary, *apperror.Error) {
	s.dashboardInput = input
	return DashboardSummary{}, nil
}

func (s *captureMetricService) RunRollup(context.Context, RollupInput) (RollupResult, *apperror.Error) {
	return RollupResult{}, nil
}

func (s *captureMetricService) RunRetention(context.Context, RetentionInput) (RetentionResult, *apperror.Error) {
	return RetentionResult{}, nil
}

func passThroughMetricAuth(c *gin.Context) {
	c.Next()
}

func passThroughMetricPermission(string) gin.HandlerFunc {
	return passThroughMetricAuth
}
