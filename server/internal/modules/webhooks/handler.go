package webhooks

import (
	"context"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/modules/auth"
	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/response"
)

type ServiceContract interface {
	ListSources(context.Context) ([]SourceSummary, *apperror.Error)
	CreateSource(context.Context, CreateSourceInput) (SourceDetail, *apperror.Error)
	RotateSourceSecrets(context.Context, RotateSourceSecretsInput) (SourceDetail, *apperror.Error)
	PauseSource(context.Context, string, AuditContext) *apperror.Error
	ResumeSource(context.Context, string, AuditContext) *apperror.Error
	DisableSource(context.Context, string, AuditContext) *apperror.Error
	ListRules(context.Context) ([]RuleSummary, *apperror.Error)
	CreateRule(context.Context, CreateRuleInput) (RuleSummary, *apperror.Error)
	UpdateRule(context.Context, UpdateRuleInput) (RuleSummary, *apperror.Error)
	PauseRule(context.Context, string, AuditContext) *apperror.Error
	ResumeRule(context.Context, string, AuditContext) *apperror.Error
	DisableRule(context.Context, string, AuditContext) *apperror.Error
	ListEvents(context.Context, ListEventsInput) (EventListResult, *apperror.Error)
	GetEvent(context.Context, EventDetailInput) (EventDetail, *apperror.Error)
	SimulateMatcher(context.Context, MatcherSimulationInput) (MatcherSimulationResult, *apperror.Error)
	ReplayEvent(context.Context, ReplayEventInput) (TriggerResult, *apperror.Error)
	Trigger(context.Context, TriggerInput) (TriggerResult, *apperror.Error)
}

type Handler struct {
	service ServiceContract
}

type createSourceRequest struct {
	Name       string `json:"name" binding:"required"`
	SourceType string `json:"sourceType"`
}

type createRuleRequest struct {
	SourceID   string   `json:"sourceId" binding:"required"`
	TargetType string   `json:"targetType"`
	TaskID     string   `json:"taskId"`
	WorkflowID string   `json:"workflowId"`
	Name       string   `json:"name" binding:"required"`
	EventType  string   `json:"eventType"`
	Matcher    *Matcher `json:"matcher"`
}

type updateRuleRequest struct {
	Name      string   `json:"name" binding:"required"`
	EventType string   `json:"eventType"`
	Matcher   *Matcher `json:"matcher"`
}

type matcherSimulationRequest struct {
	RuleID    string            `json:"ruleId"`
	EventType string            `json:"eventType"`
	Matcher   *Matcher          `json:"matcher"`
	Headers   map[string]string `json:"headers"`
	Payload   string            `json:"payload"`
}

type replayEventRequest struct {
	SimulateOnly bool   `json:"simulateOnly"`
	IdempotencyKey string `json:"idempotencyKey"`
}

func NewHandler(service ServiceContract) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, userAuth gin.HandlerFunc, requirePermission func(string) gin.HandlerFunc) {
	api.POST("/webhooks/trigger/:token", h.trigger)
	api.GET("/webhook-events", userAuth, requirePermission("webhook:read"), h.listEvents)

	protected := api.Group("/webhooks")
	protected.Use(userAuth)
	protected.GET("/sources", requirePermission("webhook:read"), h.listSources)
	protected.POST("/sources", requirePermission("webhook:manage"), h.createSource)
	protected.POST("/sources/:id/rotate-secret", requirePermission("webhook:manage"), h.rotateSourceSecret)
	protected.POST("/sources/:id/pause", requirePermission("webhook:manage"), h.pauseSource)
	protected.POST("/sources/:id/resume", requirePermission("webhook:manage"), h.resumeSource)
	protected.POST("/sources/:id/disable", requirePermission("webhook:manage"), h.disableSource)
	protected.GET("/rules", requirePermission("webhook:read"), h.listRules)
	protected.POST("/rules", requirePermission("webhook:manage"), h.createRule)
	protected.PUT("/rules/:id", requirePermission("webhook:manage"), h.updateRule)
	protected.POST("/rules/:id/pause", requirePermission("webhook:manage"), h.pauseRule)
	protected.POST("/rules/:id/resume", requirePermission("webhook:manage"), h.resumeRule)
	protected.POST("/rules/:id/disable", requirePermission("webhook:manage"), h.disableRule)
	protected.POST("/matcher/simulate", requirePermission("webhook:manage"), h.simulateMatcher)
	protected.GET("/events/:id", requirePermission("webhook:read"), h.getEvent)
	protected.POST("/events/:id/replay", requirePermission("webhook:manage"), h.replayEvent)
}

func (h *Handler) listSources(c *gin.Context) {
	sources, appErr := h.service.ListSources(c.Request.Context())
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, sources)
}

