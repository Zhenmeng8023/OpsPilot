package notifications

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/shared/apperror"
)

func TestListDeliveriesPassesFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureNotificationService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughNotificationAuth, passThroughNotificationPermission)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/notification-deliveries?status=failed&channelId=ch-1&notificationId=ntf-1", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.deliveryFilters.Status != "failed" || service.deliveryFilters.ChannelID != "ch-1" || service.deliveryFilters.NotificationID != "ntf-1" {
		t.Fatalf("unexpected delivery filters: %#v", service.deliveryFilters)
	}
}

func TestRetryDeliveryPassesID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureNotificationService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughNotificationAuth, passThroughNotificationPermission)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification-deliveries/42/retry", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.retryDeliveryID != 42 {
		t.Fatalf("unexpected retry id: %d", service.retryDeliveryID)
	}
}

func TestTestChannelPassesID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	service := &captureNotificationService{}
	router := gin.New()
	NewHandler(service).RegisterRoutes(router.Group("/api/v1"), passThroughNotificationAuth, passThroughNotificationPermission)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/notification-channels/ch-1/test", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if service.testChannelID != "ch-1" {
		t.Fatalf("unexpected test channel id: %s", service.testChannelID)
	}
}

type captureNotificationService struct {
	deliveryFilters ListDeliveriesInput
	retryDeliveryID uint64
	testChannelID   string
}

func (s *captureNotificationService) ListChannels(context.Context) ([]ChannelSummary, *apperror.Error) {
	return nil, nil
}

func (s *captureNotificationService) CreateChannel(context.Context, CreateChannelInput) (ChannelSummary, *apperror.Error) {
	return ChannelSummary{}, nil
}

func (s *captureNotificationService) TestChannel(_ context.Context, id string, _ AuditContext) (ChannelTestResult, *apperror.Error) {
	s.testChannelID = id
	return ChannelTestResult{ChannelID: id, Status: "success"}, nil
}

func (s *captureNotificationService) ListNotifications(context.Context, bool) ([]NotificationSummary, *apperror.Error) {
	return nil, nil
}

func (s *captureNotificationService) ListDeliveries(_ context.Context, input ListDeliveriesInput) ([]DeliverySummary, *apperror.Error) {
	s.deliveryFilters = input
	return nil, nil
}

func (s *captureNotificationService) MarkRead(context.Context, string) *apperror.Error {
	return nil
}

func (s *captureNotificationService) RetryDelivery(_ context.Context, id uint64, _ AuditContext) *apperror.Error {
	s.retryDeliveryID = id
	return nil
}

func passThroughNotificationAuth(c *gin.Context) {
	c.Next()
}

func passThroughNotificationPermission(string) gin.HandlerFunc {
	return passThroughNotificationAuth
}
