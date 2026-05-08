package tasks

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"

	"opspilot/server/internal/config"
	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/audit"
	"opspilot/server/internal/shared/security"
	"opspilot/server/internal/shared/uid"
)

type Service struct {
	db   *gorm.DB
	cfg  config.Config
	repo repository
}

const maxTaskTimeoutSeconds uint = 3600

var dangerousCommandPattern = regexp.MustCompile(`(?i)(^|[;&|()\r\n])\s*(sudo\s+)?([a-z]:\\[^\s;&|()]+\\|/[^\s;&|()]+/)?(rm|del|format|shutdown|reboot|mkfs(\.[a-z0-9_+-]+)?|remove-item)\b`)

type AuditContext struct {
	ActorUID      string
	IP            string
	UserAgent     string
	TraceID       string
	RequestMethod string
	RequestPath   string
}

type CreateTaskInput struct {
	Name           string
	Description    string
	ScriptID       string
	Command        string
	ScriptType     string
	TimeoutSeconds uint
	TargetAgentIDs []string
	TargetHostIDs  []string
	Audit          AuditContext
}

type ListTasksInput struct {
	Keyword     string
	Status      string
	Creator     string
	CreatedFrom string
	CreatedTo   string
	Page        int
	PageSize    int
}

type AgentIdentity struct {
	ID          uint64
	UID         string
	WorkspaceID uint64
	HostID      uint64
	Name        string
}

type LogInput struct {
	TargetID  string
	Sequence  uint64
	Stream    string
	Chunk     string
	Timestamp string
}

type ResultInput struct {
	TargetID     string
	Status       string
	ExitCode     *int
	ErrorMessage string
	StartedAt    string
	FinishedAt   string
}

type LogQuery struct {
	TaskID   string
	TargetID string
	Stream   string
	AfterID  uint64
	Limit    int
}

type TaskSummary struct {
	ID             string `json:"id"`
	TaskID         string `json:"taskId"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	Status         string `json:"status"`
	TimeoutSeconds uint   `json:"timeoutSeconds"`
	TargetCount    uint   `json:"targetCount"`
	SuccessCount   uint   `json:"successCount"`
	FailedCount    uint   `json:"failedCount"`
	CanceledCount  uint   `json:"canceledCount"`
	RunningCount   uint   `json:"runningCount"`
	QueuedCount    uint   `json:"queuedCount"`
	CreatedBy      string `json:"createdBy,omitempty"`
	CreatedAt      string `json:"createdAt"`
	QueuedAt       string `json:"queuedAt,omitempty"`
	StartedAt      string `json:"startedAt,omitempty"`
	FinishedAt     string `json:"finishedAt,omitempty"`
	ErrorMessage   string `json:"errorMessage,omitempty"`
}

type TaskListResult struct {
	Items    []TaskSummary `json:"items"`
	Total    int64         `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pageSize"`
}

type TaskDetail struct {
	TaskSummary
	Targets []TaskTargetSummary `json:"targets,omitempty"`
}

