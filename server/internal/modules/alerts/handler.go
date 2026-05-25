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
	ListSuppressionRules(context.Context) ([]SuppressionRuleSummary, *apperror.Error)
	CreateSuppressionRule(context.Context, SuppressionRuleInput) (SuppressionRuleSummary, *apperror.Error)
	UpdateSuppressionRule(context.Context, string, SuppressionRuleInput) (SuppressionRuleSummary, *apperror.Error)
	ListRoutingPolicies(context.Context) ([]RoutingPolicySummary, *apperror.Error)
	CreateRoutingPolicy(context.Context, RoutingPolicyInput) (RoutingPolicySummary, *apperror.Error)
	UpdateRoutingPolicy(context.Context, string, RoutingPolicyInput) (RoutingPolicySummary, *apperror.Error)
	ListAlertGroups(context.Context, ListAlertGroupsInput) ([]AlertGroupSummary, *apperror.Error)
	ListAlerts(context.Context, ListAlertsInput) ([]AlertSummary, *apperror.Error)
	ListAlertEvents(context.Context, string, string) ([]AlertEventSummary, *apperror.Error)
	ListAlertHistory(context.Context, AlertHistoryInput) ([]AlertHistoryPoint, *apperror.Error)
	ListNoisyRules(context.Context, NoisyRuleInput) ([]NoisyRuleSummary, *apperror.Error)
	NoiseReport(context.Context, NoisyRuleInput) (NoiseReportResult, *apperror.Error)
	RoutingExplanation(context.Context, string) (RoutingExplanationResult, *apperror.Error)
	SuppressionDryRun(context.Context, SuppressionDryRunInput) (SuppressionDryRunResult, *apperror.Error)
	RoutingDryRun(context.Context, RoutingDryRunInput) (RoutingDryRunResult, *apperror.Error)
	ListNoiseTrends(context.Context, NoiseTrendInput) ([]NoiseTrendPoint, *apperror.Error)
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

type suppressionRuleRequest struct {
	Name        string `json:"name" binding:"required"`
	RuleID      string `json:"ruleId"`
	HostID      string `json:"hostId"`
	HostGroupID string `json:"hostGroupId"`
	Severity    string `json:"severity"`
	StartsAt    string `json:"startsAt"`
	EndsAt      string `json:"endsAt"`
	Reason      string `json:"reason"`
	Status      string `json:"status"`
}

type routingPolicyRequest struct {
	Name        string `json:"name" binding:"required"`
	RuleID      string `json:"ruleId"`
	HostID      string `json:"hostId"`
	HostGroupID string `json:"hostGroupId"`
	Severity    string `json:"severity"`
	ChannelID   string `json:"channelId" binding:"required"`
	Status      string `json:"status"`
}

