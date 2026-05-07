package alerts

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"opspilot/server/internal/config"
	"opspilot/server/internal/modules/notifications"
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

type CreateRuleInput struct {
	Name            string
	MetricCode      string
	Operator        string
	Threshold       float64
	DurationSeconds uint
	CooldownSeconds uint
	Severity        string
	Audit           AuditContext
}

type UpdateRuleInput struct {
	Name            string
	MetricCode      string
	Operator        string
	Threshold       float64
	DurationSeconds uint
	CooldownSeconds uint
	Severity        string
	Audit           AuditContext
}

type AlertRuleSummary struct {
	ID              string  `json:"id"`
	Name            string  `json:"name"`
	RuleType        string  `json:"ruleType"`
	MetricCode      string  `json:"metricCode,omitempty"`
	Operator        string  `json:"operator,omitempty"`
	Threshold       float64 `json:"threshold,omitempty"`
	DurationSeconds uint    `json:"durationSeconds"`
	CooldownSeconds uint    `json:"cooldownSeconds"`
	Severity        string  `json:"severity"`
	Status          string  `json:"status"`
	CreatedBy       string  `json:"createdBy,omitempty"`
	CreatedAt       string  `json:"createdAt"`
}

type AlertSummary struct {
	ID             string `json:"id"`
	RuleID         string `json:"ruleId,omitempty"`
	RuleName       string `json:"ruleName,omitempty"`
	ResourceType   string `json:"resourceType"`
	ResourceID     uint64 `json:"resourceId,omitempty"`
	HostID         string `json:"hostId,omitempty"`
	HostName       string `json:"hostName,omitempty"`
	Title          string `json:"title"`
	Message        string `json:"message,omitempty"`
	Severity       string `json:"severity"`
	Status         string `json:"status"`
	AcknowledgedAt string `json:"acknowledgedAt,omitempty"`
	AcknowledgedBy string `json:"acknowledgedBy,omitempty"`
	SilencedUntil  string `json:"silencedUntil,omitempty"`
	SilenceReason  string `json:"silenceReason,omitempty"`
	CooldownUntil  string `json:"cooldownUntil,omitempty"`
	FirstSeenAt    string `json:"firstSeenAt"`
	LastSeenAt     string `json:"lastSeenAt"`
	ResolvedAt     string `json:"resolvedAt,omitempty"`
	CreatedAt      string `json:"createdAt"`
}

type SilenceInput struct {
	DurationSeconds uint
	Reason          string
	Audit           AuditContext
}

type EvaluationResult struct {
	Fired    int `json:"fired"`
	Resolved int `json:"resolved"`
}

type ListAlertsInput struct {
	Status   string
	Severity string
	RuleID   string
	HostID   string
}

type AlertEventSummary struct {
	ID        uint64 `json:"id"`
	EventType string `json:"eventType"`
	Message   string `json:"message,omitempty"`
	Actor     string `json:"actor,omitempty"`
	Payload   string `json:"payload,omitempty"`
	CreatedAt string `json:"createdAt"`
}

type AlertHistoryInput struct {
	Hours         int
	BucketMinutes int
	Severity      string
	RuleID        string
	HostID        string
}

type AlertHistoryPoint struct {
	BucketStart       string `json:"bucketStart"`
	FiringCount       int    `json:"firingCount"`
	ResolvedCount     int    `json:"resolvedCount"`
	AcknowledgedCount int    `json:"acknowledgedCount"`
	SilencedCount     int    `json:"silencedCount"`
}

type workspaceRecord struct {
	ID uint64
}

type userRecord struct {
	ID       uint64
	Username string
}

type ruleRecord struct {
	ID              uint64
	UID             string
	WorkspaceID     uint64
	Name            string
	RuleType        string
	MetricCode      sql.NullString
	Operator        sql.NullString
	Threshold       sql.NullFloat64
	DurationSeconds uint
	Expression      sql.NullString
	Severity        string
	Status          string
	CreatedBy       sql.NullString
	CreatedByID     sql.NullInt64
	CreatedAt       string
}

type alertRecord struct {
	ID             uint64
	UID            string
	WorkspaceID    uint64
	RuleUID        sql.NullString
	RuleName       sql.NullString
	RuleExpression sql.NullString
	ResourceType   string
	ResourceID     sql.NullInt64
	HostUID        sql.NullString
	HostName       sql.NullString
	Title          string
	Message        sql.NullString
	Severity       string
	Status         string
	Metadata       sql.NullString
	FirstSeenAt    string
	LastSeenAt     string
	ResolvedAt     sql.NullString
	CreatedAt      string
}

type alertMetadata struct {
	AcknowledgedAt string `json:"acknowledgedAt,omitempty"`
	AcknowledgedBy string `json:"acknowledgedBy,omitempty"`
	SilencedUntil  string `json:"silencedUntil,omitempty"`
	SilenceReason  string `json:"silenceReason,omitempty"`
	CooldownUntil  string `json:"cooldownUntil,omitempty"`
}

type ruleExpression struct {
	CooldownSeconds uint `json:"cooldownSeconds,omitempty"`
}

type latestMetric struct {
	HostID      uint64
	HostName    string
	MetricCode  string
	MetricValue float64
	Unit        sql.NullString
	CollectedAt string
}

type metricSeriesPoint struct {
	Value       float64
	CollectedAt string
}

type alertEventRecord struct {
	ID        uint64
	EventType string
	Message   sql.NullString
	Actor     sql.NullString
	Payload   sql.NullString
	CreatedAt string
}

func NewService(db *gorm.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

func (s *Service) ListRules(ctx context.Context) ([]AlertRuleSummary, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	rows, err := s.rules(ctx, workspace.ID, "")
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500701, "list alert rules failed", err)
	}
	out := make([]AlertRuleSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, ruleSummary(row))
	}
	return out, nil
}

