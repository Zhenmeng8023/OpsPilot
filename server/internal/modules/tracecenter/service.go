package tracecenter

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"opspilot/server/internal/config"
	"opspilot/server/internal/shared/apperror"
)

type Service struct {
	db  *gorm.DB
	cfg config.Config
}

type SearchInput struct {
	TraceID        string
	TaskRunID      string
	WorkflowRunID  string
	WebhookEventID string
}

type TimelineItem struct {
	Time      string `json:"time"`
	Category  string `json:"category"`
	Title     string `json:"title"`
	Status    string `json:"status,omitempty"`
	Reference string `json:"reference,omitempty"`
	Detail    string `json:"detail,omitempty"`
	Payload   string `json:"payload,omitempty"`
}

type SearchResult struct {
	TraceID        string         `json:"traceId,omitempty"`
	TaskRunID      string         `json:"taskRunId,omitempty"`
	WorkflowRunID  string         `json:"workflowRunId,omitempty"`
	WebhookEventID string         `json:"webhookEventId,omitempty"`
	Timeline       []TimelineItem `json:"timeline"`
}

type workspaceRecord struct {
	ID uint64
}

type workflowRunRecord struct {
	ID          uint64
	UID         string
	Status      string
	TriggerType string
	TriggerID   sql.NullInt64
	Error       sql.NullString
	WorkflowUID sql.NullString
	Workflow    sql.NullString
	CreatedAt   string
	StartedAt   sql.NullString
	FinishedAt  sql.NullString
}

type taskRunRecord struct {
	ID         uint64
	UID        string
	TaskUID    sql.NullString
	Name       string
	Trigger    sql.NullString
	Status     string
	Error      sql.NullString
	CreatedAt  string
	StartedAt  sql.NullString
	FinishedAt sql.NullString
}

type webhookEventRecord struct {
	ID        uint64
	UID       string
	SourceUID sql.NullString
	Source    sql.NullString
	EventType sql.NullString
	Status    string
	Error     sql.NullString
	Received  string
}

