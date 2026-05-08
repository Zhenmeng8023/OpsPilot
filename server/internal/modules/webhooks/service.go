package webhooks

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"

	"opspilot/server/internal/config"
	"opspilot/server/internal/modules/execution"
	"opspilot/server/internal/modules/workflows"
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

type CreateSourceInput struct {
	Name       string
	SourceType string
	Audit      AuditContext
}

type CreateRuleInput struct {
	SourceID   string
	TargetType string
	TaskID     string
	WorkflowID string
	Name       string
	EventType  string
	Matcher    *Matcher
	Audit      AuditContext
}

type UpdateRuleInput struct {
	ID        string
	Name      string
	EventType string
	Matcher   *Matcher
	Audit     AuditContext
}

type ListEventsInput struct {
	SourceID     string
	Status       string
	DeliveryID   string
	ReceivedFrom string
	ReceivedTo   string
	Page         int
	PageSize     int
}

type EventDetailInput struct {
	EventID string
}

type TriggerInput struct {
	Token           string
	EventType       string
	DeliveryID      string
	Signature       string
	SignatureHeader string
	Timestamp       string
	Nonce           string
	RemoteIP        string
	Headers         map[string]string
	Body            []byte
}

type SourceSummary struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	SourceType     string `json:"sourceType"`
	Status         string `json:"status"`
	LastReceivedAt string `json:"lastReceivedAt,omitempty"`
	CreatedBy      string `json:"createdBy,omitempty"`
	CreatedAt      string `json:"createdAt"`
}

type SourceDetail struct {
	SourceSummary
	Token         string `json:"token,omitempty"`
	SigningSecret string `json:"signingSecret,omitempty"`
}

type RuleSummary struct {
	ID           string   `json:"id"`
	SourceID     string   `json:"sourceId"`
	SourceName   string   `json:"sourceName"`
	TargetType   string   `json:"targetType"`
	TaskID       string   `json:"taskId,omitempty"`
	TaskName     string   `json:"taskName,omitempty"`
	WorkflowID   string   `json:"workflowId,omitempty"`
	WorkflowName string   `json:"workflowName,omitempty"`
	Name         string   `json:"name"`
	EventType    string   `json:"eventType,omitempty"`
	Matcher      *Matcher `json:"matcher,omitempty"`
	Status       string   `json:"status"`
	CreatedBy    string   `json:"createdBy,omitempty"`
	CreatedAt    string   `json:"createdAt"`
}

type Matcher struct {
	Conditions []MatcherCondition `json:"conditions"`
}

type MatcherCondition struct {
	Type  string `json:"type"`
	Key   string `json:"key,omitempty"`
	Path  string `json:"path,omitempty"`
	Value string `json:"value"`
}

type EventSummary struct {
	ID              string `json:"id"`
	SourceID        string `json:"sourceId,omitempty"`
	SourceName      string `json:"sourceName,omitempty"`
	EventType       string `json:"eventType,omitempty"`
	DeliveryID      string `json:"deliveryId,omitempty"`
	SourceTimestamp string `json:"sourceTimestamp,omitempty"`
	Nonce           string `json:"nonce,omitempty"`
	SignatureHeader string `json:"signatureHeader,omitempty"`
	SignatureValid  bool   `json:"signatureValid"`
	Replayed        bool   `json:"replayed"`
	Status          string `json:"status"`
	ErrorMessage    string `json:"errorMessage,omitempty"`
	RemoteIP        string `json:"remoteIp,omitempty"`
	PayloadHash     string `json:"payloadHash,omitempty"`
	ReceivedAt      string `json:"receivedAt"`
}

type EventMatchSummary struct {
	ID            uint64 `json:"id"`
	RuleID        string `json:"ruleId,omitempty"`
	RuleName      string `json:"ruleName,omitempty"`
	Matched       bool   `json:"matched"`
	Reason        string `json:"reason,omitempty"`
	TaskRunID     string `json:"taskRunId,omitempty"`
	WorkflowRunID string `json:"workflowRunId,omitempty"`
	CreatedAt     string `json:"createdAt"`
}

type EventDetail struct {
	EventSummary
	Headers []HeaderPair        `json:"headers"`
	Payload string              `json:"payload,omitempty"`
	Matches []EventMatchSummary `json:"matches"`
}

