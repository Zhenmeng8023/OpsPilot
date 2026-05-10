package tasks

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/modules/agents"
	"opspilot/server/internal/modules/auth"
	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/response"
)

type ServiceContract interface {
	List(context.Context, ListTasksInput) (TaskListResult, *apperror.Error)
	Get(context.Context, string) (TaskDetail, *apperror.Error)
	Create(context.Context, CreateTaskInput) (TaskDetail, *apperror.Error)
	Cancel(context.Context, string, AuditContext) *apperror.Error
	Targets(context.Context, string) ([]TaskTargetSummary, *apperror.Error)
	Poll(context.Context, AgentIdentity, int) ([]AgentTask, *apperror.Error)
	Claim(context.Context, AgentIdentity, string) (AgentTask, *apperror.Error)
	TargetState(context.Context, AgentIdentity, string) (TargetState, *apperror.Error)
	UploadLog(context.Context, AgentIdentity, LogInput) *apperror.Error
	ReportResult(context.Context, AgentIdentity, ResultInput) *apperror.Error
	Logs(context.Context, LogQuery) ([]TaskLogEntry, *apperror.Error)
}

type Handler struct {
	service ServiceContract
}

type createRequest struct {
	Name           string   `json:"name" binding:"required"`
	Description    string   `json:"description"`
	ScriptID       string   `json:"scriptId"`
	Command        string   `json:"command"`
	ScriptType     string   `json:"scriptType"`
	TimeoutSeconds uint     `json:"timeoutSeconds" binding:"required"`
	TargetAgentIDs []string `json:"targetAgentIds"`
	TargetHostIDs  []string `json:"targetHostIds"`
}

type logRequest struct {
	Sequence  uint64 `json:"sequence"`
	Stream    string `json:"stream" binding:"required"`
	Chunk     string `json:"chunk"`
	Timestamp string `json:"timestamp"`
}

type resultRequest struct {
	Status       string `json:"status" binding:"required"`
	ExitCode     *int   `json:"exitCode"`
	ErrorMessage string `json:"errorMessage"`
	StartedAt    string `json:"startedAt"`
	FinishedAt   string `json:"finishedAt"`
}

func NewHandler(service ServiceContract) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(api *gin.RouterGroup, userAuth gin.HandlerFunc, requirePermission func(string) gin.HandlerFunc) {
	protected := api.Group("/tasks")
	protected.Use(userAuth)
	protected.GET("", requirePermission("task:read"), h.list)
	protected.POST("", requirePermission("task:execute"), h.create)
	protected.GET("/:id", requirePermission("task:read"), h.get)
	protected.POST("/:id/cancel", requirePermission("task:cancel"), h.cancel)
	protected.GET("/:id/targets", requirePermission("task:read"), h.targets)
	protected.GET("/:id/logs", requirePermission("task:log:read"), h.logs)
	protected.GET("/:id/targets/:targetId/logs", requirePermission("task:log:read"), h.targetLogs)
	protected.GET("/:id/logs/stream", requirePermission("task:log:read"), h.streamLogs)
}

func (h *Handler) RegisterAgentRoutes(api *gin.RouterGroup, agentAuth gin.HandlerFunc) {
	agentGroup := api.Group("/agent/tasks")
	agentGroup.Use(agentAuth)
	agentGroup.GET("/poll", h.poll)
	agentGroup.POST("/:targetId/claim", h.claim)
	agentGroup.GET("/:targetId/status", h.targetState)
	agentGroup.POST("/:targetId/logs", h.uploadLog)
	agentGroup.POST("/:targetId/result", h.reportResult)
}

func (h *Handler) list(c *gin.Context) {
	tasks, appErr := h.service.List(c.Request.Context(), ListTasksInput{
		Keyword:     c.Query("keyword"),
		Status:      c.Query("status"),
		Creator:     c.Query("creator"),
		CreatedFrom: c.Query("createdFrom"),
		CreatedTo:   c.Query("createdTo"),
		Page:        int(parseUint(c.DefaultQuery("page", "1"))),
		PageSize:    int(parseUint(c.DefaultQuery("pageSize", "20"))),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, tasks)
}

func (h *Handler) get(c *gin.Context) {
	task, appErr := h.service.Get(c.Request.Context(), c.Param("id"))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, task)
}

func (h *Handler) create(c *gin.Context) {
	var req createRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	task, appErr := h.service.Create(c.Request.Context(), CreateTaskInput{
		Name:           req.Name,
		Description:    req.Description,
		ScriptID:       req.ScriptID,
		Command:        req.Command,
		ScriptType:     req.ScriptType,
		TimeoutSeconds: req.TimeoutSeconds,
		TargetAgentIDs: req.TargetAgentIDs,
		TargetHostIDs:  req.TargetHostIDs,
		Audit:          auditContext(c),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, task)
}

func (h *Handler) cancel(c *gin.Context) {
	if appErr := h.service.Cancel(c.Request.Context(), c.Param("id"), auditContext(c)); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) targets(c *gin.Context) {
	targets, appErr := h.service.Targets(c.Request.Context(), c.Param("id"))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, targets)
}