func (s *Service) CreateRule(ctx context.Context, input CreateRuleInput) (AlertRuleSummary, *apperror.Error) {
	name := strings.TrimSpace(input.Name)
	metricCode := strings.TrimSpace(input.MetricCode)
	operator := strings.TrimSpace(input.Operator)
	severity := normalizeSeverity(input.Severity)
	duration := input.DurationSeconds
	if duration == 0 {
		duration = 60
	}
	expression := ruleExpression{CooldownSeconds: input.CooldownSeconds}
	if name == "" || metricCode == "" {
		return AlertRuleSummary{}, apperror.New(http.StatusBadRequest, 400701, "rule name and metricCode are required")
	}
	if !validOperator(operator) {
		return AlertRuleSummary{}, apperror.New(http.StatusBadRequest, 400702, "unsupported alert operator")
	}
	var created AlertRuleSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
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
			`INSERT INTO alert_rules(uid, workspace_id, name, rule_type, metric_code, operator, threshold,
			                         duration_seconds, severity, expression, status, created_by)
			 VALUES (?, ?, ?, 'metric', ?, ?, ?, ?, ?, ?, 'active', ?)`,
			ruleUID, workspace.ID, name, metricCode, operator, input.Threshold, duration, severity, jsonNull(expression), nullID(actor.ID),
		).Error; err != nil {
			return err
		}
		rows, err := (&Service{db: tx, cfg: s.cfg}).rules(ctx, workspace.ID, ruleUID)
		if err != nil {
			return err
		}
		if len(rows) > 0 {
			created = ruleSummary(rows[0])
			audit.Write(ctx, tx, audit.Event{
				WorkspaceID:   workspace.ID,
				ActorUserID:   nullID(actor.ID),
				Action:        "alert.rule.create",
				ResourceType:  "alert_rule",
				ResourceID:    nullID(rows[0].ID),
				IP:            input.Audit.IP,
				UserAgent:     input.Audit.UserAgent,
				TraceID:       input.Audit.TraceID,
				RequestMethod: input.Audit.RequestMethod,
				RequestPath:   input.Audit.RequestPath,
				After:         created,
			})
		}
		return nil
	})
	if txErr != nil {
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return AlertRuleSummary{}, apperror.New(http.StatusConflict, 409701, "alert rule already exists")
		}
		return AlertRuleSummary{}, apperror.Wrap(http.StatusInternalServerError, 500702, "create alert rule failed", txErr)
	}
	return created, nil
}

