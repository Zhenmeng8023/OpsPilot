package auth

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	jwtplatform "opspilot/server/internal/platform/jwt"
	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/response"
)

const claimsKey = "auth.claims"

type ServiceContract interface {
	JWTManager() jwtplatform.Manager
	Register(context.Context, RegisterInput) (AuthResult, *apperror.Error)
	Login(context.Context, LoginInput) (AuthResult, *apperror.Error)
	Refresh(context.Context, RefreshInput) (AuthResult, *apperror.Error)
	Logout(context.Context, LogoutInput) *apperror.Error
	Me(context.Context, string) (UserProfile, *apperror.Error)
	ListUsers(context.Context, string, string) ([]UserSummary, *apperror.Error)
	CreateUser(context.Context, CreateUserInput) (UserSummary, *apperror.Error)
	UpdateUserStatus(context.Context, UpdateUserStatusInput) *apperror.Error
	UpdateUserRoles(context.Context, UpdateUserRolesInput) (UserSummary, *apperror.Error)
	ListPermissions(context.Context) ([]PermissionSummary, *apperror.Error)
	ListRoles(context.Context) ([]RoleSummary, *apperror.Error)
	CreateRole(context.Context, CreateRoleInput) (RoleSummary, *apperror.Error)
	UpdateRolePermissions(context.Context, UpdateRolePermissionsInput) (RoleSummary, *apperror.Error)
	HasPermission(context.Context, string, string) (bool, error)
}

type Handler struct {
	service ServiceContract
}

type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email"`
	Password string `json:"password" binding:"required"`
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type logoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type createUserRequest struct {
	Username string   `json:"username" binding:"required"`
	Email    string   `json:"email"`
	Password string   `json:"password" binding:"required"`
	Roles    []string `json:"roles"`
}

type updateUserStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type updateUserRolesRequest struct {
	Roles []string `json:"roles" binding:"required"`
}

type createRoleRequest struct {
	Code        string   `json:"code" binding:"required"`
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

type updateRolePermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

func NewHandler(service ServiceContract) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup) {
	authGroup := api.Group("/auth")
	authGroup.POST("/register", h.register)
	authGroup.POST("/login", h.login)
	authGroup.POST("/refresh", h.refresh)
	authGroup.POST("/logout", h.logout)

	protected := api.Group("")
	protected.Use(AuthMiddleware(h.service.JWTManager()))
	protected.GET("/me", h.me)

	protected.GET("/users", h.RequirePermission("user.read"), h.listUsers)
	protected.POST("/users", h.RequirePermission("user.write"), h.createUser)
	protected.PATCH("/users/:id/status", h.RequirePermission("user.write"), h.updateUserStatus)
	protected.PUT("/users/:id/roles", h.RequirePermission("role.write"), h.updateUserRoles)

	protected.GET("/permissions", h.RequirePermission("role.read"), h.listPermissions)
	protected.GET("/roles", h.RequirePermission("role.read"), h.listRoles)
	protected.POST("/roles", h.RequirePermission("role.write"), h.createRole)
	protected.PUT("/roles/:id/permissions", h.RequirePermission("role.write"), h.updateRolePermissions)
}