func (h *Handler) createSource(c *gin.Context) {
	var req createSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	source, appErr := h.service.CreateSource(c.Request.Context(), CreateSourceInput{
		Name:       req.Name,
		SourceType: req.SourceType,
		Audit:      auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, source)
}

func (h *Handler) rotateSourceSecret(c *gin.Context) {
	var req struct {
		RotateToken bool `json:"rotateToken"`
	}
	_ = c.ShouldBindJSON(&req)
	source, appErr := h.service.RotateSourceSecrets(c.Request.Context(), RotateSourceSecretsInput{
		SourceID:    c.Param("id"),
		RotateToken: req.RotateToken,
		Audit:       auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, source)
}

func (h *Handler) pauseSource(c *gin.Context) {
	if appErr := h.service.PauseSource(c.Request.Context(), c.Param("id"), auditContext(c)); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) resumeSource(c *gin.Context) {
	if appErr := h.service.ResumeSource(c.Request.Context(), c.Param("id"), auditContext(c)); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) disableSource(c *gin.Context) {
	if appErr := h.service.DisableSource(c.Request.Context(), c.Param("id"), auditContext(c)); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) listRules(c *gin.Context) {
	rules, appErr := h.service.ListRules(c.Request.Context())
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, rules)
}

func (h *Handler) listEvents(c *gin.Context) {
	result, appErr := h.service.ListEvents(c.Request.Context(), ListEventsInput{
		SourceID:     c.Query("sourceId"),
		Status:       c.Query("status"),
		DeliveryID:   c.Query("deliveryId"),
		ReceivedFrom: c.Query("receivedFrom"),
		ReceivedTo:   c.Query("receivedTo"),
		Page:         parseInt(c.DefaultQuery("page", "1")),
		PageSize:     parseInt(c.DefaultQuery("pageSize", "20")),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) getEvent(c *gin.Context) {
	result, appErr := h.service.GetEvent(c.Request.Context(), EventDetailInput{EventID: c.Param("id")})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) createRule(c *gin.Context) {
	var req createRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	rule, appErr := h.service.CreateRule(c.Request.Context(), CreateRuleInput{
		SourceID:   req.SourceID,
		TargetType: req.TargetType,
		TaskID:     req.TaskID,
		WorkflowID: req.WorkflowID,
		Name:       req.Name,
		EventType:  req.EventType,
		Matcher:    req.Matcher,
		Audit:      auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, rule)
}

func (h *Handler) updateRule(c *gin.Context) {
	var req updateRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	rule, appErr := h.service.UpdateRule(c.Request.Context(), UpdateRuleInput{
		ID:        c.Param("id"),
		Name:      req.Name,
		EventType: req.EventType,
		Matcher:   req.Matcher,
		Audit:     auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, rule)
}

func (h *Handler) pauseRule(c *gin.Context) {
	if appErr := h.service.PauseRule(c.Request.Context(), c.Param("id"), auditContext(c)); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) resumeRule(c *gin.Context) {
	if appErr := h.service.ResumeRule(c.Request.Context(), c.Param("id"), auditContext(c)); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) disableRule(c *gin.Context) {
	if appErr := h.service.DisableRule(c.Request.Context(), c.Param("id"), auditContext(c)); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) simulateMatcher(c *gin.Context) {
	var req matcherSimulationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	result, appErr := h.service.SimulateMatcher(c.Request.Context(), MatcherSimulationInput{
		RuleID:    req.RuleID,
		EventType: req.EventType,
		Matcher:   req.Matcher,
		Headers:   req.Headers,
		Payload:   req.Payload,
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) replayEvent(c *gin.Context) {
	var req replayEventRequest
	_ = c.ShouldBindJSON(&req)
	result, appErr := h.service.ReplayEvent(c.Request.Context(), ReplayEventInput{
		EventID:       c.Param("id"),
		SimulateOnly:  req.SimulateOnly,
		IdempotencyKey: req.IdempotencyKey,
		Audit:         auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) trigger(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	headers := map[string]string{}
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	result, appErr := h.service.Trigger(c.Request.Context(), TriggerInput{
		Token:           c.Param("token"),
		EventType:       c.GetHeader("X-Event-Type"),
		DeliveryID:      c.GetHeader("X-Delivery-Id"),
		SignatureHeader: firstHeaderName(c, "X-OpsPilot-Signature", "X-Hub-Signature-256"),
		Signature:       firstHeader(c, "X-OpsPilot-Signature", "X-Hub-Signature-256"),
		Timestamp:       c.GetHeader("X-OpsPilot-Timestamp"),
		Nonce:           c.GetHeader("X-OpsPilot-Nonce"),
		RemoteIP:        c.ClientIP(),
		Headers:         headers,
		Body:            body,
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func firstHeader(c *gin.Context, names ...string) string {
	for _, name := range names {
		if value := c.GetHeader(name); value != "" {
			return value
		}
	}
	return ""
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

func firstHeaderName(c *gin.Context, names ...string) string {
	for _, name := range names {
		if value := c.GetHeader(name); value != "" {
			return name
		}
	}
	return ""
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
