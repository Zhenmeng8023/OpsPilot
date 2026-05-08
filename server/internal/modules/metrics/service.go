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
	"opspilot/server/internal/shared/uid"
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

type TrendInput struct {
	HostID      string
	AgentID     string
	MetricCode  string
	Hours       int
	Limit       int
	Granularity string
}

type DashboardInput struct {
	Name        string `json:"name"`
	MetricCode  string `json:"metricCode"`
	HostID      string `json:"hostId"`
	AgentID     string `json:"agentId"`
	RangeHours  int    `json:"rangeHours"`
	PointLimit  int    `json:"pointLimit"`
	Granularity string `json:"granularity"`
	Status      string `json:"status"`
	ActorUID    string
}

type RollupInput struct {
	Interval string `json:"interval"`
	Hours    int    `json:"hours"`
}

type RollupResult struct {
	Interval string `json:"interval"`
	From     string `json:"from"`
	To       string `json:"to"`
	Matched  int64  `json:"matched"`
	Upserted int64  `json:"upserted"`
}

type RetentionInput struct {
	DetailDays int  `json:"detailDays"`
	RollupDays int  `json:"rollupDays"`
	DryRun     bool `json:"dryRun"`
}

type RetentionResult struct {
	DetailCutoffAt string `json:"detailCutoffAt"`
	RollupCutoffAt string `json:"rollupCutoffAt"`
	DetailDays     int    `json:"detailDays"`
	RollupDays     int    `json:"rollupDays"`
	DetailMatched  int64  `json:"detailMatched"`
	RollupMatched  int64  `json:"rollupMatched"`
	DetailDeleted  int64  `json:"detailDeleted"`
	RollupDeleted  int64  `json:"rollupDeleted"`
	DryRun         bool   `json:"dryRun"`
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

type MetricTrendPoint struct {
	CollectedAt string  `json:"collectedAt"`
	Value       float64 `json:"value"`
}

type MetricTrendSeries struct {
	HostID      string             `json:"hostId"`
	HostName    string             `json:"hostName"`
	AgentID     string             `json:"agentId,omitempty"`
	AgentName   string             `json:"agentName,omitempty"`
	MetricCode  string             `json:"metricCode"`
	Unit        string             `json:"unit,omitempty"`
	LatestValue float64            `json:"latestValue"`
	MinValue    float64            `json:"minValue"`
	MaxValue    float64            `json:"maxValue"`
	Points      []MetricTrendPoint `json:"points"`
	Granularity string             `json:"granularity"`
}

type DashboardSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	MetricCode  string `json:"metricCode,omitempty"`
	HostID      string `json:"hostId,omitempty"`
	AgentID     string `json:"agentId,omitempty"`
	RangeHours  int    `json:"rangeHours"`
	PointLimit  int    `json:"pointLimit"`
	Granularity string `json:"granularity"`
	Status      string `json:"status"`
	CreatedBy   string `json:"createdBy,omitempty"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type workspaceRecord struct {
	ID uint64
}

type trendMetricRow struct {
	ID          uint64
	HostID      string
	HostName    string
	AgentID     string
	AgentName   string
	MetricCode  string
	Value       float64
	Unit        string
	CollectedAt string
}

type dashboardRecord struct {
	UID         string
	Name        string
	MetricCode  string
	HostUID     string
	AgentUID    string
	RangeHours  int
	PointLimit  int
	Granularity string
	Status      string
	CreatedBy   string
	CreatedAt   string
	UpdatedAt   string
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

func (s *Service) ListTrends(ctx context.Context, input TrendInput) ([]MetricTrendSeries, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	metricCode := strings.TrimSpace(input.MetricCode)
	if metricCode == "" {
		return nil, apperror.New(http.StatusBadRequest, 400604, "metric code is required")
	}
	hours := normalizeTrendHours(input.Hours)
	limit := normalizeTrendPointLimit(input.Limit)
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)
	granularity := normalizeTrendGranularity(input.Granularity, hours)

	if granularity == "5m" || granularity == "1h" {
		rollupSeries, rollupErr := s.listRollupTrends(ctx, workspace.ID, input, metricCode, limit, granularity, startTime)
		if rollupErr != nil {
			return nil, rollupErr
		}
		if len(rollupSeries) > 0 || explicitTrendGranularity(input.Granularity) {
			return rollupSeries, nil
		}
	}

	args := []interface{}{workspace.ID, metricCode, startTime}
	where := "WHERE hm.workspace_id = ? AND hm.metric_code = ? AND hm.collected_at >= ?"
	if strings.TrimSpace(input.HostID) != "" {
		where += " AND h.uid = ?"
		args = append(args, strings.TrimSpace(input.HostID))
	}
	if strings.TrimSpace(input.AgentID) != "" {
		where += " AND a.uid = ?"
		args = append(args, strings.TrimSpace(input.AgentID))
	}
	args = append(args, limit)

	var rows []trendMetricRow
	if err := s.db.WithContext(ctx).Raw(
		`SELECT ranked.id, ranked.host_id, ranked.host_name, ranked.agent_id, ranked.agent_name,
		        ranked.metric_code, ranked.value, ranked.unit, ranked.collected_at
		   FROM (
		     SELECT hm.id, h.uid AS host_id, h.name AS host_name, a.uid AS agent_id, a.name AS agent_name,
		            hm.metric_code, hm.metric_value AS value, hm.unit,
		            DATE_FORMAT(hm.collected_at, '%Y-%m-%d %H:%i:%s') AS collected_at,
		            hm.collected_at AS raw_collected_at,
		            ROW_NUMBER() OVER (PARTITION BY hm.host_id, hm.metric_code ORDER BY hm.collected_at DESC, hm.id DESC) AS rn
		       FROM host_metrics hm
		       JOIN hosts h ON h.id = hm.host_id
		       LEFT JOIN agents a ON a.id = hm.agent_id
		      `+where+`
		   ) ranked
		  WHERE ranked.rn <= ?
		  ORDER BY ranked.host_name ASC, ranked.raw_collected_at ASC, ranked.id ASC`,
		args...,
	).Scan(&rows).Error; err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500604, "list metric trends failed", err)
	}

	seriesByKey := make(map[string]*MetricTrendSeries)
	order := make([]string, 0)
	for _, row := range rows {
		key := row.HostID + ":" + row.MetricCode
		item, exists := seriesByKey[key]
		if !exists {
			item = &MetricTrendSeries{
				HostID:      row.HostID,
				HostName:    row.HostName,
				AgentID:     row.AgentID,
				AgentName:   row.AgentName,
				MetricCode:  row.MetricCode,
				Unit:        row.Unit,
				MinValue:    row.Value,
				MaxValue:    row.Value,
				LatestValue: row.Value,
				Granularity: "raw",
				Points:      make([]MetricTrendPoint, 0, limit),
			}
			seriesByKey[key] = item
			order = append(order, key)
		}
		if item.Unit == "" && row.Unit != "" {
			item.Unit = row.Unit
		}
		if item.AgentID == "" && row.AgentID != "" {
			item.AgentID = row.AgentID
			item.AgentName = row.AgentName
		}
		if row.Value < item.MinValue {
			item.MinValue = row.Value
		}
		if row.Value > item.MaxValue {
			item.MaxValue = row.Value
		}
		item.LatestValue = row.Value
		item.Points = append(item.Points, MetricTrendPoint{
			CollectedAt: row.CollectedAt,
			Value:       row.Value,
		})
	}

	result := make([]MetricTrendSeries, 0, len(order))
	for _, key := range order {
		result = append(result, *seriesByKey[key])
	}
	return result, nil
}

func (s *Service) ListDashboards(ctx context.Context) ([]DashboardSummary, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	rows, err := s.dashboards(ctx, workspace.ID, "")
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500605, "list metric dashboards failed", err)
	}
	result := make([]DashboardSummary, 0, len(rows))
	for _, row := range rows {
		result = append(result, dashboardSummary(row))
	}
	return result, nil
}

func (s *Service) CreateDashboard(ctx context.Context, input DashboardInput) (DashboardSummary, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return DashboardSummary{}, appErr
	}
	normalized, appErr := normalizeDashboardInput(input, true)
	if appErr != nil {
		return DashboardSummary{}, appErr
	}
	dashboardUID, err := uid.New()
	if err != nil {
		return DashboardSummary{}, apperror.Wrap(http.StatusInternalServerError, 500606, "generate metric dashboard id failed", err)
	}
	creatorID, err := s.userIDByUID(ctx, input.ActorUID)
	if err != nil {
		return DashboardSummary{}, apperror.Wrap(http.StatusInternalServerError, 500607, "load dashboard creator failed", err)
	}
	if err := s.db.WithContext(ctx).Exec(
		`INSERT INTO metric_dashboards(uid, workspace_id, name, metric_code, host_uid, agent_uid, range_hours, point_limit, granularity, status, created_by)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		dashboardUID, workspace.ID, normalized.Name, nullString(normalized.MetricCode), nullString(normalized.HostID),
		nullString(normalized.AgentID), normalized.RangeHours, normalized.PointLimit, normalized.Granularity, normalized.Status, nullUint64(creatorID),
	).Error; err != nil {
		if isDuplicateError(err) {
			return DashboardSummary{}, apperror.New(http.StatusConflict, 409601, "metric dashboard already exists")
		}
		return DashboardSummary{}, apperror.Wrap(http.StatusInternalServerError, 500608, "create metric dashboard failed", err)
	}
	rows, err := s.dashboards(ctx, workspace.ID, dashboardUID)
	if err != nil {
		return DashboardSummary{}, apperror.Wrap(http.StatusInternalServerError, 500605, "load metric dashboard failed", err)
	}
	if len(rows) == 0 {
		return DashboardSummary{}, apperror.New(http.StatusInternalServerError, 500605, "metric dashboard was not created")
	}
	return dashboardSummary(rows[0]), nil
}

