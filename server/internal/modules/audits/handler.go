package audits

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/response"
)

type ServiceContract interface {
	List(context.Context, ListInput) (ListResult, *apperror.Error)
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
}

func (h *Handler) list(c *gin.Context) {
	logs, appErr := h.service.List(c.Request.Context(), ListInput{
		Action:       c.Query("action"),
		ActorType:    c.Query("actorType"),
		Result:       c.Query("result"),
		ResourceType: c.Query("resourceType"),
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

func parseInt(value string) int {
	parsed, _ := strconv.Atoi(value)
	return parsed
}