type TaskTargetSummary struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	AgentID      string `json:"agentId,omitempty"`
	AgentName    string `json:"agentName,omitempty"`
	AgentStatus  string `json:"agentStatus,omitempty"`
	HostID       string `json:"hostId,omitempty"`
	HostName     string `json:"hostName,omitempty"`
	ExitCode     *int   `json:"exitCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	StartedAt    string `json:"startedAt,omitempty"`
	FinishedAt   string `json:"finishedAt,omitempty"`
	CreatedAt    string `json:"createdAt"`
}

type AgentTask struct {
	TargetID       string `json:"targetId"`
	RunID          string `json:"runId"`
	Name           string `json:"name"`
	Description    string `json:"description,omitempty"`
	ScriptType     string `json:"scriptType"`
	Command        string `json:"command"`
	TimeoutSeconds uint   `json:"timeoutSeconds"`
	Status         string `json:"status"`
	AgentName      string `json:"agentName"`
	HostName       string `json:"hostName,omitempty"`
}

type TargetState struct {
	TargetID string `json:"targetId"`
	Status   string `json:"status"`
}

type TaskLogEntry struct {
	ID        uint64 `json:"id"`
	RunID     string `json:"runId"`
	TargetID  string `json:"targetId"`
	Sequence  uint64 `json:"sequence"`
	Stream    string `json:"stream"`
	Content   string `json:"content"`
	CreatedAt string `json:"createdAt"`
	AgentName string `json:"agentName,omitempty"`
	HostName  string `json:"hostName,omitempty"`
}

type targetSelection struct {
	SourceType string
	AgentID    uint64
	AgentUID   string
	AgentName  string
	HostID     sql.NullInt64
	HostUID    string
	HostName   string
	IP         string
}

func NewService(db *gorm.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg, repo: newRepository(db)}
}

func (s *Service) List(ctx context.Context, input ListTasksInput) (TaskListResult, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return TaskListResult{}, appErr
	}
	input.Page, input.PageSize = normalizePage(input.Page, input.PageSize)
	rows, total, err := s.repo.listRuns(ctx, workspace.ID, listRunsFilter{
		Keyword:     strings.TrimSpace(input.Keyword),
		Status:      strings.TrimSpace(input.Status),
		Creator:     strings.TrimSpace(input.Creator),
		CreatedFrom: normalizeDateStart(input.CreatedFrom),
		CreatedTo:   normalizeDateEnd(input.CreatedTo),
		Page:        input.Page,
		PageSize:    input.PageSize,
	})
	if err != nil {
		return TaskListResult{}, apperror.Wrap(http.StatusInternalServerError, 500301, "list tasks failed", err)
	}
	out := make([]TaskSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, summaryFromRun(row))
	}
	return TaskListResult{Items: out, Total: total, Page: input.Page, PageSize: input.PageSize}, nil
}

func (s *Service) Get(ctx context.Context, runUID string) (TaskDetail, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return TaskDetail{}, appErr
	}
	row, err := s.repo.runByUID(ctx, workspace.ID, strings.TrimSpace(runUID))
	if err != nil {
		return TaskDetail{}, apperror.Wrap(http.StatusInternalServerError, 500302, "load task failed", err)
	}
	if row.ID == 0 {
		return TaskDetail{}, apperror.New(http.StatusNotFound, 404301, "task not found")
	}
	targets, err := s.repo.targetsByRunUID(ctx, workspace.ID, runUID)
	if err != nil {
		return TaskDetail{}, apperror.Wrap(http.StatusInternalServerError, 500303, "load task targets failed", err)
	}
	return TaskDetail{TaskSummary: summaryFromRun(row), Targets: targetsFromRows(targets)}, nil
}

func (s *Service) Targets(ctx context.Context, runUID string) ([]TaskTargetSummary, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	if _, appErr := s.Get(ctx, runUID); appErr != nil {
		return nil, appErr
	}
	rows, err := s.repo.targetsByRunUID(ctx, workspace.ID, runUID)
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500303, "load task targets failed", err)
	}
	return targetsFromRows(rows), nil
}

func (s *Service) Create(ctx context.Context, input CreateTaskInput) (TaskDetail, *apperror.Error) {
	name := strings.TrimSpace(input.Name)
	command := strings.TrimSpace(input.Command)
	scriptUID := strings.TrimSpace(input.ScriptID)
	scriptType := normalizeScriptType(input.ScriptType)
	if name == "" {
		return TaskDetail{}, apperror.New(http.StatusBadRequest, 400301, "task name is required")
	}
	if input.TimeoutSeconds == 0 {
		return TaskDetail{}, apperror.New(http.StatusBadRequest, 400302, "timeoutSeconds must be greater than 0")
	}
	if input.TimeoutSeconds > maxTaskTimeoutSeconds {
		return TaskDetail{}, apperror.New(http.StatusBadRequest, 400311, "timeoutSeconds exceeds maximum allowed value")
	}
	if scriptUID == "" && command == "" {
		return TaskDetail{}, apperror.New(http.StatusBadRequest, 400303, "script or command is required")
	}
	if command != "" && !validScriptType(scriptType) {
		return TaskDetail{}, apperror.New(http.StatusBadRequest, 400304, "unsupported command type")
	}
	if command != "" {
		if err := validateCommandSafety(command, s.cfg.Command); err != nil {
			return TaskDetail{}, err
		}
	}
	if len(input.TargetAgentIDs) == 0 && len(input.TargetHostIDs) == 0 {
		return TaskDetail{}, apperror.New(http.StatusBadRequest, 400305, "task targets cannot be empty")
	}

	var created TaskDetail
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := newRepository(tx)
		workspace, err := repo.defaultWorkspace(ctx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		if workspace.ID == 0 {
			return apperror.New(http.StatusInternalServerError, 500304, "default workspace is not initialized")
		}
		actor, err := repo.userByUID(ctx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		actorID := sql.NullInt64{}
		if actor.ID != 0 {
			actorID = sql.NullInt64{Int64: int64(actor.ID), Valid: true}
		}
		script, err := s.resolveScript(ctx, tx, repo, workspace.ID, actorID, name, scriptUID, scriptType, command)
		if err != nil {
			return err
		}
		targets, err := s.resolveTargets(ctx, repo, workspace.ID, input.TargetAgentIDs, input.TargetHostIDs)
		if err != nil {
			return err
		}
		if len(targets) == 0 {
			return apperror.New(http.StatusBadRequest, 400305, "no executable agent targets found")
		}

		taskUID, err := uid.New()
		if err != nil {
			return err
		}
		taskName := name
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO tasks(uid, workspace_id, name, description, script_template_id, script_version_id,
			                   timeout_seconds, max_parallel, trigger_mode, status, created_by)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 'manual', 'active', ?)`,
			taskUID, workspace.ID, taskName, nullString(input.Description), script.TemplateID, script.VersionID,
			input.TimeoutSeconds, max(1, len(targets)), actorID,
		).Error; err != nil {
			return err
		}
		var taskID uint64
		if err := tx.WithContext(ctx).Raw("SELECT id FROM tasks WHERE uid = ? LIMIT 1", taskUID).Scan(&taskID).Error; err != nil {
			return err
		}
		runUID, err := uid.New()
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO task_runs(uid, workspace_id, task_id, script_version_id, status, trigger_type,
			                       timeout_seconds, total_targets, created_by)
			 VALUES (?, ?, ?, ?, 'pending', 'manual', ?, ?, ?)`,
			runUID, workspace.ID, taskID, script.VersionID, input.TimeoutSeconds, len(targets), actorID,
		).Error; err != nil {
			return err
		}
		var runID uint64
		if err := tx.WithContext(ctx).Raw("SELECT id FROM task_runs WHERE uid = ? LIMIT 1", runUID).Scan(&runID).Error; err != nil {
			return err
		}
		for _, target := range targets {
			if err := insertTarget(ctx, tx, taskID, runID, target); err != nil {
				return err
			}
		}
		if err := transitionRun(ctx, tx, runID, StatusPending, StatusQueued, "task queued"); err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			"UPDATE task_run_targets SET status = 'queued' WHERE run_id = ? AND status = 'pending'",
			runID,
		).Error; err != nil {
			return err
		}
		row, err := repo.runByUID(ctx, workspace.ID, runUID)
		if err != nil {
			return err
		}
		targetRows, err := repo.targetsByRunUID(ctx, workspace.ID, runUID)
		if err != nil {
			return err
		}
		created = TaskDetail{TaskSummary: summaryFromRun(row), Targets: targetsFromRows(targetRows)}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "task.create",
			ResourceType:  "task_run",
			ResourceID:    sql.NullInt64{Int64: int64(runID), Valid: true},
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			After:         created,
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return TaskDetail{}, appErr
		}
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return TaskDetail{}, apperror.New(http.StatusConflict, 409301, "task name already exists")
		}
		return TaskDetail{}, apperror.Wrap(http.StatusInternalServerError, 500305, "create task failed", txErr)
	}
	return created, nil
}

func (s *Service) Cancel(ctx context.Context, runUID string, auditCtx AuditContext) *apperror.Error {
	if strings.TrimSpace(runUID) == "" {
		return apperror.New(http.StatusBadRequest, 400001, "task id is required")
	}
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := newRepository(tx)
		workspace, err := repo.defaultWorkspace(ctx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		run, err := repo.runByUID(ctx, workspace.ID, runUID)
		if err != nil {
			return err
		}
		if run.ID == 0 {
			return apperror.New(http.StatusNotFound, 404301, "task not found")
		}
		if IsTerminal(Status(run.Status)) {
			return apperror.New(http.StatusBadRequest, 400306, "task cannot be canceled from current status")
		}
		actor, err := repo.userByUID(ctx, auditCtx.ActorUID)
		if err != nil {
			return err
		}
		actorID := sql.NullInt64{}
		if actor.ID != 0 {
			actorID = sql.NullInt64{Int64: int64(actor.ID), Valid: true}
		}
		if err := CancelRunByIDTx(ctx, tx, run.ID, actorID, "user requested"); err != nil {
			return err
		}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "task.cancel",
			ResourceType:  "task_run",
			ResourceID:    sql.NullInt64{Int64: int64(run.ID), Valid: true},
			IP:            auditCtx.IP,
			UserAgent:     auditCtx.UserAgent,
			TraceID:       auditCtx.TraceID,
			RequestMethod: auditCtx.RequestMethod,
			RequestPath:   auditCtx.RequestPath,
			Before:        summaryFromRun(run),
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return appErr
		}
		return apperror.Wrap(http.StatusInternalServerError, 500306, "cancel task failed", txErr)
	}
	return nil
}

func (s *Service) Poll(ctx context.Context, identity AgentIdentity, limit int) ([]AgentTask, *apperror.Error) {
	if limit <= 0 || limit > 50 {
		limit = 10
	}
	rows, err := s.repo.agentPoll(ctx, identity.WorkspaceID, identity.ID, limit)
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500307, "poll tasks failed", err)
	}
	out := make([]AgentTask, 0, len(rows))
	for _, row := range rows {
		out = append(out, agentTaskFromRow(row))
	}
	return out, nil
}

func (s *Service) Claim(ctx context.Context, identity AgentIdentity, targetUID string) (AgentTask, *apperror.Error) {
	targetUID = strings.TrimSpace(targetUID)
	if targetUID == "" {
		return AgentTask{}, apperror.New(http.StatusBadRequest, 400001, "target id is required")
	}
	var claimed AgentTask
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row struct {
			ID               uint64
			RunID            uint64
			Status           string
			AgentID          sql.NullInt64
			CurrentAttemptNo uint
		}
		if err := tx.WithContext(ctx).Raw(
			`SELECT rt.id, rt.run_id, rt.status, rt.agent_id, rt.current_attempt_no
			   FROM task_run_targets rt
			   JOIN task_runs tr ON tr.id = rt.run_id
			  WHERE rt.uid = ?
			    AND tr.workspace_id = ?
			  LIMIT 1
			  FOR UPDATE`,
			targetUID, identity.WorkspaceID,
		).Scan(&row).Error; err != nil {
			return err
		}
		if row.ID == 0 {
			return apperror.New(http.StatusNotFound, 404302, "task target not found")
		}
		if !row.AgentID.Valid || uint64(row.AgentID.Int64) != identity.ID {
			return apperror.New(http.StatusForbidden, 403301, "task target does not belong to this agent")
		}
		if !CanTransition(Status(row.Status), StatusRunning) {
			return apperror.New(http.StatusConflict, 409302, "task target is not claimable")
		}
		attemptNo := row.CurrentAttemptNo + 1
		attemptUID, err := uid.New()
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE task_run_targets
			    SET status = 'running', current_attempt_no = ?, started_at = COALESCE(started_at, NOW(3))
			  WHERE id = ? AND status = 'queued'`,
			attemptNo, row.ID,
		).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO task_run_attempts(uid, run_id, run_target_id, agent_id, attempt_no, status, started_at)
			 VALUES (?, ?, ?, ?, ?, 'running', NOW(3))`,
			attemptUID, row.RunID, row.ID, identity.ID, attemptNo,
		).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			"UPDATE task_runs SET status = 'running', started_at = COALESCE(started_at, NOW(3)) WHERE id = ? AND status = 'queued'",
			row.RunID,
		).Error; err != nil {
			return err
		}
		if err := writeRunEvent(ctx, tx, row.RunID, row.ID, 0, row.Status, string(StatusRunning), "agent claimed task target", nil); err != nil {
			return err
		}
		taskRow, err := newRepository(tx).agentTaskByTargetUID(ctx, identity.WorkspaceID, targetUID)
		if err != nil {
			return err
		}
		claimed = agentTaskFromRow(taskRow)
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return AgentTask{}, appErr
		}
		return AgentTask{}, apperror.Wrap(http.StatusInternalServerError, 500308, "claim task failed", txErr)
	}
	return claimed, nil
}

func (s *Service) TargetState(ctx context.Context, identity AgentIdentity, targetUID string) (TargetState, *apperror.Error) {
	targetUID = strings.TrimSpace(targetUID)
	if targetUID == "" {
		return TargetState{}, apperror.New(http.StatusBadRequest, 400001, "target id is required")
	}
	var state TargetState
	err := s.db.WithContext(ctx).Raw(
		`SELECT rt.uid AS target_id, rt.status
		   FROM task_run_targets rt
		   JOIN task_runs tr ON tr.id = rt.run_id
		  WHERE rt.uid = ?
		    AND rt.agent_id = ?
		    AND tr.workspace_id = ?
		  LIMIT 1`,
		targetUID, identity.ID, identity.WorkspaceID,
	).Scan(&state).Error
	if err != nil {
		return TargetState{}, apperror.Wrap(http.StatusInternalServerError, 500313, "load task target status failed", err)
	}
	if state.TargetID == "" {
		return TargetState{}, apperror.New(http.StatusNotFound, 404302, "task target not found")
	}
	return state, nil
}

func (s *Service) UploadLog(ctx context.Context, identity AgentIdentity, input LogInput) *apperror.Error {
	if strings.TrimSpace(input.TargetID) == "" {
		return apperror.New(http.StatusBadRequest, 400001, "target id is required")
	}
	stream := strings.TrimSpace(input.Stream)
	if stream != "stdout" && stream != "stderr" && stream != "system" {
		return apperror.New(http.StatusBadRequest, 400307, "unsupported log stream")
	}
	chunk := security.Redact(input.Chunk)
	if chunk == "" {
		return nil
	}
	timestamp := parseTime(input.Timestamp)
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		target, err := lockAgentTarget(ctx, tx, input.TargetID, identity.WorkspaceID, identity.ID)
		if err != nil {
			return err
		}
		if target.ID == 0 {
			return apperror.New(http.StatusNotFound, 404302, "task target not found")
		}
		if target.Status != string(StatusRunning) && target.Status != string(StatusCanceling) {
			return apperror.New(http.StatusConflict, 409303, "task target is not running")
		}
		attempt, err := newRepository(tx).latestAttempt(ctx, target.ID)
		if err != nil {
			return err
		}
		if attempt.ID == 0 {
			return apperror.New(http.StatusConflict, 409304, "task target has no active attempt")
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT IGNORE INTO task_run_logs(run_id, run_target_id, attempt_id, sequence, stream, content, content_hash, source_timestamp)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			target.RunID, target.ID, attempt.ID, input.Sequence, stream, chunk, checksum(chunk), timestamp,
		).Error; err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return appErr
		}
		return apperror.Wrap(http.StatusInternalServerError, 500309, "upload task log failed", txErr)
	}
	return nil
}

func (s *Service) ReportResult(ctx context.Context, identity AgentIdentity, input ResultInput) *apperror.Error {
	status := Status(strings.TrimSpace(input.Status))
	if !IsTerminal(status) {
		return apperror.New(http.StatusBadRequest, 400308, "unsupported result status")
	}
	startedAt := parseTime(input.StartedAt)
	finishedAt := parseTime(input.FinishedAt)
	errorMessage := security.Redact(input.ErrorMessage)
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		target, err := lockAgentTarget(ctx, tx, input.TargetID, identity.WorkspaceID, identity.ID)
		if err != nil {
			return err
		}
		if target.ID == 0 {
			return apperror.New(http.StatusNotFound, 404302, "task target not found")
		}
		if !CanTransition(Status(target.Status), status) {
			return apperror.New(http.StatusConflict, 409305, "illegal task target status transition")
		}
		attempt, err := newRepository(tx).latestAttempt(ctx, target.ID)
		if err != nil {
			return err
		}
		if attempt.ID == 0 {
			return apperror.New(http.StatusConflict, 409304, "task target has no active attempt")
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE task_run_targets
			    SET status = ?, exit_code = ?, error_message = ?, started_at = COALESCE(?, started_at), finished_at = COALESCE(?, NOW(3)),
			        duration_ms = CASE
			          WHEN started_at IS NOT NULL THEN TIMESTAMPDIFF(MICROSECOND, started_at, COALESCE(?, NOW(3))) / 1000
			          ELSE duration_ms
			        END
			  WHERE id = ?`,
			string(status), nullableInt(input.ExitCode), nullString(errorMessage), startedAt, finishedAt, finishedAt, target.ID,
		).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE task_run_attempts
			    SET status = ?, exit_code = ?, error_message = ?, finished_at = COALESCE(?, NOW(3))
			  WHERE id = ?`,
			string(status), nullableInt(input.ExitCode), nullString(errorMessage), finishedAt, attempt.ID,
		).Error; err != nil {
			return err
		}
		if err := writeRunEvent(ctx, tx, target.RunID, target.ID, attempt.ID, target.Status, string(status), "agent reported task result", nil); err != nil {
			return err
		}
		return refreshRunAggregate(ctx, tx, target.RunID)
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return appErr
		}
		return apperror.Wrap(http.StatusInternalServerError, 500310, "report task result failed", txErr)
	}
	return nil
}

func (s *Service) Logs(ctx context.Context, query LogQuery) ([]TaskLogEntry, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	if _, appErr := s.Get(ctx, query.TaskID); appErr != nil {
		return nil, appErr
	}
	stream := strings.TrimSpace(query.Stream)
	if stream != "" && stream != "stdout" && stream != "stderr" && stream != "system" {
		return nil, apperror.New(http.StatusBadRequest, 400307, "unsupported log stream")
	}
	rows, err := s.repo.logs(ctx, workspace.ID, query.TaskID, query.TargetID, stream, query.AfterID, normalizeLimit(query.Limit, 500, 1000))
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500311, "query task logs failed", err)
	}
	out := make([]TaskLogEntry, 0, len(rows))
	for _, row := range rows {
		out = append(out, logFromRow(row))
	}
	return out, nil
}

func (s *Service) resolveScript(ctx context.Context, tx *gorm.DB, repo repository, workspaceID uint64, actorID sql.NullInt64, taskName, scriptUID, scriptType, command string) (scriptRecord, error) {
	if scriptUID != "" {
		script, err := repo.activeScriptByUID(ctx, workspaceID, scriptUID)
		if err != nil {
			return scriptRecord{}, err
		}
		if script.TemplateID == 0 {
			return scriptRecord{}, apperror.New(http.StatusNotFound, 404201, "script not found or disabled")
		}
		if strings.TrimSpace(script.Content) == "" {
			return scriptRecord{}, apperror.New(http.StatusBadRequest, 400202, "script content cannot be empty")
		}
		if err := validateScriptApproval(script); err != nil {
			return scriptRecord{}, err
		}
		if err := validateCommandSafety(script.Content, s.cfg.Command); err != nil {
			return scriptRecord{}, err
		}
		return script, nil
	}
	templateUID, err := uid.New()
	if err != nil {
		return scriptRecord{}, err
	}
	versionUID, err := uid.New()
	if err != nil {
		return scriptRecord{}, err
	}
	inlineName := "__inline_" + limit(taskName, 80) + "_" + templateUID[:8]
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO script_templates(uid, workspace_id, name, description, script_type, risk_level, status, created_by)
		 VALUES (?, ?, ?, 'Inline command generated by task creation', ?, 'medium', 'active', ?)`,
		templateUID, workspaceID, inlineName, scriptType, actorID,
	).Error; err != nil {
		return scriptRecord{}, err
	}
	var templateID uint64
	if err := tx.WithContext(ctx).Raw("SELECT id FROM script_templates WHERE uid = ? LIMIT 1", templateUID).Scan(&templateID).Error; err != nil {
		return scriptRecord{}, err
	}
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO script_versions(uid, template_id, version_no, title, content, checksum, status, change_summary, created_by)
		 VALUES (?, ?, 1, ?, ?, ?, 'active', 'inline command', ?)`,
		versionUID, templateID, taskName, command, checksum(command), actorID,
	).Error; err != nil {
		return scriptRecord{}, err
	}
	return scriptRecord{
		TemplateID:  templateID,
		TemplateUID: templateUID,
		VersionID:   mustVersionID(ctx, tx, versionUID),
		VersionUID:  versionUID,
		Name:        inlineName,
		ScriptType:  scriptType,
		Content:     command,
		Status:      "active",
		VersionNo:   1,
	}, nil
}

func (s *Service) resolveTargets(ctx context.Context, repo repository, workspaceID uint64, agentUIDs, hostUIDs []string) ([]targetSelection, error) {
	agentRows, err := repo.targetAgentsByUIDs(ctx, workspaceID, normalizeIDs(agentUIDs))
	if err != nil {
		return nil, err
	}
	hostRows, err := repo.targetAgentsByHostUIDs(ctx, workspaceID, normalizeIDs(hostUIDs))
	if err != nil {
		return nil, err
	}
	seen := map[uint64]bool{}
	targets := make([]targetSelection, 0, len(agentRows)+len(hostRows))
	appendRow := func(row targetCandidate, source string) {
		if row.AgentID == 0 || seen[row.AgentID] {
			return
		}
		seen[row.AgentID] = true
		targets = append(targets, targetSelection{
			SourceType: source,
			AgentID:    row.AgentID,
			AgentUID:   row.AgentUID,
			AgentName:  row.AgentName,
			HostID:     row.HostID,
			HostUID:    row.HostUID.String,
			HostName:   row.HostName.String,
			IP:         row.IP.String,
		})
	}
	for _, row := range agentRows {
		appendRow(row, "agent")
	}
	for _, row := range hostRows {
		appendRow(row, "host")
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].AgentUID < targets[j].AgentUID
	})
	return targets, nil
}

func insertTarget(ctx context.Context, tx *gorm.DB, taskID, runID uint64, target targetSelection) error {
	var taskTargetID uint64
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO task_targets(task_id, target_type, agent_id, host_id, selector)
		 VALUES (?, ?, ?, ?, ?)`,
		taskID, target.SourceType, target.AgentID, target.HostID, jsonNull(target),
	).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Raw("SELECT LAST_INSERT_ID()").Scan(&taskTargetID).Error; err != nil {
		return err
	}
	runTargetUID, err := uid.New()
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec(
		`INSERT INTO task_run_targets(uid, run_id, task_target_id, agent_id, host_id, target_snapshot, status)
		 VALUES (?, ?, ?, ?, ?, ?, 'pending')`,
		runTargetUID, runID, taskTargetID, target.AgentID, target.HostID, jsonNull(target),
	).Error
}