func (s *Service) UpdateDashboard(ctx context.Context, dashboardUID string, input DashboardInput) (DashboardSummary, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return DashboardSummary{}, appErr
	}
	dashboardUID = strings.TrimSpace(dashboardUID)
	if dashboardUID == "" {
		return DashboardSummary{}, apperror.New(http.StatusBadRequest, 400605, "metric dashboard id is required")
	}
	normalized, appErr := normalizeDashboardInput(input, false)
	if appErr != nil {
		return DashboardSummary{}, appErr
	}
	exec := s.db.WithContext(ctx).Exec(
		`UPDATE metric_dashboards
		    SET name = ?, metric_code = ?, host_uid = ?, agent_uid = ?, range_hours = ?, point_limit = ?, granularity = ?, status = ?
		  WHERE workspace_id = ? AND uid = ? AND deleted_at IS NULL`,
		normalized.Name, nullString(normalized.MetricCode), nullString(normalized.HostID), nullString(normalized.AgentID),
		normalized.RangeHours, normalized.PointLimit, normalized.Granularity, normalized.Status, workspace.ID, dashboardUID,
	)
	if exec.Error != nil {
		if isDuplicateError(exec.Error) {
			return DashboardSummary{}, apperror.New(http.StatusConflict, 409601, "metric dashboard already exists")
		}
		return DashboardSummary{}, apperror.Wrap(http.StatusInternalServerError, 500609, "update metric dashboard failed", exec.Error)
	}
	if exec.RowsAffected == 0 {
		return DashboardSummary{}, apperror.New(http.StatusNotFound, 404601, "metric dashboard not found")
	}
	rows, err := s.dashboards(ctx, workspace.ID, dashboardUID)
	if err != nil {
		return DashboardSummary{}, apperror.Wrap(http.StatusInternalServerError, 500605, "load metric dashboard failed", err)
	}
	if len(rows) == 0 {
		return DashboardSummary{}, apperror.New(http.StatusNotFound, 404601, "metric dashboard not found")
	}
	return dashboardSummary(rows[0]), nil
}

