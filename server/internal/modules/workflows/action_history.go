package workflows

import (
	"context"
	"database/sql"
	"sort"
	"strings"

	"gorm.io/gorm"

	"opspilot/server/internal/shared/uid"
)

func recordWorkflowAction(ctx context.Context, tx *gorm.DB, runID uint64, traceID, actionType, nodeID, detail, result string, actorID sql.NullInt64, payload interface{}) error {
	if tx == nil || runID == 0 {
		return nil
	}
	actionType = strings.TrimSpace(actionType)
	if actionType == "" {
		return nil
	}
	actionUID, err := uid.New()
	if err != nil {
		return err
	}
	return tx.WithContext(ctx).Exec(
		`INSERT INTO workflow_run_actions(uid, workflow_run_id, trace_id, action_type, actor_id, node_id, detail, result, payload)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		actionUID,
		runID,
		nullString(traceID),
		actionType,
		actorID,
		nullString(nodeID),
		nullString(limitString(detail, 1024)),
		actionResult(result),
		jsonNull(payload),
	).Error
}

func loadWorkflowActionHistory(ctx context.Context, db *gorm.DB, runID uint64) ([]ActionHistoryItem, error) {
	var items []ActionHistoryItem
	err := db.WithContext(ctx).Raw(
		`SELECT wra.id,
		        wra.action_type AS action,
		        COALESCE(u.username, '') AS actor,
		        COALESCE(wra.detail, '') AS detail,
		        COALESCE(wra.result, '') AS result,
		        COALESCE(wra.trace_id, '') AS trace_id,
		        DATE_FORMAT(wra.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM workflow_run_actions wra
		   LEFT JOIN users u ON u.id = wra.actor_id
		  WHERE wra.workflow_run_id = ?
		  ORDER BY wra.created_at DESC, wra.id DESC
		  LIMIT 200`,
		runID,
	).Scan(&items).Error
	return items, err
}

func mergeLegacyActionHistory(audits []ActionHistoryItem, events []ActionHistoryItem) []ActionHistoryItem {
	items := make([]ActionHistoryItem, 0, len(audits)+len(events))
	items = append(items, audits...)
	items = append(items, events...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt == items[j].CreatedAt {
			return items[i].ID > items[j].ID
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
	return items
}

func actionResult(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "success"
	}
	return value
}

func legacyActionName(value string) string {
	switch strings.TrimSpace(value) {
	case "workflow.retry", "retry_requested":
		return "retry_requested"
	case "workflow.cancel", "canceled", "cancel_completed":
		return "cancel_completed"
	case "workflow.node_retry", "node_retry", "node_retry_requested":
		return "node_retry_requested"
	case "workflow.node_approve", "node_success", "approval_approved":
		return "approval_approved"
	case "workflow.node_reject", "node_failed", "approval_rejected":
		return "approval_rejected"
	case "retry_started":
		return "retry_started"
	default:
		return strings.TrimSpace(value)
	}
}