func transitionRun(ctx context.Context, tx *gorm.DB, runID uint64, from, to Status, message string) error {
	if !CanTransition(from, to) {
		return apperror.New(http.StatusConflict, 409305, "illegal task status transition")
	}
	if err := tx.WithContext(ctx).Exec(
		`UPDATE task_runs
		    SET status = ?, queued_at = CASE WHEN ? = 'queued' THEN COALESCE(queued_at, NOW(3)) ELSE queued_at END
		  WHERE id = ? AND status = ?`,
		string(to), string(to), runID, string(from),
	).Error; err != nil {
		return err
	}
	return writeRunEvent(ctx, tx, runID, 0, 0, string(from), string(to), message, nil)
}

type lockedTarget struct {
	ID      uint64
	RunID   uint64
	Status  string
	AgentID sql.NullInt64
}

func lockAgentTarget(ctx context.Context, tx *gorm.DB, targetUID string, workspaceID, agentID uint64) (lockedTarget, error) {
	var target lockedTarget
	err := tx.WithContext(ctx).Raw(
		`SELECT rt.id, rt.run_id, rt.status, rt.agent_id
		   FROM task_run_targets rt
		   JOIN task_runs tr ON tr.id = rt.run_id
		  WHERE rt.uid = ?
		    AND tr.workspace_id = ?
		    AND rt.agent_id = ?
		  LIMIT 1
		  FOR UPDATE`,
		targetUID, workspaceID, agentID,
	).Scan(&target).Error
	return target, err
}

