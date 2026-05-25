package security

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/response"
)

type ServiceContract interface {
	PermissionDiff(context.Context, PermissionDiffInput) (PermissionDiffResult, *apperror.Error)
	SecretRotationHistory(context.Context, string) ([]SecretRotationEvent, *apperror.Error)
}

type Handler struct {
	service ServiceContract
}

func NewHandler(service ServiceContract) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, userAuth gin.HandlerFunc, requirePermission func(string) gin.HandlerFunc) {
	group := api.Group("/security")
	group.Use(userAuth)
	group.GET("/permissions/diff", requirePermission("security:review"), h.permissionDiff)
	group.GET("/secrets/:id/rotation-history", requirePermission("security:review"), h.secretRotationHistory)
}

func (h *Handler) permissionDiff(c *gin.Context) {
	result, appErr := h.service.PermissionDiff(c.Request.Context(), PermissionDiffInput{
		RoleID:          c.Query("roleId"),
		PermissionCodes: splitCSV(c.Query("permissionCodes")),
	})
	if appErr != nil {
		response.FailAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) secretRotationHistory(c *gin.Context) {
	items, appErr := h.service.SecretRotationHistory(c.Request.Context(), c.Param("id"))
	if appErr != nil {
		response.FailAppError(c, appErr)
		return
	}
	response.Success(c, items)
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}