type HeaderPair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type EventListResult struct {
	Items    []EventSummary `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

type TriggerResult struct {
	EventID        string   `json:"eventId"`
	Status         string   `json:"status"`
	MatchedRules   int      `json:"matchedRules"`
	TriggeredRuns  []string `json:"triggeredRuns"`
	RejectedReason string   `json:"rejectedReason,omitempty"`
}

type sourceRecord struct {
	ID                        uint64
	UID                       string
	WorkspaceID               uint64
	Name                      string
	SourceType                string
	SigningSecret             sql.NullString
	TimestampToleranceSeconds uint
	Status                    string
	LastReceivedAt            sql.NullString
	CreatedBy                 sql.NullString
	CreatedByID               sql.NullInt64
	CreatedAt                 string
}

type ruleRecord struct {
	ID           uint64
	UID          string
	WorkspaceID  uint64
	SourceID     uint64
	SourceUID    string
	SourceName   string
	TargetType   string
	TaskID       uint64
	TaskUID      string
	TaskName     string
	WorkflowID   uint64
	WorkflowUID  string
	WorkflowName string
	Name         string
	EventType    sql.NullString
	Matcher      sql.NullString
	Status       string
	CreatedBy    sql.NullString
	CreatedByID  sql.NullInt64
	CreatedAt    string
}

type eventRecord struct {
	ID              uint64
	UID             string
	SourceUID       sql.NullString
	SourceName      sql.NullString
	EventType       sql.NullString
	DeliveryID      sql.NullString
	SourceTimestamp sql.NullString
	Nonce           sql.NullString
	SignatureHeader sql.NullString
	SignatureValid  bool
	Replayed        bool
	Status          string
	ErrorMessage    sql.NullString
	RemoteIP        sql.NullString
	PayloadHash     sql.NullString
	Headers         sql.NullString
	Payload         sql.NullString
	ReceivedAt      string
}

type eventMatchRecord struct {
	ID             uint64
	RuleUID        sql.NullString
	RuleName       sql.NullString
	Matched        bool
	Reason         sql.NullString
	TaskRunUID     sql.NullString
	WorkflowRunUID sql.NullString
	CreatedAt      string
}

type workspaceRecord struct {
	ID uint64
}

type userRecord struct {
	ID uint64
}

type taskRecord struct {
	ID uint64
}

type workflowRecord struct {
	ID     uint64
	UID    string
	Name   string
	Status string
}

func NewService(db *gorm.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

func (s *Service) ListSources(ctx context.Context) ([]SourceSummary, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	var rows []sourceRecord
	if err := s.db.WithContext(ctx).Raw(
		`SELECT ws.id, ws.uid, ws.workspace_id, ws.name, ws.source_type, ws.status,
		        ws.signing_secret, ws.timestamp_tolerance_seconds,
		        DATE_FORMAT(ws.last_received_at, '%Y-%m-%d %H:%i:%s') AS last_received_at,
		        u.username AS created_by, ws.created_by AS created_by_id,
		        DATE_FORMAT(ws.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM webhook_sources ws
		   LEFT JOIN users u ON u.id = ws.created_by
		  WHERE ws.workspace_id = ? AND ws.deleted_at IS NULL
		  ORDER BY ws.created_at DESC`,
		workspace.ID,
	).Scan(&rows).Error; err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500501, "list webhook sources failed", err)
	}
	out := make([]SourceSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, sourceSummary(row))
	}
	return out, nil
}

func (s *Service) CreateSource(ctx context.Context, input CreateSourceInput) (SourceDetail, *apperror.Error) {
	name := strings.TrimSpace(input.Name)
	sourceType := strings.TrimSpace(input.SourceType)
	if sourceType == "" {
		sourceType = "custom"
	}
	if name == "" {
		return SourceDetail{}, apperror.New(http.StatusBadRequest, 400501, "webhook source name is required")
	}
	if !validSourceType(sourceType) {
		return SourceDetail{}, apperror.New(http.StatusBadRequest, 400502, "unsupported webhook source type")
	}
	token, err := newSecret()
	if err != nil {
		return SourceDetail{}, apperror.Wrap(http.StatusInternalServerError, 500502, "issue webhook token failed", err)
	}
	signingSecret, err := newSecret()
	if err != nil {
		return SourceDetail{}, apperror.Wrap(http.StatusInternalServerError, 500502, "issue webhook signing secret failed", err)
	}
	var created SourceDetail
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		actor, err := userByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		actorID := nullID(actor.ID)
		sourceUID, err := uid.New()
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO webhook_sources(uid, workspace_id, name, source_type, token_hash, signing_secret, timestamp_tolerance_seconds, status, created_by)
			 VALUES (?, ?, ?, ?, ?, ?, 300, 'active', ?)`,
			sourceUID, workspace.ID, name, sourceType, hash(token), signingSecret, actorID,
		).Error; err != nil {
			return err
		}
		row, err := sourceByUID(ctx, tx, workspace.ID, sourceUID)
		if err != nil {
			return err
		}
		created = SourceDetail{SourceSummary: sourceSummary(row), Token: token, SigningSecret: signingSecret}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "webhook.source.create",
			ResourceType:  "webhook_source",
			ResourceID:    nullID(row.ID),
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			After:         created.SourceSummary,
		})
		return nil
	})
	if txErr != nil {
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return SourceDetail{}, apperror.New(http.StatusConflict, 409501, "webhook source already exists")
		}
		return SourceDetail{}, apperror.Wrap(http.StatusInternalServerError, 500503, "create webhook source failed", txErr)
	}
	return created, nil
}

func (s *Service) PauseSource(ctx context.Context, sourceUID string, auditCtx AuditContext) *apperror.Error {
	return s.updateSourceStatus(ctx, sourceUID, "paused", "webhook.source.pause", auditCtx)
}

func (s *Service) ResumeSource(ctx context.Context, sourceUID string, auditCtx AuditContext) *apperror.Error {
	return s.updateSourceStatus(ctx, sourceUID, "active", "webhook.source.resume", auditCtx)
}

func (s *Service) DisableSource(ctx context.Context, sourceUID string, auditCtx AuditContext) *apperror.Error {
	return s.updateSourceStatus(ctx, sourceUID, "disabled", "webhook.source.disable", auditCtx)
}

func (s *Service) updateSourceStatus(ctx context.Context, sourceUID, status, action string, auditCtx AuditContext) *apperror.Error {
	sourceUID = strings.TrimSpace(sourceUID)
	if sourceUID == "" {
		return apperror.New(http.StatusBadRequest, 400508, "webhook source id is required")
	}
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		row, err := sourceByUID(ctx, tx, workspace.ID, sourceUID)
		if err != nil {
			return err
		}
		if row.ID == 0 {
			return apperror.New(http.StatusNotFound, 404504, "webhook source not found")
		}
		if err := tx.WithContext(ctx).Exec(
			"UPDATE webhook_sources SET status = ? WHERE id = ?",
			status, row.ID,
		).Error; err != nil {
			return err
		}
		actor, err := userByUID(ctx, tx, auditCtx.ActorUID)
		if err != nil {
			return err
		}
		updated := row
		updated.Status = status
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   nullID(actor.ID),
			Action:        action,
			ResourceType:  "webhook_source",
			ResourceID:    nullID(row.ID),
			IP:            auditCtx.IP,
			UserAgent:     auditCtx.UserAgent,
			TraceID:       auditCtx.TraceID,
			RequestMethod: auditCtx.RequestMethod,
			RequestPath:   auditCtx.RequestPath,
			Before:        sourceSummary(row),
			After:         sourceSummary(updated),
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return appErr
		}
		return apperror.Wrap(http.StatusInternalServerError, 500512, "update webhook source failed", txErr)
	}
	return nil
}

func (s *Service) ListRules(ctx context.Context) ([]RuleSummary, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	rows, err := rules(ctx, s.db, workspace.ID, 0, "")
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500504, "list webhook rules failed", err)
	}
	out := make([]RuleSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, ruleSummary(row))
	}
	return out, nil
}

