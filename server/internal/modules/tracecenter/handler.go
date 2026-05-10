package tracecenter

import (
	"context"

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
	group.GET("", requirePermission("audit.read"), h.search)
}

func (h *Handler) search(c *gin.Context) {
	result, appErr := h.service.Search(c.Request.Context(), SearchInput{
		TraceID:        c.Query("traceId"),
		TaskRunID:      c.Query("taskRunId"),
		WorkflowRunID:  c.Query("workflowRunId"),
		WebhookEventID: c.Query("webhookEventId"),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func writeAppError(c *gin.Context, appErr *apperror.Error) {
	response.FailAppError(c, appErr)
}