func NewService(db *gorm.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

func (s *Service) Search(ctx context.Context, input SearchInput) (SearchResult, *apperror.Error) {
	input = normalizeInput(input)
	if input.TraceID == "" && input.TaskRunID == "" && input.WorkflowRunID == "" && input.WebhookEventID == "" {
		return SearchResult{}, apperror.New(http.StatusBadRequest, 400970, "at least one query key is required")
	}

	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return SearchResult{}, appErr
	}

	workflow, err := s.workflowByUID(ctx, workspace.ID, input.WorkflowRunID)
	if err != nil {
		return SearchResult{}, wrapErr(500970, "load workflow run failed", err)
	}
	if input.WorkflowRunID != "" && workflow.ID == 0 {
		return SearchResult{}, apperror.New(http.StatusNotFound, 404970, "workflow run not found")
	}

	task, err := s.taskByUID(ctx, workspace.ID, input.TaskRunID)
	if err != nil {
		return SearchResult{}, wrapErr(500971, "load task run failed", err)
	}
	if input.TaskRunID != "" && task.ID == 0 {
		return SearchResult{}, apperror.New(http.StatusNotFound, 404971, "task run not found")
	}

	webhook, err := s.webhookByUID(ctx, workspace.ID, input.WebhookEventID)
	if err != nil {
		return SearchResult{}, wrapErr(500972, "load webhook event failed", err)
	}
	if input.WebhookEventID != "" && webhook.ID == 0 {
		return SearchResult{}, apperror.New(http.StatusNotFound, 404972, "webhook event not found")
	}

	if webhook.ID != 0 && (workflow.ID == 0 || task.ID == 0) {
		linkedWorkflow, linkedTask, linkErr := s.matchLinkedRuns(ctx, workspace.ID, webhook.ID)
		if linkErr != nil {
			return SearchResult{}, wrapErr(500973, "load webhook links failed", linkErr)
		}
		if workflow.ID == 0 {
			workflow = linkedWorkflow
		}
		if task.ID == 0 {
			task = linkedTask
		}
	}
	if workflow.ID != 0 && task.ID == 0 {
		linkedTask, linkErr := s.taskByWorkflowRun(ctx, workspace.ID, workflow.ID)
		if linkErr != nil {
			return SearchResult{}, wrapErr(500974, "load workflow task link failed", linkErr)
		}
		if linkedTask.ID != 0 {
			task = linkedTask
		}
	}
	if task.ID != 0 && workflow.ID == 0 {
		linkedWorkflow, linkErr := s.workflowByTaskRun(ctx, workspace.ID, task.ID)
		if linkErr != nil {
			return SearchResult{}, wrapErr(500975, "load task workflow link failed", linkErr)
		}
		if linkedWorkflow.ID != 0 {
			workflow = linkedWorkflow
		}
	}

	if input.TraceID != "" && (workflow.ID == 0 || task.ID == 0 || webhook.ID == 0) {
		wf, tk, wh, inferErr := s.inferFromTrace(ctx, workspace.ID, input.TraceID, workflow, task, webhook)
		if inferErr != nil {
			return SearchResult{}, wrapErr(500976, "infer trace links failed", inferErr)
		}
		workflow, task, webhook = wf, tk, wh
	}

	timeline := make([]TimelineItem, 0, 64)
	if workflow.ID != 0 {
		items, queryErr := s.workflowTimeline(ctx, workflow.ID)
		if queryErr != nil {
			return SearchResult{}, wrapErr(500977, "load workflow timeline failed", queryErr)
		}
		timeline = append(timeline, items...)
	}
	if task.ID != 0 {
		items, queryErr := s.taskTimeline(ctx, task.ID)
		if queryErr != nil {
			return SearchResult{}, wrapErr(500978, "load task timeline failed", queryErr)
		}
		timeline = append(timeline, items...)
	}
	if webhook.ID != 0 {
		items, queryErr := s.webhookTimeline(ctx, webhook.ID)
		if queryErr != nil {
			return SearchResult{}, wrapErr(500979, "load webhook timeline failed", queryErr)
		}
		timeline = append(timeline, items...)
	}
	if workflow.ID != 0 || task.ID != 0 {
		items, queryErr := s.notificationTimeline(ctx, workspace.ID, workflow.ID, task.ID)
		if queryErr != nil {
			return SearchResult{}, wrapErr(500980, "load notification timeline failed", queryErr)
		}
		timeline = append(timeline, items...)
	}
	items, queryErr := s.auditTimeline(ctx, workspace.ID, input.TraceID, workflow.ID, task.ID, webhook.ID)
	if queryErr != nil {
		return SearchResult{}, wrapErr(500981, "load audit timeline failed", queryErr)
	}
	timeline = append(timeline, items...)

	sort.Slice(timeline, func(i, j int) bool {
		if timeline[i].Time == timeline[j].Time {
			if timeline[i].Category == timeline[j].Category {
				return timeline[i].Title < timeline[j].Title
			}
			return timeline[i].Category < timeline[j].Category
		}
		return timeline[i].Time < timeline[j].Time
	})

	return SearchResult{
		TraceID:        input.TraceID,
		TaskRunID:      task.UID,
		WorkflowRunID:  workflow.UID,
		WebhookEventID: webhook.UID,
		Timeline:       timeline,
	}, nil
}

func (s *Service) inferFromTrace(ctx context.Context, workspaceID uint64, traceID string, workflow workflowRunRecord, task taskRunRecord, webhook webhookEventRecord) (workflowRunRecord, taskRunRecord, webhookEventRecord, error) {
	var rows []struct {
		ResourceType sql.NullString `gorm:"column:resource_type"`
		ResourceID   sql.NullInt64  `gorm:"column:resource_id"`
	}
	if err := s.db.WithContext(ctx).Raw(
		`SELECT resource_type, resource_id
		   FROM audit_logs
		  WHERE workspace_id = ?
		    AND trace_id = ?
		  ORDER BY id DESC
		  LIMIT 200`,
		workspaceID, traceID,
	).Scan(&rows).Error; err != nil {
		return workflow, task, webhook, err
	}
	for _, row := range rows {
		if !row.ResourceID.Valid || !row.ResourceType.Valid {
			continue
		}
		switch row.ResourceType.String {
		case "workflow_run":
			if workflow.ID == 0 {
				record, err := s.workflowByID(ctx, workspaceID, uint64(row.ResourceID.Int64))
				if err != nil {
					return workflow, task, webhook, err
				}
				if record.ID != 0 {
					workflow = record
				}
			}
		case "task_run":
			if task.ID == 0 {
				record, err := s.taskByID(ctx, workspaceID, uint64(row.ResourceID.Int64))
				if err != nil {
					return workflow, task, webhook, err
				}
				if record.ID != 0 {
					task = record
				}
			}
		case "webhook_event":
			if webhook.ID == 0 {
				record, err := s.webhookByID(ctx, workspaceID, uint64(row.ResourceID.Int64))
				if err != nil {
					return workflow, task, webhook, err
				}
				if record.ID != 0 {
					webhook = record
				}
			}
		}
	}
	return workflow, task, webhook, nil
}