func (s *Service) RunRollup(ctx context.Context, input RollupInput) (RollupResult, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return RollupResult{}, appErr
	}
	interval, seconds := normalizeRollupInterval(input.Interval)
	if interval == "" {
		return RollupResult{}, apperror.New(http.StatusBadRequest, 400606, "unsupported metric rollup interval")
	}
	hours := input.Hours
	if hours <= 0 {
		hours = 24
	}
	if hours > 24*30 {
		hours = 24 * 30
	}
	to := time.Now()
	from := to.Add(-time.Duration(hours) * time.Hour)
	result := RollupResult{
		Interval: interval,
		From:     from.Format("2006-01-02 15:04:05"),
		To:       to.Format("2006-01-02 15:04:05"),
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Raw(
			"SELECT COUNT(*) FROM host_metrics WHERE workspace_id = ? AND collected_at >= ? AND collected_at < ?",
			workspace.ID, from, to,
		).Scan(&result.Matched).Error; err != nil {
			return err
		}
		exec := tx.WithContext(ctx).Exec(
			`INSERT INTO host_metric_rollups(workspace_id, host_id, agent_id, metric_code, interval_type, bucket_start, avg_value, min_value, max_value, sample_count, unit)
			 SELECT workspace_id, host_id, MAX(agent_id), metric_code, ?,
			        FROM_UNIXTIME(FLOOR(UNIX_TIMESTAMP(collected_at) / ?) * ?) AS bucket_start,
			        AVG(metric_value), MIN(metric_value), MAX(metric_value), COUNT(*), MAX(unit)
			   FROM host_metrics
			  WHERE workspace_id = ? AND collected_at >= ? AND collected_at < ?
			  GROUP BY workspace_id, host_id, metric_code, bucket_start
			  ON DUPLICATE KEY UPDATE
			        agent_id = VALUES(agent_id),
			        avg_value = VALUES(avg_value),
			        min_value = VALUES(min_value),
			        max_value = VALUES(max_value),
			        sample_count = VALUES(sample_count),
			        unit = VALUES(unit)`,
			interval, seconds, seconds, workspace.ID, from, to,
		)
		if exec.Error != nil {
			return exec.Error
		}
		result.Upserted = exec.RowsAffected
		return nil
	})
	if err != nil {
		return RollupResult{}, apperror.Wrap(http.StatusInternalServerError, 500610, "run metric rollup failed", err)
	}
	return result, nil
}

