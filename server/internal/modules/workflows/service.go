package workflows

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"opspilot/server/internal/config"
	"opspilot/server/internal/modules/tasks"
	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/audit"
	"opspilot/server/internal/shared/uid"
)

type Service struct {
	db  *gorm.DB
	cfg config.Config
}

type AuditContext struct {
	ActorUID      string
	IP            string
	UserAgent     string
	TraceID       string
	RequestMethod string
	RequestPath   string
}

type ListInput struct {
	Keyword  string
	Status   string
	Page     int
	PageSize int
}

type CreateInput struct {
	Name        string
	Description string
	Definition  string
	Audit       AuditContext
}

type UpdateInput struct {
	ID          string
	Name        string
	Description string
	Definition  string
	Audit       AuditContext
}

type CopyInput struct {
	ID    string
	Name  string
	Audit AuditContext
}

type RunInput struct {
	ID             string
	TriggerType    string
	IdempotencyKey string
	Input          string
	Audit          AuditContext
}

type CancelInput struct {
	ID     string
	Reason string
	Audit  AuditContext
}

type RetryInput struct {
	ID    string
	Audit AuditContext
}

type ApprovalInput struct {
	RunID   string
	NodeID  string
	Comment string
	Audit   AuditContext
}

type RetryNodeInput struct {
	RunID  string
	NodeID string
	Audit  AuditContext
}

type DefinitionSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Version     uint   `json:"version"`
	Status      string `json:"status"`
	NodeCount   int    `json:"nodeCount"`
	EdgeCount   int    `json:"edgeCount"`
	CreatedBy   string `json:"createdBy,omitempty"`
	PublishedAt string `json:"publishedAt,omitempty"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type DefinitionDetail struct {
	DefinitionSummary
	Definition string `json:"definition"`
}

type VersionSummary struct {
	ID             string `json:"id"`
	WorkflowID     string `json:"workflowId"`
	Version        uint   `json:"version"`
	Status         string `json:"status"`
	DefinitionHash string `json:"definitionHash"`
	CreatedBy      string `json:"createdBy,omitempty"`
	PublishedAt    string `json:"publishedAt,omitempty"`
	CreatedAt      string `json:"createdAt"`
}

type DefinitionListResult struct {
	Items    []DefinitionSummary `json:"items"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"pageSize"`
}

type RunSummary struct {
	ID              string `json:"id"`
	WorkflowID      string `json:"workflowId"`
	WorkflowName    string `json:"workflowName"`
	WorkflowVersion uint   `json:"workflowVersion"`
	Status          string `json:"status"`
	TriggerType     string `json:"triggerType"`
	TotalNodes      uint   `json:"totalNodes"`
	SuccessNodes    uint   `json:"successNodes"`
	FailedNodes     uint   `json:"failedNodes"`
	SkippedNodes    uint   `json:"skippedNodes"`
	ErrorMessage    string `json:"errorMessage,omitempty"`
	CreatedBy       string `json:"createdBy,omitempty"`
	QueuedAt        string `json:"queuedAt,omitempty"`
	StartedAt       string `json:"startedAt,omitempty"`
	FinishedAt      string `json:"finishedAt,omitempty"`
	CreatedAt       string `json:"createdAt"`
}

type RunNodeSummary struct {
	ID           uint64 `json:"id"`
	NodeID       string `json:"nodeId"`
	NodeType     string `json:"nodeType"`
	NodeName     string `json:"nodeName,omitempty"`
	Status       string `json:"status"`
	TaskRunID    string `json:"taskRunId,omitempty"`
	Input        string `json:"input,omitempty"`
	Output       string `json:"output,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
	Attempts     uint   `json:"attempts"`
	QueuedAt     string `json:"queuedAt,omitempty"`
	StartedAt    string `json:"startedAt,omitempty"`
	FinishedAt   string `json:"finishedAt,omitempty"`
}

type RunEventSummary struct {
	ID        uint64 `json:"id"`
	NodeID    string `json:"nodeId,omitempty"`
	EventType string `json:"eventType"`
	Message   string `json:"message,omitempty"`
	Actor     string `json:"actor,omitempty"`
	Payload   string `json:"payload,omitempty"`
	CreatedAt string `json:"createdAt"`
}

