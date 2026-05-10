package incidents

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"opspilot/server/internal/config"
	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/audit"
)

type Service struct {
	db  *gorm.DB
	cfg config.Config
}

type ListInput struct {
	Status   string
	Severity string
}

type AuditContext struct {
	ActorUID      string
	IP            string
	UserAgent     string
	TraceID       string
	RequestMethod string
	RequestPath   string
}

type LifecycleUpdateInput struct {
	Owner          string `json:"owner"`
	ImpactScope    string `json:"impactScope"`
	RootCauseClass string `json:"rootCauseClass"`
	Postmortem     string `json:"postmortem"`
	Audit          AuditContext
}

type MergeInput struct {
	TargetIncidentID string `json:"targetIncidentId"`
	Reason           string `json:"reason"`
	Audit            AuditContext
}

type CloseInput struct {
	Reason string `json:"reason"`
	Audit  AuditContext
}

type Summary struct {
	ID          string `json:"id"`
	AlertID     string `json:"alertId,omitempty"`
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	Status      string `json:"status"`
	Owner       string `json:"owner,omitempty"`
	ImpactScope string `json:"impactScope,omitempty"`
	RootCause   string `json:"rootCause,omitempty"`
	MergedInto  string `json:"mergedInto,omitempty"`
	RuleID      string `json:"ruleId,omitempty"`
	RuleName    string `json:"ruleName,omitempty"`
	HostID      string `json:"hostId,omitempty"`
	HostName    string `json:"hostName,omitempty"`
	Message     string `json:"message,omitempty"`
	FirstSeenAt string `json:"firstSeenAt"`
	LastSeenAt  string `json:"lastSeenAt"`
	ResolvedAt  string `json:"resolvedAt,omitempty"`
	AlertCount  int    `json:"alertCount"`
}

type Event struct {
	ID        uint64 `json:"id"`
	EventType string `json:"eventType"`
	Message   string `json:"message,omitempty"`
	Actor     string `json:"actor,omitempty"`
	Payload   string `json:"payload,omitempty"`
	CreatedAt string `json:"createdAt"`
}

type Detail struct {
	Summary
	Postmortem string  `json:"postmortem,omitempty"`
	Events []Event `json:"events"`
}

type workspaceRecord struct {
	ID uint64
}

type incidentRecord struct {
	ID          uint64
	UID         string
	AlertUID    sql.NullString
	RuleUID     sql.NullString
	RuleName    sql.NullString
	HostUID     sql.NullString
	HostName    sql.NullString
	Title       string
	Message     sql.NullString
	Severity    string
	Status      string
	Metadata    sql.NullString
	FirstSeenAt string
	LastSeenAt  string
	ResolvedAt  sql.NullString
	AlertCount  int
}

type incidentMetadata struct {
	Owner          string `json:"owner,omitempty"`
	ImpactScope    string `json:"impactScope,omitempty"`
	RootCauseClass string `json:"rootCauseClass,omitempty"`
	Postmortem     string `json:"postmortem,omitempty"`
	MergedInto     string `json:"mergedInto,omitempty"`
}

type eventRecord struct {
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

func (s *Service) List(ctx context.Context, input ListInput) ([]Summary, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	args := []interface{}{workspace.ID}
	where := "WHERE i.workspace_id = ? AND i.deleted_at IS NULL"
	if strings.TrimSpace(input.Status) != "" {
		where += " AND i.status = ?"
		args = append(args, strings.TrimSpace(input.Status))
	}
	if strings.TrimSpace(input.Severity) != "" {
		where += " AND i.severity = ?"
		args = append(args, strings.TrimSpace(input.Severity))
	}
	rows, err := s.queryIncidents(ctx, where, args...)
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500901, "list incidents failed", err)
	}
	out := make([]Summary, 0, len(rows))
	for _, row := range rows {
		out = append(out, summary(row))
	}
	return out, nil
}

func (s *Service) Get(ctx context.Context, incidentID string) (Detail, *apperror.Error) {
	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" {
		return Detail{}, apperror.New(http.StatusBadRequest, 400001, "incident id is required")
	}
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return Detail{}, appErr
	}
	rows, err := s.queryIncidents(ctx, "WHERE i.workspace_id = ? AND i.uid = ? AND i.deleted_at IS NULL", workspace.ID, incidentID)
	if err != nil {
		return Detail{}, apperror.Wrap(http.StatusInternalServerError, 500902, "get incident failed", err)
	}
	if len(rows) == 0 {
		return Detail{}, apperror.New(http.StatusNotFound, 404901, "incident not found")
	}
	events, err := s.events(ctx, rows[0].ID)
	if err != nil {
		return Detail{}, apperror.Wrap(http.StatusInternalServerError, 500903, "list incident timeline failed", err)
	}
	meta := parseIncidentMetadata(rows[0].Metadata)
	return Detail{Summary: summary(rows[0]), Postmortem: meta.Postmortem, Events: events}, nil
}