func (s *Service) workspace(ctx context.Context) (workspaceRecord, *apperror.Error) {
	var workspace workspaceRecord
	err := s.db.WithContext(ctx).Raw("SELECT id FROM workspaces WHERE slug = ? AND status = 'active' LIMIT 1", s.cfg.Bootstrap.WorkspaceSlug).Scan(&workspace).Error
	if err != nil {
		return workspaceRecord{}, apperror.Wrap(http.StatusInternalServerError, 500982, "load workspace failed", err)
	}
	if workspace.ID == 0 {
		return workspaceRecord{}, apperror.New(http.StatusInternalServerError, 500983, "default workspace is not initialized")
	}
	return workspace, nil
}

func normalizeInput(input SearchInput) SearchInput {
	input.TraceID = strings.TrimSpace(input.TraceID)
	input.TaskRunID = strings.TrimSpace(input.TaskRunID)
	input.WorkflowRunID = strings.TrimSpace(input.WorkflowRunID)
	input.WebhookEventID = strings.TrimSpace(input.WebhookEventID)
	return input
}

func wrapErr(code int, message string, err error) *apperror.Error {
	if appErr, ok := err.(*apperror.Error); ok {
		return appErr
	}
	return apperror.Wrap(http.StatusInternalServerError, code, message, err)
}

func (s *Service) workflowByUID(ctx context.Context, workspaceID uint64, uid string) (workflowRunRecord, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return workflowRunRecord{}, nil
	}
	return s.workflowLookup(ctx, workspaceID, "wr.uid = ?", uid)
}

func (s *Service) workflowByID(ctx context.Context, workspaceID, id uint64) (workflowRunRecord, error) {
	if id == 0 {
		return workflowRunRecord{}, nil
	}
	return s.workflowLookup(ctx, workspaceID, "wr.id = ?", id)
}

func (s *Service) workflowLookup(ctx context.Context, workspaceID uint64, where string, args ...interface{}) (workflowRunRecord, error) {
	var row workflowRunRecord
	query := `SELECT wr.id, wr.uid, wr.status, wr.trigger_type, wr.trigger_id, wr.error_message,
		        wd.uid AS workflow_uid, wd.name AS workflow,
		        DATE_FORMAT(wr.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
		        DATE_FORMAT(wr.started_at, '%Y-%m-%d %H:%i:%s') AS started_at,
		        DATE_FORMAT(wr.finished_at, '%Y-%m-%d %H:%i:%s') AS finished_at
		   FROM workflow_runs wr
		   JOIN workflow_definitions wd ON wd.id = wr.workflow_id
		  WHERE `
	queryArgs := make([]interface{}, 0, len(args)+1)
	if workspaceID != 0 {
		query += "wr.workspace_id = ? AND "
		queryArgs = append(queryArgs, workspaceID)
	}
	query += where + " LIMIT 1"
	queryArgs = append(queryArgs, args...)
	err := s.db.WithContext(ctx).Raw(query, queryArgs...).Scan(&row).Error
	return row, err
}

func (s *Service) taskByUID(ctx context.Context, workspaceID uint64, uid string) (taskRunRecord, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return taskRunRecord{}, nil
	}
	return s.taskLookup(ctx, workspaceID, "tr.uid = ?", uid)
}

func (s *Service) taskByID(ctx context.Context, workspaceID, id uint64) (taskRunRecord, error) {
	if id == 0 {
		return taskRunRecord{}, nil
	}
	return s.taskLookup(ctx, workspaceID, "tr.id = ?", id)
}