type alertDryRunRequest struct {
	RuleID      string `json:"ruleId"`
	HostID      string `json:"hostId"`
	HostGroupID string `json:"hostGroupId"`
	Severity    string `json:"severity"`
	ChannelID   string `json:"channelId"`
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
	protected.GET("/alert-suppression-rules", requirePermission("alert:read"), h.listSuppressionRules)
	protected.POST("/alert-suppression-rules", requirePermission("alert:write"), h.createSuppressionRule)
	protected.PUT("/alert-suppression-rules/:id", requirePermission("alert:write"), h.updateSuppressionRule)
	protected.GET("/alert-routing-policies", requirePermission("alert:read"), h.listRoutingPolicies)
	protected.POST("/alert-routing-policies", requirePermission("alert:write"), h.createRoutingPolicy)
	protected.PUT("/alert-routing-policies/:id", requirePermission("alert:write"), h.updateRoutingPolicy)
	protected.GET("/alert-groups", requirePermission("alert:read"), h.listAlertGroups)
	protected.GET("/alerts", requirePermission("alert:read"), h.listAlerts)
	protected.GET("/alerts/history", requirePermission("alert:read"), h.listAlertHistory)
	protected.GET("/alerts/noisy-rules", requirePermission("alert:read"), h.listNoisyRules)
	protected.GET("/alerts/noise-report", requirePermission("alert:read"), h.noiseReport)
	protected.GET("/alerts/noise-trends", requirePermission("alert:read"), h.listNoiseTrends)
	protected.POST("/alerts/suppression-dry-run", requirePermission("alert:read"), h.suppressionDryRun)
	protected.POST("/alerts/routing-dry-run", requirePermission("alert:read"), h.routingDryRun)
	protected.GET("/alerts/:id/events", requirePermission("alert:read"), h.listAlertEvents)
	protected.GET("/alerts/:id/routing-explanation", requirePermission("alert:read"), h.routingExplanation)
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

func (h *Handler) listAlertGroups(c *gin.Context) {
	items, appErr := h.service.ListAlertGroups(c.Request.Context(), ListAlertGroupsInput{
		Status:      c.Query("status"),
		Severity:    c.Query("severity"),
		RuleID:      c.Query("ruleId"),
		HostGroupID: c.Query("hostGroupId"),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, items)
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

func (h *Handler) listNoisyRules(c *gin.Context) {
	items, appErr := h.service.ListNoisyRules(c.Request.Context(), NoisyRuleInput{
		Hours: parseInt(c.DefaultQuery("hours", "24")),
		Limit: parseInt(c.DefaultQuery("limit", "10")),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, items)
}

func (h *Handler) noiseReport(c *gin.Context) {
	item, appErr := h.service.NoiseReport(c.Request.Context(), NoisyRuleInput{
		Hours: parseInt(c.DefaultQuery("hours", "24")),
		Limit: parseInt(c.DefaultQuery("limit", "10")),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
}

func (h *Handler) listNoiseTrends(c *gin.Context) {
	items, appErr := h.service.ListNoiseTrends(c.Request.Context(), NoiseTrendInput{
		Hours: parseInt(c.DefaultQuery("hours", "24")),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, items)
}

func (h *Handler) suppressionDryRun(c *gin.Context) {
	var req alertDryRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	result, appErr := h.service.SuppressionDryRun(c.Request.Context(), SuppressionDryRunInput{
		RuleID:      req.RuleID,
		HostID:      req.HostID,
		HostGroupID: req.HostGroupID,
		Severity:    req.Severity,
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) routingDryRun(c *gin.Context) {
	var req alertDryRunRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	result, appErr := h.service.RoutingDryRun(c.Request.Context(), RoutingDryRunInput{
		RuleID:      req.RuleID,
		HostID:      req.HostID,
		HostGroupID: req.HostGroupID,
		Severity:    req.Severity,
		ChannelID:   req.ChannelID,
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) listAlertEvents(c *gin.Context) {
	events, appErr := h.service.ListAlertEvents(c.Request.Context(), c.Param("id"), c.Query("eventType"))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, events)
}

func (h *Handler) routingExplanation(c *gin.Context) {
	item, appErr := h.service.RoutingExplanation(c.Request.Context(), c.Param("id"))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
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

func (h *Handler) listSuppressionRules(c *gin.Context) {
	items, appErr := h.service.ListSuppressionRules(c.Request.Context())
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, items)
}

func (h *Handler) createSuppressionRule(c *gin.Context) {
	var req suppressionRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	item, appErr := h.service.CreateSuppressionRule(c.Request.Context(), suppressionInput(req, auditContext(c)))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
}

func (h *Handler) updateSuppressionRule(c *gin.Context) {
	var req suppressionRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	item, appErr := h.service.UpdateSuppressionRule(c.Request.Context(), c.Param("id"), suppressionInput(req, auditContext(c)))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
}

func (h *Handler) listRoutingPolicies(c *gin.Context) {
	items, appErr := h.service.ListRoutingPolicies(c.Request.Context())
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, items)
}

func (h *Handler) createRoutingPolicy(c *gin.Context) {
	var req routingPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	item, appErr := h.service.CreateRoutingPolicy(c.Request.Context(), routingInput(req, auditContext(c)))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
}

func (h *Handler) updateRoutingPolicy(c *gin.Context) {
	var req routingPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	item, appErr := h.service.UpdateRoutingPolicy(c.Request.Context(), c.Param("id"), routingInput(req, auditContext(c)))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
}

func (h *Handler) acknowledge(c *gin.Context) {
	if appErr := h.service.Acknowledge(c.Request.Context(), c.Param("id"), auditContext(c)); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func suppressionInput(req suppressionRuleRequest, audit AuditContext) SuppressionRuleInput {
	return SuppressionRuleInput{
		Name:        req.Name,
		RuleID:      req.RuleID,
		HostID:      req.HostID,
		HostGroupID: req.HostGroupID,
		Severity:    req.Severity,
		StartsAt:    req.StartsAt,
		EndsAt:      req.EndsAt,
		Reason:      req.Reason,
		Status:      req.Status,
		Audit:       audit,
	}
}

func routingInput(req routingPolicyRequest, audit AuditContext) RoutingPolicyInput {
	return RoutingPolicyInput{
		Name:        req.Name,
		RuleID:      req.RuleID,
		HostID:      req.HostID,
		HostGroupID: req.HostGroupID,
		Severity:    req.Severity,
		ChannelID:   req.ChannelID,
		Status:      req.Status,
		Audit:       audit,
	}
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
	response.FailAppError(c, appErr)
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