func (h *Handler) logs(c *gin.Context) {
	logs, appErr := h.service.Logs(c.Request.Context(), LogQuery{
		TaskID:  c.Param("id"),
		Stream:  c.Query("stream"),
		AfterID: parseUint(c.Query("afterSequence")),
		Limit:   int(parseUint(c.DefaultQuery("limit", "500"))),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, logs)
}

func (h *Handler) targetLogs(c *gin.Context) {
	logs, appErr := h.service.Logs(c.Request.Context(), LogQuery{
		TaskID:   c.Param("id"),
		TargetID: c.Param("targetId"),
		Stream:   c.Query("stream"),
		AfterID:  parseUint(c.Query("afterSequence")),
		Limit:    int(parseUint(c.DefaultQuery("limit", "500"))),
	})
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, logs)
}

func (h *Handler) streamLogs(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		response.Fail(c, http.StatusInternalServerError, 500312, "streaming is not supported")
		return
	}

	cursor := parseUint(c.Query("afterSequence"))
	targetID := c.Query("targetId")
	ticker := time.NewTicker(time.Second)
	heartbeat := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	defer heartbeat.Stop()

	writeSSE(c, "ready", gin.H{"ok": true})
	flusher.Flush()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-heartbeat.C:
			writeSSE(c, "heartbeat", gin.H{"time": time.Now().Format(time.RFC3339)})
			flusher.Flush()
		case <-ticker.C:
			logs, appErr := h.service.Logs(c.Request.Context(), LogQuery{
				TaskID:   c.Param("id"),
				TargetID: targetID,
				Stream:   c.Query("stream"),
				AfterID:  cursor,
				Limit:    int(parseUint(c.DefaultQuery("limit", "500"))),
			})
			if appErr != nil {
				writeSSE(c, "error", gin.H{"message": appErr.Message})
				flusher.Flush()
				continue
			}
			for _, entry := range logs {
				if entry.ID > cursor {
					cursor = entry.ID
				}
				writeSSE(c, "log", entry)
			}
			flusher.Flush()
		}
	}
}

func (h *Handler) poll(c *gin.Context) {
	identity, ok := agentIdentity(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, 401012, "invalid agent token")
		return
	}
	tasks, appErr := h.service.Poll(c.Request.Context(), identity, int(parseUint(c.DefaultQuery("limit", "10"))))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, tasks)
}

func (h *Handler) claim(c *gin.Context) {
	identity, ok := agentIdentity(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, 401012, "invalid agent token")
		return
	}
	task, appErr := h.service.Claim(c.Request.Context(), identity, c.Param("targetId"))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, task)
}

func (h *Handler) targetState(c *gin.Context) {
	identity, ok := agentIdentity(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, 401012, "invalid agent token")
		return
	}
	state, appErr := h.service.TargetState(c.Request.Context(), identity, c.Param("targetId"))
	if appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, state)
}

func (h *Handler) uploadLog(c *gin.Context) {
	identity, ok := agentIdentity(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, 401012, "invalid agent token")
		return
	}
	var req logRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	if appErr := h.service.UploadLog(c.Request.Context(), identity, LogInput{
		TargetID:  c.Param("targetId"),
		Sequence:  req.Sequence,
		Stream:    req.Stream,
		Chunk:     req.Chunk,
		Timestamp: req.Timestamp,
	}); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

func (h *Handler) reportResult(c *gin.Context) {
	identity, ok := agentIdentity(c)
	if !ok {
		response.Fail(c, http.StatusUnauthorized, 401012, "invalid agent token")
		return
	}
	var req resultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, 400001, "invalid request body")
		return
	}
	if appErr := h.service.ReportResult(c.Request.Context(), identity, ResultInput{
		TargetID:     c.Param("targetId"),
		Status:       req.Status,
		ExitCode:     req.ExitCode,
		ErrorMessage: req.ErrorMessage,
		StartedAt:    req.StartedAt,
		FinishedAt:   req.FinishedAt,
	}); appErr != nil {
		writeAppError(c, appErr)
		return
	}
	response.Success(c, gin.H{"ok": true})
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

func agentIdentity(c *gin.Context) (AgentIdentity, bool) {
	identity, ok := agents.AgentIdentityFromContext(c)
	if !ok {
		return AgentIdentity{}, false
	}
	return AgentIdentity{
		ID:          identity.ID,
		UID:         identity.UID,
		WorkspaceID: identity.WorkspaceID,
		HostID:      identity.HostID,
		Name:        identity.Name,
	}, true
}

func parseUint(value string) uint64 {
	parsed, _ := strconv.ParseUint(value, 10, 64)
	return parsed
}

func writeSSE(c *gin.Context, event string, data interface{}) {
	bytes, _ := json.Marshal(data)
	_, _ = c.Writer.WriteString("event: " + event + "\n")
	_, _ = c.Writer.WriteString("data: " + string(bytes) + "\n\n")
}

func writeAppError(c *gin.Context, appErr *apperror.Error) {
	response.FailAppError(c, appErr)
}
