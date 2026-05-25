package tracecenter

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"gorm.io/gorm"

	"opspilot/server/internal/shared/uid"
)

const (
	syntheticWorkflowTracePrefix = "workflow-run:"
	syntheticTaskTracePrefix     = "task-run:"
	syntheticWebhookTracePrefix  = "webhook-event:"
)

func (s *Service) resolveTraceID(ctx context.Context, workspaceID uint64, traceID string, workflow workflowRunRecord, task taskRunRecord, webhook webhookEventRecord) (string, error) {
	traceID = strings.TrimSpace(traceID)
	if traceID != "" {
		return traceID, nil
	}

	args := []interface{}{workspaceID}
	refs := make([]string, 0, 3)
	if workflow.ID != 0 {
		refs = append(refs, "(resource_type = 'workflow_run' AND resource_id = ?)")
		args = append(args, workflow.ID)
	}
	if task.ID != 0 {
		refs = append(refs, "(resource_type = 'task_run' AND resource_id = ?)")
		args = append(args, task.ID)
	}
	if webhook.ID != 0 {
		refs = append(refs, "(resource_type = 'webhook_event' AND resource_id = ?)")
		args = append(args, webhook.ID)
	}
	clause := strings.Join(refs, " OR ")
	if clause != "" {
		var row struct {
			TraceID sql.NullString `gorm:"column:trace_id"`
		}
		if err := s.db.WithContext(ctx).Raw(
			`SELECT trace_id
			   FROM audit_logs
			  WHERE workspace_id = ?
			    AND trace_id IS NOT NULL
			    AND trace_id <> ''
			    AND (`+clause+`)
			  ORDER BY created_at DESC, id DESC
			  LIMIT 1`,
			args...,
		).Scan(&row).Error; err != nil {
			return "", err
		}
		if value := strings.TrimSpace(row.TraceID.String); value != "" {
			return value, nil
		}
		cacheArgs, cacheClause := traceReferenceClause(workspaceID, workflow.ID, task.ID, webhook.ID)
		if err := s.db.WithContext(ctx).Raw(
			`SELECT trace_id
			   FROM trace_events
			  WHERE workspace_id = ?
			    AND trace_id <> ''
			    AND (`+cacheClause+`)
			  ORDER BY occurred_at DESC, id DESC
			  LIMIT 1`,
			cacheArgs...,
		).Scan(&row).Error; err != nil {
			return "", err
		}
		if value := strings.TrimSpace(row.TraceID.String); value != "" {
			return value, nil
		}
	}

	switch {
	case workflow.UID != "":
		return syntheticWorkflowTracePrefix + workflow.UID, nil
	case task.UID != "":
		return syntheticTaskTracePrefix + task.UID, nil
	case webhook.UID != "":
		return syntheticWebhookTracePrefix + webhook.UID, nil
	default:
		return "", nil
	}
}

