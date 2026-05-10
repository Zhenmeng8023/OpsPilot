package incidents

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/modules/auth"
	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/response"
)

type ServiceContract interface {
	List(context.Context, ListInput) ([]Summary, *apperror.Error)
	Get(context.Context, string) (Detail, *apperror.Error)
	UpdateLifecycle(context.Context, string, LifecycleUpdateInput) (Detail, *apperror.Error)
	Merge(context.Context, string, MergeInput) (Detail, *apperror.Error)
	Close(context.Context, string, CloseInput) (Detail, *apperror.Error)
}

type Handler struct {
	service ServiceContract
}

func NewHandler(service ServiceContract) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, userAuth gin.HandlerFunc, requirePermission func(string) gin.HandlerFunc) {
	protected := api.Group("/incidents")
	protected.Use(userAuth)
	protected.GET("", requirePermission("alert:read"), h.list)
	protected.GET("/:id", requirePermission("alert:read"), h.get)
	protected.PATCH("/:id/lifecycle", requirePermission("alert:write"), h.updateLifecycle)
	protected.POST("/:id/merge", requirePermission("alert:write"), h.merge)
	protected.POST("/:id/close", requirePermission("alert:write"), h.close)
}

func (h *Handler) list(c *gin.Context) {
	items, appErr := h.service.List(c.Request.Context(), ListInput{
		Status:   c.Query("status"),
		Severity: c.Query("severity"),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, items)
}

func (h *Handler) get(c *gin.Context) {
	item, appErr := h.service.Get(c.Request.Context(), c.Param("id"))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
}

func (h *Handler) updateLifecycle(c *gin.Context) {
	var req struct {
		Owner          string `json:"owner"`
		ImpactScope    string `json:"impactScope"`
		RootCauseClass string `json:"rootCauseClass"`
		Postmortem     string `json:"postmortem"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	item, appErr := h.service.UpdateLifecycle(c.Request.Context(), c.Param("id"), LifecycleUpdateInput{
		Owner:          req.Owner,
		ImpactScope:    req.ImpactScope,
		RootCauseClass: req.RootCauseClass,
		Postmortem:     req.Postmortem,
		Audit:          auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
}

func (h *Handler) merge(c *gin.Context) {
	var req struct {
		TargetIncidentID string `json:"targetIncidentId"`
		Reason           string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	item, appErr := h.service.Merge(c.Request.Context(), c.Param("id"), MergeInput{
		TargetIncidentID: req.TargetIncidentID,
		Reason:           req.Reason,
		Audit:            auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
}

func (h *Handler) close(c *gin.Context) {
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.ShouldBindJSON(&req)
	item, appErr := h.service.Close(c.Request.Context(), c.Param("id"), CloseInput{
		Reason: req.Reason,
		Audit:  auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
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
	response.FailAppError(c, appErr)
}