type RunDetail struct {
	RunSummary
	Input      string            `json:"input,omitempty"`
	Output     string            `json:"output,omitempty"`
	Definition string            `json:"definition"`
	Nodes      []RunNodeSummary  `json:"nodes"`
	Events     []RunEventSummary `json:"events"`
}

type RunListResult struct {
	Items    []RunSummary `json:"items"`
	Total    int64        `json:"total"`
	Page     int          `json:"page"`
	PageSize int          `json:"pageSize"`
}

type workspaceRecord struct {
	ID uint64
}

type workflowRecord struct {
	ID          uint64
	UID         string
	WorkspaceID uint64
	Name        string
	Description sql.NullString
	Definition  string
	Version     uint
	Status      string
	CreatedByID sql.NullInt64
	CreatedBy   sql.NullString
	PublishedAt sql.NullString
	CreatedAt   string
	UpdatedAt   string
}

type versionRecord struct {
	ID             uint64
	UID            string
	WorkflowUID    string
	VersionNo      uint
	Status         string
	DefinitionHash string
	CreatedBy      sql.NullString
	PublishedAt    sql.NullString
	CreatedAt      string
}

type runRecord struct {
	ID                 uint64
	WorkflowDBID       uint64
	UID                string
	WorkflowUID        string
	WorkflowName       string
	WorkflowVersion    uint
	DefinitionSnapshot string
	Status             string
	TriggerType        string
	Input              sql.NullString
	Output             sql.NullString
	TotalNodes         uint
	SuccessNodes       uint
	FailedNodes        uint
	SkippedNodes       uint
	ErrorMessage       sql.NullString
	CreatedBy          sql.NullString
	QueuedAt           sql.NullString
	StartedAt          sql.NullString
	FinishedAt         sql.NullString
	CreatedAt          string
}

type nodeRecord struct {
	ID           uint64
	NodeID       string
	NodeType     string
	NodeName     sql.NullString
	Status       string
	TaskRunUID   sql.NullString
	Input        sql.NullString
	Output       sql.NullString
	ErrorMessage sql.NullString
	Attempts     uint
	QueuedAt     sql.NullString
	StartedAt    sql.NullString
	FinishedAt   sql.NullString
}

type eventRecord struct {
	ID        uint64
	NodeID    sql.NullString
	EventType string
	Message   sql.NullString
	Actor     sql.NullString
	Payload   sql.NullString
	CreatedAt string
}

func NewService(db *gorm.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

func (s *Service) List(ctx context.Context, input ListInput) (DefinitionListResult, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return DefinitionListResult{}, appErr
	}
	input.Page, input.PageSize = normalizePage(input.Page, input.PageSize)
	where, args := workflowListWhere(workspace.ID, input.Keyword, input.Status)
	var total int64
	if err := s.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM workflow_definitions wd "+where, args...).Scan(&total).Error; err != nil {
		return DefinitionListResult{}, apperror.Wrap(http.StatusInternalServerError, 501001, "count workflows failed", err)
	}
	rows, err := s.workflowRows(ctx, where+" ORDER BY wd.updated_at DESC LIMIT ? OFFSET ?", append(args, input.PageSize, (input.Page-1)*input.PageSize)...)
	if err != nil {
		return DefinitionListResult{}, apperror.Wrap(http.StatusInternalServerError, 501002, "list workflows failed", err)
	}
	items := make([]DefinitionSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, definitionSummary(row))
	}
	return DefinitionListResult{Items: items, Total: total, Page: input.Page, PageSize: input.PageSize}, nil
}

func (s *Service) Get(ctx context.Context, id string) (DefinitionDetail, *apperror.Error) {
	row, appErr := s.workflowByUID(ctx, id)
	if appErr != nil {
		return DefinitionDetail{}, appErr
	}
	return DefinitionDetail{DefinitionSummary: definitionSummary(row), Definition: row.Definition}, nil
}