func (s *Service) taskLookup(ctx context.Context, workspaceID uint64, where string, args ...interface{}) (taskRunRecord, error) {
	var row taskRunRecord
	query := `SELECT tr.id, tr.uid, tr.status, tr.error_message, tr.trigger_type AS trigger,
		        t.uid AS task_uid, t.name,
		        DATE_FORMAT(tr.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
		        DATE_FORMAT(tr.started_at, '%Y-%m-%d %H:%i:%s') AS started_at,
		        DATE_FORMAT(tr.finished_at, '%Y-%m-%d %H:%i:%s') AS finished_at
		   FROM task_runs tr
		   JOIN tasks t ON t.id = tr.task_id
		  WHERE `
	queryArgs := make([]interface{}, 0, len(args)+1)
	if workspaceID != 0 {
		query += "tr.workspace_id = ? AND "
		queryArgs = append(queryArgs, workspaceID)
	}
	query += where + " LIMIT 1"
	queryArgs = append(queryArgs, args...)
	err := s.db.WithContext(ctx).Raw(query, queryArgs...).Scan(&row).Error
	return row, err
}

func (s *Service) webhookByUID(ctx context.Context, workspaceID uint64, uid string) (webhookEventRecord, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return webhookEventRecord{}, nil
	}
	return s.webhookLookup(ctx, workspaceID, "we.uid = ?", uid)
}

func (s *Service) webhookByID(ctx context.Context, workspaceID, id uint64) (webhookEventRecord, error) {
	if id == 0 {
		return webhookEventRecord{}, nil
	}
	return s.webhookLookup(ctx, workspaceID, "we.id = ?", id)
}

func (s *Service) webhookLookup(ctx context.Context, workspaceID uint64, where string, args ...interface{}) (webhookEventRecord, error) {
	var row webhookEventRecord
	query := `SELECT we.id, we.uid, we.status, we.event_type, we.error_message,
		        ws.uid AS source_uid, ws.name AS source,
		        DATE_FORMAT(we.received_at, '%Y-%m-%d %H:%i:%s') AS received
		   FROM webhook_events we
		   LEFT JOIN webhook_sources ws ON ws.id = we.source_id
		  WHERE `
	queryArgs := make([]interface{}, 0, len(args)+1)
	if workspaceID != 0 {
		query += "we.workspace_id = ? AND "
		queryArgs = append(queryArgs, workspaceID)
	}
	query += where + " LIMIT 1"
	queryArgs = append(queryArgs, args...)
	err := s.db.WithContext(ctx).Raw(query, queryArgs...).Scan(&row).Error
	return row, err
}

func (s *Service) matchLinkedRuns(ctx context.Context, workspaceID, webhookEventID uint64) (workflowRunRecord, taskRunRecord, error) {
	var link struct {
		WorkflowRunID sql.NullInt64 `gorm:"column:workflow_run_id"`
		TaskRunID     sql.NullInt64 `gorm:"column:task_run_id"`
	}
	if err := s.db.WithContext(ctx).Raw(
		`SELECT workflow_run_id, task_run_id
		   FROM webhook_event_matches
		  WHERE event_id = ?
		  ORDER BY id DESC
		  LIMIT 1`,
		webhookEventID,
	).Scan(&link).Error; err != nil {
		return workflowRunRecord{}, taskRunRecord{}, err
	}
	var workflow workflowRunRecord
	var task taskRunRecord
	var err error
	if link.WorkflowRunID.Valid {
		workflow, err = s.workflowByID(ctx, workspaceID, uint64(link.WorkflowRunID.Int64))
		if err != nil {
			return workflowRunRecord{}, taskRunRecord{}, err
		}
	}
	if link.TaskRunID.Valid {
		task, err = s.taskByID(ctx, workspaceID, uint64(link.TaskRunID.Int64))
		if err != nil {
			return workflowRunRecord{}, taskRunRecord{}, err
		}
	}
	return workflow, task, nil
}

func (s *Service) taskByWorkflowRun(ctx context.Context, workspaceID, workflowRunID uint64) (taskRunRecord, error) {
	if workflowRunID == 0 {
		return taskRunRecord{}, nil
	}
	var taskRunID uint64
	if err := s.db.WithContext(ctx).Raw(
		`SELECT task_run_id
		   FROM workflow_run_nodes
		  WHERE run_id = ?
		    AND task_run_id IS NOT NULL
		  ORDER BY id DESC
		  LIMIT 1`,
		workflowRunID,
	).Scan(&taskRunID).Error; err != nil {
		return taskRunRecord{}, err
	}
	if taskRunID == 0 {
		return taskRunRecord{}, nil
	}
	return s.taskByID(ctx, workspaceID, taskRunID)
}