func (s *Service) inferFromTraceCache(ctx context.Context, workspaceID uint64, traceID string, workflow workflowRunRecord, task taskRunRecord, webhook webhookEventRecord) (workflowRunRecord, taskRunRecord, webhookEventRecord, error) {
	traceID = strings.TrimSpace(traceID)
	if traceID == "" {
		return workflow, task, webhook, nil
	}
	var rows []struct {
		WorkflowRunID sql.NullInt64 `gorm:"column:workflow_run_id"`
		TaskRunID     sql.NullInt64 `gorm:"column:task_run_id"`
		WebhookEvent  sql.NullInt64 `gorm:"column:webhook_event_id"`
	}
	if err := s.db.WithContext(ctx).Raw(
		`SELECT workflow_run_id, task_run_id, webhook_event_id
		   FROM trace_events
		  WHERE workspace_id = ?
		    AND trace_id = ?
		  ORDER BY occurred_at DESC, id DESC
		  LIMIT 200`,
		workspaceID, traceID,
	).Scan(&rows).Error; err != nil {
		return workflow, task, webhook, err
	}
	for _, row := range rows {
		if workflow.ID == 0 && row.WorkflowRunID.Valid {
			record, err := s.workflowByID(ctx, workspaceID, uint64(row.WorkflowRunID.Int64))
			if err != nil {
				return workflow, task, webhook, err
			}
			if record.ID != 0 {
				workflow = record
			}
		}
		if task.ID == 0 && row.TaskRunID.Valid {
			record, err := s.taskByID(ctx, workspaceID, uint64(row.TaskRunID.Int64))
			if err != nil {
				return workflow, task, webhook, err
			}
			if record.ID != 0 {
				task = record
			}
		}
		if webhook.ID == 0 && row.WebhookEvent.Valid {
			record, err := s.webhookByID(ctx, workspaceID, uint64(row.WebhookEvent.Int64))
			if err != nil {
				return workflow, task, webhook, err
			}
			if record.ID != 0 {
				webhook = record
			}
		}
		if workflow.ID != 0 && task.ID != 0 && webhook.ID != 0 {
			break
		}
	}
	return workflow, task, webhook, nil
}

func (s *Service) syncTraceCache(ctx context.Context, workspaceID uint64, traceID string, workflow workflowRunRecord, task taskRunRecord, webhook webhookEventRecord, timeline []TimelineItem) error {
	traceID = strings.TrimSpace(traceID)
	if traceID == "" || workspaceID == 0 {
		return nil
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		args, clause := traceReferenceClause(workspaceID, workflow.ID, task.ID, webhook.ID)
		if clause == "" {
			clause = "trace_id = ?"
			args = []interface{}{workspaceID, traceID}
		} else {
			clause = "(trace_id = ? OR " + clause + ")"
			args = append([]interface{}{workspaceID, traceID}, args[1:]...)
		}
		if err := tx.WithContext(ctx).Exec(
			`DELETE FROM trace_events
			  WHERE workspace_id = ?
			    AND `+clause,
			args...,
		).Error; err != nil {
			return err
		}
		for _, item := range timeline {
			eventUID, err := uid.New()
			if err != nil {
				return err
			}
			if err := tx.WithContext(ctx).Exec(
				`INSERT INTO trace_events(
				    uid, workspace_id, trace_id, workflow_run_id, task_run_id, webhook_event_id,
				    source_type, source_id, category, title, status, reference_value, detail, payload, occurred_at
				  ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				eventUID,
				workspaceID,
				traceID,
				nullUint64(workflow.ID),
				nullUint64(task.ID),
				nullUint64(webhook.ID),
				strings.TrimSpace(item.Category),
				sql.NullInt64{},
				strings.TrimSpace(item.Category),
				strings.TrimSpace(item.Title),
				nullString(item.Status),
				nullString(item.Reference),
				nullString(item.Detail),
				nullString(item.Payload),
				parseTimelineTime(item.Time),
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func traceReferenceClause(workspaceID, workflowRunID, taskRunID, webhookEventID uint64) ([]interface{}, string) {
	args := []interface{}{workspaceID}
	refs := make([]string, 0, 3)
	if workflowRunID != 0 {
		refs = append(refs, "workflow_run_id = ?")
		args = append(args, workflowRunID)
	}
	if taskRunID != 0 {
		refs = append(refs, "task_run_id = ?")
		args = append(args, taskRunID)
	}
	if webhookEventID != 0 {
		refs = append(refs, "webhook_event_id = ?")
		args = append(args, webhookEventID)
	}
	return args, strings.Join(refs, " OR ")
}

func parseTimelineTime(value string) time.Time {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Now()
	}
	if parsed, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local); err == nil {
		return parsed
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	return time.Now()
}

func nullUint64(value uint64) sql.NullInt64 {
	if value == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(value), Valid: true}
}

func nullString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	return sql.NullString{String: value, Valid: value != ""}
}
