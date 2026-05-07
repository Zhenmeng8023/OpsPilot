package notifications

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/modules/auth"
	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/response"
)

type ServiceContract interface {
	ListChannels(context.Context) ([]ChannelSummary, *apperror.Error)
	CreateChannel(context.Context, CreateChannelInput) (ChannelSummary, *apperror.Error)
	TestChannel(context.Context, string, AuditContext) (ChannelTestResult, *apperror.Error)
	ListNotifications(context.Context, bool) ([]NotificationSummary, *apperror.Error)
	ListDeliveries(context.Context, ListDeliveriesInput) ([]DeliverySummary, *apperror.Error)
	MarkRead(context.Context, string) *apperror.Error
	RetryDelivery(context.Context, uint64, AuditContext) *apperror.Error
}

type Handler struct {
	service ServiceContract
}

type createChannelRequest struct {
	Name        string                 `json:"name" binding:"required"`
	ChannelType string                 `json:"channelType"`
	Config      map[string]interface{} `json:"config"`
}

func NewHandler(service ServiceContract) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, userAuth gin.HandlerFunc, requirePermission func(string) gin.HandlerFunc) {
	protected := api.Group("")
	protected.Use(userAuth)
	protected.GET("/notification-channels", requirePermission("notification:read"), h.listChannels)
	protected.POST("/notification-channels", requirePermission("notification:write"), h.createChannel)
	protected.POST("/notification-channels/:id/test", requirePermission("notification:write"), h.testChannel)
	protected.GET("/notifications", requirePermission("notification:read"), h.listNotifications)
	protected.POST("/notifications/:id/read", requirePermission("notification:read"), h.markRead)
	protected.GET("/notification-deliveries", requirePermission("notification:read"), h.listDeliveries)
	protected.POST("/notification-deliveries/:id/retry", requirePermission("notification:write"), h.retryDelivery)
}

func (h *Handler) listChannels(c *gin.Context) {
	channels, appErr := h.service.ListChannels(c.Request.Context())
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, channels)
}

func (h *Handler) createChannel(c *gin.Context) {
	var req createChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	channel, appErr := h.service.CreateChannel(c.Request.Context(), CreateChannelInput{
		Name:        req.Name,
		ChannelType: req.ChannelType,
		Config:      req.Config,
		Audit:       auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, channel)
}

func (h *Handler) testChannel(c *gin.Context) {
	result, appErr := h.service.TestChannel(c.Request.Context(), c.Param("id"), auditContext(c))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) listNotifications(c *gin.Context) {
	items, appErr := h.service.ListNotifications(c.Request.Context(), c.Query("unread") == "true")
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, items)
}

func (h *Handler) markRead(c *gin.Context) {
	if appErr := h.service.MarkRead(c.Request.Context(), c.Param("id")); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) listDeliveries(c *gin.Context) {
	items, appErr := h.service.ListDeliveries(c.Request.Context(), ListDeliveriesInput{
		Status:         c.Query("status"),
		ChannelID:      c.Query("channelId"),
		NotificationID: c.Query("notificationId"),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, items)
}

func (h *Handler) retryDelivery(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if appErr := h.service.RetryDelivery(c.Request.Context(), id, auditContext(c)); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func auditContext(c *gin.Context) AuditContext {
	actorUID := ""
	if claims, ok := auth.ClaimsFromContext(c); ok {
		actorUID = claims.UserID
	}
	return AuditContext{
		ActorUID:      actorUID,
		IP:            c.ClientIP(),
		UserAgent:     c.Request.UserAgent(),
		TraceID:       c.GetString("traceId"),
		RequestMethod: c.Request.Method,
		RequestPath:   c.Request.URL.Path,
	}
}

func writeAppError(c *gin.Context, appErr *apperror.Error) {
	response.Fail(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
}