func (s *Service) UpdateRule(ctx context.Context, ruleUID string, input UpdateRuleInput) (AlertRuleSummary, *apperror.Error) {
	ruleUID = strings.TrimSpace(ruleUID)
	if ruleUID == "" {
		return AlertRuleSummary{}, apperror.New(http.StatusBadRequest, 400001, "rule id is required")
	}
	name := strings.TrimSpace(input.Name)
	metricCode := strings.TrimSpace(input.MetricCode)
	operator := strings.TrimSpace(input.Operator)
	severity := normalizeSeverity(input.Severity)
	duration := input.DurationSeconds
	if duration == 0 {
		duration = 60
	}
	if name == "" || metricCode == "" {
		return AlertRuleSummary{}, apperror.New(http.StatusBadRequest, 400701, "rule name and metricCode are required")
	}
	if !validOperator(operator) {
		return AlertRuleSummary{}, apperror.New(http.StatusBadRequest, 400702, "unsupported alert operator")
	}
	expression := ruleExpression{CooldownSeconds: input.CooldownSeconds}
	var updated AlertRuleSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		actor, err := userByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		rows, err := (&Service{db: tx, cfg: s.cfg}).rules(ctx, workspace.ID, ruleUID)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return apperror.New(http.StatusNotFound, 404702, "alert rule not found")
		}
		current := rows[0]
		if current.Status == "disabled" || current.Status == "archived" {
			return apperror.New(http.StatusBadRequest, 400706, "disabled alert rules cannot be updated")
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE alert_rules
			    SET name = ?, metric_code = ?, operator = ?, threshold = ?, duration_seconds = ?, severity = ?, expression = ?, updated_at = NOW(3)
			  WHERE id = ?`,
			name, metricCode, operator, input.Threshold, duration, severity, jsonNull(expression), current.ID,
		).Error; err != nil {
			return err
		}
		rows, err = (&Service{db: tx, cfg: s.cfg}).rules(ctx, workspace.ID, ruleUID)
		if err != nil {
			return err
		}
		updated = ruleSummary(rows[0])
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   nullID(actor.ID),
			Action:        "alert.rule.update",
			ResourceType:  "alert_rule",
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
			return AlertRuleSummary{}, appErr
		}
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return AlertRuleSummary{}, apperror.New(http.StatusConflict, 409701, "alert rule already exists")
		}
		return AlertRuleSummary{}, apperror.Wrap(http.StatusInternalServerError, 500709, "update alert rule failed", txErr)
	}
	return updated, nil
}

func (s *Service) UpdateRuleStatus(ctx context.Context, ruleUID, status string, auditCtx AuditContext) *apperror.Error {
	ruleUID = strings.TrimSpace(ruleUID)
	status = strings.TrimSpace(status)
	if ruleUID == "" {
		return apperror.New(http.StatusBadRequest, 400001, "rule id is required")
	}
	if status != "paused" && status != "active" && status != "disabled" {
		return apperror.New(http.StatusBadRequest, 400707, "unsupported alert rule status")
	}
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		actor, err := userByUID(ctx, tx, auditCtx.ActorUID)
		if err != nil {
			return err
		}
		rows, err := (&Service{db: tx, cfg: s.cfg}).rules(ctx, workspace.ID, ruleUID)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return apperror.New(http.StatusNotFound, 404702, "alert rule not found")
		}
		current := rows[0]
		if current.Status == status {
			return nil
		}
		if current.Status == "disabled" && status != "disabled" {
			return apperror.New(http.StatusBadRequest, 400708, "disabled alert rules cannot be resumed")
		}
		if err := tx.WithContext(ctx).Exec(
			"UPDATE alert_rules SET status = ?, updated_at = NOW(3) WHERE id = ?",
			status, current.ID,
		).Error; err != nil {
			return err
		}
		updated := ruleSummary(current)
		updated.Status = status
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   nullID(actor.ID),
			Action:        "alert.rule.status",
			ResourceType:  "alert_rule",
			ResourceID:    nullID(current.ID),
			IP:            auditCtx.IP,
			UserAgent:     auditCtx.UserAgent,
			TraceID:       auditCtx.TraceID,
			RequestMethod: auditCtx.RequestMethod,
			RequestPath:   auditCtx.RequestPath,
			Before:        ruleSummary(current),
			After:         updated,
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return appErr
		}
		return apperror.Wrap(http.StatusInternalServerError, 500710, "update alert rule status failed", txErr)
	}
	return nil
}

func (s *Service) ListAlerts(ctx context.Context, input ListAlertsInput) ([]AlertSummary, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	args := []interface{}{workspace.ID}
	where := "WHERE a.workspace_id = ?"
	if strings.TrimSpace(input.Status) != "" {
		where += " AND a.status = ?"
		args = append(args, strings.TrimSpace(input.Status))
	}
	if strings.TrimSpace(input.Severity) != "" {
		where += " AND a.severity = ?"
		args = append(args, strings.TrimSpace(input.Severity))
	}
	if strings.TrimSpace(input.RuleID) != "" {
		where += " AND ar.uid = ?"
		args = append(args, strings.TrimSpace(input.RuleID))
	}
	if strings.TrimSpace(input.HostID) != "" {
		where += " AND h.uid = ?"
		args = append(args, strings.TrimSpace(input.HostID))
	}
	var rows []alertRecord
	if err := s.db.WithContext(ctx).Raw(
		`SELECT a.id, a.uid, a.workspace_id, ar.uid AS rule_uid, ar.name AS rule_name, a.resource_type, a.resource_id,
		        h.uid AS host_uid, h.name AS host_name,
		        a.title, a.message, a.severity, a.status, a.metadata,
		        DATE_FORMAT(a.first_seen_at, '%Y-%m-%d %H:%i:%s') AS first_seen_at,
		        DATE_FORMAT(a.last_seen_at, '%Y-%m-%d %H:%i:%s') AS last_seen_at,
		        DATE_FORMAT(a.resolved_at, '%Y-%m-%d %H:%i:%s') AS resolved_at,
		        DATE_FORMAT(a.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM alerts a
		   LEFT JOIN alert_rules ar ON ar.id = a.alert_rule_id
		   LEFT JOIN hosts h ON a.resource_type = 'host' AND h.id = a.resource_id
		  `+where+`
		  ORDER BY a.last_seen_at DESC
		  LIMIT 200`,
		args...,
	).Scan(&rows).Error; err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500703, "list alerts failed", err)
	}
	out := make([]AlertSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, alertSummary(row))
	}
	return out, nil
}

func (s *Service) ListAlertEvents(ctx context.Context, alertUID, eventType string) ([]AlertEventSummary, *apperror.Error) {
	alertUID = strings.TrimSpace(alertUID)
	if alertUID == "" {
		return nil, apperror.New(http.StatusBadRequest, 400001, "alert id is required")
	}
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	args := []interface{}{workspace.ID, alertUID}
	where := "WHERE a.workspace_id = ? AND a.uid = ?"
	if strings.TrimSpace(eventType) != "" {
		where += " AND ae.event_type = ?"
		args = append(args, strings.TrimSpace(eventType))
	}
	var rows []alertEventRecord
	if err := s.db.WithContext(ctx).Raw(
		`SELECT ae.id, ae.event_type, ae.message, u.username AS actor, CAST(ae.payload AS CHAR) AS payload,
		        DATE_FORMAT(ae.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM alert_events ae
		   JOIN alerts a ON a.id = ae.alert_id
		   LEFT JOIN users u ON u.id = ae.actor_id
		  `+where+`
		  ORDER BY ae.created_at DESC, ae.id DESC
		  LIMIT 100`,
		args...,
	).Scan(&rows).Error; err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500711, "list alert events failed", err)
	}
	out := make([]AlertEventSummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, alertEventSummary(row))
	}
	return out, nil
}

func (s *Service) ListAlertHistory(ctx context.Context, input AlertHistoryInput) ([]AlertHistoryPoint, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	hours := normalizeHistoryHours(input.Hours)
	bucketMinutes := normalizeHistoryBucketMinutes(input.BucketMinutes)
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)
	args := []interface{}{bucketMinutes, bucketMinutes, workspace.ID, startTime}
	where := "WHERE a.workspace_id = ? AND ae.created_at >= ?"
	if strings.TrimSpace(input.Severity) != "" {
		where += " AND a.severity = ?"
		args = append(args, strings.TrimSpace(input.Severity))
	}
	if strings.TrimSpace(input.RuleID) != "" {
		where += " AND ar.uid = ?"
		args = append(args, strings.TrimSpace(input.RuleID))
	}
	if strings.TrimSpace(input.HostID) != "" {
		where += " AND h.uid = ?"
		args = append(args, strings.TrimSpace(input.HostID))
	}
	var rows []struct {
		BucketStart       string
		FiringCount       int
		ResolvedCount     int
		AcknowledgedCount int
		SilencedCount     int
	}
	if err := s.db.WithContext(ctx).Raw(
		`SELECT DATE_FORMAT(
		            FROM_UNIXTIME(FLOOR(UNIX_TIMESTAMP(ae.created_at) / (? * 60)) * (? * 60)),
		            '%Y-%m-%d %H:%i:%s'
		        ) AS bucket_start,
		        SUM(CASE WHEN ae.event_type = 'firing' THEN 1 ELSE 0 END) AS firing_count,
		        SUM(CASE WHEN ae.event_type = 'resolved' THEN 1 ELSE 0 END) AS resolved_count,
		        SUM(CASE WHEN ae.event_type = 'acknowledged' THEN 1 ELSE 0 END) AS acknowledged_count,
		        SUM(CASE WHEN ae.event_type = 'silenced' THEN 1 ELSE 0 END) AS silenced_count
		   FROM alert_events ae
		   JOIN alerts a ON a.id = ae.alert_id
		   LEFT JOIN alert_rules ar ON ar.id = a.alert_rule_id
		   LEFT JOIN hosts h ON a.resource_type = 'host' AND h.id = a.resource_id
		  `+where+`
		  GROUP BY bucket_start
		  ORDER BY bucket_start ASC`,
		args...,
	).Scan(&rows).Error; err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500712, "list alert history failed", err)
	}
	result := make([]AlertHistoryPoint, 0, len(rows))
	for _, row := range rows {
		result = append(result, AlertHistoryPoint{
			BucketStart:       row.BucketStart,
			FiringCount:       row.FiringCount,
			ResolvedCount:     row.ResolvedCount,
			AcknowledgedCount: row.AcknowledgedCount,
			SilencedCount:     row.SilencedCount,
		})
	}
	return result, nil
}

func (s *Service) Resolve(ctx context.Context, alertUID string, auditCtx AuditContext) *apperror.Error {
	alertUID = strings.TrimSpace(alertUID)
	if alertUID == "" {
		return apperror.New(http.StatusBadRequest, 400001, "alert id is required")
	}
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		actor, err := userByUID(ctx, tx, auditCtx.ActorUID)
		if err != nil {
			return err
		}
		row, err := alertByUID(ctx, tx, workspace.ID, alertUID)
		if err != nil {
			return err
		}
		if row.ID == 0 {
			return apperror.New(http.StatusNotFound, 404701, "alert not found")
		}
		if row.Status == "resolved" {
			return nil
		}
		if err := resolveAlertRecord(ctx, tx, row, "alert resolved manually", nullID(actor.ID), ruleCooldownSeconds(row.RuleExpression)); err != nil {
			return err
		}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   nullID(actor.ID),
			Action:        "alert.resolve",
			ResourceType:  "alert",
			ResourceID:    nullID(row.ID),
			IP:            auditCtx.IP,
			UserAgent:     auditCtx.UserAgent,
			TraceID:       auditCtx.TraceID,
			RequestMethod: auditCtx.RequestMethod,
			RequestPath:   auditCtx.RequestPath,
			After:         map[string]interface{}{"status": "resolved"},
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return appErr
		}
		return apperror.Wrap(http.StatusInternalServerError, 500704, "resolve alert failed", txErr)
	}
	return nil
}

func (s *Service) Acknowledge(ctx context.Context, alertUID string, auditCtx AuditContext) *apperror.Error {
	alertUID = strings.TrimSpace(alertUID)
	if alertUID == "" {
		return apperror.New(http.StatusBadRequest, 400001, "alert id is required")
	}
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		actor, err := userByUID(ctx, tx, auditCtx.ActorUID)
		if err != nil {
			return err
		}
		row, err := alertByUID(ctx, tx, workspace.ID, alertUID)
		if err != nil {
			return err
		}
		if row.ID == 0 {
			return apperror.New(http.StatusNotFound, 404701, "alert not found")
		}
		if row.Status == "resolved" {
			return apperror.New(http.StatusBadRequest, 400703, "resolved alerts cannot be acknowledged")
		}
		meta := parseAlertMetadata(row.Metadata)
		meta.AcknowledgedAt = nowString()
		meta.AcknowledgedBy = actor.Username
		meta.SilencedUntil = ""
		meta.SilenceReason = ""
		if err := tx.WithContext(ctx).Exec(
			"UPDATE alerts SET status = 'acknowledged', metadata = ?, updated_at = NOW(3) WHERE id = ?",
			jsonNull(meta), row.ID,
		).Error; err != nil {
			return err
		}
		if err := writeAlertEvent(ctx, tx, row.ID, "acknowledged", "alert acknowledged", nullID(actor.ID), meta); err != nil {
			return err
		}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   nullID(actor.ID),
			Action:        "alert.acknowledge",
			ResourceType:  "alert",
			ResourceID:    nullID(row.ID),
			IP:            auditCtx.IP,
			UserAgent:     auditCtx.UserAgent,
			TraceID:       auditCtx.TraceID,
			RequestMethod: auditCtx.RequestMethod,
			RequestPath:   auditCtx.RequestPath,
			After:         map[string]interface{}{"status": "acknowledged", "metadata": meta},
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return appErr
		}
		return apperror.Wrap(http.StatusInternalServerError, 500706, "acknowledge alert failed", txErr)
	}
	return nil
}

func (s *Service) Silence(ctx context.Context, alertUID string, input SilenceInput) *apperror.Error {
	alertUID = strings.TrimSpace(alertUID)
	if alertUID == "" {
		return apperror.New(http.StatusBadRequest, 400001, "alert id is required")
	}
	duration := input.DurationSeconds
	if duration == 0 {
		duration = 3600
	}
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		actor, err := userByUID(ctx, tx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		row, err := alertByUID(ctx, tx, workspace.ID, alertUID)
		if err != nil {
			return err
		}
		if row.ID == 0 {
			return apperror.New(http.StatusNotFound, 404701, "alert not found")
		}
		if row.Status == "resolved" {
			return apperror.New(http.StatusBadRequest, 400704, "resolved alerts cannot be silenced")
		}
		meta := parseAlertMetadata(row.Metadata)
		meta.SilencedUntil = time.Now().Add(time.Duration(duration) * time.Second).Format("2006-01-02 15:04:05")
		meta.SilenceReason = strings.TrimSpace(input.Reason)
		if err := tx.WithContext(ctx).Exec(
			"UPDATE alerts SET status = 'silenced', metadata = ?, updated_at = NOW(3) WHERE id = ?",
			jsonNull(meta), row.ID,
		).Error; err != nil {
			return err
		}
		if err := writeAlertEvent(ctx, tx, row.ID, "silenced", "alert silenced", nullID(actor.ID), meta); err != nil {
			return err
		}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   nullID(actor.ID),
			Action:        "alert.silence",
			ResourceType:  "alert",
			ResourceID:    nullID(row.ID),
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			After:         map[string]interface{}{"status": "silenced", "metadata": meta},
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return appErr
		}
		return apperror.Wrap(http.StatusInternalServerError, 500707, "silence alert failed", txErr)
	}
	return nil
}

func (s *Service) Unsilence(ctx context.Context, alertUID string, auditCtx AuditContext) *apperror.Error {
	alertUID = strings.TrimSpace(alertUID)
	if alertUID == "" {
		return apperror.New(http.StatusBadRequest, 400001, "alert id is required")
	}
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		actor, err := userByUID(ctx, tx, auditCtx.ActorUID)
		if err != nil {
			return err
		}
		row, err := alertByUID(ctx, tx, workspace.ID, alertUID)
		if err != nil {
			return err
		}
		if row.ID == 0 {
			return apperror.New(http.StatusNotFound, 404701, "alert not found")
		}
		if row.Status == "resolved" {
			return apperror.New(http.StatusBadRequest, 400705, "resolved alerts cannot be unsilenced")
		}
		meta := parseAlertMetadata(row.Metadata)
		meta.SilencedUntil = ""
		meta.SilenceReason = ""
		nextStatus := "firing"
		if meta.AcknowledgedAt != "" {
			nextStatus = "acknowledged"
		}
		if err := tx.WithContext(ctx).Exec(
			"UPDATE alerts SET status = ?, metadata = ?, updated_at = NOW(3) WHERE id = ?",
			nextStatus, jsonNull(meta), row.ID,
		).Error; err != nil {
			return err
		}
		if err := writeAlertEvent(ctx, tx, row.ID, "unsilenced", "alert unsilenced", nullID(actor.ID), meta); err != nil {
			return err
		}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   nullID(actor.ID),
			Action:        "alert.unsilence",
			ResourceType:  "alert",
			ResourceID:    nullID(row.ID),
			IP:            auditCtx.IP,
			UserAgent:     auditCtx.UserAgent,
			TraceID:       auditCtx.TraceID,
			RequestMethod: auditCtx.RequestMethod,
			RequestPath:   auditCtx.RequestPath,
			After:         map[string]interface{}{"status": nextStatus, "metadata": meta},
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return appErr
		}
		return apperror.Wrap(http.StatusInternalServerError, 500708, "unsilence alert failed", txErr)
	}
	return nil
}

func (s *Service) Evaluate(ctx context.Context) (EvaluationResult, error) {
	var result EvaluationResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspaces, err := activeWorkspaces(ctx, tx)
		if err != nil {
			return err
		}
		for _, workspace := range workspaces {
			rules, err := (&Service{db: tx, cfg: s.cfg}).rules(ctx, workspace.ID, "")
			if err != nil {
				return err
			}
			for _, rule := range rules {
				if rule.Status != "active" || rule.RuleType != "metric" || !rule.MetricCode.Valid || !rule.Threshold.Valid {
					continue
				}
				fired, resolved, err := evaluateMetricRule(ctx, tx, rule)
				if err != nil {
					return err
				}
				result.Fired += fired
				result.Resolved += resolved
			}
		}
		return nil
	})
	return result, err
}

func evaluateMetricRule(ctx context.Context, tx *gorm.DB, rule ruleRecord) (int, int, error) {
	var rows []latestMetric
	if err := tx.WithContext(ctx).Raw(
		`SELECT hm.host_id, h.name AS host_name, hm.metric_code, hm.metric_value, hm.unit,
		        DATE_FORMAT(hm.collected_at, '%Y-%m-%d %H:%i:%s') AS collected_at
		   FROM host_metrics hm
		   JOIN hosts h ON h.id = hm.host_id
		   JOIN (
		     SELECT host_id, metric_code, MAX(collected_at) AS max_collected_at
		       FROM host_metrics
		      WHERE workspace_id = ? AND metric_code = ?
		      GROUP BY host_id, metric_code
		   ) latest ON latest.host_id = hm.host_id
		            AND latest.metric_code = hm.metric_code
		            AND latest.max_collected_at = hm.collected_at
		  WHERE hm.workspace_id = ? AND hm.metric_code = ?`,
		rule.WorkspaceID, rule.MetricCode.String, rule.WorkspaceID, rule.MetricCode.String,
	).Scan(&rows).Error; err != nil {
		return 0, 0, err
	}
	fired := 0
	resolved := 0
	for _, metric := range rows {
		fp := alertFingerprint(rule.ID, metric.HostID, rule.MetricCode.String)
		sustained, _, streakDuration, err := metricRuleSustained(ctx, tx, rule, metric)
		if err != nil {
			return fired, resolved, err
		}
		if sustained {
			title := fmt.Sprintf("%s %s %.2f", rule.MetricCode.String, rule.Operator.String, rule.Threshold.Float64)
			message := fmt.Sprintf("%s value %.2f%s at %s", metric.HostName, metric.MetricValue, metric.Unit.String, metric.CollectedAt)
			if streakDuration > 0 {
				message = fmt.Sprintf("%s for %ds", message, uint(streakDuration.Seconds()))
			}
			created, err := upsertFiringAlert(ctx, tx, rule, metric.HostID, title, message, fp, metric)
			if err != nil {
				return fired, resolved, err
			}
			if created {
				fired++
			}
			continue
		}
		changed, err := resolveOpenAlert(ctx, tx, rule, fp, "metric threshold recovered or duration window reset", sql.NullInt64{})
		if err != nil {
			return fired, resolved, err
		}
		if changed {
			resolved++
		}
	}
	return fired, resolved, nil
}

func metricRuleSustained(ctx context.Context, tx *gorm.DB, rule ruleRecord, latest latestMetric) (bool, time.Time, time.Duration, error) {
	latestAt, err := parseDBTime(latest.CollectedAt)
	if err != nil {
		return false, time.Time{}, 0, err
	}
	if !compare(latest.MetricValue, rule.Operator.String, rule.Threshold.Float64) {
		return false, time.Time{}, 0, nil
	}
	if rule.DurationSeconds == 0 {
		return true, latestAt, 0, nil
	}
	points, err := recentMetricSeries(ctx, tx, rule.WorkspaceID, latest.HostID, rule.MetricCode.String, maxMetricSeriesPoints(rule.DurationSeconds))
	if err != nil {
		return false, time.Time{}, 0, err
	}
	ok, streakStart := sustainedViolation(points, rule.Operator.String, rule.Threshold.Float64, time.Duration(rule.DurationSeconds)*time.Second)
	return ok, streakStart, latestAt.Sub(streakStart), nil
}

func recentMetricSeries(ctx context.Context, tx *gorm.DB, workspaceID, hostID uint64, metricCode string, limit int) ([]metricSeriesPoint, error) {
	var rows []metricSeriesPoint
	err := tx.WithContext(ctx).Raw(
		`SELECT hm.metric_value AS value,
		        DATE_FORMAT(hm.collected_at, '%Y-%m-%d %H:%i:%s') AS collected_at
		   FROM host_metrics hm
		  WHERE hm.workspace_id = ? AND hm.host_id = ? AND hm.metric_code = ?
		  ORDER BY hm.collected_at DESC, hm.id DESC
		  LIMIT ?`,
		workspaceID, hostID, metricCode, limit,
	).Scan(&rows).Error
	return rows, err
}

func sustainedViolation(points []metricSeriesPoint, operator string, threshold float64, duration time.Duration) (bool, time.Time) {
	if len(points) == 0 {
		return false, time.Time{}
	}
	latestAt, err := parseDBTime(points[0].CollectedAt)
	if err != nil {
		return false, time.Time{}
	}
	if !compare(points[0].Value, operator, threshold) {
		return false, time.Time{}
	}
	if duration <= 0 {
		return true, latestAt
	}
	streakStart := latestAt
	for _, point := range points {
		if !compare(point.Value, operator, threshold) {
			break
		}
		at, err := parseDBTime(point.CollectedAt)
		if err != nil {
			break
		}
		streakStart = at
	}
	return latestAt.Sub(streakStart) >= duration, streakStart
}

func upsertFiringAlert(ctx context.Context, tx *gorm.DB, rule ruleRecord, hostID uint64, title, message, fingerprint string, metric latestMetric) (bool, error) {
	var existing struct {
		ID       uint64
		Status   string
		Metadata sql.NullString
	}
	if err := tx.WithContext(ctx).Raw(
		"SELECT id, status, metadata FROM alerts WHERE workspace_id = ? AND fingerprint = ? AND status IN ('firing', 'acknowledged', 'silenced') LIMIT 1",
		rule.WorkspaceID, fingerprint,
	).Scan(&existing).Error; err != nil {
		return false, err
	}
	if existing.ID != 0 {
		meta := parseAlertMetadata(existing.Metadata)
		nextStatus := existing.Status
		if existing.Status == "silenced" && silenceExpired(meta) {
			nextStatus = "firing"
			meta.SilencedUntil = ""
			meta.SilenceReason = ""
		}
		if err := tx.WithContext(ctx).Exec(
			"UPDATE alerts SET status = ?, metadata = ?, last_seen_at = NOW(3), message = ?, updated_at = NOW(3) WHERE id = ?",
			nextStatus, jsonNull(meta), message, existing.ID,
		).Error; err != nil {
			return false, err
		}
		return false, nil
	}
	suppressed, cooldownUntil := cooldownSuppressed(ctx, tx, rule.WorkspaceID, fingerprint)
	alertUID, err := uid.New()
	if err != nil {
		return false, err
	}
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO alerts(uid, workspace_id, alert_rule_id, resource_type, resource_id, title, message,
		                    severity, status, fingerprint, first_seen_at, last_seen_at, metadata)
		 VALUES (?, ?, ?, 'host', ?, ?, ?, ?, 'firing', ?, NOW(3), NOW(3), ?)`,
		alertUID, rule.WorkspaceID, rule.ID, hostID, title, message, rule.Severity, fingerprint, jsonNull(alertMetadata{}),
	).Error; err != nil {
		return false, err
	}
	var alertID uint64
	if err := tx.WithContext(ctx).Raw("SELECT id FROM alerts WHERE uid = ? LIMIT 1", alertUID).Scan(&alertID).Error; err != nil {
		return false, err
	}
	meta := alertMetadata{CooldownUntil: cooldownUntil}
	if err := writeAlertEvent(ctx, tx, alertID, "firing", "metric threshold violated", sql.NullInt64{}, metric); err != nil {
		return false, err
	}
	if suppressed {
		if err := tx.WithContext(ctx).Exec("UPDATE alerts SET metadata = ? WHERE id = ?", jsonNull(meta), alertID).Error; err != nil {
			return false, err
		}
		if err := writeAlertEvent(ctx, tx, alertID, "cooldown_suppressed", "notification suppressed by cooldown", sql.NullInt64{}, meta); err != nil {
			return false, err
		}
	} else {
		if err := notifications.EnqueueForAlert(ctx, tx, rule.WorkspaceID, alertID, title, message, rule.Severity); err != nil {
			return false, err
		}
	}
	return true, nil
}

func resolveOpenAlert(ctx context.Context, tx *gorm.DB, rule ruleRecord, fingerprint string, reason string, actorID sql.NullInt64) (bool, error) {
	var alert alertRecord
	if err := tx.WithContext(ctx).Raw(
		`SELECT a.id, a.uid, a.workspace_id, ar.uid AS rule_uid, ar.name AS rule_name, a.resource_type, a.resource_id,
		        a.title, a.message, a.severity, a.status, a.metadata,
		        DATE_FORMAT(a.first_seen_at, '%Y-%m-%d %H:%i:%s') AS first_seen_at,
		        DATE_FORMAT(a.last_seen_at, '%Y-%m-%d %H:%i:%s') AS last_seen_at,
		        DATE_FORMAT(a.resolved_at, '%Y-%m-%d %H:%i:%s') AS resolved_at,
		        DATE_FORMAT(a.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM alerts a
		   LEFT JOIN alert_rules ar ON ar.id = a.alert_rule_id
		  WHERE a.workspace_id = ? AND a.fingerprint = ? AND a.status IN ('firing', 'acknowledged', 'silenced')
		  LIMIT 1`,
		rule.WorkspaceID, fingerprint,
	).Scan(&alert).Error; err != nil {
		return false, err
	}
	if alert.ID == 0 {
		return false, nil
	}
	return true, resolveAlertRecord(ctx, tx, alert, reason, actorID, ruleCooldownSeconds(rule.Expression))
}

func (s *Service) rules(ctx context.Context, workspaceID uint64, ruleUID string) ([]ruleRecord, error) {
	args := []interface{}{workspaceID}
	where := "WHERE ar.workspace_id = ? AND ar.deleted_at IS NULL"
	if ruleUID != "" {
		where += " AND ar.uid = ?"
		args = append(args, ruleUID)
	}
	var rows []ruleRecord
	err := s.db.WithContext(ctx).Raw(
		`SELECT ar.id, ar.uid, ar.workspace_id, ar.name, ar.rule_type, ar.metric_code, ar.operator,
		        ar.threshold, ar.duration_seconds, ar.expression, ar.severity, ar.status,
		        u.username AS created_by, ar.created_by AS created_by_id,
		        DATE_FORMAT(ar.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM alert_rules ar
		   LEFT JOIN users u ON u.id = ar.created_by
		  `+where+`
		  ORDER BY ar.created_at DESC`,
		args...,
	).Scan(&rows).Error
	return rows, err
}

func (s *Service) workspace(ctx context.Context) (workspaceRecord, *apperror.Error) {
	workspace, err := defaultWorkspace(ctx, s.db, s.cfg.Bootstrap.WorkspaceSlug)
	if err != nil {
		return workspaceRecord{}, apperror.Wrap(http.StatusInternalServerError, 500705, "load workspace failed", err)
	}
	if workspace.ID == 0 {
		return workspaceRecord{}, apperror.New(http.StatusInternalServerError, 500705, "default workspace is not initialized")
	}
	return workspace, nil
}

func activeWorkspaces(ctx context.Context, db *gorm.DB) ([]workspaceRecord, error) {
	var rows []workspaceRecord
	err := db.WithContext(ctx).Raw("SELECT id FROM workspaces WHERE status = 'active'").Scan(&rows).Error
	return rows, err
}

func defaultWorkspace(ctx context.Context, db *gorm.DB, slug string) (workspaceRecord, error) {
	var workspace workspaceRecord
	err := db.WithContext(ctx).Raw("SELECT id FROM workspaces WHERE slug = ? AND status = 'active' LIMIT 1", slug).Scan(&workspace).Error
	return workspace, err
}

func userByUID(ctx context.Context, db *gorm.DB, uid string) (userRecord, error) {
	var user userRecord
	err := db.WithContext(ctx).Raw("SELECT id, username FROM users WHERE uid = ? AND status = 'active' LIMIT 1", uid).Scan(&user).Error
	return user, err
}

func writeAlertEvent(ctx context.Context, tx *gorm.DB, alertID uint64, eventType, message string, actorID sql.NullInt64, payload interface{}) error {
	return tx.WithContext(ctx).Exec(
		"INSERT INTO alert_events(alert_id, event_type, message, actor_id, payload) VALUES (?, ?, ?, ?, ?)",
		alertID, eventType, nullString(message), actorID, jsonNull(payload),
	).Error
}

func alertByUID(ctx context.Context, tx *gorm.DB, workspaceID uint64, alertUID string) (alertRecord, error) {
	var row alertRecord
	err := tx.WithContext(ctx).Raw(
		`SELECT a.id, a.uid, a.workspace_id, ar.uid AS rule_uid, ar.name AS rule_name, ar.expression AS rule_expression,
		        a.resource_type, a.resource_id, a.title, a.message, a.severity, a.status, a.metadata,
		        DATE_FORMAT(a.first_seen_at, '%Y-%m-%d %H:%i:%s') AS first_seen_at,
		        DATE_FORMAT(a.last_seen_at, '%Y-%m-%d %H:%i:%s') AS last_seen_at,
		        DATE_FORMAT(a.resolved_at, '%Y-%m-%d %H:%i:%s') AS resolved_at,
		        DATE_FORMAT(a.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM alerts a
		   LEFT JOIN alert_rules ar ON ar.id = a.alert_rule_id
		  WHERE a.workspace_id = ? AND a.uid = ?
		  LIMIT 1
		  FOR UPDATE`,
		workspaceID, alertUID,
	).Scan(&row).Error
	return row, err
}

func resolveAlertRecord(ctx context.Context, tx *gorm.DB, row alertRecord, reason string, actorID sql.NullInt64, cooldownSeconds uint) error {
	meta := parseAlertMetadata(row.Metadata)
	if cooldownSeconds > 0 {
		meta.CooldownUntil = time.Now().Add(time.Duration(cooldownSeconds) * time.Second).Format("2006-01-02 15:04:05")
	}
	if err := tx.WithContext(ctx).Exec(
		"UPDATE alerts SET status = 'resolved', resolved_at = NOW(3), resolved_by = ?, metadata = ?, updated_at = NOW(3) WHERE id = ?",
		actorID, jsonNull(meta), row.ID,
	).Error; err != nil {
		return err
	}
	if err := writeAlertEvent(ctx, tx, row.ID, "resolved", reason, actorID, meta); err != nil {
		return err
	}
	message := strings.TrimSpace(row.Message.String)
	if message == "" {
		message = reason
	}
	return notifications.EnqueueForAlert(ctx, tx, row.WorkspaceID, row.ID, "Resolved: "+row.Title, message, "info")
}

func ruleSummary(row ruleRecord) AlertRuleSummary {
	return AlertRuleSummary{
		ID:              row.UID,
		Name:            row.Name,
		RuleType:        row.RuleType,
		MetricCode:      row.MetricCode.String,
		Operator:        row.Operator.String,
		Threshold:       row.Threshold.Float64,
		DurationSeconds: row.DurationSeconds,
		CooldownSeconds: ruleCooldownSeconds(row.Expression),
		Severity:        row.Severity,
		Status:          row.Status,
		CreatedBy:       row.CreatedBy.String,
		CreatedAt:       row.CreatedAt,
	}
}

func alertEventSummary(row alertEventRecord) AlertEventSummary {
	return AlertEventSummary{
		ID:        row.ID,
		EventType: row.EventType,
		Message:   row.Message.String,
		Actor:     row.Actor.String,
		Payload:   row.Payload.String,
		CreatedAt: row.CreatedAt,
	}
}

func alertSummary(row alertRecord) AlertSummary {
	resourceID := uint64(0)
	if row.ResourceID.Valid {
		resourceID = uint64(row.ResourceID.Int64)
	}
	meta := parseAlertMetadata(row.Metadata)
	return AlertSummary{
		ID:             row.UID,
		RuleID:         row.RuleUID.String,
		RuleName:       row.RuleName.String,
		ResourceType:   row.ResourceType,
		ResourceID:     resourceID,
		HostID:         row.HostUID.String,
		HostName:       row.HostName.String,
		Title:          row.Title,
		Message:        row.Message.String,
		Severity:       row.Severity,
		Status:         row.Status,
		AcknowledgedAt: meta.AcknowledgedAt,
		AcknowledgedBy: meta.AcknowledgedBy,
		SilencedUntil:  meta.SilencedUntil,
		SilenceReason:  meta.SilenceReason,
		CooldownUntil:  meta.CooldownUntil,
		FirstSeenAt:    row.FirstSeenAt,
		LastSeenAt:     row.LastSeenAt,
		ResolvedAt:     row.ResolvedAt.String,
		CreatedAt:      row.CreatedAt,
	}
}

func parseAlertMetadata(raw sql.NullString) alertMetadata {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return alertMetadata{}
	}
	var meta alertMetadata
	if err := json.Unmarshal([]byte(raw.String), &meta); err != nil {
		return alertMetadata{}
	}
	return meta
}

