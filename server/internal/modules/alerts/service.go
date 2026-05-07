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
	Severity        string  `json:"severity"`
	Status          string  `json:"status"`
	CreatedBy       string  `json:"createdBy,omitempty"`
	CreatedAt       string  `json:"createdAt"`
}

type AlertSummary struct {
	ID           string `json:"id"`
	RuleID       string `json:"ruleId,omitempty"`
	RuleName     string `json:"ruleName,omitempty"`
	ResourceType string `json:"resourceType"`
	ResourceID   uint64 `json:"resourceId,omitempty"`
	Title        string `json:"title"`
	Message      string `json:"message,omitempty"`
	Severity     string `json:"severity"`
	Status       string `json:"status"`
	FirstSeenAt  string `json:"firstSeenAt"`
	LastSeenAt   string `json:"lastSeenAt"`
	ResolvedAt   string `json:"resolvedAt,omitempty"`
	CreatedAt    string `json:"createdAt"`
}

type EvaluationResult struct {
	Fired    int `json:"fired"`
	Resolved int `json:"resolved"`
}

type workspaceRecord struct {
	ID uint64
}

type userRecord struct {
	ID uint64
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
	Severity        string
	Status          string
	CreatedBy       sql.NullString
	CreatedByID     sql.NullInt64
	CreatedAt       string
}

type alertRecord struct {
	ID           uint64
	UID          string
	RuleUID      sql.NullString
	RuleName     sql.NullString
	ResourceType string
	ResourceID   sql.NullInt64
	Title        string
	Message      sql.NullString
	Severity     string
	Status       string
	FirstSeenAt  string
	LastSeenAt   string
	ResolvedAt   sql.NullString
	CreatedAt    string
}

type latestMetric struct {
	HostID      uint64
	HostName    string
	MetricCode  string
	MetricValue float64
	Unit        sql.NullString
	CollectedAt string
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
			                         duration_seconds, severity, status, created_by)
			 VALUES (?, ?, ?, 'metric', ?, ?, ?, ?, ?, 'active', ?)`,
			ruleUID, workspace.ID, name, metricCode, operator, input.Threshold, duration, severity, nullID(actor.ID),
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

func (s *Service) ListAlerts(ctx context.Context, status string) ([]AlertSummary, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	args := []interface{}{workspace.ID}
	where := "WHERE a.workspace_id = ?"
	if strings.TrimSpace(status) != "" {
		where += " AND a.status = ?"
		args = append(args, strings.TrimSpace(status))
	}
	var rows []alertRecord
	if err := s.db.WithContext(ctx).Raw(
		`SELECT a.id, a.uid, ar.uid AS rule_uid, ar.name AS rule_name, a.resource_type, a.resource_id,
		        a.title, a.message, a.severity, a.status,
		        DATE_FORMAT(a.first_seen_at, '%Y-%m-%d %H:%i:%s') AS first_seen_at,
		        DATE_FORMAT(a.last_seen_at, '%Y-%m-%d %H:%i:%s') AS last_seen_at,
		        DATE_FORMAT(a.resolved_at, '%Y-%m-%d %H:%i:%s') AS resolved_at,
		        DATE_FORMAT(a.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM alerts a
		   LEFT JOIN alert_rules ar ON ar.id = a.alert_rule_id
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
		var row struct {
			ID uint64
		}
		if err := tx.WithContext(ctx).Raw(
			"SELECT id FROM alerts WHERE workspace_id = ? AND uid = ? LIMIT 1 FOR UPDATE",
			workspace.ID, alertUID,
		).Scan(&row).Error; err != nil {
			return err
		}
		if row.ID == 0 {
			return apperror.New(http.StatusNotFound, 404701, "alert not found")
		}
		if err := tx.WithContext(ctx).Exec(
			"UPDATE alerts SET status = 'resolved', resolved_at = NOW(3), resolved_by = ? WHERE id = ? AND status <> 'resolved'",
			nullID(actor.ID), row.ID,
		).Error; err != nil {
			return err
		}
		return writeAlertEvent(ctx, tx, row.ID, "resolved", "alert resolved manually", nullID(actor.ID), nil)
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return appErr
		}
		return apperror.Wrap(http.StatusInternalServerError, 500704, "resolve alert failed", txErr)
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
		violating := compare(metric.MetricValue, rule.Operator.String, rule.Threshold.Float64)
		fp := alertFingerprint(rule.ID, metric.HostID, rule.MetricCode.String)
		if violating {
			title := fmt.Sprintf("%s %s %.2f", rule.MetricCode.String, rule.Operator.String, rule.Threshold.Float64)
			message := fmt.Sprintf("%s value %.2f%s at %s", metric.HostName, metric.MetricValue, metric.Unit.String, metric.CollectedAt)
			created, err := upsertFiringAlert(ctx, tx, rule, metric.HostID, title, message, fp, metric)
			if err != nil {
				return fired, resolved, err
			}
			if created {
				fired++
			}
			continue
		}
		changed, err := resolveOpenAlert(ctx, tx, rule.WorkspaceID, fp)
		if err != nil {
			return fired, resolved, err
		}
		if changed {
			resolved++
		}
	}
	return fired, resolved, nil
}