func (s *Service) Create(ctx context.Context, input CreateInput) (DefinitionDetail, *apperror.Error) {
	name := strings.TrimSpace(input.Name)
	description := strings.TrimSpace(input.Description)
	if name == "" {
		return DefinitionDetail{}, apperror.New(http.StatusBadRequest, 401001, "workflow name is required")
	}
	_, normalized, err := normalizeAndValidateDefinition(input.Definition)
	if err != nil {
		return DefinitionDetail{}, apperror.New(http.StatusBadRequest, 401002, err.Error())
	}
	var created DefinitionDetail
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		actorID, err := audit.UserIDByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		workflowUID, err := uid.New()
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO workflow_definitions(uid, workspace_id, name, description, definition, version, status, created_by)
			 VALUES (?, ?, ?, ?, ?, 1, 'draft', ?)`,
			workflowUID, workspace.ID, name, nullString(description), normalized, actorID,
		).Error; err != nil {
			return err
		}
		row, err := workflowRowByUID(ctx, tx, workspace.ID, workflowUID)
		if err != nil {
			return err
		}
		if err := insertWorkflowVersion(ctx, tx, row.ID, row.Version, row.Definition, "draft", actorID); err != nil {
			return err
		}
		created = DefinitionDetail{DefinitionSummary: definitionSummary(row), Definition: row.Definition}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "workflow.create",
			ResourceType:  "workflow",
			ResourceID:    sql.NullInt64{Int64: int64(row.ID), Valid: row.ID != 0},
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
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return DefinitionDetail{}, apperror.New(http.StatusConflict, 409001, "workflow name already exists")
		}
		return DefinitionDetail{}, wrapAppError(txErr, 501003, "create workflow failed")
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (DefinitionDetail, *apperror.Error) {
	name := strings.TrimSpace(input.Name)
	description := strings.TrimSpace(input.Description)
	id := strings.TrimSpace(input.ID)
	if id == "" {
		return DefinitionDetail{}, apperror.New(http.StatusBadRequest, 401003, "workflow id is required")
	}
	if name == "" {
		return DefinitionDetail{}, apperror.New(http.StatusBadRequest, 401001, "workflow name is required")
	}
	_, normalized, err := normalizeAndValidateDefinition(input.Definition)
	if err != nil {
		return DefinitionDetail{}, apperror.New(http.StatusBadRequest, 401002, err.Error())
	}
	var updated DefinitionDetail
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		before, err := workflowRowByUID(ctx, tx, workspace.ID, id)
		if err != nil {
			return err
		}
		if before.ID == 0 {
			return apperror.New(http.StatusNotFound, 404001, "workflow not found")
		}
		if before.Status == "archived" {
			return apperror.New(http.StatusConflict, 409002, "archived workflow cannot be updated")
		}
		actorID, err := audit.UserIDByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE workflow_definitions
			    SET name = ?, description = ?, definition = ?, version = version + 1,
			        status = CASE WHEN status = 'active' THEN 'draft' ELSE status END
			  WHERE id = ?`,
			name, nullString(description), normalized, before.ID,
		).Error; err != nil {
			return err
		}
		after, err := workflowRowByUID(ctx, tx, workspace.ID, id)
		if err != nil {
			return err
		}
		if err := insertWorkflowVersion(ctx, tx, after.ID, after.Version, after.Definition, "draft", actorID); err != nil {
			return err
		}
		updated = DefinitionDetail{DefinitionSummary: definitionSummary(after), Definition: after.Definition}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "workflow.update",
			ResourceType:  "workflow",
			ResourceID:    sql.NullInt64{Int64: int64(before.ID), Valid: true},
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			Before:        DefinitionDetail{DefinitionSummary: definitionSummary(before), Definition: before.Definition},
			After:         updated,
		})
		return nil
	})
	if txErr != nil {
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return DefinitionDetail{}, apperror.New(http.StatusConflict, 409001, "workflow name already exists")
		}
		return DefinitionDetail{}, wrapAppError(txErr, 501004, "update workflow failed")
	}
	return updated, nil
}

