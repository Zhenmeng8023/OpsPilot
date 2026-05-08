package workflows

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"opspilot/server/internal/shared/apperror"
)

func (s *Service) workspace(ctx context.Context) (workspaceRecord, *apperror.Error) {
	workspace, err := defaultWorkspace(ctx, s.db, s.cfg.Bootstrap.WorkspaceSlug)
	if err != nil {
		if appErr, ok := err.(*apperror.Error); ok {
			return workspaceRecord{}, appErr
		}
		return workspaceRecord{}, apperror.Wrap(http.StatusInternalServerError, 501011, "load workspace failed", err)
	}
	return workspace, nil
}

func defaultWorkspace(ctx context.Context, db *gorm.DB, slug string) (workspaceRecord, error) {
	var workspace workspaceRecord
	err := db.WithContext(ctx).Raw("SELECT id FROM workspaces WHERE slug = ? AND status = 'active' LIMIT 1", slug).Scan(&workspace).Error
	if err != nil {
		return workspaceRecord{}, err
	}
	if workspace.ID == 0 {
		return workspaceRecord{}, apperror.New(http.StatusInternalServerError, 501012, "default workspace is not initialized")
	}
	return workspace, nil
}

func (s *Service) workflowByUID(ctx context.Context, id string) (workflowRecord, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return workflowRecord{}, appErr
	}
	row, err := workflowRowByUID(ctx, s.db, workspace.ID, strings.TrimSpace(id))
	if err != nil {
		return workflowRecord{}, apperror.Wrap(http.StatusInternalServerError, 501013, "get workflow failed", err)
	}
	if row.ID == 0 {
		return workflowRecord{}, apperror.New(http.StatusNotFound, 404001, "workflow not found")
	}
	return row, nil
}

func workflowRowByUID(ctx context.Context, db *gorm.DB, workspaceID uint64, uid string) (workflowRecord, error) {
	rows, err := workflowRows(ctx, db, "WHERE wd.workspace_id = ? AND wd.uid = ? AND wd.deleted_at IS NULL LIMIT 1", workspaceID, uid)
	if err != nil || len(rows) == 0 {
		return workflowRecord{}, err
	}
	return rows[0], nil
}

func workflowRowByID(ctx context.Context, db *gorm.DB, workspaceID, id uint64) (workflowRecord, error) {
	rows, err := workflowRows(ctx, db, "WHERE wd.workspace_id = ? AND wd.id = ? AND wd.deleted_at IS NULL LIMIT 1", workspaceID, id)
	if err != nil || len(rows) == 0 {
		return workflowRecord{}, err
	}
	return rows[0], nil
}

func (s *Service) workflowRows(ctx context.Context, where string, args ...interface{}) ([]workflowRecord, error) {
	return workflowRows(ctx, s.db, where, args...)
}

func workflowRows(ctx context.Context, db *gorm.DB, where string, args ...interface{}) ([]workflowRecord, error) {
	var rows []workflowRecord
	err := db.WithContext(ctx).Raw(
		`SELECT wd.id, wd.uid, wd.workspace_id, wd.name, wd.description, CAST(wd.definition AS CHAR) AS definition,
		        wd.version, wd.status, wd.created_by AS created_by_id, u.username AS created_by,
		        DATE_FORMAT(wd.published_at, '%Y-%m-%d %H:%i:%s') AS published_at,
		        DATE_FORMAT(wd.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
		        DATE_FORMAT(wd.updated_at, '%Y-%m-%d %H:%i:%s') AS updated_at
		   FROM workflow_definitions wd
		   LEFT JOIN users u ON u.id = wd.created_by
		  `+where,
		args...,
	).Scan(&rows).Error
	return rows, err
}

func workflowListWhere(workspaceID uint64, keyword, status string) (string, []interface{}) {
	args := []interface{}{workspaceID}
	where := "WHERE wd.workspace_id = ? AND wd.deleted_at IS NULL"
	if strings.TrimSpace(keyword) != "" {
		where += " AND (wd.name LIKE ? OR wd.description LIKE ?)"
		like := "%" + strings.TrimSpace(keyword) + "%"
		args = append(args, like, like)
	}
	if strings.TrimSpace(status) != "" {
		where += " AND wd.status = ?"
		args = append(args, strings.TrimSpace(status))
	}
	return where, args
}

