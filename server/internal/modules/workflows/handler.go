package workflows

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
	List(context.Context, ListInput) (DefinitionListResult, *apperror.Error)
	Get(context.Context, string) (DefinitionDetail, *apperror.Error)
	Create(context.Context, CreateInput) (DefinitionDetail, *apperror.Error)
	Update(context.Context, UpdateInput) (DefinitionDetail, *apperror.Error)
	Publish(context.Context, string, AuditContext) (DefinitionDetail, *apperror.Error)
	Run(context.Context, RunInput) (RunDetail, *apperror.Error)
	ListRuns(context.Context, ListInput) (RunListResult, *apperror.Error)
	GetRun(context.Context, string) (RunDetail, *apperror.Error)
	CancelRun(context.Context, CancelInput) (RunDetail, *apperror.Error)
	RetryRun(context.Context, RetryInput) (RunDetail, *apperror.Error)
}

type Handler struct {
	service ServiceContract
}

type definitionRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Definition  string `json:"definition" binding:"required"`
}

type runRequest struct {
	TriggerType    string `json:"triggerType"`
	IdempotencyKey string `json:"idempotencyKey"`
	Input          string `json:"input"`
}

type cancelRequest struct {
	Reason string `json:"reason"`
}

func NewHandler(service ServiceContract) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, userAuth gin.HandlerFunc, requirePermission func(string) gin.HandlerFunc) {
	definitions := api.Group("/workflows")
	definitions.Use(userAuth)
	definitions.GET("", requirePermission("workflow:read"), h.list)
	definitions.POST("", requirePermission("workflow:write"), h.create)
	definitions.GET("/:id", requirePermission("workflow:read"), h.get)
	definitions.PUT("/:id", requirePermission("workflow:write"), h.update)
	definitions.POST("/:id/publish", requirePermission("workflow:write"), h.publish)
	definitions.POST("/:id/run", requirePermission("workflow:execute"), h.run)

	runs := api.Group("/workflow-runs")
	runs.Use(userAuth)
	runs.GET("", requirePermission("workflow:read"), h.listRuns)
	runs.GET("/:id", requirePermission("workflow:read"), h.getRun)
	runs.POST("/:id/cancel", requirePermission("workflow:cancel"), h.cancelRun)
	runs.POST("/:id/retry", requirePermission("workflow:execute"), h.retryRun)
}

func (h *Handler) list(c *gin.Context) {
	result, appErr := h.service.List(c.Request.Context(), ListInput{
		Keyword:  c.Query("keyword"),
		Status:   c.Query("status"),
		Page:     int(parseUint(c.DefaultQuery("page", "1"))),
		PageSize: int(parseUint(c.DefaultQuery("pageSize", "20"))),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) get(c *gin.Context) {
	item, appErr := h.service.Get(c.Request.Context(), c.Param("id"))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
}

func (h *Handler) create(c *gin.Context) {
	var req definitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	item, appErr := h.service.Create(c.Request.Context(), CreateInput{
		Name:        req.Name,
		Description: req.Description,
		Definition:  req.Definition,
		Audit:       auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
}

func (h *Handler) update(c *gin.Context) {
	var req definitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	item, appErr := h.service.Update(c.Request.Context(), UpdateInput{
		ID:          c.Param("id"),
		Name:        req.Name,
		Description: req.Description,
		Definition:  req.Definition,
		Audit:       auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
}

func (h *Handler) publish(c *gin.Context) {
	item, appErr := h.service.Publish(c.Request.Context(), c.Param("id"), auditContext(c))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
}

func (h *Handler) run(c *gin.Context) {
	var req runRequest
	_ = c.ShouldBindJSON(&req)
	item, appErr := h.service.Run(c.Request.Context(), RunInput{
		ID:             c.Param("id"),
		TriggerType:    req.TriggerType,
		IdempotencyKey: req.IdempotencyKey,
		Input:          req.Input,
		Audit:          auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
}

func (h *Handler) listRuns(c *gin.Context) {
	result, appErr := h.service.ListRuns(c.Request.Context(), ListInput{
		Keyword:  c.Query("keyword"),
		Status:   c.Query("status"),
		Page:     int(parseUint(c.DefaultQuery("page", "1"))),
		PageSize: int(parseUint(c.DefaultQuery("pageSize", "20"))),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) getRun(c *gin.Context) {
	item, appErr := h.service.GetRun(c.Request.Context(), c.Param("id"))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
}

func (h *Handler) cancelRun(c *gin.Context) {
	var req cancelRequest
	_ = c.ShouldBindJSON(&req)
	item, appErr := h.service.CancelRun(c.Request.Context(), CancelInput{
		ID:     c.Param("id"),
		Reason: req.Reason,
		Audit:  auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
}

func (h *Handler) retryRun(c *gin.Context) {
	item, appErr := h.service.RetryRun(c.Request.Context(), RetryInput{
		ID:    c.Param("id"),
		Audit: auditContext(c),
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

func parseUint(value string) uint64 {
	parsed, _ := strconv.ParseUint(value, 10, 64)
	return parsed
}

func writeAppError(c *gin.Context, appErr *apperror.Error) {
	response.Fail(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
}