func (s *Service) RunRetention(ctx context.Context, input RetentionInput) (RetentionResult, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return RetentionResult{}, appErr
	}
	detailDays := input.DetailDays
	if detailDays <= 0 {
		detailDays = s.cfg.Metric.DetailRetentionDays
	}
	rollupDays := input.RollupDays
	if rollupDays <= 0 {
		rollupDays = s.cfg.Metric.RollupRetentionDays
	}
	detailCutoff := time.Now().Add(-time.Duration(detailDays) * 24 * time.Hour)
	rollupCutoff := time.Now().Add(-time.Duration(rollupDays) * 24 * time.Hour)
	result := RetentionResult{
		DetailCutoffAt: detailCutoff.Format("2006-01-02 15:04:05"),
		RollupCutoffAt: rollupCutoff.Format("2006-01-02 15:04:05"),
		DetailDays:     detailDays,
		RollupDays:     rollupDays,
		DryRun:         input.DryRun,
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.WithContext(ctx).Raw(
			"SELECT COUNT(*) FROM host_metrics WHERE workspace_id = ? AND collected_at < ?",
			workspace.ID, detailCutoff,
		).Scan(&result.DetailMatched).Error; err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Raw(
			"SELECT COUNT(*) FROM host_metric_rollups WHERE workspace_id = ? AND bucket_start < ?",
			workspace.ID, rollupCutoff,
		).Scan(&result.RollupMatched).Error; err != nil {
			return err
		}
		if input.DryRun {
			return nil
		}
		if result.DetailMatched > 0 {
			exec := tx.WithContext(ctx).Exec(
				"DELETE FROM host_metrics WHERE workspace_id = ? AND collected_at < ?",
				workspace.ID, detailCutoff,
			)
			if exec.Error != nil {
				return exec.Error
			}
			result.DetailDeleted = exec.RowsAffected
		}
		if result.RollupMatched > 0 {
			exec := tx.WithContext(ctx).Exec(
				"DELETE FROM host_metric_rollups WHERE workspace_id = ? AND bucket_start < ?",
				workspace.ID, rollupCutoff,
			)
			if exec.Error != nil {
				return exec.Error
			}
			result.RollupDeleted = exec.RowsAffected
		}
		return nil
	})
	if err != nil {
		return RetentionResult{}, apperror.Wrap(http.StatusInternalServerError, 500611, "run metric retention failed", err)
	}
	return result, nil
}

