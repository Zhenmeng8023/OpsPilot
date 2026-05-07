package execution

import (
	"context"
	"database/sql"
	"errors"

	"gorm.io/gorm"

	"opspilot/server/internal/shared/uid"
)

func CreateRunFromTask(ctx context.Context, tx *gorm.DB, workspaceID, taskID uint64, triggerType string, triggerID, actorID sql.NullInt64) (uint64, string, error) {
	var task struct {
		ID             uint64
		ScriptVersionID uint64
		TimeoutSeconds uint
	}
	if err := tx.WithContext(ctx).Raw(
		`SELECT id, script_version_id, timeout_seconds
		   FROM tasks
		  WHERE workspace_id = ? AND id = ? AND status = 'active' AND deleted_at IS NULL
		  LIMIT 1`,
		workspaceID, taskID,
	).Scan(&task).Error; err != nil {
		return 0, "", err
	}
	if task.ID == 0 {
		return 0, "", errors.New("task definition not found")
	}
	var targetCount int
	if err := tx.WithContext(ctx).Raw("SELECT COUNT(*) FROM task_targets WHERE task_id = ?", taskID).Scan(&targetCount).Error; err != nil {
		return 0, "", err
	}
	if targetCount == 0 {
		return 0, "", errors.New("task definition has no targets")
	}
	runUID, err := uid.New()
	if err != nil {
		return 0, "", err
	}
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO task_runs(uid, workspace_id, task_id, script_version_id, status, trigger_type, trigger_id,
		                       timeout_seconds, total_targets, created_by, queued_at)
		 VALUES (?, ?, ?, ?, 'queued', ?, ?, ?, ?, ?, NOW(3))`,
		runUID, workspaceID, taskID, task.ScriptVersionID, triggerType, triggerID, task.TimeoutSeconds, targetCount, actorID,
	).Error; err != nil {
		return 0, "", err
	}
	var runID uint64
	if err := tx.WithContext(ctx).Raw("SELECT id FROM task_runs WHERE uid = ? LIMIT 1", runUID).Scan(&runID).Error; err != nil {
		return 0, "", err
	}
	var targets []struct {
		ID      uint64
		AgentID sql.NullInt64
		HostID  sql.NullInt64
		Selector sql.NullString
	}
	if err := tx.WithContext(ctx).Raw("SELECT id, agent_id, host_id, selector FROM task_targets WHERE task_id = ?", taskID).Scan(&targets).Error; err != nil {
		return 0, "", err
	}
	for _, target := range targets {
		targetUID, err := uid.New()
		if err != nil {
			return 0, "", err
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO task_run_targets(uid, run_id, task_target_id, agent_id, host_id, target_snapshot, status)
			 VALUES (?, ?, ?, ?, ?, ?, 'queued')`,
			targetUID, runID, target.ID, target.AgentID, target.HostID, target.Selector,
		).Error; err != nil {
			return 0, "", err
		}
	}
	_ = tx.WithContext(ctx).Exec(
		`INSERT INTO task_run_events(run_id, event_type, from_status, to_status, message)
		 VALUES (?, 'status_transition', 'pending', 'queued', 'triggered run queued')`,
		runID,
	).Error
	return runID, runUID, nil
}
