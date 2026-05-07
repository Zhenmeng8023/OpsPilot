package scripts

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"opspilot/server/internal/config"
	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/audit"
	"opspilot/server/internal/shared/uid"
)

type Service struct {
	db   *gorm.DB
	cfg  config.Config
	repo repository
}

type AuditContext struct {
	ActorUID      string
	IP            string
	UserAgent     string
	TraceID       string
	RequestMethod string
	RequestPath   string
}

type CreateInput struct {
	Name          string
	Description   string
	ScriptType    string
	Content       string
	ChangeSummary string
	Audit         AuditContext
}

type UpdateInput struct {
	ID            string
	Name          string
	Description   string
	ScriptType    string
	Content       string
	ChangeSummary string
	Audit         AuditContext
}

type ListInput struct {
	Keyword string
	Status  string
}

type ScriptSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	ScriptType  string `json:"scriptType"`
	Status      string `json:"status"`
	Version     uint   `json:"version"`
	CreatedBy   string `json:"createdBy,omitempty"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type ScriptDetail struct {
	ScriptSummary
	Content       string `json:"content"`
	ChangeSummary string `json:"changeSummary,omitempty"`
}

type ApprovalSummary struct {
	ID         uint64 `json:"id"`
	VersionID  string `json:"versionId"`
	ScriptID   string `json:"scriptId"`
	ScriptName string `json:"scriptName"`
	Version    uint   `json:"version"`
	Status     string `json:"status"`
	Comment    string `json:"comment,omitempty"`
	Approver   string `json:"approver,omitempty"`
	ApprovedAt string `json:"approvedAt,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

func NewService(db *gorm.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg, repo: newRepository(db)}
}

func (s *Service) List(ctx context.Context, input ListInput) ([]ScriptSummary, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	rows, err := s.repo.listScripts(ctx, workspace.ID, strings.TrimSpace(input.Keyword), strings.TrimSpace(input.Status))
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500201, "list scripts failed", err)
	}
	out := make([]ScriptSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, summaryFromRecord(row))
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, scriptUID string) (ScriptDetail, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return ScriptDetail{}, appErr
	}
	row, err := s.repo.scriptByUID(ctx, workspace.ID, strings.TrimSpace(scriptUID))
	if err != nil {
		return ScriptDetail{}, apperror.Wrap(http.StatusInternalServerError, 500202, "load script failed", err)
	}
	if row.ID == 0 {
		return ScriptDetail{}, apperror.New(http.StatusNotFound, 404201, "script not found")
	}
	return detailFromRecord(row), nil
}