func (s *Service) workflowByTaskRun(ctx context.Context, workspaceID, taskRunID uint64) (workflowRunRecord, error) {
	if taskRunID == 0 {
		return workflowRunRecord{}, nil
	}
	var workflowRunID uint64
	if err := s.db.WithContext(ctx).Raw(
		`SELECT run_id
		   FROM workflow_run_nodes
		  WHERE task_run_id = ?
		  ORDER BY id DESC
		  LIMIT 1`,
		taskRunID,
	).Scan(&workflowRunID).Error; err != nil {
		return workflowRunRecord{}, err
	}
	if workflowRunID == 0 {
		return workflowRunRecord{}, nil
	}
	return s.workflowByID(ctx, workspaceID, workflowRunID)
}

func (s *Service) workflowTimeline(ctx context.Context, workflowRunID uint64) ([]TimelineItem, error) {
	row, err := s.workflowByID(ctx, 0, workflowRunID)
	if err != nil {
		return nil, err
	}
	items := make([]TimelineItem, 0, 24)
	if row.ID != 0 {
		items = append(items, TimelineItem{
			Time:      row.CreatedAt,
			Category:  "workflow",
			Title:     "workflow run created",
			Status:    row.Status,
			Reference: row.UID,
			Detail:    fmt.Sprintf("workflow=%s trigger=%s", fallback(row.Workflow.String, row.WorkflowUID.String), row.TriggerType),
		})
		if row.StartedAt.Valid {
			items = append(items, TimelineItem{
				Time:      row.StartedAt.String,
				Category:  "workflow",
				Title:     "workflow run started",
				Reference: row.UID,
			})
		}
		if row.FinishedAt.Valid {
			items = append(items, TimelineItem{
				Time:      row.FinishedAt.String,
				Category:  "workflow",
				Title:     "workflow run finished",
				Status:    row.Status,
				Reference: row.UID,
				Detail:    row.Error.String,
			})
		}
	}

	var nodes []struct {
		NodeID      string         `gorm:"column:node_id"`
		NodeType    string         `gorm:"column:node_type"`
		NodeName    sql.NullString `gorm:"column:node_name"`
		Status      string         `gorm:"column:status"`
		TaskRunUID  sql.NullString `gorm:"column:task_run_uid"`
		Error       sql.NullString `gorm:"column:error_message"`
		OccurredAt  string         `gorm:"column:occurred_at"`
	}
	if err := s.db.WithContext(ctx).Raw(
		`SELECT wrn.node_id, wrn.node_type, wrn.node_name, wrn.status, tr.uid AS task_run_uid, wrn.error_message,
		        DATE_FORMAT(COALESCE(wrn.finished_at, wrn.started_at, wrn.queued_at, wrn.created_at), '%Y-%m-%d %H:%i:%s') AS occurred_at
		   FROM workflow_run_nodes wrn
		   LEFT JOIN task_runs tr ON tr.id = wrn.task_run_id
		  WHERE wrn.run_id = ?
		  ORDER BY wrn.id ASC`,
		workflowRunID,
	).Scan(&nodes).Error; err != nil {
		return nil, err
	}
	for _, node := range nodes {
		items = append(items, TimelineItem{
			Time:      node.OccurredAt,
			Category:  "workflow-node",
			Title:     fallback(node.NodeName.String, node.NodeID),
			Status:    node.Status,
			Reference: node.TaskRunUID.String,
			Detail:    fmt.Sprintf("nodeId=%s nodeType=%s %s", node.NodeID, node.NodeType, fallback(node.Error.String, "")),
		})
	}

	var events []struct {
		ID        uint64         `gorm:"column:id"`
		NodeID    sql.NullString `gorm:"column:node_id"`
		EventType string         `gorm:"column:event_type"`
		Message   sql.NullString `gorm:"column:message"`
		Actor     sql.NullString `gorm:"column:actor"`
		Payload   sql.NullString `gorm:"column:payload"`
		CreatedAt string         `gorm:"column:created_at"`
	}
	if err := s.db.WithContext(ctx).Raw(
		`SELECT wre.id, wre.node_id, wre.event_type, wre.message, u.username AS actor,
		        CAST(wre.payload AS CHAR) AS payload,
		        DATE_FORMAT(wre.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM workflow_run_events wre
		   LEFT JOIN users u ON u.id = wre.actor_id
		  WHERE wre.run_id = ?
		  ORDER BY wre.created_at ASC, wre.id ASC`,
		workflowRunID,
	).Scan(&events).Error; err != nil {
		return nil, err
	}
	for _, event := range events {
		detail := strings.TrimSpace(event.Message.String)
		if event.Actor.Valid && event.Actor.String != "" {
			detail = strings.TrimSpace(detail + " actor=" + event.Actor.String)
		}
		items = append(items, TimelineItem{
			Time:      event.CreatedAt,
			Category:  "workflow-event",
			Title:     event.EventType,
			Reference: event.NodeID.String,
			Detail:    detail,
			Payload:   event.Payload.String,
		})
	}

	return items, nil
}

