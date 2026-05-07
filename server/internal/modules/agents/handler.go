package agents

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/modules/auth"
	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/response"
)

const agentIdentityKey = "agent.identity"

type Handler struct {
	service *Service
}

type registerRequest struct {
	BootstrapSecret string                 `json:"bootstrapSecret"`
	EnrollmentToken string                 `json:"enrollmentToken"`
	Name            string                 `json:"name"`
	Hostname        string                 `json:"hostname"`
	IP              string                 `json:"ip"`
	OS              string                 `json:"os"`
	OSType          string                 `json:"osType"`
	OSName          string                 `json:"osName"`
	OSVersion       string                 `json:"osVersion"`
	Arch            string                 `json:"arch"`
	Version         string                 `json:"version"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type createEnrollmentTokenRequest struct {
	MaxUses           uint   `json:"maxUses"`
	ExpiresInSeconds  uint   `json:"expiresInSeconds"`
	BindWorkspaceSlug string `json:"bindWorkspaceSlug"`
}

type heartbeatRequest struct {
	Status       string                 `json:"status"`
	RunningTasks int                    `json:"runningTasks"`
	Name         string                 `json:"name"`
	Hostname     string                 `json:"hostname"`
	IP           string                 `json:"ip"`
	OS           string                 `json:"os"`
	OSType       string                 `json:"osType"`
	OSName       string                 `json:"osName"`
	OSVersion    string                 `json:"osVersion"`
	Arch         string                 `json:"arch"`
	Version      string                 `json:"version"`
	Metadata     map[string]interface{} `json:"metadata"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, userAuth gin.HandlerFunc, requirePermission func(string) gin.HandlerFunc) {
	api.POST("/agents/register", h.register)
	api.POST("/agents/heartbeat", h.AgentAuthMiddleware(), h.heartbeat)

	protected := api.Group("")
	protected.Use(userAuth)
	protected.GET("/agents", requirePermission("agent:read"), h.listAgents)
	protected.GET("/hosts", requirePermission("host:read"), h.listHosts)
	protected.POST("/agents/offline-scan", requirePermission("agent:write"), h.markOffline)
	protected.POST("/agents/:id/disable", requirePermission("agent:write"), h.disableAgent)
	protected.POST("/agents/:id/revoke-token", requirePermission("agent:write"), h.revokeAgentToken)
	protected.GET("/agent-enrollment-tokens", requirePermission("agent:read"), h.listEnrollmentTokens)
	protected.POST("/agent-enrollment-tokens", requirePermission("agent:write"), h.createEnrollmentToken)
	protected.POST("/agent-enrollment-tokens/:id/revoke", requirePermission("agent:write"), h.revokeEnrollmentToken)
}

func (h *Handler) register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	if req.BootstrapSecret == "" {
		req.BootstrapSecret = c.GetHeader("X-Agent-Bootstrap-Secret")
	}
	if req.IP == "" {
		req.IP = c.ClientIP()
	}
	result, appErr := h.service.Register(c.Request.Context(), RegisterInput{
		BootstrapSecret: req.BootstrapSecret,
		EnrollmentToken: req.EnrollmentToken,
		Audit: AuditContext{
			IP:            c.ClientIP(),
			UserAgent:     c.Request.UserAgent(),
			TraceID:       traceID(c),
			RequestMethod: c.Request.Method,
			RequestPath:   c.Request.URL.Path,
		},
		HostInfoInput: hostInfoFromRegister(req),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) heartbeat(c *gin.Context) {
	var req heartbeatRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	if req.IP == "" {
		req.IP = c.ClientIP()
	}
	identity, ok := AgentIdentityFromContext(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, 401012, "invalid agent token")
		return
	}
	result, appErr := h.service.Heartbeat(c.Request.Context(), identity, HeartbeatInput{
		Status:        req.Status,
		HostInfoInput: hostInfoFromHeartbeat(req),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) listAgents(c *gin.Context) {
	agents, appErr := h.service.ListAgents(c.Request.Context(), listInput(c))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, agents)
}

func (h *Handler) listHosts(c *gin.Context) {
	hosts, appErr := h.service.ListHosts(c.Request.Context(), listInput(c))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, hosts)
}

