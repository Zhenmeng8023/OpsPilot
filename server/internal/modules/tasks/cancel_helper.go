package tasks

import (
	"context"
	"database/sql"
	"strings"

	"gorm.io/gorm"
)

type cancelRunRecord struct {
	ID     uint64
	Status string
}

func CancelRunByIDTx(ctx context.Context, tx *gorm.DB, runID uint64, actorID sql.NullInt64, reason string) error {
	var run cancelRunRecord
	if err := tx.WithContext(ctx).Raw("SELECT id, status FROM task_runs WHERE id = ? LIMIT 1", runID).Scan(&run).Error; err != nil {
		return err
	}
	if run.ID == 0 || IsTerminal(Status(run.Status)) {
		return nil
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		reason = "workflow canceled"
	}
	if err := tx.WithContext(ctx).Exec(
		`UPDATE task_run_targets
		    SET status = 'canceled', finished_at = NOW(3)
		  WHERE run_id = ? AND status IN ('pending', 'queued')`,
		run.ID,
	).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Exec(
		`UPDATE task_run_targets
		    SET status = 'canceling'
		  WHERE run_id = ? AND status IN ('running', 'canceling')`,
		run.ID,
	).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Exec(
		`UPDATE task_run_attempts ta
		    JOIN task_run_targets rt ON rt.id = ta.run_target_id
		    SET ta.status = 'canceling'
		  WHERE rt.run_id = ? AND ta.status = 'running'`,
		run.ID,
	).Error; err != nil {
		return err
	}
	if err := tx.WithContext(ctx).Exec(
		`UPDATE task_runs
		    SET cancel_requested_by = ?, cancel_reason = ?
		  WHERE id = ?`,
		actorID, reason, run.ID,
	).Error; err != nil {
		return err
	}
	if err := refreshRunAggregate(ctx, tx, run.ID); err != nil {
		return err
	}
	return writeRunEvent(ctx, tx, run.ID, 0, 0, run.Status, string(StatusCanceled), "task cancellation requested", map[string]string{"reason": reason})
}