func (s *Service) taskTimeline(ctx context.Context, taskRunID uint64) ([]TimelineItem, error) {
	row, err := s.taskByID(ctx, 0, taskRunID)
	if err != nil {
		return nil, err
	}
	items := make([]TimelineItem, 0, 24)
	if row.ID != 0 {
		items = append(items, TimelineItem{
			Time:      row.CreatedAt,
			Category:  "task",
			Title:     "task run created",
			Status:    row.Status,
			Reference: row.UID,
			Detail:    fmt.Sprintf("task=%s trigger=%s", fallback(row.Name, row.TaskUID.String), row.Trigger.String),
		})
		if row.StartedAt.Valid {
			items = append(items, TimelineItem{
				Time:      row.StartedAt.String,
				Category:  "task",
				Title:     "task run started",
				Reference: row.UID,
			})
		}
		if row.FinishedAt.Valid {
			items = append(items, TimelineItem{
				Time:      row.FinishedAt.String,
				Category:  "task",
				Title:     "task run finished",
				Status:    row.Status,
				Reference: row.UID,
				Detail:    row.Error.String,
			})
		}
	}

	var targetAgg []struct {
		Status string `gorm:"column:status"`
		Count  int    `gorm:"column:count"`
	}
	if err := s.db.WithContext(ctx).Raw(
		`SELECT status, COUNT(*) AS count
		   FROM task_run_targets
		  WHERE run_id = ?
		  GROUP BY status
		  ORDER BY status ASC`,
		taskRunID,
	).Scan(&targetAgg).Error; err != nil {
		return nil, err
	}
	if len(targetAgg) > 0 {
		summary := make([]string, 0, len(targetAgg))
		for _, item := range targetAgg {
			summary = append(summary, item.Status+"="+strconv.Itoa(item.Count))
		}
		items = append(items, TimelineItem{
			Time:      row.CreatedAt,
			Category:  "task-targets",
			Title:     "task target distribution",
			Reference: row.UID,
			Detail:    strings.Join(summary, ", "),
		})
	}

	var events []struct {
		ID        uint64         `gorm:"column:id"`
		EventType string         `gorm:"column:event_type"`
		From      sql.NullString `gorm:"column:from_status"`
		To        sql.NullString `gorm:"column:to_status"`
		Message   sql.NullString `gorm:"column:message"`
		Payload   sql.NullString `gorm:"column:payload"`
		TargetUID sql.NullString `gorm:"column:target_uid"`
		CreatedAt string         `gorm:"column:created_at"`
	}
	if err := s.db.WithContext(ctx).Raw(
		`SELECT tre.id, tre.event_type, tre.from_status, tre.to_status, tre.message,
		        CAST(tre.payload AS CHAR) AS payload,
		        trt.uid AS target_uid,
		        DATE_FORMAT(tre.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM task_run_events tre
		   LEFT JOIN task_run_targets trt ON trt.id = tre.run_target_id
		  WHERE tre.run_id = ?
		  ORDER BY tre.created_at ASC, tre.id ASC`,
		taskRunID,
	).Scan(&events).Error; err != nil {
		return nil, err
	}
	for _, event := range events {
		detail := strings.TrimSpace(event.Message.String)
		if event.From.Valid || event.To.Valid {
			detail = strings.TrimSpace(detail + " " + event.From.String + "->" + event.To.String)
		}
		items = append(items, TimelineItem{
			Time:      event.CreatedAt,
			Category:  "task-event",
			Title:     event.EventType,
			Status:    event.To.String,
			Reference: event.TargetUID.String,
			Detail:    detail,
			Payload:   event.Payload.String,
		})
	}

	return items, nil
}