func (s *Service) CreateRule(ctx context.Context, input CreateRuleInput) (RuleSummary, *apperror.Error) {
	name := strings.TrimSpace(input.Name)
	sourceUID := strings.TrimSpace(input.SourceID)
	targetType := normalizeWebhookTargetType(input.TargetType)
	taskUID := strings.TrimSpace(input.TaskID)
	workflowUID := strings.TrimSpace(input.WorkflowID)
	eventType := strings.TrimSpace(input.EventType)
	if name == "" || sourceUID == "" {
		return RuleSummary{}, apperror.New(http.StatusBadRequest, 400503, "sourceId and name are required")
	}
	if targetType == "workflow" && workflowUID == "" {
		return RuleSummary{}, apperror.New(http.StatusBadRequest, 400508, "workflowId is required")
	}
	if targetType == "task" && taskUID == "" {
		return RuleSummary{}, apperror.New(http.StatusBadRequest, 400402, "taskId is required")
	}
	matcher, appErr := normalizeMatcher(input.Matcher)
	if appErr != nil {
		return RuleSummary{}, appErr
	}
	var created RuleSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		source, err := sourceByUID(ctx, tx, workspace.ID, sourceUID)
		if err != nil {
			return err
		}
		if source.ID == 0 {
			return apperror.New(http.StatusNotFound, 404501, "webhook source not found")
		}
		taskID := sql.NullInt64{}
		workflowID := sql.NullInt64{}
		if targetType == "workflow" {
			workflow, err := workflowByUID(ctx, tx, workspace.ID, workflowUID)
			if err != nil {
				return err
			}
			if workflow.ID == 0 {
				return apperror.New(http.StatusNotFound, 404505, "workflow definition not found")
			}
			if workflow.Status != "active" {
				return apperror.New(http.StatusConflict, 409504, "workflow must be active before it can be attached to a webhook rule")
			}
			workflowID = nullID(workflow.ID)
		} else {
			task, err := taskByUID(ctx, tx, workspace.ID, taskUID)
			if err != nil {
				return err
			}
			if task.ID == 0 {
				return apperror.New(http.StatusNotFound, 404401, "task definition not found")
			}
			taskID = nullID(task.ID)
		}
		actor, err := userByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		ruleUID, err := uid.New()
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO webhook_rules(uid, workspace_id, source_id, task_id, workflow_id, target_type, name, event_type, matcher, status, created_by)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'active', ?)`,
			ruleUID, workspace.ID, source.ID, taskID, workflowID, targetType, name, nullString(eventType), matcherSQL(matcher), nullID(actor.ID),
		).Error; err != nil {
			return err
		}
		rows, err := rules(ctx, tx, workspace.ID, source.ID, ruleUID)
		if err != nil {
			return err
		}
		if len(rows) > 0 {
			created = ruleSummary(rows[0])
		}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   nullID(actor.ID),
			Action:        "webhook.rule.create",
			ResourceType:  "webhook_rule",
			ResourceID:    nullID(rows[0].ID),
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
			return RuleSummary{}, appErr
		}
		return RuleSummary{}, apperror.Wrap(http.StatusInternalServerError, 500505, "create webhook rule failed", txErr)
	}
	return created, nil
}

func (s *Service) UpdateRule(ctx context.Context, input UpdateRuleInput) (RuleSummary, *apperror.Error) {
	ruleUID := strings.TrimSpace(input.ID)
	name := strings.TrimSpace(input.Name)
	eventType := strings.TrimSpace(input.EventType)
	if ruleUID == "" {
		return RuleSummary{}, apperror.New(http.StatusBadRequest, 400504, "webhook rule id is required")
	}
	if name == "" {
		return RuleSummary{}, apperror.New(http.StatusBadRequest, 400506, "webhook rule name is required")
	}
	matcher, appErr := normalizeMatcher(input.Matcher)
	if appErr != nil {
		return RuleSummary{}, appErr
	}
	var updated RuleSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		rows, err := rules(ctx, tx, workspace.ID, 0, ruleUID)
		if err != nil {
			return err
		}
		if len(rows) == 0 || rows[0].ID == 0 {
			return apperror.New(http.StatusNotFound, 404503, "webhook rule not found")
		}
		current := rows[0]
		if current.Status == "disabled" {
			return apperror.New(http.StatusBadRequest, 400507, "disabled webhook rule cannot be edited")
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE webhook_rules
			    SET name = ?, event_type = ?, matcher = ?
			  WHERE id = ?`,
			name, nullString(eventType), matcherSQL(matcher), current.ID,
		).Error; err != nil {
			return err
		}
		rows, err = rules(ctx, tx, workspace.ID, 0, ruleUID)
		if err != nil {
			return err
		}
		if len(rows) > 0 {
			updated = ruleSummary(rows[0])
		}
		actor, err := userByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   nullID(actor.ID),
			Action:        "webhook.rule.update",
			ResourceType:  "webhook_rule",
			ResourceID:    nullID(current.ID),
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			Before:        ruleSummary(current),
			After:         updated,
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return RuleSummary{}, appErr
		}
		return RuleSummary{}, apperror.Wrap(http.StatusInternalServerError, 500511, "update webhook rule failed", txErr)
	}
	return updated, nil
}

func (s *Service) PauseRule(ctx context.Context, ruleUID string, auditCtx AuditContext) *apperror.Error {
	return s.updateRuleStatus(ctx, ruleUID, "paused", "webhook.rule.pause", auditCtx)
}

func (s *Service) ResumeRule(ctx context.Context, ruleUID string, auditCtx AuditContext) *apperror.Error {
	return s.updateRuleStatus(ctx, ruleUID, "active", "webhook.rule.resume", auditCtx)
}

func (s *Service) DisableRule(ctx context.Context, ruleUID string, auditCtx AuditContext) *apperror.Error {
	return s.updateRuleStatus(ctx, ruleUID, "disabled", "webhook.rule.disable", auditCtx)
}

func (s *Service) updateRuleStatus(ctx context.Context, ruleUID, status, action string, auditCtx AuditContext) *apperror.Error {
	ruleUID = strings.TrimSpace(ruleUID)
	if ruleUID == "" {
		return apperror.New(http.StatusBadRequest, 400504, "webhook rule id is required")
	}
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		rows, err := rules(ctx, tx, workspace.ID, 0, ruleUID)
		if err != nil {
			return err
		}
		if len(rows) == 0 || rows[0].ID == 0 {
			return apperror.New(http.StatusNotFound, 404503, "webhook rule not found")
		}
		current := rows[0]
		if err := tx.WithContext(ctx).Exec(
			"UPDATE webhook_rules SET status = ? WHERE id = ?",
			status, current.ID,
		).Error; err != nil {
			return err
		}
		actor, err := userByUID(ctx, tx, auditCtx.ActorUID)
		if err != nil {
			return err
		}
		updated := current
		updated.Status = status
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   nullID(actor.ID),
			Action:        action,
			ResourceType:  "webhook_rule",
			ResourceID:    nullID(current.ID),
			IP:            auditCtx.IP,
			UserAgent:     auditCtx.UserAgent,
			TraceID:       auditCtx.TraceID,
			RequestMethod: auditCtx.RequestMethod,
			RequestPath:   auditCtx.RequestPath,
			Before:        ruleSummary(current),
			After:         ruleSummary(updated),
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return appErr
		}
		return apperror.Wrap(http.StatusInternalServerError, 500506, "update webhook rule failed", txErr)
	}
	return nil
}