func refreshRunAggregate(ctx context.Context, tx *gorm.DB, runID uint64) error {
	return tx.WithContext(ctx).Exec(
		`UPDATE task_runs tr
		    JOIN (
		      SELECT run_id,
		             COUNT(*) AS total_count,
		             SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) AS success_count,
		             SUM(CASE WHEN status IN ('failed', 'timeout') THEN 1 ELSE 0 END) AS failed_count,
		             SUM(CASE WHEN status = 'timeout' THEN 1 ELSE 0 END) AS timeout_count,
		             SUM(CASE WHEN status = 'canceled' THEN 1 ELSE 0 END) AS canceled_count,
		             SUM(CASE WHEN status IN ('pending', 'queued', 'running', 'canceling') THEN 1 ELSE 0 END) AS active_count,
		             SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END) AS running_count,
		             SUM(CASE WHEN status = 'canceling' THEN 1 ELSE 0 END) AS canceling_count
		        FROM task_run_targets
		       WHERE run_id = ?
		       GROUP BY run_id
		    ) agg ON agg.run_id = tr.id
		    SET tr.total_targets = agg.total_count,
		        tr.success_targets = agg.success_count,
		        tr.failed_targets = agg.failed_count,
		        tr.canceled_targets = agg.canceled_count,
		        tr.status = CASE
		          WHEN agg.canceling_count > 0 THEN 'canceling'
		          WHEN agg.active_count > 0 AND agg.running_count > 0 THEN 'running'
		          WHEN agg.active_count > 0 THEN 'queued'
		          WHEN agg.failed_count > 0 AND agg.timeout_count = agg.failed_count THEN 'timeout'
		          WHEN agg.failed_count > 0 THEN 'failed'
		          WHEN agg.canceled_count > 0 THEN 'canceled'
		          ELSE 'success'
		        END,
		        tr.finished_at = CASE WHEN agg.active_count = 0 THEN COALESCE(tr.finished_at, NOW(3)) ELSE tr.finished_at END
		  WHERE tr.id = ?`,
		runID, runID,
	).Error
}