func (h *Handler) register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	result, appErr := h.service.Register(c.Request.Context(), RegisterInput{
		Username:  req.Username,
		Email:     req.Email,
		Password:  req.Password,
		IP:        c.ClientIP(),
		UserAgent: c.Request.UserAgent(),
		TraceID:   traceID(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	result, appErr := h.service.Login(c.Request.Context(), LoginInput{
		Identifier: req.Username,
		Password:   req.Password,
		IP:         c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
		TraceID:    traceID(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	result, appErr := h.service.Refresh(c.Request.Context(), RefreshInput{
		RefreshToken: req.RefreshToken,
		IP:           c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) logout(c *gin.Context) {
	var req logoutRequest
	_ = c.ShouldBindJSON(&req)
	if appErr := h.service.Logout(c.Request.Context(), LogoutInput{RefreshToken: req.RefreshToken}); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) me(c *gin.Context) {
	claims, ok := ClaimsFromContext(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, 401003, "unauthorized")
		return
	}
	profile, appErr := h.service.Me(c.Request.Context(), claims.UserID)
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, profile)
}

func (h *Handler) listUsers(c *gin.Context) {
	users, appErr := h.service.ListUsers(c.Request.Context(), c.Query("keyword"), c.Query("status"))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, users)
}

func (h *Handler) createUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	user, appErr := h.service.CreateUser(c.Request.Context(), CreateUserInput{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password,
		Roles:    req.Roles,
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, user)
}

func (h *Handler) updateUserStatus(c *gin.Context) {
	var req updateUserStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	claims, _ := ClaimsFromContext(c)
	appErr := h.service.UpdateUserStatus(c.Request.Context(), UpdateUserStatusInput{
		UserID: c.Param("id"),
		Status: req.Status,
		Actor:  claims.UserID,
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) updateUserRoles(c *gin.Context) {
	var req updateUserRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	user, appErr := h.service.UpdateUserRoles(c.Request.Context(), UpdateUserRolesInput{
		UserID: c.Param("id"),
		Roles:  req.Roles,
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, user)
}

func (h *Handler) listPermissions(c *gin.Context) {
	permissions, appErr := h.service.ListPermissions(c.Request.Context())
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, permissions)
}

func (h *Handler) listRoles(c *gin.Context) {
	roles, appErr := h.service.ListRoles(c.Request.Context())
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, roles)
}

func (h *Handler) createRole(c *gin.Context) {
	var req createRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	role, appErr := h.service.CreateRole(c.Request.Context(), CreateRoleInput{
		Code:        req.Code,
		Name:        req.Name,
		Description: req.Description,
		Permissions: req.Permissions,
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, role)
}

func (h *Handler) updateRolePermissions(c *gin.Context) {
	var req updateRolePermissionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	claims, _ := ClaimsFromContext(c)
	role, appErr := h.service.UpdateRolePermissions(c.Request.Context(), UpdateRolePermissionsInput{
		RoleID:      c.Param("id"),
		Permissions: req.Permissions,
		Actor:       claims.UserID,
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, role)
}

func AuthMiddleware(manager jwtplatform.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || strings.TrimSpace(token) == "" {
			response.Fail(c, http.StatusUnauthorized, 401003, "unauthorized")
			c.Abort()
			return
		}
		claims, err := manager.ParseAccessToken(strings.TrimSpace(token))
		if err != nil {
			response.Fail(c, http.StatusUnauthorized, 401003, "unauthorized")
			c.Abort()
			return
		}
		c.Set(claimsKey, claims)
		c.Next()
	}
}

func (h *Handler) RequirePermission(permission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, ok := ClaimsFromContext(c)
		if !ok {
			response.Fail(c, http.StatusUnauthorized, 401003, "unauthorized")
			c.Abort()
			return
		}
		allowed, err := h.service.HasPermission(c.Request.Context(), claims.UserID, permission)
		if err != nil {
			response.Fail(c, http.StatusInternalServerError, 500001, "permission check failed")
			c.Abort()
			return
		}
		if !allowed {
			response.Fail(c, http.StatusForbidden, 403002, "permission denied")
			c.Abort()
			return
		}
		c.Next()
	}
}

func ClaimsFromContext(c *gin.Context) (*jwtplatform.Claims, bool) {
	value, exists := c.Get(claimsKey)
	if !exists {
		return nil, false
	}
	claims, ok := value.(*jwtplatform.Claims)
	return claims, ok
}

func writeAppError(c *gin.Context, appErr *apperror.Error) {
	response.FailAppError(c, appErr)
}

func traceID(c *gin.Context) string {
	value, exists := c.Get("traceId")
	if !exists {
		return ""
	}
	traceID, _ := value.(string)
	return traceID
}