func upsertFiringAlert(ctx context.Context, tx *gorm.DB, rule ruleRecord, hostID uint64, title, message, fingerprint string, metric latestMetric) (bool, error) {
	var existing struct {
		ID uint64
	}
	if err := tx.WithContext(ctx).Raw(
		"SELECT id FROM alerts WHERE workspace_id = ? AND fingerprint = ? AND status IN ('firing', 'acknowledged', 'silenced') LIMIT 1",
		rule.WorkspaceID, fingerprint,
	).Scan(&existing).Error; err != nil {
		return false, err
	}
	if existing.ID != 0 {
		if err := tx.WithContext(ctx).Exec(
			"UPDATE alerts SET last_seen_at = NOW(3), message = ?, updated_at = NOW(3) WHERE id = ?",
			message, existing.ID,
		).Error; err != nil {
			return false, err
		}
		return false, nil
	}
	alertUID, err := uid.New()
	if err != nil {
		return false, err
	}
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO alerts(uid, workspace_id, alert_rule_id, resource_type, resource_id, title, message,
		                    severity, status, fingerprint, first_seen_at, last_seen_at, metadata)
		 VALUES (?, ?, ?, 'host', ?, ?, ?, ?, 'firing', ?, NOW(3), NOW(3), ?)`,
		alertUID, rule.WorkspaceID, rule.ID, hostID, title, message, rule.Severity, fingerprint, jsonNull(metric),
	).Error; err != nil {
		return false, err
	}
	var alertID uint64
	if err := tx.WithContext(ctx).Raw("SELECT id FROM alerts WHERE uid = ? LIMIT 1", alertUID).Scan(&alertID).Error; err != nil {
		return false, err
	}
	if err := writeAlertEvent(ctx, tx, alertID, "firing", "metric threshold violated", sql.NullInt64{}, metric); err != nil {
		return false, err
	}
	if err := notifications.EnqueueForAlert(ctx, tx, rule.WorkspaceID, alertID, title, message, rule.Severity); err != nil {
		return false, err
	}
	return true, nil
}

func resolveOpenAlert(ctx context.Context, tx *gorm.DB, workspaceID uint64, fingerprint string) (bool, error) {
	var alert struct {
		ID uint64
	}
	if err := tx.WithContext(ctx).Raw(
		"SELECT id FROM alerts WHERE workspace_id = ? AND fingerprint = ? AND status IN ('firing', 'acknowledged', 'silenced') LIMIT 1",
		workspaceID, fingerprint,
	).Scan(&alert).Error; err != nil {
		return false, err
	}
	if alert.ID == 0 {
		return false, nil
	}
	if err := tx.WithContext(ctx).Exec("UPDATE alerts SET status = 'resolved', resolved_at = NOW(3), updated_at = NOW(3) WHERE id = ?", alert.ID).Error; err != nil {
		return false, err
	}
	return true, writeAlertEvent(ctx, tx, alert.ID, "resolved", "metric threshold recovered", sql.NullInt64{}, nil)
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
		        ar.threshold, ar.duration_seconds, ar.severity, ar.status,
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
	err := db.WithContext(ctx).Raw("SELECT id FROM users WHERE uid = ? AND status = 'active' LIMIT 1", uid).Scan(&user).Error
	return user, err
}

func writeAlertEvent(ctx context.Context, tx *gorm.DB, alertID uint64, eventType, message string, actorID sql.NullInt64, payload interface{}) error {
	return tx.WithContext(ctx).Exec(
		"INSERT INTO alert_events(alert_id, event_type, message, actor_id, payload) VALUES (?, ?, ?, ?, ?)",
		alertID, eventType, nullString(message), actorID, jsonNull(payload),
	).Error
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
		Severity:        row.Severity,
		Status:          row.Status,
		CreatedBy:       row.CreatedBy.String,
		CreatedAt:       row.CreatedAt,
	}
}

func alertSummary(row alertRecord) AlertSummary {
	resourceID := uint64(0)
	if row.ResourceID.Valid {
		resourceID = uint64(row.ResourceID.Int64)
	}
	return AlertSummary{
		ID:           row.UID,
		RuleID:       row.RuleUID.String,
		RuleName:     row.RuleName.String,
		ResourceType: row.ResourceType,
		ResourceID:   resourceID,
		Title:        row.Title,
		Message:      row.Message.String,
		Severity:     row.Severity,
		Status:       row.Status,
		FirstSeenAt:  row.FirstSeenAt,
		LastSeenAt:   row.LastSeenAt,
		ResolvedAt:   row.ResolvedAt.String,
		CreatedAt:    row.CreatedAt,
	}
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
