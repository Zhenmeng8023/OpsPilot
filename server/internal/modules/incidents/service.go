package incidents

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"gorm.io/gorm"

	"opspilot/server/internal/config"
	"opspilot/server/internal/shared/apperror"
)

type Service struct {
	db  *gorm.DB
	cfg config.Config
}

type ListInput struct {
	Status   string
	Severity string
}

type Summary struct {
	ID          string `json:"id"`
	AlertID     string `json:"alertId,omitempty"`
	Title       string `json:"title"`
	Severity    string `json:"severity"`
	Status      string `json:"status"`
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
	FirstSeenAt string
	LastSeenAt  string
	ResolvedAt  sql.NullString
	AlertCount  int
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
	return Detail{Summary: summary(rows[0]), Events: events}, nil
}

func (s *Service) queryIncidents(ctx context.Context, where string, args ...interface{}) ([]incidentRecord, error) {
	var rows []incidentRecord
	err := s.db.WithContext(ctx).Raw(
		`SELECT i.id, i.uid, MIN(a.uid) AS alert_uid, ar.uid AS rule_uid, ar.name AS rule_name, h.uid AS host_uid, h.name AS host_name,
		        i.title, i.message, i.severity, i.status,
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
		  GROUP BY i.id, i.uid, ar.uid, ar.name, h.uid, h.name, i.title, i.message, i.severity, i.status,
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
	return Summary{
		ID:          row.UID,
		AlertID:     row.AlertUID.String,
		Title:       row.Title,
		Severity:    row.Severity,
		Status:      row.Status,
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
