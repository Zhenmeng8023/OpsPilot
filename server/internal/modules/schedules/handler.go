package schedules

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
	List(context.Context, ListInput) (ScheduleListResult, *apperror.Error)
	Create(context.Context, CreateInput) (ScheduleSummary, *apperror.Error)
	Preview(context.Context, PreviewInput) (PreviewResult, *apperror.Error)
	ListTriggers(context.Context, string, int) ([]TriggerSummary, *apperror.Error)
	Pause(context.Context, string, AuditContext) *apperror.Error
	Resume(context.Context, string, AuditContext) *apperror.Error
	Disable(context.Context, string, AuditContext) *apperror.Error
}

type Handler struct {
	service ServiceContract
}

type createRequest struct {
	Name          string `json:"name" binding:"required"`
	TaskID        string `json:"taskId" binding:"required"`
	CronExpr      string `json:"cronExpr" binding:"required"`
	Timezone      string `json:"timezone"`
	MisfirePolicy string `json:"misfirePolicy"`
}

type previewRequest struct {
	CronExpr string `json:"cronExpr" binding:"required"`
	Timezone string `json:"timezone"`
	Count    int    `json:"count"`
}

func NewHandler(service ServiceContract) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, userAuth gin.HandlerFunc, requirePermission func(string) gin.HandlerFunc) {
	protected := api.Group("/schedules")
	protected.Use(userAuth)
	protected.GET("", requirePermission("schedule:read"), h.list)
	protected.POST("", requirePermission("schedule:write"), h.create)
	protected.POST("/preview", requirePermission("schedule:read"), h.preview)
	protected.GET("/:id/triggers", requirePermission("schedule:read"), h.listTriggers)
	protected.POST("/:id/pause", requirePermission("schedule:write"), h.pause)
	protected.POST("/:id/resume", requirePermission("schedule:write"), h.resume)
	protected.POST("/:id/disable", requirePermission("schedule:write"), h.disable)
}

func (h *Handler) list(c *gin.Context) {
	result, appErr := h.service.List(c.Request.Context(), ListInput{
		Keyword:  c.Query("keyword"),
		Status:   c.Query("status"),
		TaskID:   c.Query("taskId"),
		Page:     int(parseUint(c.DefaultQuery("page", "1"))),
		PageSize: int(parseUint(c.DefaultQuery("pageSize", "20"))),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) create(c *gin.Context) {
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	created, appErr := h.service.Create(c.Request.Context(), CreateInput{
		Name:          req.Name,
		TaskID:        req.TaskID,
		CronExpr:      req.CronExpr,
		Timezone:      req.Timezone,
		MisfirePolicy: req.MisfirePolicy,
		Audit:         auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, created)
}

func (h *Handler) preview(c *gin.Context) {
	var req previewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	result, appErr := h.service.Preview(c.Request.Context(), PreviewInput{
		CronExpr: req.CronExpr,
		Timezone: req.Timezone,
		Count:    req.Count,
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) listTriggers(c *gin.Context) {
	items, appErr := h.service.ListTriggers(c.Request.Context(), c.Param("id"), int(parseUint(c.DefaultQuery("limit", "20"))))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, items)
}

func (h *Handler) pause(c *gin.Context) {
	if appErr := h.service.Pause(c.Request.Context(), c.Param("id"), auditContext(c)); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) resume(c *gin.Context) {
	if appErr := h.service.Resume(c.Request.Context(), c.Param("id"), auditContext(c)); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) disable(c *gin.Context) {
	if appErr := h.service.Disable(c.Request.Context(), c.Param("id"), auditContext(c)); appErr != nil {
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

func parseUint(value string) uint64 {
	parsed, _ := strconv.ParseUint(value, 10, 64)
	return parsed
}

func writeAppError(c *gin.Context, appErr *apperror.Error) {
	response.Fail(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
}
