package metrics

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"opspilot/server/internal/config"
	"opspilot/server/internal/modules/agents"
	"opspilot/server/internal/shared/apperror"
)

type Service struct {
	db  *gorm.DB
	cfg config.Config
}

type MetricInput struct {
	Code        string                 `json:"code"`
	Value       float64                `json:"value"`
	Unit        string                 `json:"unit,omitempty"`
	Dimensions  map[string]interface{} `json:"dimensions,omitempty"`
	CollectedAt string                 `json:"collectedAt,omitempty"`
}

type ListInput struct {
	HostID     string
	AgentID    string
	MetricCode string
	Limit      int
}

type MetricSummary struct {
	ID          uint64  `json:"id"`
	HostID      string  `json:"hostId"`
	HostName    string  `json:"hostName"`
	AgentID     string  `json:"agentId,omitempty"`
	AgentName   string  `json:"agentName,omitempty"`
	MetricCode  string  `json:"metricCode"`
	Value       float64 `json:"value"`
	Unit        string  `json:"unit,omitempty"`
	CollectedAt string  `json:"collectedAt"`
	CreatedAt   string  `json:"createdAt"`
}

type workspaceRecord struct {
	ID uint64
}

func NewService(db *gorm.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

func (s *Service) Upload(ctx context.Context, identity agents.AgentIdentity, metrics []MetricInput) *apperror.Error {
	if len(metrics) == 0 {
		return apperror.New(http.StatusBadRequest, 400601, "metrics cannot be empty")
	}
	if len(metrics) > 100 {
		return apperror.New(http.StatusBadRequest, 400602, "too many metrics")
	}
	for _, metric := range metrics {
		if strings.TrimSpace(metric.Code) == "" {
			return apperror.New(http.StatusBadRequest, 400603, "metric code is required")
		}
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, metric := range metrics {
			collectedAt := parseTime(metric.CollectedAt)
			if !collectedAt.Valid {
				collectedAt = sql.NullTime{Time: time.Now(), Valid: true}
			}
			if err := tx.WithContext(ctx).Exec(
				`INSERT INTO host_metrics(workspace_id, host_id, agent_id, metric_code, metric_value, unit, dimensions, collected_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
				identity.WorkspaceID,
				identity.HostID,
				identity.ID,
				strings.TrimSpace(metric.Code),
				metric.Value,
				nullString(metric.Unit),
				jsonNull(metric.Dimensions),
				collectedAt,
			).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return apperror.Wrap(http.StatusInternalServerError, 500601, "upload metrics failed", err)
	}
	return nil
}

func (s *Service) List(ctx context.Context, input ListInput) ([]MetricSummary, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	limit := input.Limit
	if limit <= 0 || limit > 1000 {
		limit = 200
	}
	args := []interface{}{workspace.ID}
	where := "WHERE hm.workspace_id = ?"
	if strings.TrimSpace(input.HostID) != "" {
		where += " AND h.uid = ?"
		args = append(args, strings.TrimSpace(input.HostID))
	}
	if strings.TrimSpace(input.AgentID) != "" {
		where += " AND a.uid = ?"
		args = append(args, strings.TrimSpace(input.AgentID))
	}
	if strings.TrimSpace(input.MetricCode) != "" {
		where += " AND hm.metric_code = ?"
		args = append(args, strings.TrimSpace(input.MetricCode))
	}
	args = append(args, limit)
	var rows []MetricSummary
	if err := s.db.WithContext(ctx).Raw(
		`SELECT hm.id, h.uid AS host_id, h.name AS host_name, a.uid AS agent_id, a.name AS agent_name,
		        hm.metric_code, hm.metric_value AS value, hm.unit,
		        DATE_FORMAT(hm.collected_at, '%Y-%m-%d %H:%i:%s') AS collected_at,
		        DATE_FORMAT(hm.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM host_metrics hm
		   JOIN hosts h ON h.id = hm.host_id
		   LEFT JOIN agents a ON a.id = hm.agent_id
		  `+where+`
		  ORDER BY hm.collected_at DESC, hm.id DESC
		  LIMIT ?`,
		args...,
	).Scan(&rows).Error; err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500602, "list metrics failed", err)
	}
	return rows, nil
}

func (s *Service) workspace(ctx context.Context) (workspaceRecord, *apperror.Error) {
	var workspace workspaceRecord
	err := s.db.WithContext(ctx).Raw(
		"SELECT id FROM workspaces WHERE slug = ? AND status = 'active' LIMIT 1",
		s.cfg.Bootstrap.WorkspaceSlug,
	).Scan(&workspace).Error
	if err != nil {
		return workspaceRecord{}, apperror.Wrap(http.StatusInternalServerError, 500603, "load workspace failed", err)
	}
	if workspace.ID == 0 {
		return workspaceRecord{}, apperror.New(http.StatusInternalServerError, 500603, "default workspace is not initialized")
	}
	return workspace, nil
}

func parseTime(value string) sql.NullTime {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullTime{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05"} {
		parsed, err := time.Parse(layout, value)
		if err == nil {
			return sql.NullTime{Time: parsed, Valid: true}
		}
	}
	return sql.NullTime{}
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