func (s *Service) ListEvents(ctx context.Context, input ListEventsInput) (EventListResult, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return EventListResult{}, appErr
	}
	page, pageSize := normalizeEventPage(input.Page, input.PageSize)
	rows, total, err := eventRows(ctx, s.db, workspace.ID, eventFilter{
		SourceUID:    strings.TrimSpace(input.SourceID),
		Status:       strings.TrimSpace(input.Status),
		DeliveryID:   strings.TrimSpace(input.DeliveryID),
		ReceivedFrom: normalizeTimeFilter(input.ReceivedFrom),
		ReceivedTo:   normalizeTimeFilter(input.ReceivedTo),
		Page:         page,
		PageSize:     pageSize,
	})
	if err != nil {
		return EventListResult{}, apperror.Wrap(http.StatusInternalServerError, 500509, "list webhook events failed", err)
	}
	items := make([]EventSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, eventSummary(row))
	}
	return EventListResult{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

func (s *Service) GetEvent(ctx context.Context, input EventDetailInput) (EventDetail, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return EventDetail{}, appErr
	}
	eventID := strings.TrimSpace(input.EventID)
	if eventID == "" {
		return EventDetail{}, apperror.New(http.StatusBadRequest, 400504, "event id is required")
	}
	row, err := eventByUID(ctx, s.db, workspace.ID, eventID)
	if err != nil {
		return EventDetail{}, apperror.Wrap(http.StatusInternalServerError, 500510, "load webhook event failed", err)
	}
	if row.ID == 0 {
		return EventDetail{}, apperror.New(http.StatusNotFound, 404502, "webhook event not found")
	}
	matches, err := eventMatches(ctx, s.db, row.ID)
	if err != nil {
		return EventDetail{}, apperror.Wrap(http.StatusInternalServerError, 500510, "load webhook event matches failed", err)
	}
	return EventDetail{
		EventSummary: eventSummary(row),
		Headers:      parseHeaderPairs(row.Headers.String),
		Payload:      row.Payload.String,
		Matches:      eventMatchSummaries(matches),
	}, nil
}