func (s *Service) UpdateLifecycle(ctx context.Context, incidentID string, input LifecycleUpdateInput) (Detail, *apperror.Error) {
	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" {
		return Detail{}, apperror.New(http.StatusBadRequest, 400001, "incident id is required")
	}
	var updated Detail
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, appErr := s.workspace(ctx)
		if appErr != nil {
			return appErr
		}
		rows, err := (&Service{db: tx, cfg: s.cfg}).queryIncidents(ctx, "WHERE i.workspace_id = ? AND i.uid = ? AND i.deleted_at IS NULL", workspace.ID, incidentID)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return apperror.New(http.StatusNotFound, 404901, "incident not found")
		}
		current := rows[0]
		meta := parseIncidentMetadata(current.Metadata)
		meta.Owner = limit(strings.TrimSpace(input.Owner), 128)
		meta.ImpactScope = limit(strings.TrimSpace(input.ImpactScope), 1024)
		meta.RootCauseClass = limit(strings.TrimSpace(input.RootCauseClass), 128)
		meta.Postmortem = limit(strings.TrimSpace(input.Postmortem), 8192)

		if err := tx.WithContext(ctx).Exec(
			"UPDATE incidents SET metadata = ?, updated_at = NOW(3) WHERE id = ?",
			marshalIncidentMetadata(meta), current.ID,
		).Error; err != nil {
			return err
		}
		actorID, _ := audit.UserIDByUID(ctx, tx, input.Audit.ActorUID)
		if err := tx.WithContext(ctx).Exec(
			"INSERT INTO incident_events(incident_id, event_type, message, actor_id, payload) VALUES (?, 'lifecycle_updated', ?, ?, ?)",
			current.ID, nullString("incident lifecycle updated"), actorID, marshalIncidentMetadata(meta),
		).Error; err != nil {
			return err
		}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "incident.lifecycle.update",
			ResourceType:  "incident",
			ResourceID:    sql.NullInt64{Int64: int64(current.ID), Valid: true},
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			Metadata:      meta,
		})
		out, appErr := (&Service{db: tx, cfg: s.cfg}).Get(ctx, incidentID)
		if appErr != nil {
			return appErr
		}
		updated = out
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return Detail{}, appErr
		}
		return Detail{}, apperror.Wrap(http.StatusInternalServerError, 500906, "update incident lifecycle failed", txErr)
	}
	return updated, nil
}