func (s *Service) Publish(ctx context.Context, id string, auditCtx AuditContext) (DefinitionDetail, *apperror.Error) {
	id = strings.TrimSpace(id)
	var published DefinitionDetail
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		before, err := workflowRowByUID(ctx, tx, workspace.ID, id)
		if err != nil {
			return err
		}
		if before.ID == 0 {
			return apperror.New(http.StatusNotFound, 404001, "workflow not found")
		}
		if before.Status == "archived" {
			return apperror.New(http.StatusConflict, 409003, "archived workflow cannot be published")
		}
		if _, _, err := normalizeAndValidateDefinition(before.Definition); err != nil {
			return apperror.New(http.StatusBadRequest, 401002, err.Error())
		}
		actorID, err := audit.UserIDByUID(ctx, tx, auditCtx.ActorUID)
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec("UPDATE workflow_definitions SET status = 'active', published_at = NOW(3) WHERE id = ?", before.ID).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE workflow_versions
			    SET status = 'published', published_at = NOW(3)
			  WHERE workflow_id = ? AND version_no = ?`,
			before.ID, before.Version,
		).Error; err != nil {
			return err
		}
		after, err := workflowRowByUID(ctx, tx, workspace.ID, id)
		if err != nil {
			return err
		}
		published = DefinitionDetail{DefinitionSummary: definitionSummary(after), Definition: after.Definition}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "workflow.publish",
			ResourceType:  "workflow",
			ResourceID:    sql.NullInt64{Int64: int64(before.ID), Valid: true},
			IP:            auditCtx.IP,
			UserAgent:     auditCtx.UserAgent,
			TraceID:       auditCtx.TraceID,
			RequestMethod: auditCtx.RequestMethod,
			RequestPath:   auditCtx.RequestPath,
			Before:        DefinitionDetail{DefinitionSummary: definitionSummary(before), Definition: before.Definition},
			After:         published,
		})
		return nil
	})
	if txErr != nil {
		return DefinitionDetail{}, wrapAppError(txErr, 501005, "publish workflow failed")
	}
	return published, nil
}

func (s *Service) Disable(ctx context.Context, id string, auditCtx AuditContext) (DefinitionDetail, *apperror.Error) {
	id = strings.TrimSpace(id)
	var disabled DefinitionDetail
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		before, err := workflowRowByUID(ctx, tx, workspace.ID, id)
		if err != nil {
			return err
		}
		if before.ID == 0 {
			return apperror.New(http.StatusNotFound, 404001, "workflow not found")
		}
		if before.Status == "archived" {
			return apperror.New(http.StatusConflict, 409011, "archived workflow cannot be disabled")
		}
		actorID, err := audit.UserIDByUID(ctx, tx, auditCtx.ActorUID)
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec("UPDATE workflow_definitions SET status = 'disabled' WHERE id = ?", before.ID).Error; err != nil {
			return err
		}
		after, err := workflowRowByUID(ctx, tx, workspace.ID, id)
		if err != nil {
			return err
		}
		disabled = DefinitionDetail{DefinitionSummary: definitionSummary(after), Definition: after.Definition}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "workflow.disable",
			ResourceType:  "workflow",
			ResourceID:    sql.NullInt64{Int64: int64(before.ID), Valid: true},
			IP:            auditCtx.IP,
			UserAgent:     auditCtx.UserAgent,
			TraceID:       auditCtx.TraceID,
			RequestMethod: auditCtx.RequestMethod,
			RequestPath:   auditCtx.RequestPath,
			Before:        DefinitionDetail{DefinitionSummary: definitionSummary(before), Definition: before.Definition},
			After:         disabled,
		})
		return nil
	})
	if txErr != nil {
		return DefinitionDetail{}, wrapAppError(txErr, 501018, "disable workflow failed")
	}
	return disabled, nil
}

func (s *Service) Copy(ctx context.Context, input CopyInput) (DefinitionDetail, *apperror.Error) {
	id := strings.TrimSpace(input.ID)
	name := strings.TrimSpace(input.Name)
	var copied DefinitionDetail
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		source, err := workflowRowByUID(ctx, tx, workspace.ID, id)
		if err != nil {
			return err
		}
		if source.ID == 0 {
			return apperror.New(http.StatusNotFound, 404001, "workflow not found")
		}
		actorID, err := audit.UserIDByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		if name == "" {
			generated, err := nextWorkflowCopyName(ctx, tx, workspace.ID, source.Name)
			if err != nil {
				return err
			}
			name = generated
		}
		workflowUID, err := uid.New()
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO workflow_definitions(uid, workspace_id, name, description, definition, version, status, created_by)
			 VALUES (?, ?, ?, ?, ?, 1, 'draft', ?)`,
			workflowUID, workspace.ID, name, source.Description, source.Definition, actorID,
		).Error; err != nil {
			return err
		}
		row, err := workflowRowByUID(ctx, tx, workspace.ID, workflowUID)
		if err != nil {
			return err
		}
		if err := insertWorkflowVersion(ctx, tx, row.ID, row.Version, row.Definition, "draft", actorID); err != nil {
			return err
		}
		copied = DefinitionDetail{DefinitionSummary: definitionSummary(row), Definition: row.Definition}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "workflow.copy",
			ResourceType:  "workflow",
			ResourceID:    sql.NullInt64{Int64: int64(row.ID), Valid: true},
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			Before:        DefinitionDetail{DefinitionSummary: definitionSummary(source), Definition: source.Definition},
			After:         copied,
		})
		return nil
	})
	if txErr != nil {
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return DefinitionDetail{}, apperror.New(http.StatusConflict, 409001, "workflow name already exists")
		}
		return DefinitionDetail{}, wrapAppError(txErr, 501019, "copy workflow failed")
	}
	return copied, nil
}