func ruleCooldownSeconds(raw sql.NullString) uint {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return 0
	}
	var expr ruleExpression
	if err := json.Unmarshal([]byte(raw.String), &expr); err != nil {
		return 0
	}
	return expr.CooldownSeconds
}

func cooldownSuppressed(ctx context.Context, tx *gorm.DB, workspaceID uint64, fingerprint string) (bool, string) {
	var row struct {
		Metadata sql.NullString
	}
	if err := tx.WithContext(ctx).Raw(
		"SELECT metadata FROM alerts WHERE workspace_id = ? AND fingerprint = ? ORDER BY id DESC LIMIT 1",
		workspaceID, fingerprint,
	).Scan(&row).Error; err != nil {
		return false, ""
	}
	meta := parseAlertMetadata(row.Metadata)
	if meta.CooldownUntil == "" {
		return false, ""
	}
	until, err := time.ParseInLocation("2006-01-02 15:04:05", meta.CooldownUntil, time.Local)
	if err != nil {
		return false, ""
	}
	return until.After(time.Now()), meta.CooldownUntil
}

func silenceExpired(meta alertMetadata) bool {
	if meta.SilencedUntil == "" {
		return false
	}
	until, err := time.ParseInLocation("2006-01-02 15:04:05", meta.SilencedUntil, time.Local)
	if err != nil {
		return true
	}
	return !until.After(time.Now())
}

