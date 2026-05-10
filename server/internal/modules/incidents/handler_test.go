package incidents

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/shared/apperror"
)

func TestUpdateLifecyclePassesPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureIncidentService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughIncidentAuth, passThroughIncidentPermission)

	req := httptest.NewRequest(http.MethodPatch, "/api/v1/incidents/inc-1/lifecycle", bytes.NewBufferString(`{"owner":"oncall-a","impactScope":"payments","rootCauseClass":"dependency","postmortem":"draft"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.lifecycleIncidentID != "inc-1" {
		t.Fatalf("unexpected incident id: %q", service.lifecycleIncidentID)
	}
	if service.lifecycleInput.Owner != "oncall-a" || service.lifecycleInput.ImpactScope != "payments" || service.lifecycleInput.RootCauseClass != "dependency" || service.lifecycleInput.Postmortem != "draft" {
		t.Fatalf("unexpected lifecycle input: %#v", service.lifecycleInput)
	}
}

func TestMergePassesPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureIncidentService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughIncidentAuth, passThroughIncidentPermission)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/inc-1/merge", bytes.NewBufferString(`{"targetIncidentId":"inc-2","reason":"duplicate"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.mergeIncidentID != "inc-1" {
		t.Fatalf("unexpected source incident id: %q", service.mergeIncidentID)
	}
	if service.mergeInput.TargetIncidentID != "inc-2" || service.mergeInput.Reason != "duplicate" {
		t.Fatalf("unexpected merge input: %#v", service.mergeInput)
	}
}

func TestClosePassesPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureIncidentService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughIncidentAuth, passThroughIncidentPermission)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/incidents/inc-3/close", bytes.NewBufferString(`{"reason":"mitigated"}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.closeIncidentID != "inc-3" {
		t.Fatalf("unexpected incident id: %q", service.closeIncidentID)
	}
	if service.closeInput.Reason != "mitigated" {
		t.Fatalf("unexpected close input: %#v", service.closeInput)
	}
}

type captureIncidentService struct {
	lifecycleIncidentID string
	lifecycleInput      LifecycleUpdateInput
	mergeIncidentID     string
	mergeInput          MergeInput
	closeIncidentID     string
	closeInput          CloseInput
}

func (s *captureIncidentService) List(context.Context, ListInput) ([]Summary, *apperror.Error) {
	return nil, nil
}

func (s *captureIncidentService) Get(context.Context, string) (Detail, *apperror.Error) {
	return Detail{}, nil
}

func (s *captureIncidentService) UpdateLifecycle(_ context.Context, incidentID string, input LifecycleUpdateInput) (Detail, *apperror.Error) {
	s.lifecycleIncidentID = incidentID
	s.lifecycleInput = input
	return Detail{}, nil
}

func (s *captureIncidentService) Merge(_ context.Context, incidentID string, input MergeInput) (Detail, *apperror.Error) {
	s.mergeIncidentID = incidentID
	s.mergeInput = input
	return Detail{}, nil
}

func (s *captureIncidentService) Close(_ context.Context, incidentID string, input CloseInput) (Detail, *apperror.Error) {
	s.closeIncidentID = incidentID
	s.closeInput = input
	return Detail{}, nil
}

func passThroughIncidentAuth(c *gin.Context) {
	c.Next()
}

func passThroughIncidentPermission(string) gin.HandlerFunc {
	return passThroughIncidentAuth
}
