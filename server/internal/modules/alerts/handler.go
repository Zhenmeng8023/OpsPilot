package alerts

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/modules/auth"
	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/response"
)

type ServiceContract interface {
	ListRules(context.Context) ([]AlertRuleSummary, *apperror.Error)
	CreateRule(context.Context, CreateRuleInput) (AlertRuleSummary, *apperror.Error)
	ListAlerts(context.Context, string) ([]AlertSummary, *apperror.Error)
	Resolve(context.Context, string, AuditContext) *apperror.Error
}

type Handler struct {
	service ServiceContract
}

type createRuleRequest struct {
	Name            string  `json:"name" binding:"required"`
	MetricCode      string  `json:"metricCode" binding:"required"`
	Operator        string  `json:"operator" binding:"required"`
	Threshold       float64 `json:"threshold"`
	DurationSeconds uint    `json:"durationSeconds"`
	Severity        string  `json:"severity"`
}

func NewHandler(service ServiceContract) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, userAuth gin.HandlerFunc, requirePermission func(string) gin.HandlerFunc) {
	protected := api.Group("")
	protected.Use(userAuth)
	protected.GET("/alert-rules", requirePermission("alert:read"), h.listRules)
	protected.POST("/alert-rules", requirePermission("alert:write"), h.createRule)
	protected.GET("/alerts", requirePermission("alert:read"), h.listAlerts)
	protected.POST("/alerts/:id/resolve", requirePermission("alert:write"), h.resolve)
}

func (h *Handler) listRules(c *gin.Context) {
	rules, appErr := h.service.ListRules(c.Request.Context())
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, rules)
}

func (h *Handler) createRule(c *gin.Context) {
	var req createRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	rule, appErr := h.service.CreateRule(c.Request.Context(), CreateRuleInput{
		Name:            req.Name,
		MetricCode:      req.MetricCode,
		Operator:        req.Operator,
		Threshold:       req.Threshold,
		DurationSeconds: req.DurationSeconds,
		Severity:        req.Severity,
		Audit:           auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, rule)
}

func (h *Handler) listAlerts(c *gin.Context) {
	alerts, appErr := h.service.ListAlerts(c.Request.Context(), c.Query("status"))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, alerts)
}

func (h *Handler) resolve(c *gin.Context) {
	if appErr := h.service.Resolve(c.Request.Context(), c.Param("id"), auditContext(c)); appErr != nil {
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