func nowString() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func parseDBTime(value string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02 15:04:05", strings.TrimSpace(value), time.Local)
}

func maxMetricSeriesPoints(durationSeconds uint) int {
	estimated := int(durationSeconds/15) + 12
	if estimated < 24 {
		return 24
	}
	if estimated > 480 {
		return 480
	}
	return estimated
}

func normalizeHistoryHours(value int) int {
	if value <= 0 {
		return 24
	}
	if value > 24*14 {
		return 24 * 14
	}
	return value
}

func normalizeHistoryBucketMinutes(value int) int {
	if value <= 0 {
		return 60
	}
	if value < 5 {
		return 5
	}
	if value > 240 {
		return 240
	}
	return value
}

func compare(value float64, operator string, threshold float64) bool {
	switch operator {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return false
	}
}

func validOperator(value string) bool {
	switch value {
	case ">", ">=", "<", "<=", "==", "!=":
		return true
	default:
		return false
	}
}

func normalizeSeverity(value string) string {
	switch strings.TrimSpace(value) {
	case "info", "warning", "critical":
		return strings.TrimSpace(value)
	default:
		return "warning"
	}
}

func alertFingerprint(ruleID, hostID uint64, metricCode string) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%d:%s", ruleID, hostID, metricCode)))
	return hex.EncodeToString(sum[:])
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

func jsonNull(value interface{}) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	bytes, err := json.Marshal(value)
	if err != nil || string(bytes) == "null" {
		return sql.NullString{}
	}
	return sql.NullString{String: string(bytes), Valid: true}
}
