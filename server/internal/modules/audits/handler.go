package audits

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
	List(context.Context, ListInput) (ListResult, *apperror.Error)
	Export(context.Context, ListInput, string, AuditContext) (ExportResult, *apperror.Error)
	RunRetention(context.Context, RetentionInput) (RetentionResult, *apperror.Error)
}

type Handler struct {
	service ServiceContract
}

func NewHandler(service ServiceContract) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, userAuth gin.HandlerFunc, requirePermission func(string) gin.HandlerFunc) {
	protected := api.Group("/audit-logs")
	protected.Use(userAuth)
	protected.GET("", requirePermission("audit.read"), h.list)
	protected.GET("/export", requirePermission("audit.read"), h.export)
	protected.POST("/retention/run", requirePermission("audit.read"), h.runRetention)
}

func (h *Handler) list(c *gin.Context) {
	logs, appErr := h.service.List(c.Request.Context(), ListInput{
		Action:       c.Query("action"),
		Actor:        c.Query("actor"),
		ActorType:    c.Query("actorType"),
		Result:       c.Query("result"),
		ResourceType: c.Query("resourceType"),
		ResourceID:   c.Query("resourceId"),
		TraceID:      c.Query("traceId"),
		Keyword:      c.Query("keyword"),
		CreatedFrom:  c.Query("createdFrom"),
		CreatedTo:    c.Query("createdTo"),
		Page:         parseInt(c.DefaultQuery("page", "1")),
		PageSize:     parseInt(c.DefaultQuery("pageSize", "20")),
	})
	if appErr != nil {
		response.Fail(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
		return
	}
	response.Success(c, logs)
}

func (h *Handler) export(c *gin.Context) {
	result, appErr := h.service.Export(c.Request.Context(), ListInput{
		Action:       c.Query("action"),
		Actor:        c.Query("actor"),
		ActorType:    c.Query("actorType"),
		Result:       c.Query("result"),
		ResourceType: c.Query("resourceType"),
		ResourceID:   c.Query("resourceId"),
		TraceID:      c.Query("traceId"),
		Keyword:      c.Query("keyword"),
		CreatedFrom:  c.Query("createdFrom"),
		CreatedTo:    c.Query("createdTo"),
	}, c.DefaultQuery("format", "csv"), auditContext(c))
	if appErr != nil {
		response.Fail(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
		return
	}
	c.Header("Content-Type", result.ContentType)
	c.Header("Content-Disposition", `attachment; filename="`+result.FileName+`"`)
	c.Data(http.StatusOK, result.ContentType, result.Content)
}

func (h *Handler) runRetention(c *gin.Context) {
	var req struct {
		Days   int  `json:"days"`
		DryRun bool `json:"dryRun"`
	}
	_ = c.ShouldBindJSON(&req)
	result, appErr := h.service.RunRetention(c.Request.Context(), RetentionInput{
		Days:   req.Days,
		DryRun: req.DryRun,
		Audit:  auditContext(c),
	})
	if appErr != nil {
		response.Fail(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
		return
	}
	response.Success(c, result)
}

func parseInt(value string) int {
	parsed, _ := strconv.Atoi(value)
	return parsed
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