func writeRunEvent(ctx context.Context, tx *gorm.DB, runID, targetID, attemptID uint64, from, to, message string, payload interface{}) error {
	return tx.WithContext(ctx).Exec(
		`INSERT INTO task_run_events(run_id, run_target_id, attempt_id, event_type, from_status, to_status, message, payload)
		 VALUES (?, ?, ?, 'status_transition', ?, ?, ?, ?)`,
		runID, nullUint64(targetID), nullUint64(attemptID), nullString(from), nullString(to), nullString(message), jsonNull(payload),
	).Error
}

func (s *Service) workspace(ctx context.Context) (workspaceRecord, *apperror.Error) {
	workspace, err := s.repo.defaultWorkspace(ctx, s.cfg.Bootstrap.WorkspaceSlug)
	if err != nil {
		return workspaceRecord{}, apperror.Wrap(http.StatusInternalServerError, 500304, "load workspace failed", err)
	}
	if workspace.ID == 0 {
		return workspaceRecord{}, apperror.New(http.StatusInternalServerError, 500304, "default workspace is not initialized")
	}
	return workspace, nil
}

func summaryFromRun(row runRecord) TaskSummary {
	return TaskSummary{
		ID:             row.UID,
		TaskID:         row.TaskUID,
		Name:           row.TaskName,
		Description:    row.Description.String,
		Status:         row.Status,
		TimeoutSeconds: row.TimeoutSeconds,
		TargetCount:    row.TotalTargets,
		SuccessCount:   row.SuccessTargets,
		FailedCount:    row.FailedTargets,
		CanceledCount:  row.CanceledTargets,
		RunningCount:   row.RunningTargets,
		QueuedCount:    row.QueuedTargets,
		CreatedBy:      row.CreatedBy.String,
		CreatedAt:      row.CreatedAt,
		QueuedAt:       row.QueuedAt.String,
		StartedAt:      row.StartedAt.String,
		FinishedAt:     row.FinishedAt.String,
		ErrorMessage:   row.ErrorMessage.String,
	}
}

