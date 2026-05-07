package metrics

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/modules/agents"
	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/response"
)

type ServiceContract interface {
	Upload(context.Context, agents.AgentIdentity, []MetricInput) *apperror.Error
	List(context.Context, ListInput) ([]MetricSummary, *apperror.Error)
}

type Handler struct {
	service ServiceContract
}

type uploadRequest struct {
	Metrics []MetricInput `json:"metrics" binding:"required"`
}

func NewHandler(service ServiceContract) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, userAuth gin.HandlerFunc, agentAuth gin.HandlerFunc, requirePermission func(string) gin.HandlerFunc) {
	agentGroup := api.Group("/agent/metrics")
	agentGroup.Use(agentAuth)
	agentGroup.POST("", h.upload)

	protected := api.Group("/metrics")
	protected.Use(userAuth)
	protected.GET("/hosts", requirePermission("metric.read"), h.list)
}

func (h *Handler) upload(c *gin.Context) {
	var req uploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	identity, ok := agents.AgentIdentityFromContext(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, 401012, "invalid agent token")
		return
	}
	if appErr := h.service.Upload(c.Request.Context(), identity, req.Metrics); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) list(c *gin.Context) {
	items, appErr := h.service.List(c.Request.Context(), ListInput{
		HostID:     c.Query("hostId"),
		AgentID:    c.Query("agentId"),
		MetricCode: c.Query("metricCode"),
		Limit:      parseInt(c.DefaultQuery("limit", "200")),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, items)
}

func parseInt(value string) int {
	parsed := 0
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0
		}
		parsed = parsed*10 + int(ch-'0')
	}
	return parsed
}

func writeAppError(c *gin.Context, appErr *apperror.Error) {
	response.Fail(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
}
