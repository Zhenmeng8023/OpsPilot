package metrics

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/modules/agents"
	"opspilot/server/internal/modules/auth"
	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/response"
)

type ServiceContract interface {
	Upload(context.Context, agents.AgentIdentity, []MetricInput) *apperror.Error
	List(context.Context, ListInput) ([]MetricSummary, *apperror.Error)
	ListTrends(context.Context, TrendInput) ([]MetricTrendSeries, *apperror.Error)
	ListDashboards(context.Context) ([]DashboardSummary, *apperror.Error)
	CreateDashboard(context.Context, DashboardInput) (DashboardSummary, *apperror.Error)
	UpdateDashboard(context.Context, string, DashboardInput) (DashboardSummary, *apperror.Error)
	RunRollup(context.Context, RollupInput) (RollupResult, *apperror.Error)
	RunRetention(context.Context, RetentionInput) (RetentionResult, *apperror.Error)
}

type Handler struct {
	service ServiceContract
}

type uploadRequest struct {
	Metrics []MetricInput `json:"metrics" binding:"required"`
}

type dashboardRequest struct {
	Name        string `json:"name" binding:"required"`
	MetricCode  string `json:"metricCode"`
	HostID      string `json:"hostId"`
	AgentID     string `json:"agentId"`
	RangeHours  int    `json:"rangeHours"`
	PointLimit  int    `json:"pointLimit"`
	Granularity string `json:"granularity"`
	Status      string `json:"status"`
}

type rollupRequest struct {
	Interval string `json:"interval"`
	Hours    int    `json:"hours"`
}

type retentionRequest struct {
	DetailDays int  `json:"detailDays"`
	RollupDays int  `json:"rollupDays"`
	DryRun     bool `json:"dryRun"`
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
	protected.GET("/hosts", requirePermission("metric:read"), h.list)
	protected.GET("/hosts/trends", requirePermission("metric:read"), h.listTrends)
	protected.GET("/dashboards", requirePermission("metric:read"), h.listDashboards)
	protected.POST("/dashboards", requirePermission("metric:write"), h.createDashboard)
	protected.PUT("/dashboards/:id", requirePermission("metric:write"), h.updateDashboard)
	protected.POST("/rollups/run", requirePermission("metric:write"), h.runRollup)
	protected.POST("/retention/run", requirePermission("metric:write"), h.runRetention)
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

func (h *Handler) listTrends(c *gin.Context) {
	items, appErr := h.service.ListTrends(c.Request.Context(), TrendInput{
		HostID:      c.Query("hostId"),
		AgentID:     c.Query("agentId"),
		MetricCode:  c.Query("metricCode"),
		Hours:       parseInt(c.DefaultQuery("hours", "24")),
		Limit:       parseInt(c.DefaultQuery("limit", "120")),
		Granularity: c.DefaultQuery("granularity", "auto"),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, items)
}

func (h *Handler) listDashboards(c *gin.Context) {
	items, appErr := h.service.ListDashboards(c.Request.Context())
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, items)
}

func (h *Handler) createDashboard(c *gin.Context) {
	var req dashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	item, appErr := h.service.CreateDashboard(c.Request.Context(), DashboardInput{
		Name:        req.Name,
		MetricCode:  req.MetricCode,
		HostID:      req.HostID,
		AgentID:     req.AgentID,
		RangeHours:  req.RangeHours,
		PointLimit:  req.PointLimit,
		Granularity: req.Granularity,
		Status:      req.Status,
		ActorUID:    actorUID(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
}

func (h *Handler) updateDashboard(c *gin.Context) {
	var req dashboardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	item, appErr := h.service.UpdateDashboard(c.Request.Context(), c.Param("id"), DashboardInput{
		Name:        req.Name,
		MetricCode:  req.MetricCode,
		HostID:      req.HostID,
		AgentID:     req.AgentID,
		RangeHours:  req.RangeHours,
		PointLimit:  req.PointLimit,
		Granularity: req.Granularity,
		Status:      req.Status,
		ActorUID:    actorUID(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
}

func (h *Handler) runRollup(c *gin.Context) {
	var req rollupRequest
	_ = c.ShouldBindJSON(&req)
	result, appErr := h.service.RunRollup(c.Request.Context(), RollupInput{Interval: req.Interval, Hours: req.Hours})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) runRetention(c *gin.Context) {
	var req retentionRequest
	_ = c.ShouldBindJSON(&req)
	result, appErr := h.service.RunRetention(c.Request.Context(), RetentionInput{
		DetailDays: req.DetailDays,
		RollupDays: req.RollupDays,
		DryRun:     req.DryRun,
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
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

func actorUID(c *gin.Context) string {
	if claims, ok := auth.ClaimsFromContext(c); ok {
		return claims.UserID
	}
	return ""
}