func runListWhere(workspaceID uint64, keyword, status string) (string, []interface{}) {
	args := []interface{}{workspaceID}
	where := "WHERE wr.workspace_id = ?"
	if strings.TrimSpace(keyword) != "" {
		where += " AND (wr.uid LIKE ? OR wd.name LIKE ?)"
		like := "%" + strings.TrimSpace(keyword) + "%"
		args = append(args, like, like)
	}
	if strings.TrimSpace(status) != "" {
		where += " AND wr.status = ?"
		args = append(args, strings.TrimSpace(status))
	}
	return where, args
}

func workflowVersionRows(ctx context.Context, db *gorm.DB, workflowID uint64) ([]versionRecord, error) {
	var rows []versionRecord
	err := db.WithContext(ctx).Raw(
		`SELECT wv.id, wv.uid, wd.uid AS workflow_uid, wv.version_no, wv.status, wv.definition_hash,
		        u.username AS created_by,
		        DATE_FORMAT(wv.published_at, '%Y-%m-%d %H:%i:%s') AS published_at,
		        DATE_FORMAT(wv.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM workflow_versions wv
		   JOIN workflow_definitions wd ON wd.id = wv.workflow_id
		   LEFT JOIN users u ON u.id = wv.created_by
		  WHERE wv.workflow_id = ?
		  ORDER BY wv.version_no DESC`,
		workflowID,
	).Scan(&rows).Error
	return rows, err
}

func nextWorkflowCopyName(ctx context.Context, db *gorm.DB, workspaceID uint64, sourceName string) (string, error) {
	base := strings.TrimSpace(sourceName)
	if base == "" {
		base = "Workflow"
	}
	candidate := base + " Copy"
	for index := 0; index < 100; index++ {
		name := candidate
		if index > 0 {
			name = candidate + " " + fmt.Sprint(index+1)
		}
		var count int64
		if err := db.WithContext(ctx).Raw(
			"SELECT COUNT(*) FROM workflow_definitions WHERE workspace_id = ? AND name = ? AND deleted_at IS NULL",
			workspaceID, name,
		).Scan(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return name, nil
		}
	}
	return "", apperror.New(http.StatusConflict, 409012, "workflow copy name is exhausted")
}

func runRows(ctx context.Context, db *gorm.DB, where string, args ...interface{}) ([]runRecord, error) {
	var rows []runRecord
	err := db.WithContext(ctx).Raw(
		`SELECT wr.id, wr.workflow_id AS workflow_db_id, wr.uid, wd.uid AS workflow_uid, wd.name AS workflow_name, wr.workflow_version,
		        CAST(wr.definition_snapshot AS CHAR) AS definition_snapshot, wr.status, wr.trigger_type,
		        CAST(wr.input AS CHAR) AS input, CAST(wr.output AS CHAR) AS output,
		        wr.total_nodes, wr.success_nodes, wr.failed_nodes, wr.skipped_nodes, wr.error_message,
		        u.username AS created_by,
		        DATE_FORMAT(wr.queued_at, '%Y-%m-%d %H:%i:%s') AS queued_at,
		        DATE_FORMAT(wr.started_at, '%Y-%m-%d %H:%i:%s') AS started_at,
		        DATE_FORMAT(wr.finished_at, '%Y-%m-%d %H:%i:%s') AS finished_at,
		        DATE_FORMAT(wr.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM workflow_runs wr
		   JOIN workflow_definitions wd ON wd.id = wr.workflow_id
		   LEFT JOIN users u ON u.id = wr.created_by
		  `+where,
		args...,
	).Scan(&rows).Error
	return rows, err
}

func runByUID(ctx context.Context, db *gorm.DB, workspaceID uint64, uid string) (runRecord, error) {
	rows, err := runRows(ctx, db, "WHERE wr.workspace_id = ? AND wr.uid = ? LIMIT 1", workspaceID, uid)
	if err != nil || len(rows) == 0 {
		return runRecord{}, err
	}
	return rows[0], nil
}