func (s *Service) Create(ctx context.Context, input CreateInput) (ScriptDetail, *apperror.Error) {
	name := strings.TrimSpace(input.Name)
	content := strings.TrimSpace(input.Content)
	scriptType := normalizeScriptType(input.ScriptType)
	if name == "" {
		return ScriptDetail{}, apperror.New(http.StatusBadRequest, 400201, "script name is required")
	}
	if content == "" {
		return ScriptDetail{}, apperror.New(http.StatusBadRequest, 400202, "script content cannot be empty")
	}
	if !validScriptType(scriptType) {
		return ScriptDetail{}, apperror.New(http.StatusBadRequest, 400203, "unsupported script type")
	}

	var created ScriptDetail
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := newRepository(tx)
		workspace, err := repo.defaultWorkspace(ctx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		if workspace.ID == 0 {
			return apperror.New(http.StatusInternalServerError, 500203, "default workspace is not initialized")
		}
		actorID, err := repo.userIDByUID(ctx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		templateUID, err := uid.New()
		if err != nil {
			return err
		}
		versionUID, err := uid.New()
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO script_templates(uid, workspace_id, name, description, script_type, risk_level, status, created_by)
			 VALUES (?, ?, ?, ?, ?, 'medium', 'active', ?)`,
			templateUID, workspace.ID, name, nullString(input.Description), scriptType, actorID,
		).Error; err != nil {
			return err
		}
		var templateID uint64
		if err := tx.WithContext(ctx).Raw("SELECT id FROM script_templates WHERE uid = ? LIMIT 1", templateUID).Scan(&templateID).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO script_versions(uid, template_id, version_no, title, content, checksum, status, change_summary, created_by)
			 VALUES (?, ?, 1, ?, ?, ?, 'active', ?, ?)`,
			versionUID, templateID, name, content, checksum(content), nullString(input.ChangeSummary), actorID,
		).Error; err != nil {
			return err
		}
		row, err := repo.scriptByUID(ctx, workspace.ID, templateUID)
		if err != nil {
			return err
		}
		created = detailFromRecord(row)
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "script.create",
			ResourceType:  "script_template",
			ResourceID:    sql.NullInt64{Int64: int64(templateID), Valid: true},
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
			return ScriptDetail{}, appErr
		}
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return ScriptDetail{}, apperror.New(http.StatusConflict, 409201, "script name already exists")
		}
		return ScriptDetail{}, apperror.Wrap(http.StatusInternalServerError, 500204, "create script failed", txErr)
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, input UpdateInput) (ScriptDetail, *apperror.Error) {
	name := strings.TrimSpace(input.Name)
	content := strings.TrimSpace(input.Content)
	scriptType := normalizeScriptType(input.ScriptType)
	if strings.TrimSpace(input.ID) == "" {
		return ScriptDetail{}, apperror.New(http.StatusBadRequest, 400001, "script id is required")
	}
	if name == "" {
		return ScriptDetail{}, apperror.New(http.StatusBadRequest, 400201, "script name is required")
	}
	if content == "" {
		return ScriptDetail{}, apperror.New(http.StatusBadRequest, 400202, "script content cannot be empty")
	}
	if !validScriptType(scriptType) {
		return ScriptDetail{}, apperror.New(http.StatusBadRequest, 400203, "unsupported script type")
	}

	var updated ScriptDetail
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := newRepository(tx)
		workspace, err := repo.defaultWorkspace(ctx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		current, err := repo.scriptByUID(ctx, workspace.ID, input.ID)
		if err != nil {
			return err
		}
		if current.ID == 0 {
			return apperror.New(http.StatusNotFound, 404201, "script not found")
		}
		if current.Status == "disabled" {
			return apperror.New(http.StatusBadRequest, 400204, "disabled script cannot be edited")
		}
		actorID, err := repo.userIDByUID(ctx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE script_templates
			    SET name = ?, description = ?, script_type = ?, status = 'active'
			  WHERE id = ?`,
			name, nullString(input.Description), scriptType, current.ID,
		).Error; err != nil {
			return err
		}
		latest, err := repo.latestVersion(ctx, current.ID)
		if err != nil {
			return err
		}
		if latest.Checksum != checksum(content) {
			versionUID, err := uid.New()
			if err != nil {
				return err
			}
			if err := tx.WithContext(ctx).Exec(
				"UPDATE script_versions SET status = 'deprecated' WHERE template_id = ? AND status = 'active'",
				current.ID,
			).Error; err != nil {
				return err
			}
			if err := tx.WithContext(ctx).Exec(
				`INSERT INTO script_versions(uid, template_id, version_no, title, content, checksum, status, change_summary, created_by)
				 VALUES (?, ?, ?, ?, ?, ?, 'active', ?, ?)`,
				versionUID, current.ID, latest.VersionNo+1, name, content, checksum(content), nullString(input.ChangeSummary), actorID,
			).Error; err != nil {
				return err
			}
		}
		row, err := repo.scriptByUID(ctx, workspace.ID, input.ID)
		if err != nil {
			return err
		}
		updated = detailFromRecord(row)
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "script.update",
			ResourceType:  "script_template",
			ResourceID:    sql.NullInt64{Int64: int64(current.ID), Valid: true},
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			Before:        detailFromRecord(current),
			After:         updated,
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return ScriptDetail{}, appErr
		}
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return ScriptDetail{}, apperror.New(http.StatusConflict, 409201, "script name already exists")
		}
		return ScriptDetail{}, apperror.Wrap(http.StatusInternalServerError, 500205, "update script failed", txErr)
	}
	return updated, nil
}

func (s *Service) Disable(ctx context.Context, scriptUID string, auditCtx AuditContext) *apperror.Error {
	if strings.TrimSpace(scriptUID) == "" {
		return apperror.New(http.StatusBadRequest, 400001, "script id is required")
	}
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := newRepository(tx)
		workspace, err := repo.defaultWorkspace(ctx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		row, err := repo.scriptByUID(ctx, workspace.ID, scriptUID)
		if err != nil {
			return err
		}
		if row.ID == 0 {
			return apperror.New(http.StatusNotFound, 404201, "script not found")
		}
		actorID, err := repo.userIDByUID(ctx, auditCtx.ActorUID)
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec("UPDATE script_templates SET status = 'disabled' WHERE id = ?", row.ID).Error; err != nil {
			return err
		}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "script.disable",
			ResourceType:  "script_template",
			ResourceID:    sql.NullInt64{Int64: int64(row.ID), Valid: true},
			IP:            auditCtx.IP,
			UserAgent:     auditCtx.UserAgent,
			TraceID:       auditCtx.TraceID,
			RequestMethod: auditCtx.RequestMethod,
			RequestPath:   auditCtx.RequestPath,
			Before:        detailFromRecord(row),
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return appErr
		}
		return apperror.Wrap(http.StatusInternalServerError, 500206, "disable script failed", txErr)
	}
	return nil
}

func (s *Service) ListApprovals(ctx context.Context, status string) ([]ApprovalSummary, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	rows, err := s.repo.listApprovals(ctx, workspace.ID, strings.TrimSpace(status))
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500207, "list script approvals failed", err)
	}
	out := make([]ApprovalSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, approvalFromRecord(row))
	}
	return out, nil
}

func (s *Service) RequestApproval(ctx context.Context, scriptUID string, auditCtx AuditContext) (ApprovalSummary, *apperror.Error) {
	var created ApprovalSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := newRepository(tx)
		workspace, err := repo.defaultWorkspace(ctx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		script, err := repo.scriptByUID(ctx, workspace.ID, scriptUID)
		if err != nil {
			return err
		}
		if script.ID == 0 || script.LatestVersionID == 0 {
			return apperror.New(http.StatusNotFound, 404201, "script not found")
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO script_approvals(script_version_id, status)
			 SELECT ?, 'pending'
			 WHERE NOT EXISTS (
			   SELECT 1 FROM script_approvals WHERE script_version_id = ? AND status = 'pending'
			 )`,
			script.LatestVersionID, script.LatestVersionID,
		).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec("UPDATE script_templates SET approval_required = 1 WHERE id = ?", script.ID).Error; err != nil {
			return err
		}
		rows, err := repo.listApprovals(ctx, workspace.ID, "pending")
		if err != nil {
			return err
		}
		for _, row := range rows {
			if row.VersionUID == script.LatestVersionUID.String {
				created = approvalFromRecord(row)
				break
			}
		}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			Action:        "script.approval.request",
			ResourceType:  "script_template",
			ResourceID:    sql.NullInt64{Int64: int64(script.ID), Valid: true},
			TraceID:       auditCtx.TraceID,
			RequestMethod: auditCtx.RequestMethod,
			RequestPath:   auditCtx.RequestPath,
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return ApprovalSummary{}, appErr
		}
		return ApprovalSummary{}, apperror.Wrap(http.StatusInternalServerError, 500208, "request script approval failed", txErr)
	}
	return created, nil
}

