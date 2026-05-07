package webhooks

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"opspilot/server/internal/config"
	"opspilot/server/internal/modules/execution"
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
	SourceID  string
	TaskID    string
	Name      string
	EventType string
	Audit     AuditContext
}

type TriggerInput struct {
	Token      string
	EventType  string
	DeliveryID string
	RemoteIP   string
	Headers    map[string]string
	Body       []byte
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
	Token string `json:"token,omitempty"`
}

type RuleSummary struct {
	ID         string `json:"id"`
	SourceID   string `json:"sourceId"`
	SourceName string `json:"sourceName"`
	TaskID     string `json:"taskId"`
	TaskName   string `json:"taskName"`
	Name       string `json:"name"`
	EventType  string `json:"eventType,omitempty"`
	Status     string `json:"status"`
	CreatedBy  string `json:"createdBy,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

type TriggerResult struct {
	EventID        string   `json:"eventId"`
	Status         string   `json:"status"`
	MatchedRules   int      `json:"matchedRules"`
	TriggeredRuns  []string `json:"triggeredRuns"`
	RejectedReason string   `json:"rejectedReason,omitempty"`
}

type sourceRecord struct {
	ID             uint64
	UID            string
	WorkspaceID    uint64
	Name           string
	SourceType     string
	Status         string
	LastReceivedAt sql.NullString
	CreatedBy      sql.NullString
	CreatedByID    sql.NullInt64
	CreatedAt      string
}

type ruleRecord struct {
	ID          uint64
	UID         string
	WorkspaceID uint64
	SourceID    uint64
	SourceUID   string
	SourceName  string
	TaskID      uint64
	TaskUID     string
	TaskName    string
	Name        string
	EventType   sql.NullString
	Status      string
	CreatedBy   sql.NullString
	CreatedByID sql.NullInt64
	CreatedAt   string
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
	token, tokenHash, err := newSecret()
	if err != nil {
		return SourceDetail{}, apperror.Wrap(http.StatusInternalServerError, 500502, "issue webhook token failed", err)
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
			`INSERT INTO webhook_sources(uid, workspace_id, name, source_type, token_hash, status, created_by)
			 VALUES (?, ?, ?, ?, ?, 'active', ?)`,
			sourceUID, workspace.ID, name, sourceType, tokenHash, actorID,
		).Error; err != nil {
			return err
		}
		row, err := sourceByUID(ctx, tx, workspace.ID, sourceUID)
		if err != nil {
			return err
		}
		created = SourceDetail{SourceSummary: sourceSummary(row), Token: token}
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
	taskUID := strings.TrimSpace(input.TaskID)
	eventType := strings.TrimSpace(input.EventType)
	if name == "" || sourceUID == "" || taskUID == "" {
		return RuleSummary{}, apperror.New(http.StatusBadRequest, 400503, "sourceId, taskId and name are required")
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
		task, err := taskByUID(ctx, tx, workspace.ID, taskUID)
		if err != nil {
			return err
		}
		if task.ID == 0 {
			return apperror.New(http.StatusNotFound, 404401, "task definition not found")
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
			`INSERT INTO webhook_rules(uid, workspace_id, source_id, task_id, name, event_type, status, created_by)
			 VALUES (?, ?, ?, ?, ?, ?, 'active', ?)`,
			ruleUID, workspace.ID, source.ID, task.ID, name, nullString(eventType), nullID(actor.ID),
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

func (s *Service) Trigger(ctx context.Context, input TriggerInput) (TriggerResult, *apperror.Error) {
	tokenHash := hash(input.Token)
	if strings.TrimSpace(input.Token) == "" {
		return TriggerResult{}, apperror.New(http.StatusUnauthorized, 401501, "webhook token is required")
	}
	payloadHash := hashBytes(input.Body)
	eventType := strings.TrimSpace(input.EventType)
	if eventType == "" {
		eventType = input.Headers["X-Event-Type"]
	}
	eventUID, err := uid.New()
	if err != nil {
		return TriggerResult{}, apperror.Wrap(http.StatusInternalServerError, 500506, "create webhook event id failed", err)
	}
	var result TriggerResult
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var source sourceRecord
		if err := tx.WithContext(ctx).Raw(
			`SELECT id, uid, workspace_id, name, source_type, status
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
		if input.DeliveryID != "" {
			var duplicateID uint64
			if err := tx.WithContext(ctx).Raw(
				"SELECT id FROM webhook_events WHERE source_id = ? AND delivery_id = ? LIMIT 1",
				source.ID, input.DeliveryID,
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
			`INSERT INTO webhook_events(uid, workspace_id, source_id, event_type, delivery_id,
			                            signature_valid, replayed, remote_ip, headers, payload, payload_hash, status)
			 VALUES (?, ?, ?, ?, ?, 1, 0, ?, ?, ?, ?, 'received')`,
			eventUID, source.WorkspaceID, source.ID, nullString(eventType), nullString(input.DeliveryID),
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
			matched++
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

func sourceByUID(ctx context.Context, db *gorm.DB, workspaceID uint64, uid string) (sourceRecord, error) {
	var source sourceRecord
	err := db.WithContext(ctx).Raw(
		`SELECT ws.id, ws.uid, ws.workspace_id, ws.name, ws.source_type, ws.status,
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
		        wr.task_id, t.uid AS task_uid, t.name AS task_name, wr.name, wr.event_type, wr.status,
		        u.username AS created_by, wr.created_by AS created_by_id,
		        DATE_FORMAT(wr.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM webhook_rules wr
		   JOIN webhook_sources ws ON ws.id = wr.source_id
		   JOIN tasks t ON t.id = wr.task_id
		   LEFT JOIN users u ON u.id = wr.created_by
		  `+where+`
		  ORDER BY wr.created_at DESC`,
		args...,
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
	return RuleSummary{
		ID:         row.UID,
		SourceID:   row.SourceUID,
		SourceName: row.SourceName,
		TaskID:     row.TaskUID,
		TaskName:   row.TaskName,
		Name:       row.Name,
		EventType:  row.EventType.String,
		Status:     row.Status,
		CreatedBy:  row.CreatedBy.String,
		CreatedAt:  row.CreatedAt,
	}
}

func newSecret() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	return token, hash(token), nil
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