func (s *Service) Merge(ctx context.Context, incidentID string, input MergeInput) (Detail, *apperror.Error) {
	incidentID = strings.TrimSpace(incidentID)
	targetID := strings.TrimSpace(input.TargetIncidentID)
	if incidentID == "" || targetID == "" {
		return Detail{}, apperror.New(http.StatusBadRequest, 400001, "incident id and targetIncidentId are required")
	}
	if incidentID == targetID {
		return Detail{}, apperror.New(http.StatusBadRequest, 400902, "incident cannot be merged into itself")
	}
	var merged Detail
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, appErr := s.workspace(ctx)
		if appErr != nil {
			return appErr
		}
		sourceRows, err := (&Service{db: tx, cfg: s.cfg}).queryIncidents(ctx, "WHERE i.workspace_id = ? AND i.uid = ? AND i.deleted_at IS NULL", workspace.ID, incidentID)
		if err != nil {
			return err
		}
		targetRows, err := (&Service{db: tx, cfg: s.cfg}).queryIncidents(ctx, "WHERE i.workspace_id = ? AND i.uid = ? AND i.deleted_at IS NULL", workspace.ID, targetID)
		if err != nil {
			return err
		}
		if len(sourceRows) == 0 || len(targetRows) == 0 {
			return apperror.New(http.StatusNotFound, 404901, "incident not found")
		}
		source := sourceRows[0]
		target := targetRows[0]

		sourceMeta := parseIncidentMetadata(source.Metadata)
		sourceMeta.MergedInto = target.UID
		if err := tx.WithContext(ctx).Exec(
			"UPDATE incidents SET status = 'resolved', resolved_at = NOW(3), last_seen_at = NOW(3), metadata = ?, updated_at = NOW(3) WHERE id = ?",
			marshalIncidentMetadata(sourceMeta), source.ID,
		).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT IGNORE INTO incident_alerts(incident_id, alert_id)
			 SELECT ?, alert_id FROM incident_alerts WHERE incident_id = ?`,
			target.ID, source.ID,
		).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			"DELETE FROM incident_alerts WHERE incident_id = ?",
			source.ID,
		).Error; err != nil {
			return err
		}
		actorID, _ := audit.UserIDByUID(ctx, tx, input.Audit.ActorUID)
		reason := strings.TrimSpace(input.Reason)
		if reason == "" {
			reason = "incident merged"
		}
		payload := map[string]string{"targetIncidentId": target.UID, "reason": reason}
		if err := tx.WithContext(ctx).Exec(
			"INSERT INTO incident_events(incident_id, event_type, message, actor_id, payload) VALUES (?, 'merged', ?, ?, ?)",
			source.ID, nullString(reason), actorID, jsonNull(payload),
		).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			"INSERT INTO incident_events(incident_id, event_type, message, actor_id, payload) VALUES (?, 'merged_from', ?, ?, ?)",
			target.ID, nullString(source.UID), actorID, jsonNull(map[string]string{"sourceIncidentId": source.UID}),
		).Error; err != nil {
			return err
		}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "incident.merge",
			ResourceType:  "incident",
			ResourceID:    sql.NullInt64{Int64: int64(source.ID), Valid: true},
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			Metadata:      payload,
		})
		out, appErr := (&Service{db: tx, cfg: s.cfg}).Get(ctx, incidentID)
		if appErr != nil {
			return appErr
		}
		merged = out
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return Detail{}, appErr
		}
		return Detail{}, apperror.Wrap(http.StatusInternalServerError, 500907, "merge incident failed", txErr)
	}
	return merged, nil
}

func (s *Service) Close(ctx context.Context, incidentID string, input CloseInput) (Detail, *apperror.Error) {
	incidentID = strings.TrimSpace(incidentID)
	if incidentID == "" {
		return Detail{}, apperror.New(http.StatusBadRequest, 400001, "incident id is required")
	}
	var closed Detail
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, appErr := s.workspace(ctx)
		if appErr != nil {
			return appErr
		}
		rows, err := (&Service{db: tx, cfg: s.cfg}).queryIncidents(ctx, "WHERE i.workspace_id = ? AND i.uid = ? AND i.deleted_at IS NULL", workspace.ID, incidentID)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return apperror.New(http.StatusNotFound, 404901, "incident not found")
		}
		current := rows[0]
		if err := tx.WithContext(ctx).Exec(
			"UPDATE incidents SET status = 'resolved', resolved_at = NOW(3), last_seen_at = NOW(3), updated_at = NOW(3) WHERE id = ?",
			current.ID,
		).Error; err != nil {
			return err
		}
		actorID, _ := audit.UserIDByUID(ctx, tx, input.Audit.ActorUID)
		reason := strings.TrimSpace(input.Reason)
		if reason == "" {
			reason = "incident closed"
		}
		if err := tx.WithContext(ctx).Exec(
			"INSERT INTO incident_events(incident_id, event_type, message, actor_id, payload) VALUES (?, 'closed', ?, ?, ?)",
			current.ID, nullString(reason), actorID, jsonNull(map[string]string{"reason": reason}),
		).Error; err != nil {
			return err
		}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "incident.close",
			ResourceType:  "incident",
			ResourceID:    sql.NullInt64{Int64: int64(current.ID), Valid: true},
			IP:            input.Audit.IP,
			UserAgent:     input.Audit.UserAgent,
			TraceID:       input.Audit.TraceID,
			RequestMethod: input.Audit.RequestMethod,
			RequestPath:   input.Audit.RequestPath,
			Metadata:      map[string]string{"reason": reason},
		})
		out, appErr := (&Service{db: tx, cfg: s.cfg}).Get(ctx, incidentID)
		if appErr != nil {
			return appErr
		}
		closed = out
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return Detail{}, appErr
		}
		return Detail{}, apperror.Wrap(http.StatusInternalServerError, 500908, "close incident failed", txErr)
	}
	return closed, nil
}

func (s *Service) queryIncidents(ctx context.Context, where string, args ...interface{}) ([]incidentRecord, error) {
	var rows []incidentRecord
	err := s.db.WithContext(ctx).Raw(
		`SELECT i.id, i.uid, MIN(a.uid) AS alert_uid, ar.uid AS rule_uid, ar.name AS rule_name, h.uid AS host_uid, h.name AS host_name,
		        i.title, i.message, i.severity, i.status, i.metadata,
		        DATE_FORMAT(i.first_seen_at, '%Y-%m-%d %H:%i:%s') AS first_seen_at,
		        DATE_FORMAT(i.last_seen_at, '%Y-%m-%d %H:%i:%s') AS last_seen_at,
		        DATE_FORMAT(i.resolved_at, '%Y-%m-%d %H:%i:%s') AS resolved_at,
		        COUNT(ia.id) AS alert_count
		   FROM incidents i
		   LEFT JOIN alert_rules ar ON ar.id = i.alert_rule_id
		   LEFT JOIN hosts h ON i.resource_type = 'host' AND h.id = i.resource_id
		   LEFT JOIN incident_alerts ia ON ia.incident_id = i.id
		   LEFT JOIN alerts a ON a.id = ia.alert_id
		  `+where+`
		  GROUP BY i.id, i.uid, ar.uid, ar.name, h.uid, h.name, i.title, i.message, i.severity, i.status, i.metadata,
		           i.first_seen_at, i.last_seen_at, i.resolved_at
		  ORDER BY CASE i.status WHEN 'open' THEN 0 WHEN 'acknowledged' THEN 1 WHEN 'silenced' THEN 2 ELSE 3 END,
		           i.last_seen_at DESC
		  LIMIT 200`,
		args...,
	).Scan(&rows).Error
	return rows, err
}

func (s *Service) events(ctx context.Context, incidentID uint64) ([]Event, error) {
	var rows []eventRecord
	err := s.db.WithContext(ctx).Raw(
		`SELECT ie.id, ie.event_type, ie.message, u.username AS actor, CAST(ie.payload AS CHAR) AS payload,
		        DATE_FORMAT(ie.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM incident_events ie
		   LEFT JOIN users u ON u.id = ie.actor_id
		  WHERE ie.incident_id = ?
		  ORDER BY ie.created_at ASC, ie.id ASC
		  LIMIT 200`,
		incidentID,
	).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]Event, 0, len(rows))
	for _, row := range rows {
		out = append(out, Event{
			ID:        row.ID,
			EventType: row.EventType,
			Message:   row.Message.String,
			Actor:     row.Actor.String,
			Payload:   row.Payload.String,
			CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

func (s *Service) workspace(ctx context.Context) (workspaceRecord, *apperror.Error) {
	var workspace workspaceRecord
	err := s.db.WithContext(ctx).Raw("SELECT id FROM workspaces WHERE slug = ? AND status = 'active' LIMIT 1", s.cfg.Bootstrap.WorkspaceSlug).Scan(&workspace).Error
	if err != nil {
		return workspaceRecord{}, apperror.Wrap(http.StatusInternalServerError, 500904, "load workspace failed", err)
	}
	if workspace.ID == 0 {
		return workspaceRecord{}, apperror.New(http.StatusInternalServerError, 500905, "default workspace is not initialized")
	}
	return workspace, nil
}

func summary(row incidentRecord) Summary {
	meta := parseIncidentMetadata(row.Metadata)
	return Summary{
		ID:          row.UID,
		AlertID:     row.AlertUID.String,
		Title:       row.Title,
		Severity:    row.Severity,
		Status:      row.Status,
		Owner:       meta.Owner,
		ImpactScope: meta.ImpactScope,
		RootCause:   meta.RootCauseClass,
		MergedInto:  meta.MergedInto,
		RuleID:      row.RuleUID.String,
		RuleName:    row.RuleName.String,
		HostID:      row.HostUID.String,
		HostName:    row.HostName.String,
		Message:     row.Message.String,
		FirstSeenAt: row.FirstSeenAt,
		LastSeenAt:  row.LastSeenAt,
		ResolvedAt:  row.ResolvedAt.String,
		AlertCount:  row.AlertCount,
	}
}

func parseIncidentMetadata(raw sql.NullString) incidentMetadata {
	if !raw.Valid || strings.TrimSpace(raw.String) == "" {
		return incidentMetadata{}
	}
	var out incidentMetadata
	if err := json.Unmarshal([]byte(raw.String), &out); err != nil {
		return incidentMetadata{}
	}
	out.Owner = strings.TrimSpace(out.Owner)
	out.ImpactScope = strings.TrimSpace(out.ImpactScope)
	out.RootCauseClass = strings.TrimSpace(out.RootCauseClass)
	out.Postmortem = strings.TrimSpace(out.Postmortem)
	out.MergedInto = strings.TrimSpace(out.MergedInto)
	return out
}

func marshalIncidentMetadata(value incidentMetadata) sql.NullString {
	bytes, err := json.Marshal(value)
	if err != nil || string(bytes) == "null" {
		return sql.NullString{}
	}
	return sql.NullString{String: string(bytes), Valid: true}
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

func limit(value string, max int) string {
	if len(value) <= max {
		return value
	}
	return value[:max]
}
