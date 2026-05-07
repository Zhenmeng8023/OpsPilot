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
	UpdateRule(context.Context, string, UpdateRuleInput) (AlertRuleSummary, *apperror.Error)
	UpdateRuleStatus(context.Context, string, string, AuditContext) *apperror.Error
	ListAlerts(context.Context, ListAlertsInput) ([]AlertSummary, *apperror.Error)
	ListAlertEvents(context.Context, string, string) ([]AlertEventSummary, *apperror.Error)
	ListAlertHistory(context.Context, AlertHistoryInput) ([]AlertHistoryPoint, *apperror.Error)
	Acknowledge(context.Context, string, AuditContext) *apperror.Error
	Silence(context.Context, string, SilenceInput) *apperror.Error
	Unsilence(context.Context, string, AuditContext) *apperror.Error
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
	CooldownSeconds uint    `json:"cooldownSeconds"`
	Severity        string  `json:"severity"`
}

type silenceAlertRequest struct {
	DurationSeconds uint   `json:"durationSeconds"`
	Reason          string `json:"reason"`
}

func NewHandler(service ServiceContract) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, userAuth gin.HandlerFunc, requirePermission func(string) gin.HandlerFunc) {
	protected := api.Group("")
	protected.Use(userAuth)
	protected.GET("/alert-rules", requirePermission("alert:read"), h.listRules)
	protected.POST("/alert-rules", requirePermission("alert:write"), h.createRule)
	protected.PUT("/alert-rules/:id", requirePermission("alert:write"), h.updateRule)
	protected.POST("/alert-rules/:id/pause", requirePermission("alert:write"), h.pauseRule)
	protected.POST("/alert-rules/:id/resume", requirePermission("alert:write"), h.resumeRule)
	protected.POST("/alert-rules/:id/disable", requirePermission("alert:write"), h.disableRule)
	protected.GET("/alerts", requirePermission("alert:read"), h.listAlerts)
	protected.GET("/alerts/history", requirePermission("alert:read"), h.listAlertHistory)
	protected.GET("/alerts/:id/events", requirePermission("alert:read"), h.listAlertEvents)
	protected.POST("/alerts/:id/ack", requirePermission("alert:write"), h.acknowledge)
	protected.POST("/alerts/:id/silence", requirePermission("alert:write"), h.silence)
	protected.POST("/alerts/:id/unsilence", requirePermission("alert:write"), h.unsilence)
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
		CooldownSeconds: req.CooldownSeconds,
		Severity:        req.Severity,
		Audit:           auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, rule)
}

func (h *Handler) updateRule(c *gin.Context) {
	var req createRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	rule, appErr := h.service.UpdateRule(c.Request.Context(), c.Param("id"), UpdateRuleInput{
		Name:            req.Name,
		MetricCode:      req.MetricCode,
		Operator:        req.Operator,
		Threshold:       req.Threshold,
		DurationSeconds: req.DurationSeconds,
		CooldownSeconds: req.CooldownSeconds,
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
	alerts, appErr := h.service.ListAlerts(c.Request.Context(), ListAlertsInput{
		Status:   c.Query("status"),
		Severity: c.Query("severity"),
		RuleID:   c.Query("ruleId"),
		HostID:   c.Query("hostId"),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, alerts)
}

func (h *Handler) listAlertHistory(c *gin.Context) {
	items, appErr := h.service.ListAlertHistory(c.Request.Context(), AlertHistoryInput{
		Hours:         parseInt(c.DefaultQuery("hours", "24")),
		BucketMinutes: parseInt(c.DefaultQuery("bucketMinutes", "60")),
		Severity:      c.Query("severity"),
		RuleID:        c.Query("ruleId"),
		HostID:        c.Query("hostId"),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, items)
}

func (h *Handler) listAlertEvents(c *gin.Context) {
	events, appErr := h.service.ListAlertEvents(c.Request.Context(), c.Param("id"), c.Query("eventType"))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, events)
}

func (h *Handler) pauseRule(c *gin.Context) {
	if appErr := h.service.UpdateRuleStatus(c.Request.Context(), c.Param("id"), "paused", auditContext(c)); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) resumeRule(c *gin.Context) {
	if appErr := h.service.UpdateRuleStatus(c.Request.Context(), c.Param("id"), "active", auditContext(c)); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) disableRule(c *gin.Context) {
	if appErr := h.service.UpdateRuleStatus(c.Request.Context(), c.Param("id"), "disabled", auditContext(c)); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) acknowledge(c *gin.Context) {
	if appErr := h.service.Acknowledge(c.Request.Context(), c.Param("id"), auditContext(c)); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) silence(c *gin.Context) {
	var req silenceAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	if appErr := h.service.Silence(c.Request.Context(), c.Param("id"), SilenceInput{
		DurationSeconds: req.DurationSeconds,
		Reason:          req.Reason,
		Audit:           auditContext(c),
	}); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) unsilence(c *gin.Context) {
	if appErr := h.service.Unsilence(c.Request.Context(), c.Param("id"), auditContext(c)); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
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
