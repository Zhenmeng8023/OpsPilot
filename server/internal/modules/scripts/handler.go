package scripts

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
	List(context.Context, ListInput) (ScriptListResult, *apperror.Error)
	Get(context.Context, string) (ScriptDetail, *apperror.Error)
	Create(context.Context, CreateInput) (ScriptDetail, *apperror.Error)
	Update(context.Context, UpdateInput) (ScriptDetail, *apperror.Error)
	Disable(context.Context, string, AuditContext) *apperror.Error
	ListApprovals(context.Context, string) ([]ApprovalSummary, *apperror.Error)
	RequestApproval(context.Context, string, AuditContext) (ApprovalSummary, *apperror.Error)
	DecideApproval(context.Context, uint64, bool, string, AuditContext) (ApprovalSummary, *apperror.Error)
}

type Handler struct {
	service ServiceContract
}

type createRequest struct {
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	ScriptType    string `json:"scriptType"`
	Content       string `json:"content" binding:"required"`
	ChangeSummary string `json:"changeSummary"`
}

type updateRequest struct {
	Name          string `json:"name" binding:"required"`
	Description   string `json:"description"`
	ScriptType    string `json:"scriptType"`
	Content       string `json:"content" binding:"required"`
	ChangeSummary string `json:"changeSummary"`
}

type decideApprovalRequest struct {
	Comment string `json:"comment"`
}

func NewHandler(service ServiceContract) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, userAuth gin.HandlerFunc, requirePermission func(string) gin.HandlerFunc) {
	protected := api.Group("/scripts")
	protected.Use(userAuth)
	protected.GET("", requirePermission("script:read"), h.list)
	protected.POST("", requirePermission("script:write"), h.create)
	protected.GET("/:id", requirePermission("script:read"), h.get)
	protected.PUT("/:id", requirePermission("script:write"), h.update)
	protected.POST("/:id/disable", requirePermission("script:write"), h.disable)
	protected.POST("/:id/approval", requirePermission("script:write"), h.requestApproval)

	approvals := api.Group("/script-approvals")
	approvals.Use(userAuth)
	approvals.GET("", requirePermission("script:read"), h.listApprovals)
	approvals.POST("/:id/approve", requirePermission("script:approve"), h.approve)
	approvals.POST("/:id/reject", requirePermission("script:approve"), h.reject)
}

func (h *Handler) list(c *gin.Context) {
	scripts, appErr := h.service.List(c.Request.Context(), ListInput{
		Keyword:        c.Query("keyword"),
		ScriptType:     c.Query("type"),
		Status:         c.Query("status"),
		ApprovalStatus: c.Query("approvalStatus"),
		Page:           int(parseUint(c.DefaultQuery("page", "1"))),
		PageSize:       int(parseUint(c.DefaultQuery("pageSize", "20"))),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, scripts)
}

func (h *Handler) get(c *gin.Context) {
	script, appErr := h.service.Get(c.Request.Context(), c.Param("id"))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, script)
}

func (h *Handler) create(c *gin.Context) {
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	script, appErr := h.service.Create(c.Request.Context(), CreateInput{
		Name:          req.Name,
		Description:   req.Description,
		ScriptType:    req.ScriptType,
		Content:       req.Content,
		ChangeSummary: req.ChangeSummary,
		Audit:         auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, script)
}

func (h *Handler) update(c *gin.Context) {
	var req updateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	script, appErr := h.service.Update(c.Request.Context(), UpdateInput{
		ID:            c.Param("id"),
		Name:          req.Name,
		Description:   req.Description,
		ScriptType:    req.ScriptType,
		Content:       req.Content,
		ChangeSummary: req.ChangeSummary,
		Audit:         auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, script)
}

func (h *Handler) disable(c *gin.Context) {
	if appErr := h.service.Disable(c.Request.Context(), c.Param("id"), auditContext(c)); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) listApprovals(c *gin.Context) {
	approvals, appErr := h.service.ListApprovals(c.Request.Context(), c.Query("status"))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, approvals)
}

func (h *Handler) requestApproval(c *gin.Context) {
	approval, appErr := h.service.RequestApproval(c.Request.Context(), c.Param("id"), auditContext(c))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, approval)
}

func (h *Handler) approve(c *gin.Context) {
	h.decide(c, true)
}

func (h *Handler) reject(c *gin.Context) {
	h.decide(c, false)
}

func (h *Handler) decide(c *gin.Context, approve bool) {
	var req decideApprovalRequest
	_ = c.ShouldBindJSON(&req)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	approval, appErr := h.service.DecideApproval(c.Request.Context(), id, approve, req.Comment, auditContext(c))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, approval)
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

func parseUint(value string) uint64 {
	parsed, _ := strconv.ParseUint(value, 10, 64)
	return parsed
}