func (s *Service) listRollupTrends(ctx context.Context, workspaceID uint64, input TrendInput, metricCode string, limit int, granularity string, startTime time.Time) ([]MetricTrendSeries, *apperror.Error) {
	args := []interface{}{workspaceID, metricCode, granularity, startTime}
	where := "WHERE mr.workspace_id = ? AND mr.metric_code = ? AND mr.interval_type = ? AND mr.bucket_start >= ?"
	if strings.TrimSpace(input.HostID) != "" {
		where += " AND h.uid = ?"
		args = append(args, strings.TrimSpace(input.HostID))
	}
	if strings.TrimSpace(input.AgentID) != "" {
		where += " AND a.uid = ?"
		args = append(args, strings.TrimSpace(input.AgentID))
	}
	args = append(args, limit)

	var rows []trendMetricRow
	if err := s.db.WithContext(ctx).Raw(
		`SELECT ranked.id, ranked.host_id, ranked.host_name, ranked.agent_id, ranked.agent_name,
		        ranked.metric_code, ranked.value, ranked.unit, ranked.collected_at
		   FROM (
		     SELECT mr.id, h.uid AS host_id, h.name AS host_name, a.uid AS agent_id, a.name AS agent_name,
		            mr.metric_code, mr.avg_value AS value, mr.unit,
		            DATE_FORMAT(mr.bucket_start, '%Y-%m-%d %H:%i:%s') AS collected_at,
		            mr.bucket_start AS raw_collected_at,
		            ROW_NUMBER() OVER (PARTITION BY mr.host_id, mr.metric_code ORDER BY mr.bucket_start DESC, mr.id DESC) AS rn
		       FROM host_metric_rollups mr
		       JOIN hosts h ON h.id = mr.host_id
		       LEFT JOIN agents a ON a.id = mr.agent_id
		      `+where+`
		   ) ranked
		  WHERE ranked.rn <= ?
		  ORDER BY ranked.host_name ASC, ranked.raw_collected_at ASC, ranked.id ASC`,
		args...,
	).Scan(&rows).Error; err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500604, "list metric rollup trends failed", err)
	}
	return buildTrendSeries(rows, limit, granularity), nil
}

func buildTrendSeries(rows []trendMetricRow, limit int, granularity string) []MetricTrendSeries {
	seriesByKey := make(map[string]*MetricTrendSeries)
	order := make([]string, 0)
	for _, row := range rows {
		key := row.HostID + ":" + row.MetricCode
		item, exists := seriesByKey[key]
		if !exists {
			item = &MetricTrendSeries{
				HostID:      row.HostID,
				HostName:    row.HostName,
				AgentID:     row.AgentID,
				AgentName:   row.AgentName,
				MetricCode:  row.MetricCode,
				Unit:        row.Unit,
				MinValue:    row.Value,
				MaxValue:    row.Value,
				LatestValue: row.Value,
				Granularity: granularity,
				Points:      make([]MetricTrendPoint, 0, limit),
			}
			seriesByKey[key] = item
			order = append(order, key)
		}
		if item.Unit == "" && row.Unit != "" {
			item.Unit = row.Unit
		}
		if item.AgentID == "" && row.AgentID != "" {
			item.AgentID = row.AgentID
			item.AgentName = row.AgentName
		}
		if row.Value < item.MinValue {
			item.MinValue = row.Value
		}
		if row.Value > item.MaxValue {
			item.MaxValue = row.Value
		}
		item.LatestValue = row.Value
		item.Points = append(item.Points, MetricTrendPoint{
			CollectedAt: row.CollectedAt,
			Value:       row.Value,
		})
	}
	result := make([]MetricTrendSeries, 0, len(order))
	for _, key := range order {
		result = append(result, *seriesByKey[key])
	}
	return result
}