func (s *Service) ListVersions(ctx context.Context, id string) ([]VersionSummary, *apperror.Error) {
	workflow, appErr := s.workflowByUID(ctx, id)
	if appErr != nil {
		return nil, appErr
	}
	rows, err := workflowVersionRows(ctx, s.db, workflow.ID)
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 501020, "list workflow versions failed", err)
	}
	items := make([]VersionSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, versionSummary(row))
	}
	return items, nil
}

func insertWorkflowVersion(ctx context.Context, tx *gorm.DB, workflowID uint64, version uint, definition string, status string, actorID sql.NullInt64) error {
	versionUID, err := uid.New()
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec(
		`INSERT INTO workflow_versions(uid, workflow_id, version_no, definition, definition_hash, status, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   definition = VALUES(definition),
		   definition_hash = VALUES(definition_hash),
		   status = VALUES(status),
		   created_by = VALUES(created_by)`,
		versionUID, workflowID, version, definition, definitionHash(definition), normalizeWorkflowVersionStatus(status), actorID,
	).Error
}

func definitionHash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", sum)
}

func normalizeWorkflowVersionStatus(value string) string {
	switch strings.TrimSpace(value) {
	case "published", "deprecated", "archived":
		return strings.TrimSpace(value)
	default:
		return "draft"
	}
}

func (s *Service) Run(ctx context.Context, input RunInput) (RunDetail, *apperror.Error) {
	input.ID = strings.TrimSpace(input.ID)
	triggerType := normalizeTriggerType(input.TriggerType)
	inputJSON := strings.TrimSpace(input.Input)
	if inputJSON != "" && !json.Valid([]byte(inputJSON)) {
		return RunDetail{}, apperror.New(http.StatusBadRequest, 401004, "workflow input must be valid JSON")
	}
	var detail RunDetail
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		workflow, err := workflowRowByUID(ctx, tx, workspace.ID, input.ID)
		if err != nil {
			return err
		}
		if workflow.ID == 0 {
			return apperror.New(http.StatusNotFound, 404001, "workflow not found")
		}
		if workflow.Status != "active" {
			return errWorkflowNotActive()
		}
		_, normalized, err := normalizeAndValidateDefinition(workflow.Definition)
		if err != nil {
			return apperror.New(http.StatusBadRequest, 401002, err.Error())
		}
		actorID, err := audit.UserIDByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		runID, runUID, err := createWorkflowRunWithSnapshotTx(
			ctx,
			tx,
			workspace.ID,
			workflow.ID,
			workflow.Version,
			normalized,
			triggerType,
			strings.TrimSpace(input.IdempotencyKey),
			inputJSON,
			actorID,
			nil,
		)
		if err != nil {
			return err
		}
		loaded, err := runDetailByUID(ctx, tx, workspace.ID, runUID)
		if err != nil {
			return err
		}
		detail = loaded
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "workflow.run",
			ResourceType:  "workflow_run",
			ResourceID:    sql.NullInt64{Int64: int64(runID), Valid: runID != 0},
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			After:         detail.RunSummary,
		})
		return nil
	})
	if txErr != nil {
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return RunDetail{}, apperror.New(http.StatusConflict, 409005, "workflow run idempotency key already exists")
		}
		return RunDetail{}, wrapAppError(txErr, 501006, "create workflow run failed")
	}
	return detail, nil
}