func (s *Service) webhookTimeline(ctx context.Context, webhookEventID uint64) ([]TimelineItem, error) {
	row, err := s.webhookByID(ctx, 0, webhookEventID)
	if err != nil {
		return nil, err
	}
	items := make([]TimelineItem, 0, 16)
	if row.ID != 0 {
		items = append(items, TimelineItem{
			Time:      row.Received,
			Category:  "webhook",
			Title:     "webhook event received",
			Status:    row.Status,
			Reference: row.UID,
			Detail:    fmt.Sprintf("source=%s type=%s %s", fallback(row.Source.String, row.SourceUID.String), row.EventType.String, fallback(row.Error.String, "")),
		})
	}

	var matches []struct {
		Matched        int            `gorm:"column:matched"`
		Reason         sql.NullString `gorm:"column:reason"`
		RuleUID        sql.NullString `gorm:"column:rule_uid"`
		TaskRunUID     sql.NullString `gorm:"column:task_run_uid"`
		WorkflowRunUID sql.NullString `gorm:"column:workflow_run_uid"`
		CreatedAt      string         `gorm:"column:created_at"`
	}
	if err := s.db.WithContext(ctx).Raw(
		`SELECT wem.matched, wem.reason, wr.uid AS rule_uid, tr.uid AS task_run_uid, wfr.uid AS workflow_run_uid,
		        DATE_FORMAT(wem.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM webhook_event_matches wem
		   LEFT JOIN webhook_rules wr ON wr.id = wem.rule_id
		   LEFT JOIN task_runs tr ON tr.id = wem.task_run_id
		   LEFT JOIN workflow_runs wfr ON wfr.id = wem.workflow_run_id
		  WHERE wem.event_id = ?
		  ORDER BY wem.id ASC`,
		webhookEventID,
	).Scan(&matches).Error; err != nil {
		return nil, err
	}
	for _, match := range matches {
		ref := strings.TrimSpace(match.WorkflowRunUID.String)
		if ref == "" {
			ref = match.TaskRunUID.String
		}
		if ref == "" {
			ref = match.RuleUID.String
		}
		items = append(items, TimelineItem{
			Time:      match.CreatedAt,
			Category:  "webhook-match",
			Title:     ternary(match.Matched > 0, "webhook matched", "webhook not matched"),
			Status:    ternary(match.Matched > 0, "matched", "ignored"),
			Reference: ref,
			Detail:    match.Reason.String,
		})
	}
	return items, nil
}

func (s *Service) notificationTimeline(ctx context.Context, workspaceID, workflowRunID, taskRunID uint64) ([]TimelineItem, error) {
	filters := make([]string, 0, 2)
	args := make([]interface{}, 0, 3)
	args = append(args, workspaceID)
	if workflowRunID != 0 {
		filters = append(filters, "(n.resource_type = 'workflow_run' AND n.resource_id = ?)")
		args = append(args, workflowRunID)
	}
	if taskRunID != 0 {
		filters = append(filters, "(n.resource_type = 'task_run' AND n.resource_id = ?)")
		args = append(args, taskRunID)
	}
	if len(filters) == 0 {
		return nil, nil
	}

	var rows []struct {
		NotificationUID string         `gorm:"column:notification_uid"`
		Title           string         `gorm:"column:title"`
		Content         sql.NullString `gorm:"column:content"`
		Severity        string         `gorm:"column:severity"`
		CreatedAt       string         `gorm:"column:created_at"`
		DeliveryStatus  sql.NullString `gorm:"column:delivery_status"`
		DeliveredAt     sql.NullString `gorm:"column:delivered_at"`
		DeliveryError   sql.NullString `gorm:"column:delivery_error"`
		ChannelUID      sql.NullString `gorm:"column:channel_uid"`
	}
	query := `SELECT n.uid AS notification_uid, n.title, n.content, n.severity,
	                 DATE_FORMAT(n.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
	                 nd.status AS delivery_status,
	                 DATE_FORMAT(nd.delivered_at, '%Y-%m-%d %H:%i:%s') AS delivered_at,
	                 nd.error_message AS delivery_error,
	                 nc.uid AS channel_uid
	            FROM notifications n
	            LEFT JOIN notification_deliveries nd ON nd.notification_id = n.id
	            LEFT JOIN notification_channels nc ON nc.id = nd.channel_id
	           WHERE n.workspace_id = ?
	             AND (` + strings.Join(filters, " OR ") + `)
	           ORDER BY n.created_at ASC, n.id ASC, nd.id ASC`
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]TimelineItem, 0, len(rows))
	for _, row := range rows {
		at := row.CreatedAt
		if row.DeliveredAt.Valid {
			at = row.DeliveredAt.String
		}
		detail := strings.TrimSpace(row.Content.String)
		if row.ChannelUID.Valid {
			detail = strings.TrimSpace(detail + " channel=" + row.ChannelUID.String)
		}
		if row.DeliveryError.Valid {
			detail = strings.TrimSpace(detail + " error=" + row.DeliveryError.String)
		}
		items = append(items, TimelineItem{
			Time:      at,
			Category:  "notification",
			Title:     row.Title,
			Status:    fallback(row.DeliveryStatus.String, row.Severity),
			Reference: row.NotificationUID,
			Detail:    detail,
		})
	}
	return items, nil
}

