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

type heartbeatRequest struct {
	Status    string                 `json:"status"`
	Name      string                 `json:"name"`
	Hostname  string                 `json:"hostname"`
	IP        string                 `json:"ip"`
	OS        string                 `json:"os"`
	OSType    string                 `json:"osType"`
	OSName    string                 `json:"osName"`
	OSVersion string                 `json:"osVersion"`
	Arch      string                 `json:"arch"`
	Version   string                 `json:"version"`
	Metadata  map[string]interface{} `json:"metadata"`
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, userAuth gin.HandlerFunc, requirePermission func(string) gin.HandlerFunc) {
	api.POST("/agents/register", h.register)
	api.POST("/agents/heartbeat", h.AgentAuthMiddleware(), h.heartbeat)

	protected := api.Group("")
	protected.Use(userAuth)
	protected.GET("/agents", requirePermission("agent.read"), h.listAgents)
	protected.GET("/hosts", requirePermission("agent.read"), h.listHosts)
	protected.POST("/agents/offline-scan", requirePermission("agent.disable"), h.markOffline)
	protected.POST("/agents/:id/disable", requirePermission("agent.disable"), h.disableAgent)
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
		HostInfoInput:   hostInfoFromRegister(req),
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
	agents, appErr := h.service.ListAgents(c.Request.Context())
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, agents)
}

func (h *Handler) listHosts(c *gin.Context) {
	hosts, appErr := h.service.ListHosts(c.Request.Context())
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, hosts)
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

func writeAppError(c *gin.Context, appErr *apperror.Error) {
	response.Fail(c, appErr.HTTPStatus, appErr.Code, appErr.Message)
}
