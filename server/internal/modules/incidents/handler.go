package incidents

import (
	"context"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/response"
)

type ServiceContract interface {
	List(context.Context, ListInput) ([]Summary, *apperror.Error)
	Get(context.Context, string) (Detail, *apperror.Error)
}

type Handler struct {
	service ServiceContract
}

func NewHandler(service ServiceContract) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, userAuth gin.HandlerFunc, requirePermission func(string) gin.HandlerFunc) {
	protected := api.Group("/incidents")
	protected.Use(userAuth)
	protected.GET("", requirePermission("alert:read"), h.list)
	protected.GET("/:id", requirePermission("alert:read"), h.get)
}

func (h *Handler) list(c *gin.Context) {
	items, appErr := h.service.List(c.Request.Context(), ListInput{
		Status:   c.Query("status"),
		Severity: c.Query("severity"),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, items)
}

func (h *Handler) get(c *gin.Context) {
	item, appErr := h.service.Get(c.Request.Context(), c.Param("id"))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, item)
}

func writeAppError(c *gin.Context, appErr *apperror.Error) {
	response.Fail(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
}