func (s *Service) DecideApproval(ctx context.Context, approvalID uint64, approve bool, comment string, auditCtx AuditContext) (ApprovalSummary, *apperror.Error) {
	if approvalID == 0 {
		return ApprovalSummary{}, apperror.New(http.StatusBadRequest, 400205, "approval id is required")
	}
	nextStatus := "rejected"
	if approve {
		nextStatus = "approved"
	}
	var updated ApprovalSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := newRepository(tx)
		workspace, err := repo.defaultWorkspace(ctx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		actorID, err := repo.userIDByUID(ctx, auditCtx.ActorUID)
		if err != nil {
			return err
		}
		var row struct {
			ID        uint64
			VersionID uint64
			Status    string
		}
		if err := tx.WithContext(ctx).Raw("SELECT id, script_version_id AS version_id, status FROM script_approvals WHERE id = ? FOR UPDATE", approvalID).Scan(&row).Error; err != nil {
			return err
		}
		if row.ID == 0 {
			return apperror.New(http.StatusNotFound, 404202, "approval not found")
		}
		if row.Status != "pending" {
			return apperror.New(http.StatusConflict, 409202, "approval is already decided")
		}
		if err := tx.WithContext(ctx).Exec(
			"UPDATE script_approvals SET status = ?, comment = ?, approver_id = ?, approved_at = NOW(3) WHERE id = ?",
			nextStatus, nullString(comment), actorID, approvalID,
		).Error; err != nil {
			return err
		}
		if approve {
			if err := tx.WithContext(ctx).Exec("UPDATE script_versions SET status = 'active' WHERE id = ?", row.VersionID).Error; err != nil {
				return err
			}
		}
		rows, err := repo.listApprovals(ctx, workspace.ID, nextStatus)
		if err != nil {
			return err
		}
		for _, approval := range rows {
			if approval.ID == approvalID {
				updated = approvalFromRecord(approval)
				break
			}
		}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "script.approval." + nextStatus,
			ResourceType:  "script_approval",
			ResourceID:    sql.NullInt64{Int64: int64(approvalID), Valid: true},
			After:         updated,
			TraceID:       auditCtx.TraceID,
			RequestMethod: auditCtx.RequestMethod,
			RequestPath:   auditCtx.RequestPath,
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return ApprovalSummary{}, appErr
		}
		return ApprovalSummary{}, apperror.Wrap(http.StatusInternalServerError, 500209, "decide script approval failed", txErr)
	}
	return updated, nil
}