func (s *Service) RetryRun(ctx context.Context, input RetryInput) (RunDetail, *apperror.Error) {
	var retried RunDetail
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		run, err := runByUID(ctx, tx, workspace.ID, strings.TrimSpace(input.ID))
		if err != nil {
			return err
		}
		if run.ID == 0 {
			return apperror.New(http.StatusNotFound, 404002, "workflow run not found")
		}
		if activeWorkflowStatus(run.Status) {
			return apperror.New(http.StatusConflict, 409007, "active workflow run cannot be retried")
		}
		if !retryableWorkflowStatus(run.Status) {
			return apperror.New(http.StatusConflict, 409008, "workflow run is not retryable")
		}
		actorID, err := audit.UserIDByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		runID, runUID, err := createWorkflowRunWithSnapshotTx(
			ctx,
			tx,
			workspace.ID,
			run.WorkflowDBID,
			run.WorkflowVersion,
			run.DefinitionSnapshot,
			"retry",
			"",
			run.Input.String,
			actorID,
			map[string]interface{}{"sourceRunId": run.UID},
		)
		if err != nil {
			return err
		}
		loaded, err := runDetailByUID(ctx, tx, workspace.ID, runUID)
		if err != nil {
			return err
		}
		retried = loaded
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "workflow.retry",
			ResourceType:  "workflow_run",
			ResourceID:    sql.NullInt64{Int64: int64(runID), Valid: runID != 0},
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			After:         retried.RunSummary,
		})
		return nil
	})
	if txErr != nil {
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return RunDetail{}, apperror.New(http.StatusConflict, 409005, "workflow run idempotency key already exists")
		}
		return RunDetail{}, wrapAppError(txErr, 501016, "retry workflow run failed")
	}
	return retried, nil
}

func (s *Service) RetryNode(ctx context.Context, input RetryNodeInput) (RunDetail, *apperror.Error) {
	var detail RunDetail
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		run, err := runByUID(ctx, tx, workspace.ID, strings.TrimSpace(input.RunID))
		if err != nil {
			return err
		}
		if run.ID == 0 {
			return apperror.New(http.StatusNotFound, 404002, "workflow run not found")
		}
		if activeWorkflowStatus(run.Status) {
			return apperror.New(http.StatusConflict, 409011, "active workflow run node cannot be retried")
		}
		if !retryableWorkflowStatus(run.Status) {
			return apperror.New(http.StatusConflict, 409012, "workflow run node is not retryable")
		}
		var def Definition
		if err := json.Unmarshal([]byte(run.DefinitionSnapshot), &def); err != nil {
			return err
		}
		targetNodeID := strings.TrimSpace(input.NodeID)
		if targetNodeID == "" {
			return apperror.New(http.StatusBadRequest, 401008, "workflow node id is required")
		}
		nodes, err := workflowNodeStates(ctx, tx, run.ID)
		if err != nil {
			return err
		}
		var target *workflowRunNodeState
		for index := range nodes {
			if nodes[index].NodeID == targetNodeID {
				target = &nodes[index]
				break
			}
		}
		if target == nil {
			return apperror.New(http.StatusNotFound, 404004, "workflow node not found")
		}
		if !retryableNodeStatus(target.Status) {
			return apperror.New(http.StatusConflict, 409013, "workflow node is not retryable")
		}
		resetIDs := downstreamNodeIDs(def, targetNodeID)
		if len(resetIDs) == 0 {
			return apperror.New(http.StatusNotFound, 404004, "workflow node not found")
		}
		actorID, err := audit.UserIDByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE workflow_run_nodes
			    SET status = 'pending',
			        task_run_id = NULL,
			        input = NULL,
			        output = NULL,
			        error_message = NULL,
			        queued_at = NULL,
			        started_at = NULL,
			        finished_at = NULL,
			        duration_ms = NULL
			  WHERE run_id = ? AND node_id IN ?`,
			run.ID,
			resetIDs,
		).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE workflow_runs
			    SET status = 'queued', error_message = NULL, finished_at = NULL, duration_ms = NULL
			  WHERE id = ?`,
			run.ID,
		).Error; err != nil {
			return err
		}
		if err := writeWorkflowEvent(ctx, tx, run.ID, targetNodeID, "node_retry", "Workflow node retry requested", actorID, map[string]interface{}{"resetNodeIds": resetIDs}); err != nil {
			return err
		}
		runInput, err := parseWorkflowInput(run.Input.String)
		if err != nil {
			return err
		}
		for index := 0; index <= len(def.Nodes); index++ {
			changed, err := progressReadyNodes(ctx, tx, workspace.ID, run.ID, def, runInput)
			if err != nil {
				return err
			}
			if !changed {
				break
			}
		}
		if err := refreshWorkflowRunAggregate(ctx, tx, run.ID); err != nil {
			return err
		}
		loaded, err := runDetailByUID(ctx, tx, workspace.ID, run.UID)
		if err != nil {
			return err
		}
		detail = loaded
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "workflow.node_retry",
			ResourceType:  "workflow_run",
			ResourceID:    sql.NullInt64{Int64: int64(run.ID), Valid: run.ID != 0},
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			After:         map[string]interface{}{"runId": run.UID, "nodeId": targetNodeID, "resetNodeIds": resetIDs},
		})
		return nil
	})
	if txErr != nil {
		return RunDetail{}, wrapAppError(txErr, 501018, "retry workflow node failed")
	}
	return detail, nil
}