func (s *Service) Trigger(ctx context.Context, input TriggerInput) (TriggerResult, *apperror.Error) {
	if strings.TrimSpace(input.Token) == "" {
		return TriggerResult{}, apperror.New(http.StatusUnauthorized, 401501, "webhook token is required")
	}
	tokenHash := hash(input.Token)
	payloadHash := hashBytes(input.Body)
	eventType := strings.TrimSpace(input.EventType)
	if eventType == "" {
		eventType = input.Headers["X-Event-Type"]
	}
	deliveryID := strings.TrimSpace(input.DeliveryID)
	timestampValue := strings.TrimSpace(input.Timestamp)
	nonce := strings.TrimSpace(input.Nonce)
	signature := strings.TrimSpace(input.Signature)
	signatureHeader := strings.TrimSpace(input.SignatureHeader)
	eventUID, err := uid.New()
	if err != nil {
		return TriggerResult{}, apperror.Wrap(http.StatusInternalServerError, 500506, "create webhook event id failed", err)
	}
	var result TriggerResult
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var source sourceRecord
		if err := tx.WithContext(ctx).Raw(
			`SELECT id, uid, workspace_id, name, source_type, signing_secret, timestamp_tolerance_seconds, status
			   FROM webhook_sources
			  WHERE token_hash = ? AND deleted_at IS NULL
			  LIMIT 1
			  FOR UPDATE`,
			tokenHash,
		).Scan(&source).Error; err != nil {
			return err
		}
		if source.ID == 0 || source.Status != "active" {
			return apperror.New(http.StatusUnauthorized, 401502, "invalid webhook token")
		}
		signingSecret := strings.TrimSpace(source.SigningSecret.String)
		legacyMode := signingSecret == ""
		if legacyMode {
			signingSecret = strings.TrimSpace(input.Token)
		}
		if !validSignature(signingSecret, input.Body, signature) {
			if err := insertRejectedEvent(ctx, tx, rejectedEventInput{
				EventUID:        eventUID,
				WorkspaceID:     source.WorkspaceID,
				SourceID:        source.ID,
				EventType:       eventType,
				DeliveryID:      deliveryID,
				SourceTimestamp: nil,
				Nonce:           nonce,
				SignatureHeader: signatureHeader,
				SignatureValid:  false,
				Replayed:        false,
				RemoteIP:        input.RemoteIP,
				Headers:         input.Headers,
				Body:            input.Body,
				PayloadHash:     payloadHash,
				Status:          "rejected",
				ErrorMessage:    "invalid webhook signature",
			}); err != nil {
				return err
			}
			return apperror.New(http.StatusUnauthorized, 401503, "invalid webhook signature")
		}
		var sourceTimestamp *time.Time
		tolerance := time.Duration(source.TimestampToleranceSeconds) * time.Second
		if tolerance <= 0 {
			tolerance = 5 * time.Minute
		}
		if !legacyMode || timestampValue != "" {
			parsedTimestamp, err := parseWebhookTimestamp(timestampValue)
			if err != nil {
				if insertErr := insertRejectedEvent(ctx, tx, rejectedEventInput{
					EventUID:        eventUID,
					WorkspaceID:     source.WorkspaceID,
					SourceID:        source.ID,
					EventType:       eventType,
					DeliveryID:      deliveryID,
					SourceTimestamp: nil,
					Nonce:           nonce,
					SignatureHeader: signatureHeader,
					SignatureValid:  true,
					Replayed:        false,
					RemoteIP:        input.RemoteIP,
					Headers:         input.Headers,
					Body:            input.Body,
					PayloadHash:     payloadHash,
					Status:          "rejected",
					ErrorMessage:    "invalid webhook timestamp",
				}); insertErr != nil {
					return insertErr
				}
				return apperror.New(http.StatusUnauthorized, 401504, "invalid webhook timestamp")
			}
			now := time.Now()
			if parsedTimestamp.Before(now.Add(-tolerance)) || parsedTimestamp.After(now.Add(tolerance)) {
				if insertErr := insertRejectedEvent(ctx, tx, rejectedEventInput{
					EventUID:        eventUID,
					WorkspaceID:     source.WorkspaceID,
					SourceID:        source.ID,
					EventType:       eventType,
					DeliveryID:      deliveryID,
					SourceTimestamp: &parsedTimestamp,
					Nonce:           nonce,
					SignatureHeader: signatureHeader,
					SignatureValid:  true,
					Replayed:        false,
					RemoteIP:        input.RemoteIP,
					Headers:         input.Headers,
					Body:            input.Body,
					PayloadHash:     payloadHash,
					Status:          "rejected",
					ErrorMessage:    "webhook timestamp outside allowed tolerance",
				}); insertErr != nil {
					return insertErr
				}
				return apperror.New(http.StatusUnauthorized, 401504, "invalid webhook timestamp")
			}
			sourceTimestamp = &parsedTimestamp
		}
		if !legacyMode || nonce != "" {
			if nonce == "" {
				if err := insertRejectedEvent(ctx, tx, rejectedEventInput{
					EventUID:        eventUID,
					WorkspaceID:     source.WorkspaceID,
					SourceID:        source.ID,
					EventType:       eventType,
					DeliveryID:      deliveryID,
					SourceTimestamp: sourceTimestamp,
					Nonce:           "",
					SignatureHeader: signatureHeader,
					SignatureValid:  true,
					Replayed:        false,
					RemoteIP:        input.RemoteIP,
					Headers:         input.Headers,
					Body:            input.Body,
					PayloadHash:     payloadHash,
					Status:          "rejected",
					ErrorMessage:    "webhook nonce is required",
				}); err != nil {
					return err
				}
				return apperror.New(http.StatusUnauthorized, 401505, "webhook nonce is required")
			}
			var duplicateNonceID uint64
			if err := tx.WithContext(ctx).Raw(
				`SELECT id
				   FROM webhook_events
				  WHERE source_id = ?
				    AND nonce = ?
				    AND received_at > DATE_SUB(NOW(3), INTERVAL ? SECOND)
				  LIMIT 1`,
				source.ID, nonce, int(tolerance.Seconds()),
			).Scan(&duplicateNonceID).Error; err != nil {
				return err
			}
			if duplicateNonceID != 0 {
				if err := insertRejectedEvent(ctx, tx, rejectedEventInput{
					EventUID:        eventUID,
					WorkspaceID:     source.WorkspaceID,
					SourceID:        source.ID,
					EventType:       eventType,
					DeliveryID:      deliveryID,
					SourceTimestamp: sourceTimestamp,
					Nonce:           nonce,
					SignatureHeader: signatureHeader,
					SignatureValid:  true,
					Replayed:        true,
					RemoteIP:        input.RemoteIP,
					Headers:         input.Headers,
					Body:            input.Body,
					PayloadHash:     payloadHash,
					Status:          "rejected",
					ErrorMessage:    "webhook nonce replay detected",
				}); err != nil {
					return err
				}
				return apperror.New(http.StatusConflict, 409503, "webhook nonce replay detected")
			}
		}
		if deliveryID != "" {
			var duplicateID uint64
			if err := tx.WithContext(ctx).Raw(
				"SELECT id FROM webhook_events WHERE source_id = ? AND delivery_id = ? LIMIT 1",
				source.ID, deliveryID,
			).Scan(&duplicateID).Error; err != nil {
				return err
			}
			if duplicateID != 0 {
				return apperror.New(http.StatusConflict, 409502, "webhook delivery replay detected")
			}
		}
		var recentCount int
		if err := tx.WithContext(ctx).Raw(
			"SELECT COUNT(*) FROM webhook_events WHERE source_id = ? AND received_at > DATE_SUB(NOW(3), INTERVAL 1 MINUTE)",
			source.ID,
		).Scan(&recentCount).Error; err != nil {
			return err
		}
		if recentCount >= 120 {
			return apperror.New(http.StatusTooManyRequests, 429501, "webhook rate limit exceeded")
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO webhook_events(uid, workspace_id, source_id, event_type, delivery_id, source_timestamp, nonce, signature_header,
			                            signature_valid, replayed, remote_ip, headers, payload, payload_hash, status)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1, 0, ?, ?, ?, ?, 'received')`,
			eventUID, source.WorkspaceID, source.ID, nullString(eventType), nullString(deliveryID), nullTime(sourceTimestamp), nullString(nonce), nullString(signatureHeader),
			nullString(input.RemoteIP), jsonNull(input.Headers), jsonRaw(input.Body), payloadHash,
		).Error; err != nil {
			return err
		}
		var eventID uint64
		if err := tx.WithContext(ctx).Raw("SELECT id FROM webhook_events WHERE uid = ? LIMIT 1", eventUID).Scan(&eventID).Error; err != nil {
			return err
		}
		rows, err := rules(ctx, tx, source.WorkspaceID, source.ID, "")
		if err != nil {
			return err
		}
		payload := parsePayloadObject(input.Body)
		triggeredRuns := []string{}
		matched := 0
		for _, rule := range rows {
			if rule.Status != "active" {
				continue
			}
			if rule.EventType.Valid && eventType != "" && rule.EventType.String != eventType {
				_ = tx.WithContext(ctx).Exec(
					"INSERT INTO webhook_event_matches(event_id, rule_id, matched, reason) VALUES (?, ?, 0, 'event_type_mismatch')",
					eventID, rule.ID,
				).Error
				continue
			}
			ruleMatcher, err := parseMatcher(rule.Matcher.String)
			if err != nil {
				_ = tx.WithContext(ctx).Exec(
					"INSERT INTO webhook_event_matches(event_id, rule_id, matched, reason) VALUES (?, ?, 0, 'matcher_invalid')",
					eventID, rule.ID,
				).Error
				continue
			}
			if ok, reason := matchRule(ruleMatcher, eventType, input.Headers, payload); !ok {
				_ = tx.WithContext(ctx).Exec(
					"INSERT INTO webhook_event_matches(event_id, rule_id, matched, reason) VALUES (?, ?, 0, ?)",
					eventID, rule.ID, limitString(reason, 1024),
				).Error
				continue
			}
			matched++
			if normalizeWebhookTargetType(rule.TargetType) == "workflow" {
				runID, runUID, err := workflows.CreateTriggeredRunTx(
					ctx,
					tx,
					source.WorkspaceID,
					rule.WorkflowID,
					"webhook",
					"",
					"",
					rule.CreatedByID,
					map[string]interface{}{"webhookEventId": eventUID, "webhookRuleId": rule.UID},
				)
				if err != nil {
					_ = tx.WithContext(ctx).Exec(
						"INSERT INTO webhook_event_matches(event_id, rule_id, matched, reason) VALUES (?, ?, 0, ?)",
						eventID, rule.ID, limitString(err.Error(), 1024),
					).Error
					continue
				}
				triggeredRuns = append(triggeredRuns, runUID)
				_ = tx.WithContext(ctx).Exec(
					"INSERT INTO webhook_event_matches(event_id, rule_id, workflow_run_id, matched, reason) VALUES (?, ?, ?, 1, 'triggered')",
					eventID, rule.ID, runID,
				).Error
				continue
			}
			runID, runUID, err := execution.CreateRunFromTask(ctx, tx, source.WorkspaceID, rule.TaskID, "webhook", nullID(eventID), rule.CreatedByID)
			if err != nil {
				_ = tx.WithContext(ctx).Exec(
					"INSERT INTO webhook_event_matches(event_id, rule_id, matched, reason) VALUES (?, ?, 0, ?)",
					eventID, rule.ID, limitString(err.Error(), 1024),
				).Error
				continue
			}
			triggeredRuns = append(triggeredRuns, runUID)
			_ = tx.WithContext(ctx).Exec(
				"INSERT INTO webhook_event_matches(event_id, rule_id, task_run_id, matched, reason) VALUES (?, ?, ?, 1, 'triggered')",
				eventID, rule.ID, runID,
			).Error
		}
		status := "ignored"
		if len(triggeredRuns) > 0 {
			status = "triggered"
		} else if matched > 0 {
			status = "failed"
		}
		if err := tx.WithContext(ctx).Exec(
			"UPDATE webhook_events SET status = ? WHERE id = ?",
			status, eventID,
		).Error; err != nil {
			return err
		}
		_ = tx.WithContext(ctx).Exec("UPDATE webhook_sources SET last_received_at = NOW(3) WHERE id = ?", source.ID).Error
		result = TriggerResult{EventID: eventUID, Status: status, MatchedRules: matched, TriggeredRuns: triggeredRuns}
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return TriggerResult{}, appErr
		}
		return TriggerResult{}, apperror.Wrap(http.StatusInternalServerError, 500507, "trigger webhook failed", txErr)
	}
	return result, nil
}

func (s *Service) workspace(ctx context.Context) (workspaceRecord, *apperror.Error) {
	workspace, err := defaultWorkspace(ctx, s.db, s.cfg.Bootstrap.WorkspaceSlug)
	if err != nil {
		return workspaceRecord{}, apperror.Wrap(http.StatusInternalServerError, 500508, "load workspace failed", err)
	}
	if workspace.ID == 0 {
		return workspaceRecord{}, apperror.New(http.StatusInternalServerError, 500508, "default workspace is not initialized")
	}
	return workspace, nil
}

func defaultWorkspace(ctx context.Context, db *gorm.DB, slug string) (workspaceRecord, error) {
	var workspace workspaceRecord
	err := db.WithContext(ctx).Raw("SELECT id FROM workspaces WHERE slug = ? AND status = 'active' LIMIT 1", slug).Scan(&workspace).Error
	return workspace, err
}

func userByUID(ctx context.Context, db *gorm.DB, uid string) (userRecord, error) {
	var user userRecord
	err := db.WithContext(ctx).Raw("SELECT id FROM users WHERE uid = ? AND status = 'active' LIMIT 1", uid).Scan(&user).Error
	return user, err
}

func taskByUID(ctx context.Context, db *gorm.DB, workspaceID uint64, uid string) (taskRecord, error) {
	var task taskRecord
	err := db.WithContext(ctx).Raw(
		"SELECT id FROM tasks WHERE workspace_id = ? AND uid = ? AND status = 'active' AND deleted_at IS NULL LIMIT 1",
		workspaceID, uid,
	).Scan(&task).Error
	return task, err
}

func workflowByUID(ctx context.Context, db *gorm.DB, workspaceID uint64, uid string) (workflowRecord, error) {
	var workflow workflowRecord
	err := db.WithContext(ctx).Raw(
		`SELECT id, uid, name, status
		   FROM workflow_definitions
		  WHERE workspace_id = ? AND uid = ? AND deleted_at IS NULL
		  LIMIT 1`,
		workspaceID, uid,
	).Scan(&workflow).Error
	return workflow, err
}

func sourceByUID(ctx context.Context, db *gorm.DB, workspaceID uint64, uid string) (sourceRecord, error) {
	var source sourceRecord
	err := db.WithContext(ctx).Raw(
		`SELECT ws.id, ws.uid, ws.workspace_id, ws.name, ws.source_type, ws.signing_secret, ws.timestamp_tolerance_seconds, ws.status,
		        DATE_FORMAT(ws.last_received_at, '%Y-%m-%d %H:%i:%s') AS last_received_at,
		        u.username AS created_by, ws.created_by AS created_by_id,
		        DATE_FORMAT(ws.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM webhook_sources ws
		   LEFT JOIN users u ON u.id = ws.created_by
		  WHERE ws.workspace_id = ? AND ws.uid = ? AND ws.deleted_at IS NULL
		  LIMIT 1`,
		workspaceID, uid,
	).Scan(&source).Error
	return source, err
}

func rules(ctx context.Context, db *gorm.DB, workspaceID, sourceID uint64, ruleUID string) ([]ruleRecord, error) {
	args := []interface{}{workspaceID}
	where := "WHERE wr.workspace_id = ? AND wr.deleted_at IS NULL"
	if sourceID != 0 {
		where += " AND wr.source_id = ?"
		args = append(args, sourceID)
	}
	if ruleUID != "" {
		where += " AND wr.uid = ?"
		args = append(args, ruleUID)
	}
	var rows []ruleRecord
	err := db.WithContext(ctx).Raw(
		`SELECT wr.id, wr.uid, wr.workspace_id, wr.source_id, ws.uid AS source_uid, ws.name AS source_name,
		        wr.target_type, wr.task_id, t.uid AS task_uid, t.name AS task_name, wr.workflow_id, wd.uid AS workflow_uid, wd.name AS workflow_name,
		        wr.name, wr.event_type, CAST(wr.matcher AS CHAR) AS matcher, wr.status,
		        u.username AS created_by, wr.created_by AS created_by_id,
		        DATE_FORMAT(wr.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM webhook_rules wr
		   JOIN webhook_sources ws ON ws.id = wr.source_id
		   LEFT JOIN tasks t ON t.id = wr.task_id
		   LEFT JOIN workflow_definitions wd ON wd.id = wr.workflow_id
		   LEFT JOIN users u ON u.id = wr.created_by
		  `+where+`
		  ORDER BY wr.created_at DESC`,
		args...,
	).Scan(&rows).Error
	return rows, err
}

type eventFilter struct {
	SourceUID    string
	Status       string
	DeliveryID   string
	ReceivedFrom string
	ReceivedTo   string
	Page         int
	PageSize     int
}

func eventRows(ctx context.Context, db *gorm.DB, workspaceID uint64, filter eventFilter) ([]eventRecord, int64, error) {
	args := []interface{}{workspaceID}
	where := "WHERE we.workspace_id = ?"
	if filter.SourceUID != "" {
		where += " AND ws.uid = ?"
		args = append(args, filter.SourceUID)
	}
	if filter.Status != "" {
		where += " AND we.status = ?"
		args = append(args, filter.Status)
	}
	if filter.DeliveryID != "" {
		where += " AND we.delivery_id LIKE ?"
		args = append(args, "%"+filter.DeliveryID+"%")
	}
	if filter.ReceivedFrom != "" {
		where += " AND we.received_at >= ?"
		args = append(args, filter.ReceivedFrom)
	}
	if filter.ReceivedTo != "" {
		where += " AND we.received_at <= ?"
		args = append(args, filter.ReceivedTo)
	}

	var total int64
	if err := db.WithContext(ctx).Raw(
		`SELECT COUNT(*)
		   FROM webhook_events we
		   LEFT JOIN webhook_sources ws ON ws.id = we.source_id
		  `+where,
		args...,
	).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	page, pageSize := normalizeEventPage(filter.Page, filter.PageSize)
	offset := (page - 1) * pageSize
	queryArgs := append(append([]interface{}{}, args...), pageSize, offset)
	var rows []eventRecord
	err := db.WithContext(ctx).Raw(
		`SELECT we.id, we.uid, ws.uid AS source_uid, ws.name AS source_name, we.event_type, we.delivery_id,
		        DATE_FORMAT(we.source_timestamp, '%Y-%m-%d %H:%i:%s') AS source_timestamp,
		        we.nonce, we.signature_header, we.signature_valid, we.replayed, we.status, we.error_message,
		        we.remote_ip, we.payload_hash, CAST(we.headers AS CHAR) AS headers,
		        DATE_FORMAT(we.received_at, '%Y-%m-%d %H:%i:%s') AS received_at
		   FROM webhook_events we
		   LEFT JOIN webhook_sources ws ON ws.id = we.source_id
		  `+where+`
		  ORDER BY we.received_at DESC, we.id DESC
		  LIMIT ? OFFSET ?`,
		queryArgs...,
	).Scan(&rows).Error
	return rows, total, err
}

func eventByUID(ctx context.Context, db *gorm.DB, workspaceID uint64, eventUID string) (eventRecord, error) {
	var row eventRecord
	err := db.WithContext(ctx).Raw(
		`SELECT we.id, we.uid, ws.uid AS source_uid, ws.name AS source_name, we.event_type, we.delivery_id,
		        DATE_FORMAT(we.source_timestamp, '%Y-%m-%d %H:%i:%s') AS source_timestamp,
		        we.nonce, we.signature_header, we.signature_valid, we.replayed, we.status, we.error_message,
		        we.remote_ip, we.payload_hash, CAST(we.headers AS CHAR) AS headers, CAST(we.payload AS CHAR) AS payload,
		        DATE_FORMAT(we.received_at, '%Y-%m-%d %H:%i:%s') AS received_at
		   FROM webhook_events we
		   LEFT JOIN webhook_sources ws ON ws.id = we.source_id
		  WHERE we.workspace_id = ? AND we.uid = ?
		  LIMIT 1`,
		workspaceID, eventUID,
	).Scan(&row).Error
	return row, err
}

func eventMatches(ctx context.Context, db *gorm.DB, eventID uint64) ([]eventMatchRecord, error) {
	var rows []eventMatchRecord
	err := db.WithContext(ctx).Raw(
		`SELECT wem.id, wr.uid AS rule_uid, wr.name AS rule_name, wem.matched, wem.reason,
		        tr.uid AS task_run_uid, wfr.uid AS workflow_run_uid,
		        DATE_FORMAT(wem.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM webhook_event_matches wem
		   LEFT JOIN webhook_rules wr ON wr.id = wem.rule_id
		   LEFT JOIN task_runs tr ON tr.id = wem.task_run_id
		   LEFT JOIN workflow_runs wfr ON wfr.id = wem.workflow_run_id
		  WHERE wem.event_id = ?
		  ORDER BY wem.created_at ASC, wem.id ASC`,
		eventID,
	).Scan(&rows).Error
	return rows, err
}

func sourceSummary(row sourceRecord) SourceSummary {
	return SourceSummary{
		ID:             row.UID,
		Name:           row.Name,
		SourceType:     row.SourceType,
		Status:         row.Status,
		LastReceivedAt: row.LastReceivedAt.String,
		CreatedBy:      row.CreatedBy.String,
		CreatedAt:      row.CreatedAt,
	}
}

func ruleSummary(row ruleRecord) RuleSummary {
	matcher, _ := parseMatcher(row.Matcher.String)
	return RuleSummary{
		ID:           row.UID,
		SourceID:     row.SourceUID,
		SourceName:   row.SourceName,
		TargetType:   normalizeWebhookTargetType(row.TargetType),
		TaskID:       row.TaskUID,
		TaskName:     row.TaskName,
		WorkflowID:   row.WorkflowUID,
		WorkflowName: row.WorkflowName,
		Name:         row.Name,
		EventType:    row.EventType.String,
		Matcher:      matcher,
		Status:       row.Status,
		CreatedBy:    row.CreatedBy.String,
		CreatedAt:    row.CreatedAt,
	}
}

func eventSummary(row eventRecord) EventSummary {
	return EventSummary{
		ID:              row.UID,
		SourceID:        row.SourceUID.String,
		SourceName:      row.SourceName.String,
		EventType:       row.EventType.String,
		DeliveryID:      row.DeliveryID.String,
		SourceTimestamp: row.SourceTimestamp.String,
		Nonce:           row.Nonce.String,
		SignatureHeader: row.SignatureHeader.String,
		SignatureValid:  row.SignatureValid,
		Replayed:        row.Replayed,
		Status:          row.Status,
		ErrorMessage:    row.ErrorMessage.String,
		RemoteIP:        row.RemoteIP.String,
		PayloadHash:     row.PayloadHash.String,
		ReceivedAt:      row.ReceivedAt,
	}
}

func eventMatchSummaries(rows []eventMatchRecord) []EventMatchSummary {
	out := make([]EventMatchSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, EventMatchSummary{
			ID:            row.ID,
			RuleID:        row.RuleUID.String,
			RuleName:      row.RuleName.String,
			Matched:       row.Matched,
			Reason:        row.Reason.String,
			TaskRunID:     row.TaskRunUID.String,
			WorkflowRunID: row.WorkflowRunUID.String,
			CreatedAt:     row.CreatedAt,
		})
	}
	return out
}

func normalizeWebhookTargetType(value string) string {
	if strings.TrimSpace(value) == "workflow" {
		return "workflow"
	}
	return "task"
}

func validSignature(secret string, body []byte, signature string) bool {
	signature = strings.TrimSpace(signature)
	if signature == "" {
		return false
	}
	signature = strings.TrimPrefix(signature, "sha256=")
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(secret)))
	_, _ = mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(strings.ToLower(signature)))
}

func newSecret() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func hash(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])
}

func hashBytes(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func jsonNull(value interface{}) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	bytes, err := json.Marshal(value)
	if err != nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(bytes), Valid: true}
}

func jsonRaw(value []byte) sql.NullString {
	if len(value) == 0 {
		return sql.NullString{}
	}
	if !json.Valid(value) {
		bytes, err := json.Marshal(string(value))
		if err != nil {
			return sql.NullString{}
		}
		return sql.NullString{String: string(bytes), Valid: true}
	}
	return sql.NullString{String: string(value), Valid: true}
}

func nullTime(value *time.Time) sql.NullTime {
	if value == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *value, Valid: true}
}

func nullID(id uint64) sql.NullInt64 {
	if id == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(id), Valid: true}
}

func nullString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	return sql.NullString{String: value, Valid: value != ""}
}

func validSourceType(value string) bool {
	switch value {
	case "github", "gitee", "gitlab", "custom":
		return true
	default:
		return false
	}
}

func limitString(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}

func normalizeEventPage(page, pageSize int) (int, int) {
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

func normalizeTimeFilter(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, "T", " ")
	return value
}

func parseHeaderPairs(value string) []HeaderPair {
	value = strings.TrimSpace(value)
	if value == "" {
		return []HeaderPair{}
	}
	headers := map[string]string{}
	if err := json.Unmarshal([]byte(value), &headers); err != nil {
		return []HeaderPair{}
	}
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	out := make([]HeaderPair, 0, len(keys))
	for _, key := range keys {
		out = append(out, HeaderPair{Key: key, Value: headers[key]})
	}
	return out
}

func normalizeMatcher(input *Matcher) (*Matcher, *apperror.Error) {
	if input == nil || len(input.Conditions) == 0 {
		return nil, nil
	}
	conditions := make([]MatcherCondition, 0, len(input.Conditions))
	for _, item := range input.Conditions {
		condition := MatcherCondition{
			Type:  strings.TrimSpace(item.Type),
			Key:   strings.TrimSpace(item.Key),
			Path:  strings.TrimSpace(item.Path),
			Value: strings.TrimSpace(item.Value),
		}
		if condition.Type == "" || condition.Value == "" {
			return nil, apperror.New(http.StatusBadRequest, 400505, "matcher type and value are required")
		}
		switch condition.Type {
		case "header_equals":
			if condition.Key == "" {
				return nil, apperror.New(http.StatusBadRequest, 400505, "matcher key is required")
			}
			condition.Key = http.CanonicalHeaderKey(condition.Key)
			condition.Path = ""
		case "payload_equals", "payload_contains":
			if condition.Path == "" {
				return nil, apperror.New(http.StatusBadRequest, 400505, "matcher path is required")
			}
			condition.Key = ""
		case "event_type_equals", "ref_equals", "branch_equals":
			condition.Key = ""
			condition.Path = ""
		default:
			return nil, apperror.New(http.StatusBadRequest, 400505, "unsupported matcher type")
		}
		conditions = append(conditions, condition)
	}
	return &Matcher{Conditions: conditions}, nil
}

func matcherSQL(matcher *Matcher) sql.NullString {
	if matcher == nil || len(matcher.Conditions) == 0 {
		return sql.NullString{}
	}
	bytes, err := json.Marshal(matcher)
	if err != nil {
		return sql.NullString{}
	}
	return sql.NullString{String: string(bytes), Valid: true}
}

func parseMatcher(value string) (*Matcher, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	var matcher Matcher
	if err := json.Unmarshal([]byte(value), &matcher); err != nil {
		return nil, err
	}
	if len(matcher.Conditions) == 0 {
		return nil, nil
	}
	for index := range matcher.Conditions {
		matcher.Conditions[index].Type = strings.TrimSpace(matcher.Conditions[index].Type)
		matcher.Conditions[index].Key = strings.TrimSpace(matcher.Conditions[index].Key)
		matcher.Conditions[index].Path = strings.TrimSpace(matcher.Conditions[index].Path)
		matcher.Conditions[index].Value = strings.TrimSpace(matcher.Conditions[index].Value)
	}
	return &matcher, nil
}

func parsePayloadObject(body []byte) interface{} {
	if len(body) == 0 || !json.Valid(body) {
		return nil
	}
	var payload interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	return payload
}

func matchRule(matcher *Matcher, eventType string, headers map[string]string, payload interface{}) (bool, string) {
	if matcher == nil || len(matcher.Conditions) == 0 {
		return true, ""
	}
	for _, condition := range matcher.Conditions {
		switch condition.Type {
		case "event_type_equals":
			if strings.TrimSpace(eventType) != condition.Value {
				return false, "event_type_condition_mismatch"
			}
		case "header_equals":
			headerValue := firstHeaderValue(headers, condition.Key)
			if headerValue != condition.Value {
				return false, "header_mismatch:" + condition.Key
			}
		case "payload_equals":
			payloadValue, ok := payloadValueAtPath(payload, condition.Path)
			if !ok || payloadValue != condition.Value {
				return false, "payload_mismatch:" + condition.Path
			}
		case "payload_contains":
			payloadValue, ok := payloadValueAtPath(payload, condition.Path)
			if !ok || !strings.Contains(payloadValue, condition.Value) {
				return false, "payload_contains_mismatch:" + condition.Path
			}
		case "ref_equals":
			refValue, ok := payloadValueAtPath(payload, "ref")
			if !ok || refValue != condition.Value {
				return false, "ref_mismatch"
			}
		case "branch_equals":
			refValue, ok := payloadValueAtPath(payload, "ref")
			if !ok || branchFromRef(refValue) != condition.Value {
				return false, "branch_mismatch"
			}
		default:
			return false, "matcher_invalid"
		}
	}
	return true, ""
}

func branchFromRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if strings.HasPrefix(ref, "refs/heads/") {
		return strings.TrimPrefix(ref, "refs/heads/")
	}
	return ref
}

func firstHeaderValue(headers map[string]string, key string) string {
	if headers == nil {
		return ""
	}
	canonical := http.CanonicalHeaderKey(strings.TrimSpace(key))
	for headerKey, value := range headers {
		if http.CanonicalHeaderKey(headerKey) == canonical {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func payloadValueAtPath(payload interface{}, path string) (string, bool) {
	current := payload
	for _, segment := range strings.Split(strings.TrimSpace(path), ".") {
		if segment == "" {
			return "", false
		}
		node, ok := current.(map[string]interface{})
		if !ok {
			return "", false
		}
		current, ok = node[segment]
		if !ok {
			return "", false
		}
	}
	switch value := current.(type) {
	case string:
		return value, true
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64), true
	case bool:
		return strconv.FormatBool(value), true
	default:
		bytes, err := json.Marshal(value)
		if err != nil {
			return "", false
		}
		return string(bytes), true
	}
}

type rejectedEventInput struct {
	EventUID        string
	WorkspaceID     uint64
	SourceID        uint64
	EventType       string
	DeliveryID      string
	SourceTimestamp *time.Time
	Nonce           string
	SignatureHeader string
	SignatureValid  bool
	Replayed        bool
	RemoteIP        string
	Headers         map[string]string
	Body            []byte
	PayloadHash     string
	Status          string
	ErrorMessage    string
}

func insertRejectedEvent(ctx context.Context, tx *gorm.DB, input rejectedEventInput) error {
	insert := func(deliveryID string) error {
		return tx.WithContext(ctx).Exec(
			`INSERT INTO webhook_events(uid, workspace_id, source_id, event_type, delivery_id, source_timestamp, nonce, signature_header,
			                            signature_valid, replayed, remote_ip, headers, payload, payload_hash, status, error_message)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			input.EventUID,
			input.WorkspaceID,
			input.SourceID,
			nullString(input.EventType),
			nullString(deliveryID),
			nullTime(input.SourceTimestamp),
			nullString(input.Nonce),
			nullString(input.SignatureHeader),
			input.SignatureValid,
			input.Replayed,
			nullString(input.RemoteIP),
			jsonNull(input.Headers),
			jsonRaw(input.Body),
			input.PayloadHash,
			input.Status,
			nullString(limitString(input.ErrorMessage, 1024)),
		).Error
	}
	err := insert(input.DeliveryID)
	if err == nil {
		return nil
	}
	if input.DeliveryID != "" && isDuplicateWebhookDeliveryError(err) {
		return insert("")
	}
	return err
}

func isDuplicateWebhookDeliveryError(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "uk_webhook_events_delivery")
}

func parseWebhookTimestamp(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, errors.New("timestamp is required")
	}
	if unix, err := strconv.ParseInt(value, 10, 64); err == nil {
		switch len(value) {
		case 13:
			return time.UnixMilli(unix).UTC(), nil
		case 10:
			return time.Unix(unix, 0).UTC(), nil
		}
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed.UTC(), nil
	}
	return time.Time{}, errors.New("unsupported timestamp format")
}