func (s *Service) workspace(ctx context.Context) (workspaceRecord, *apperror.Error) {
	workspace, err := s.repo.defaultWorkspace(ctx, s.cfg.Bootstrap.WorkspaceSlug)
	if err != nil {
		return workspaceRecord{}, apperror.Wrap(http.StatusInternalServerError, 500203, "load workspace failed", err)
	}
	if workspace.ID == 0 {
		return workspaceRecord{}, apperror.New(http.StatusInternalServerError, 500203, "default workspace is not initialized")
	}
	return workspace, nil
}

func summaryFromRecord(row scriptRecord) ScriptSummary {
	return ScriptSummary{
		ID:          row.UID,
		Name:        row.Name,
		Description: row.Description.String,
		ScriptType:  row.ScriptType,
		Status:      row.Status,
		Version:     row.LatestVersion,
		CreatedBy:   row.CreatedBy.String,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

func detailFromRecord(row scriptRecord) ScriptDetail {
	return ScriptDetail{
		ScriptSummary: summaryFromRecord(row),
		Content:       row.Content.String,
		ChangeSummary: row.ChangeSummary.String,
	}
}

func approvalFromRecord(row approvalRecord) ApprovalSummary {
	return ApprovalSummary{
		ID:         row.ID,
		VersionID:  row.VersionUID,
		ScriptID:   row.ScriptUID,
		ScriptName: row.ScriptName,
		Version:    row.VersionNo,
		Status:     row.Status,
		Comment:    row.Comment.String,
		Approver:   row.Approver.String,
		ApprovedAt: row.ApprovedAt.String,
		CreatedAt:  row.CreatedAt,
	}
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

func nullString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	return sql.NullString{String: value, Valid: value != ""}
}