func targetsFromRows(rows []runTargetRecord) []TaskTargetSummary {
	targets := make([]TaskTargetSummary, 0, len(rows))
	for _, row := range rows {
		var exitCode *int
		if row.ExitCode.Valid {
			value := int(row.ExitCode.Int64)
			exitCode = &value
		}
		targets = append(targets, TaskTargetSummary{
			ID:           row.UID,
			Status:       row.Status,
			AgentID:      row.AgentUID.String,
			AgentName:    row.AgentName.String,
			AgentStatus:  row.AgentStatus.String,
			HostID:       row.HostUID.String,
			HostName:     row.HostName.String,
			ExitCode:     exitCode,
			ErrorMessage: row.ErrorMessage.String,
			StartedAt:    row.StartedAt.String,
			FinishedAt:   row.FinishedAt.String,
			CreatedAt:    row.CreatedAt,
		})
	}
	return targets
}

func agentTaskFromRow(row agentTaskRecord) AgentTask {
	return AgentTask{
		TargetID:       row.TargetID,
		RunID:          row.RunID,
		Name:           row.TaskName,
		Description:    row.Description.String,
		ScriptType:     row.ScriptType,
		Command:        row.Command,
		TimeoutSeconds: row.TimeoutSeconds,
		Status:         row.Status,
		AgentName:      row.AgentName,
		HostName:       row.HostName.String,
	}
}

