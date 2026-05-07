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
	ListRules(context.Context) ([]RuleSummary, *apperror.Error)
	CreateRule(context.Context, CreateRuleInput) (RuleSummary, *apperror.Error)
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
	SourceID  string `json:"sourceId" binding:"required"`
	TaskID    string `json:"taskId" binding:"required"`
	Name      string `json:"name" binding:"required"`
	EventType string `json:"eventType"`
}

func NewHandler(service ServiceContract) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, userAuth gin.HandlerFunc, requirePermission func(string) gin.HandlerFunc) {
	api.POST("/webhooks/trigger/:token", h.trigger)

	protected := api.Group("/webhooks")
	protected.Use(userAuth)
	protected.GET("/sources", requirePermission("webhook:read"), h.listSources)
	protected.POST("/sources", requirePermission("webhook:manage"), h.createSource)
	protected.GET("/rules", requirePermission("webhook:read"), h.listRules)
	protected.POST("/rules", requirePermission("webhook:manage"), h.createRule)
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
		SourceID:  req.SourceID,
		TaskID:    req.TaskID,
		Name:      req.Name,
		EventType: req.EventType,
		Audit:     auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, rule)
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
		Token:      c.Param("token"),
		EventType:  c.GetHeader("X-Event-Type"),
		DeliveryID: c.GetHeader("X-Delivery-Id"),
		Signature:  firstHeader(c, "X-OpsPilot-Signature", "X-Hub-Signature-256"),
		RemoteIP:   c.ClientIP(),
		Headers:    headers,
		Body:       body,
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