func (s *Service) dashboards(ctx context.Context, workspaceID uint64, dashboardUID string) ([]dashboardRecord, error) {
	args := []interface{}{workspaceID}
	where := "WHERE md.workspace_id = ? AND md.deleted_at IS NULL"
	if dashboardUID != "" {
		where += " AND md.uid = ?"
		args = append(args, dashboardUID)
	}
	var rows []dashboardRecord
	err := s.db.WithContext(ctx).Raw(
		`SELECT md.uid, md.name, COALESCE(md.metric_code, '') AS metric_code,
		        COALESCE(md.host_uid, '') AS host_uid, COALESCE(md.agent_uid, '') AS agent_uid,
		        md.range_hours, md.point_limit, md.granularity, md.status,
		        COALESCE(u.username, '') AS created_by,
		        DATE_FORMAT(md.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
		        DATE_FORMAT(md.updated_at, '%Y-%m-%d %H:%i:%s') AS updated_at
		   FROM metric_dashboards md
		   LEFT JOIN users u ON u.id = md.created_by
		  `+where+`
		  ORDER BY md.updated_at DESC, md.id DESC`,
		args...,
	).Scan(&rows).Error
	return rows, err
}

func dashboardSummary(row dashboardRecord) DashboardSummary {
	return DashboardSummary{
		ID:          row.UID,
		Name:        row.Name,
		MetricCode:  row.MetricCode,
		HostID:      row.HostUID,
		AgentID:     row.AgentUID,
		RangeHours:  row.RangeHours,
		PointLimit:  row.PointLimit,
		Granularity: row.Granularity,
		Status:      row.Status,
		CreatedBy:   row.CreatedBy,
		CreatedAt:   row.CreatedAt,
		UpdatedAt:   row.UpdatedAt,
	}
}

type normalizedDashboardInput struct {
	Name        string
	MetricCode  string
	HostID      string
	AgentID     string
	RangeHours  int
	PointLimit  int
	Granularity string
	Status      string
}

func normalizeDashboardInput(input DashboardInput, creating bool) (normalizedDashboardInput, *apperror.Error) {
	out := normalizedDashboardInput{
		Name:        strings.TrimSpace(input.Name),
		MetricCode:  strings.TrimSpace(input.MetricCode),
		HostID:      strings.TrimSpace(input.HostID),
		AgentID:     strings.TrimSpace(input.AgentID),
		RangeHours:  normalizeTrendHours(input.RangeHours),
		PointLimit:  normalizeTrendPointLimit(input.PointLimit),
		Granularity: normalizeDashboardGranularity(input.Granularity),
		Status:      strings.TrimSpace(input.Status),
	}
	if out.Name == "" {
		return out, apperror.New(http.StatusBadRequest, 400607, "metric dashboard name is required")
	}
	if creating && out.MetricCode == "" {
		return out, apperror.New(http.StatusBadRequest, 400604, "metric code is required")
	}
	if out.Status == "" {
		out.Status = "active"
	}
	if out.Status != "active" && out.Status != "disabled" && out.Status != "archived" {
		return out, apperror.New(http.StatusBadRequest, 400608, "unsupported metric dashboard status")
	}
	return out, nil
}

func (s *Service) userIDByUID(ctx context.Context, userUID string) (uint64, error) {
	userUID = strings.TrimSpace(userUID)
	if userUID == "" {
		return 0, nil
	}
	var id uint64
	err := s.db.WithContext(ctx).Raw("SELECT id FROM users WHERE uid = ? LIMIT 1", userUID).Scan(&id).Error
	return id, err
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

func nullUint64(value uint64) sql.NullInt64 {
	if value == 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(value), Valid: true}
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

func normalizeTrendHours(value int) int {
	if value <= 0 {
		return 24
	}
	if value > 24*14 {
		return 24 * 14
	}
	return value
}

func normalizeTrendPointLimit(value int) int {
	if value <= 0 {
		return 120
	}
	if value < 10 {
		return 10
	}
	if value > 240 {
		return 240
	}
	return value
}

func normalizeTrendGranularity(value string, hours int) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "raw":
		return "raw"
	case "5m":
		return "5m"
	case "1h":
		return "1h"
	case "", "auto":
		if hours > 72 {
			return "1h"
		}
		if hours > 24 {
			return "5m"
		}
		return "raw"
	default:
		return "raw"
	}
}

func explicitTrendGranularity(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "raw", "5m", "1h":
		return true
	default:
		return false
	}
}

func normalizeDashboardGranularity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "raw", "5m", "1h":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "auto"
	}
}

func normalizeRollupInterval(value string) (string, int) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "5m", "":
		return "5m", 300
	case "1h":
		return "1h", 3600
	default:
		return "", 0
	}
}

func isDuplicateError(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "duplicate") || strings.Contains(message, "unique")
}
