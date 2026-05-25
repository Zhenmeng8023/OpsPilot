package tracecenter

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/response"
)

type ServiceContract interface {
	Search(context.Context, SearchInput) (SearchResult, *apperror.Error)
}

type Handler struct {
	service ServiceContract
}

func NewHandler(service ServiceContract) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, userAuth gin.HandlerFunc, requirePermission func(string) gin.HandlerFunc) {
	group := api.Group("/trace-center")
	group.Use(userAuth)
	group.GET("", requirePermission("traces:read"), h.search)

	traces := api.Group("/traces")
	traces.Use(userAuth)
	traces.GET("/lookup", requirePermission("traces:read"), h.lookup)
	traces.GET("/:id", requirePermission("traces:read"), h.getTrace)
	traces.GET("/:id/events", requirePermission("traces:read"), h.getTraceEvents)
}

func (h *Handler) search(c *gin.Context) {
	result, appErr := h.service.Search(c.Request.Context(), searchInput(c))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) getTrace(c *gin.Context) {
	result, appErr := h.service.Search(c.Request.Context(), SearchInput{TraceID: c.Param("id")})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) getTraceEvents(c *gin.Context) {
	result, appErr := h.service.Search(c.Request.Context(), SearchInput{TraceID: c.Param("id")})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	page := positiveInt(c.DefaultQuery("page", "1"), 1)
	pageSize := positiveInt(c.DefaultQuery("pageSize", "50"), 50)
	if pageSize > 200 {
		pageSize = 200
	}
	start := (page - 1) * pageSize
	end := start + pageSize
	total := len(result.Timeline)
	if start > total {
		start = total
	}
	if end > total {
		end = total
	}
	response.Success(c, gin.H{
		"traceId":  result.TraceID,
		"items":    result.Timeline[start:end],
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

func (h *Handler) lookup(c *gin.Context) {
	entityType := strings.TrimSpace(c.Query("entityType"))
	entityID := strings.TrimSpace(c.Query("entityId"))
	input := searchInput(c)
	switch entityType {
	case "workflow_run", "workflowRun":
		input.WorkflowRunID = entityID
	case "task_run", "taskRun":
		input.TaskRunID = entityID
	case "webhook_event", "webhookEvent":
		input.WebhookEventID = entityID
	case "trace":
		input.TraceID = entityID
	}
	if input.TraceID == "" && input.TaskRunID == "" && input.WorkflowRunID == "" && input.WebhookEventID == "" {
		response.Fail(c, http.StatusBadRequest, 400970, "entityType/entityId or query key is required")
		return
	}
	result, appErr := h.service.Search(c.Request.Context(), input)
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func searchInput(c *gin.Context) SearchInput {
	return SearchInput{
		TraceID:        c.Query("traceId"),
		TaskRunID:      c.Query("taskRunId"),
		WorkflowRunID:  c.Query("workflowRunId"),
		WebhookEventID: c.Query("webhookEventId"),
	}
}

func positiveInt(value string, fallback int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func writeAppError(c *gin.Context, appErr *apperror.Error) {
	response.FailAppError(c, appErr)
}