func logFromRow(row logRecord) TaskLogEntry {
	return TaskLogEntry{
		ID:        row.ID,
		RunID:     row.RunID,
		TargetID:  row.TargetID,
		Sequence:  row.Sequence,
		Stream:    row.Stream,
		Content:   row.Content,
		CreatedAt: row.CreatedAt,
		AgentName: row.AgentName.String,
		HostName:  row.HostName.String,
	}
}

func normalizeIDs(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func normalizeScriptType(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return "shell"
	}
	return value
}

func validScriptType(value string) bool {
	switch value {
	case "shell", "powershell", "bash":
		return true
	default:
		return false
	}
}

func checksum(content string) string {
	sum := sha256.Sum256([]byte(content))
	return hex.EncodeToString(sum[:])
}

func parseTime(value string) sql.NullTime {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullTime{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return sql.NullTime{Time: parsed, Valid: true}
		}
	}
	return sql.NullTime{}
}

func nullableInt(value *int) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
}

func nullString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	return sql.NullString{String: value, Valid: value != ""}
}

func nullUint64(value uint64) sql.NullInt64 {
	if value == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(value), Valid: true}
}

func jsonNull(value interface{}) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	bytes, err := json.Marshal(value)
	if err != nil || string(bytes) == "null" {
		return sql.NullString{}
	}
	return sql.NullString{String: string(bytes), Valid: true}
}