func runIDByUID(ctx context.Context, db *gorm.DB, workspaceID uint64, uid string) (uint64, error) {
	var id uint64
	err := db.WithContext(ctx).Raw("SELECT id FROM workflow_runs WHERE workspace_id = ? AND uid = ? LIMIT 1", workspaceID, uid).Scan(&id).Error
	return id, err
}

func taskIDByUID(ctx context.Context, db *gorm.DB, workspaceID uint64, uid string) (uint64, error) {
	var id uint64
	err := db.WithContext(ctx).Raw(
		"SELECT id FROM tasks WHERE workspace_id = ? AND uid = ? AND status = 'active' AND deleted_at IS NULL LIMIT 1",
		workspaceID,
		strings.TrimSpace(uid),
	).Scan(&id).Error
	return id, err
}

func runDetailByUID(ctx context.Context, db *gorm.DB, workspaceID uint64, uid string) (RunDetail, error) {
	row, err := runByUID(ctx, db, workspaceID, uid)
	if err != nil || row.ID == 0 {
		return RunDetail{}, err
	}
	nodes, err := runNodes(ctx, db, row.ID)
	if err != nil {
		return RunDetail{}, err
	}
	events, err := runEvents(ctx, db, row.ID)
	if err != nil {
		return RunDetail{}, err
	}
	return RunDetail{
		RunSummary: runSummary(row),
		Input:      row.Input.String,
		Output:     row.Output.String,
		Definition: row.DefinitionSnapshot,
		Nodes:      nodes,
		Events:     events,
	}, nil
}

func runNodes(ctx context.Context, db *gorm.DB, runID uint64) ([]RunNodeSummary, error) {
	var rows []nodeRecord
	err := db.WithContext(ctx).Raw(
		`SELECT wrn.id, wrn.node_id, wrn.node_type, wrn.node_name, wrn.status, tr.uid AS task_run_uid,
		        CAST(wrn.input AS CHAR) AS input, CAST(wrn.output AS CHAR) AS output,
		        wrn.error_message, wrn.attempts,
		        DATE_FORMAT(wrn.queued_at, '%Y-%m-%d %H:%i:%s') AS queued_at,
		        DATE_FORMAT(wrn.started_at, '%Y-%m-%d %H:%i:%s') AS started_at,
		        DATE_FORMAT(wrn.finished_at, '%Y-%m-%d %H:%i:%s') AS finished_at
		   FROM workflow_run_nodes wrn
		   LEFT JOIN task_runs tr ON tr.id = wrn.task_run_id
		  WHERE wrn.run_id = ?
		  ORDER BY wrn.id ASC`,
		runID,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]RunNodeSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, RunNodeSummary{
			ID:           row.ID,
			NodeID:       row.NodeID,
			NodeType:     row.NodeType,
			NodeName:     row.NodeName.String,
			Status:       row.Status,
			TaskRunID:    row.TaskRunUID.String,
			Input:        row.Input.String,
			Output:       row.Output.String,
			ErrorMessage: row.ErrorMessage.String,
			Attempts:     row.Attempts,
			QueuedAt:     row.QueuedAt.String,
			StartedAt:    row.StartedAt.String,
			FinishedAt:   row.FinishedAt.String,
		})
	}
	return out, nil
}

func runEvents(ctx context.Context, db *gorm.DB, runID uint64) ([]RunEventSummary, error) {
	var rows []eventRecord
	err := db.WithContext(ctx).Raw(
		`SELECT wre.id, wre.node_id, wre.event_type, wre.message, u.username AS actor, CAST(wre.payload AS CHAR) AS payload,
		        DATE_FORMAT(wre.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM workflow_run_events wre
		   LEFT JOIN users u ON u.id = wre.actor_id
		  WHERE wre.run_id = ?
		  ORDER BY wre.created_at ASC, wre.id ASC`,
		runID,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]RunEventSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, RunEventSummary{
			ID:        row.ID,
			NodeID:    row.NodeID.String,
			EventType: row.EventType,
			Message:   row.Message.String,
			Actor:     row.Actor.String,
			Payload:   row.Payload.String,
			CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

func nullString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	return sql.NullString{String: value, Valid: value != ""}
}