func (s *Service) auditTimeline(ctx context.Context, workspaceID uint64, traceID string, workflowRunID, taskRunID, webhookEventID uint64) ([]TimelineItem, error) {
	traceID = strings.TrimSpace(traceID)
	args := []interface{}{workspaceID}
	clause := "(al.workspace_id = ? OR al.workspace_id IS NULL)"
	refs := make([]string, 0, 4)
	if traceID != "" {
		refs = append(refs, "al.trace_id = ?")
		args = append(args, traceID)
	}
	if workflowRunID != 0 {
		refs = append(refs, "(al.resource_type = 'workflow_run' AND al.resource_id = ?)")
		args = append(args, workflowRunID)
	}
	if taskRunID != 0 {
		refs = append(refs, "(al.resource_type = 'task_run' AND al.resource_id = ?)")
		args = append(args, taskRunID)
	}
	if webhookEventID != 0 {
		refs = append(refs, "(al.resource_type = 'webhook_event' AND al.resource_id = ?)")
		args = append(args, webhookEventID)
	}
	if len(refs) == 0 {
		return nil, nil
	}
	clause += " AND (" + strings.Join(refs, " OR ") + ")"

	var rows []struct {
		Action       string         `gorm:"column:action"`
		Result       string         `gorm:"column:result"`
		TraceID      sql.NullString `gorm:"column:trace_id"`
		ResourceType sql.NullString `gorm:"column:resource_type"`
		ResourceID   sql.NullInt64  `gorm:"column:resource_id"`
		ActorType    string         `gorm:"column:actor_type"`
		ActorUser    sql.NullString `gorm:"column:actor_user"`
		ActorAgent   sql.NullString `gorm:"column:actor_agent"`
		Request      sql.NullString `gorm:"column:request_path"`
		Metadata     sql.NullString `gorm:"column:metadata"`
		CreatedAt    string         `gorm:"column:created_at"`
	}
	query := `SELECT al.action, al.result, al.trace_id, al.resource_type, al.resource_id, al.actor_type,
	                 u.username AS actor_user, ag.name AS actor_agent, al.request_path,
	                 CAST(al.metadata AS CHAR) AS metadata,
	                 DATE_FORMAT(al.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
	            FROM audit_logs al
	            LEFT JOIN users u ON u.id = al.actor_user_id
	            LEFT JOIN agents ag ON ag.id = al.actor_agent_id
	           WHERE ` + clause + `
	           ORDER BY al.created_at ASC, al.id ASC
	           LIMIT 400`
	if err := s.db.WithContext(ctx).Raw(query, args...).Scan(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]TimelineItem, 0, len(rows))
	for _, row := range rows {
		actor := row.ActorType
		if row.ActorUser.Valid && row.ActorUser.String != "" {
			actor = actor + ":" + row.ActorUser.String
		} else if row.ActorAgent.Valid && row.ActorAgent.String != "" {
			actor = actor + ":" + row.ActorAgent.String
		}
		ref := strings.TrimSpace(row.TraceID.String)
		if ref == "" && row.ResourceType.Valid && row.ResourceID.Valid {
			ref = row.ResourceType.String + "#" + strconv.FormatInt(row.ResourceID.Int64, 10)
		}
		detail := "actor=" + actor
		if row.Request.Valid && row.Request.String != "" {
			detail += " path=" + row.Request.String
		}
		items = append(items, TimelineItem{
			Time:      row.CreatedAt,
			Category:  "audit",
			Title:     row.Action,
			Status:    row.Result,
			Reference: ref,
			Detail:    detail,
			Payload:   row.Metadata.String,
		})
	}
	return items, nil
}

func fallback(value, alt string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return strings.TrimSpace(alt)
}

func ternary(ok bool, yes, no string) string {
	if ok {
		return yes
	}
	return no
}