func mustVersionID(ctx context.Context, tx *gorm.DB, versionUID string) uint64 {
	var id uint64
	_ = tx.WithContext(ctx).Raw("SELECT id FROM script_versions WHERE uid = ? LIMIT 1", versionUID).Scan(&id).Error
	return id
}

func limit(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func normalizePage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func normalizeLimit(value, fallback, maxValue int) int {
	if value <= 0 {
		value = fallback
	}
	if value > maxValue {
		value = maxValue
	}
	return value
}

func normalizeDateStart(value string) string {
	return strings.TrimSpace(value)
}

func normalizeDateEnd(value string) string {
	value = strings.TrimSpace(value)
	if len(value) == len("2006-01-02") {
		return value + " 23:59:59"
	}
	return value
}

func validateScriptApproval(script scriptRecord) error {
	if !script.ApprovalRequired {
		return nil
	}
	status := strings.ToLower(strings.TrimSpace(script.LatestApprovalStatus.String))
	switch status {
	case "approved":
		return nil
	case "pending":
		return apperror.New(http.StatusConflict, 409306, "script approval is pending")
	case "rejected", "canceled":
		return apperror.New(http.StatusBadRequest, 400309, "script latest revision is not approved")
	default:
		return apperror.New(http.StatusBadRequest, 400309, "script requires approval before execution")
	}
}

func validateCommandSafety(command string, policy config.CommandPolicyConfig) *apperror.Error {
	if dangerousCommandPattern.MatchString(command) {
		return apperror.New(http.StatusBadRequest, 400310, "command contains high-risk operation and is blocked")
	}
	for _, pattern := range policy.DenyPatterns {
		matched, err := regexp.MatchString(pattern, command)
		if err != nil {
			return apperror.Wrap(http.StatusInternalServerError, 500314, "invalid command deny pattern", err)
		}
		if matched {
			return apperror.New(http.StatusBadRequest, 400310, "command is blocked by denylist policy")
		}
	}
	if len(policy.AllowPatterns) > 0 {
		for _, pattern := range policy.AllowPatterns {
			matched, err := regexp.MatchString(pattern, command)
			if err != nil {
				return apperror.Wrap(http.StatusInternalServerError, 500315, "invalid command allow pattern", err)
			}
			if matched {
				return nil
			}
		}
		return apperror.New(http.StatusBadRequest, 400312, "command is not allowed by allowlist policy")
	}
	return nil
}