func (s *Service) ApproveNode(ctx context.Context, input ApprovalInput) (RunDetail, *apperror.Error) {
	return s.resolveApprovalNode(ctx, input, true)
}

func (s *Service) RejectNode(ctx context.Context, input ApprovalInput) (RunDetail, *apperror.Error) {
	return s.resolveApprovalNode(ctx, input, false)
}

func (s *Service) resolveApprovalNode(ctx context.Context, input ApprovalInput, approved bool) (RunDetail, *apperror.Error) {
	var detail RunDetail
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		run, err := runByUID(ctx, tx, workspace.ID, strings.TrimSpace(input.RunID))
		if err != nil {
			return err
		}
		if run.ID == 0 {
			return apperror.New(http.StatusNotFound, 404002, "workflow run not found")
		}
		if !activeWorkflowStatus(run.Status) {
			return apperror.New(http.StatusConflict, 409009, "workflow run is not waiting for approval")
		}
		nodes, err := workflowNodeStates(ctx, tx, run.ID)
		if err != nil {
			return err
		}
		targetNodeID := strings.TrimSpace(input.NodeID)
		var target *workflowRunNodeState
		for index := range nodes {
			if nodes[index].NodeID == targetNodeID {
				target = &nodes[index]
				break
			}
		}
		if target == nil {
			return apperror.New(http.StatusNotFound, 404004, "workflow node not found")
		}
		if target.NodeType != "approval" || target.Status != "running" {
			return apperror.New(http.StatusConflict, 409010, "workflow node is not waiting for approval")
		}
		actorID, err := audit.UserIDByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		comment := limitString(strings.TrimSpace(input.Comment), 512)
		status := "success"
		message := "Approval granted"
		errorMessage := sql.NullString{}
		if !approved {
			status = "failed"
			message = "Approval rejected"
			errorMessage = nullString(message)
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE workflow_run_nodes
			    SET status = ?, output = ?, error_message = ?, finished_at = COALESCE(finished_at, NOW(3))
			  WHERE id = ? AND status = 'running'`,
			status,
			jsonStringOrNull(map[string]interface{}{
				"decision": map[string]interface{}{
					"approved": approved,
					"comment":  comment,
					"actorId":  actorID.Int64,
				},
			}),
			errorMessage,
			target.ID,
		).Error; err != nil {
			return err
		}
		if err := writeWorkflowEvent(ctx, tx, run.ID, target.NodeID, "node_"+status, message, actorID, map[string]string{"comment": comment}); err != nil {
			return err
		}
		var def Definition
		if err := json.Unmarshal([]byte(run.DefinitionSnapshot), &def); err != nil {
			return err
		}
		runInput, err := parseWorkflowInput(run.Input.String)
		if err != nil {
			return err
		}
		for index := 0; index <= len(def.Nodes); index++ {
			changed, err := progressReadyNodes(ctx, tx, workspace.ID, run.ID, def, runInput)
			if err != nil {
				return err
			}
			if !changed {
				break
			}
		}
		if err := refreshWorkflowRunAggregate(ctx, tx, run.ID); err != nil {
			return err
		}
		loaded, err := runDetailByUID(ctx, tx, workspace.ID, run.UID)
		if err != nil {
			return err
		}
		detail = loaded
		return nil
	})
	if txErr != nil {
		return RunDetail{}, wrapAppError(txErr, 501017, "resolve approval node failed")
	}
	return detail, nil
}

func (s *Service) ListRuns(ctx context.Context, input ListInput) (RunListResult, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return RunListResult{}, appErr
	}
	if err := reconcileActiveRuns(ctx, s.db, workspace.ID); err != nil {
		return RunListResult{}, apperror.Wrap(http.StatusInternalServerError, 501014, "reconcile workflow runs failed", err)
	}
	input.Page, input.PageSize = normalizePage(input.Page, input.PageSize)
	where, args := runListWhere(workspace.ID, input.Keyword, input.Status)
	var total int64
	if err := s.db.WithContext(ctx).Raw("SELECT COUNT(*) FROM workflow_runs wr JOIN workflow_definitions wd ON wd.id = wr.workflow_id "+where, args...).Scan(&total).Error; err != nil {
		return RunListResult{}, apperror.Wrap(http.StatusInternalServerError, 501007, "count workflow runs failed", err)
	}
	rows, err := runRows(ctx, s.db, where+" ORDER BY wr.created_at DESC LIMIT ? OFFSET ?", append(args, input.PageSize, (input.Page-1)*input.PageSize)...)
	if err != nil {
		return RunListResult{}, apperror.Wrap(http.StatusInternalServerError, 501008, "list workflow runs failed", err)
	}
	items := make([]RunSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, runSummary(row))
	}
	return RunListResult{Items: items, Total: total, Page: input.Page, PageSize: input.PageSize}, nil
}

func (s *Service) GetRun(ctx context.Context, id string) (RunDetail, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return RunDetail{}, appErr
	}
	if err := reconcileRunByUID(ctx, s.db, workspace.ID, strings.TrimSpace(id)); err != nil {
		return RunDetail{}, apperror.Wrap(http.StatusInternalServerError, 501015, "reconcile workflow run failed", err)
	}
	detail, err := runDetailByUID(ctx, s.db, workspace.ID, strings.TrimSpace(id))
	if err != nil {
		return RunDetail{}, apperror.Wrap(http.StatusInternalServerError, 501009, "get workflow run failed", err)
	}
	if detail.ID == "" {
		return RunDetail{}, apperror.New(http.StatusNotFound, 404002, "workflow run not found")
	}
	return detail, nil
}

func (s *Service) CancelRun(ctx context.Context, input CancelInput) (RunDetail, *apperror.Error) {
	var canceled RunDetail
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		run, err := runByUID(ctx, tx, workspace.ID, strings.TrimSpace(input.ID))
		if err != nil {
			return err
		}
		if run.ID == 0 {
			return apperror.New(http.StatusNotFound, 404002, "workflow run not found")
		}
		if !cancelableStatus(run.Status) {
			return apperror.New(http.StatusConflict, 409006, "workflow run cannot be canceled")
		}
		actorID, err := audit.UserIDByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		reason := limitString(strings.TrimSpace(input.Reason), 512)
		var taskRunIDs []uint64
		if err := tx.WithContext(ctx).Raw(
			`SELECT DISTINCT task_run_id
			   FROM workflow_run_nodes
			  WHERE run_id = ? AND task_run_id IS NOT NULL`,
			run.ID,
		).Scan(&taskRunIDs).Error; err != nil {
			return err
		}
		for _, taskRunID := range taskRunIDs {
			if err := tasks.CancelRunByIDTx(ctx, tx, taskRunID, actorID, reason); err != nil {
				return err
			}
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE workflow_runs
			    SET status = 'canceled', cancel_requested_by = ?, cancel_reason = ?, finished_at = COALESCE(finished_at, NOW(3))
			  WHERE id = ?`,
			actorID, nullString(reason), run.ID,
		).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE workflow_run_nodes
			    SET status = 'canceled', finished_at = COALESCE(finished_at, NOW(3))
			  WHERE run_id = ? AND status IN ('pending', 'queued', 'running')`,
			run.ID,
		).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO workflow_run_events(run_id, event_type, message, actor_id, payload)
			 VALUES (?, 'canceled', 'Workflow run canceled', ?, JSON_OBJECT('reason', ?, 'taskRuns', ?))`,
			run.ID, actorID, reason, len(taskRunIDs),
		).Error; err != nil {
			return err
		}
		loaded, err := runDetailByUID(ctx, tx, workspace.ID, input.ID)
		if err != nil {
			return err
		}
		canceled = loaded
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "workflow.cancel",
			ResourceType:  "workflow_run",
			ResourceID:    sql.NullInt64{Int64: int64(run.ID), Valid: true},
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			After:         canceled.RunSummary,
		})
		return nil
	})
	if txErr != nil {
		return RunDetail{}, wrapAppError(txErr, 501010, "cancel workflow run failed")
	}
	return canceled, nil
}