func (h *Handler) revokeAgentToken(c *gin.Context) {
	actorID := ""
	if claims, ok := auth.ClaimsFromContext(c); ok {
		actorID = claims.UserID
	}
	if appErr := h.service.RevokeAgentToken(c.Request.Context(), c.Param("id"), actorID); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) listEnrollmentTokens(c *gin.Context) {
	tokens, appErr := h.service.ListEnrollmentTokens(c.Request.Context())
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, tokens)
}

func (h *Handler) createEnrollmentToken(c *gin.Context) {
	var req createEnrollmentTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	actorID := ""
	if claims, ok := auth.ClaimsFromContext(c); ok {
		actorID = claims.UserID
	}
	token, appErr := h.service.CreateEnrollmentToken(c.Request.Context(), CreateEnrollmentTokenInput{
		MaxUses:           req.MaxUses,
		ExpiresInSeconds:  req.ExpiresInSeconds,
		BindWorkspaceSlug: req.BindWorkspaceSlug,
		Audit: AuditContext{
			ActorUID:      actorID,
			IP:            c.ClientIP(),
			UserAgent:     c.Request.UserAgent(),
			TraceID:       traceID(c),
			RequestMethod: c.Request.Method,
			RequestPath:   c.Request.URL.Path,
		},
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, token)
}

func (h *Handler) revokeEnrollmentToken(c *gin.Context) {
	actorID := ""
	if claims, ok := auth.ClaimsFromContext(c); ok {
		actorID = claims.UserID
	}
	if appErr := h.service.RevokeEnrollmentToken(c.Request.Context(), c.Param("id"), actorID); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) disableAgent(c *gin.Context) {
	actorID := ""
	if claims, ok := auth.ClaimsFromContext(c); ok {
		actorID = claims.UserID
	}
	if appErr := h.service.DisableAgent(c.Request.Context(), c.Param("id"), actorID); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) markOffline(c *gin.Context) {
	result, appErr := h.service.MarkOffline(c.Request.Context())
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, result)
}

func (h *Handler) AgentAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		token, ok := strings.CutPrefix(header, "Bearer ")
		if !ok || strings.TrimSpace(token) == "" {
			response.Fail(c, http.StatusUnauthorized, 401011, "agent token is required")
			c.Abort()
			return
		}
		identity, appErr := h.service.AuthenticateAgent(c.Request.Context(), token)
		if appErr != nil {
			writeAppError(c, appErr)
			c.Abort()
			return
		}
		c.Set(agentIdentityKey, identity)
		c.Next()
	}
}

func AgentIdentityFromContext(c *gin.Context) (AgentIdentity, bool) {
	value, exists := c.Get(agentIdentityKey)
	if !exists {
		return AgentIdentity{}, false
	}
	identity, ok := value.(AgentIdentity)
	return identity, ok
}

func hostInfoFromRegister(req registerRequest) HostInfoInput {
	return HostInfoInput{
		Name:      req.Name,
		Hostname:  req.Hostname,
		IP:        req.IP,
		OS:        req.OS,
		OSType:    req.OSType,
		OSName:    req.OSName,
		OSVersion: req.OSVersion,
		Arch:      req.Arch,
		Version:   req.Version,
		Metadata:  req.Metadata,
	}
}

func hostInfoFromHeartbeat(req heartbeatRequest) HostInfoInput {
	metadata := req.Metadata
	if metadata == nil {
		metadata = map[string]interface{}{}
	}
	metadata["runningTasks"] = req.RunningTasks
	return HostInfoInput{
		Name:      req.Name,
		Hostname:  req.Hostname,
		IP:        req.IP,
		OS:        req.OS,
		OSType:    req.OSType,
		OSName:    req.OSName,
		OSVersion: req.OSVersion,
		Arch:      req.Arch,
		Version:   req.Version,
		Metadata:  metadata,
	}
}

func listInput(c *gin.Context) ListInput {
	return ListInput{
		Keyword:  c.Query("keyword"),
		Status:   c.Query("status"),
		Page:     parseInt(c.DefaultQuery("page", "1")),
		PageSize: parseInt(c.DefaultQuery("pageSize", "20")),
	}
}

func parseInt(value string) int {
	parsed := 0
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return 0
		}
		parsed = parsed*10 + int(ch-'0')
	}
	return parsed
}

func traceID(c *gin.Context) string {
	value, exists := c.Get("traceId")
	if !exists {
		return ""
	}
	traceID, _ := value.(string)
	return traceID
}

func writeAppError(c *gin.Context, appErr *apperror.Error) {
	response.Fail(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
}
