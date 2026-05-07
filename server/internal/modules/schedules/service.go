package schedules

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"gorm.io/gorm"

	"opspilot/server/internal/config"
	"opspilot/server/internal/modules/execution"
	"opspilot/server/internal/shared/apperror"
	"opspilot/server/internal/shared/audit"
	"opspilot/server/internal/shared/uid"
)

type Service struct {
	db   *gorm.DB
	cfg  config.Config
	repo repository
}

type AuditContext struct {
	ActorUID      string
	IP            string
	UserAgent     string
	TraceID       string
	RequestMethod string
	RequestPath   string
}

type ListInput struct {
	Keyword  string
	Status   string
	TaskID   string
	Page     int
	PageSize int
}

type CreateInput struct {
	Name     string
	TaskID   string
	CronExpr string
	Timezone string
	Audit    AuditContext
}

type ScheduleSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	TaskID       string `json:"taskId"`
	TaskName     string `json:"taskName"`
	ScheduleType string `json:"scheduleType"`
	CronExpr     string `json:"cronExpr"`
	Timezone     string `json:"timezone"`
	Status       string `json:"status"`
	NextFireAt   string `json:"nextFireAt,omitempty"`
	LastFireAt   string `json:"lastFireAt,omitempty"`
	CreatedBy    string `json:"createdBy,omitempty"`
	CreatedAt    string `json:"createdAt"`
}

type ScheduleListResult struct {
	Items    []ScheduleSummary `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

type FireResult struct {
	Fired  int `json:"fired"`
	Failed int `json:"failed"`
}

func NewService(db *gorm.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg, repo: newRepository(db)}
}

func (s *Service) List(ctx context.Context, input ListInput) (ScheduleListResult, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return ScheduleListResult{}, appErr
	}
	input.Page, input.PageSize = normalizePage(input.Page, input.PageSize)
	rows, total, err := s.repo.listSchedules(ctx, workspace.ID, listFilter{
		Keyword:  strings.TrimSpace(input.Keyword),
		Status:   strings.TrimSpace(input.Status),
		TaskUID:  strings.TrimSpace(input.TaskID),
		Page:     input.Page,
		PageSize: input.PageSize,
	})
	if err != nil {
		return ScheduleListResult{}, apperror.Wrap(http.StatusInternalServerError, 500401, "list schedules failed", err)
	}
	items := make([]ScheduleSummary, 0, len(rows))
	for _, row := range rows {
		items = append(items, summaryFromRecord(row))
	}
	return ScheduleListResult{Items: items, Total: total, Page: input.Page, PageSize: input.PageSize}, nil
}

func (s *Service) Create(ctx context.Context, input CreateInput) (ScheduleSummary, *apperror.Error) {
	name := strings.TrimSpace(input.Name)
	taskUID := strings.TrimSpace(input.TaskID)
	cronExpr := strings.TrimSpace(input.CronExpr)
	timezone := normalizeTimezone(input.Timezone)
	if name == "" {
		return ScheduleSummary{}, apperror.New(http.StatusBadRequest, 400401, "schedule name is required")
	}
	if taskUID == "" {
		return ScheduleSummary{}, apperror.New(http.StatusBadRequest, 400402, "taskId is required")
	}
	if cronExpr == "" {
		return ScheduleSummary{}, apperror.New(http.StatusBadRequest, 400403, "cronExpr is required")
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return ScheduleSummary{}, apperror.New(http.StatusBadRequest, 400404, "invalid timezone")
	}
	nextFireAt, err := nextCronTime(cronExpr, loc, time.Now())
	if err != nil {
		return ScheduleSummary{}, apperror.New(http.StatusBadRequest, 400405, "invalid cronExpr")
	}

	var created ScheduleSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := newRepository(tx)
		workspace, err := repo.defaultWorkspace(ctx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		if workspace.ID == 0 {
			return apperror.New(http.StatusInternalServerError, 500402, "default workspace is not initialized")
		}
		task, err := repo.taskByUID(ctx, workspace.ID, taskUID)
		if err != nil {
			return err
		}
		if task.ID == 0 {
			return apperror.New(http.StatusNotFound, 404401, "task definition not found")
		}
		actor, err := repo.userByUID(ctx, input.Audit.ActorUID)
		if err != nil {
			return err
		}
		actorID := sql.NullInt64{}
		if actor.ID != 0 {
			actorID = sql.NullInt64{Int64: int64(actor.ID), Valid: true}
		}
		scheduleUID, err := uid.New()
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO schedules(uid, workspace_id, task_id, name, schedule_type, cron_expr, timezone, status, next_fire_at, created_by)
			 VALUES (?, ?, ?, ?, 'cron', ?, ?, 'active', ?, ?)`,
			scheduleUID, workspace.ID, task.ID, name, cronExpr, timezone, nextFireAt, actorID,
		).Error; err != nil {
			return err
		}
		row, err := repo.scheduleByUID(ctx, workspace.ID, scheduleUID)
		if err != nil {
			return err
		}
		created = summaryFromRecord(row)
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        "schedule.create",
			ResourceType:  "schedule",
			ResourceID:    sql.NullInt64{Int64: int64(row.ID), Valid: row.ID != 0},
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
			return ScheduleSummary{}, appErr
		}
		if strings.Contains(strings.ToLower(txErr.Error()), "duplicate") {
			return ScheduleSummary{}, apperror.New(http.StatusConflict, 409401, "schedule name already exists")
		}
		return ScheduleSummary{}, apperror.Wrap(http.StatusInternalServerError, 500403, "create schedule failed", txErr)
	}
	return created, nil
}

func (s *Service) Pause(ctx context.Context, scheduleUID string, auditCtx AuditContext) *apperror.Error {
	return s.updateStatus(ctx, scheduleUID, "paused", "schedule.pause", auditCtx)
}

func (s *Service) Resume(ctx context.Context, scheduleUID string, auditCtx AuditContext) *apperror.Error {
	return s.updateStatus(ctx, scheduleUID, "active", "schedule.resume", auditCtx)
}

func (s *Service) Disable(ctx context.Context, scheduleUID string, auditCtx AuditContext) *apperror.Error {
	return s.updateStatus(ctx, scheduleUID, "disabled", "schedule.disable", auditCtx)
}

func (s *Service) updateStatus(ctx context.Context, scheduleUID, status, action string, auditCtx AuditContext) *apperror.Error {
	scheduleUID = strings.TrimSpace(scheduleUID)
	if scheduleUID == "" {
		return apperror.New(http.StatusBadRequest, 400001, "schedule id is required")
	}
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		repo := newRepository(tx)
		workspace, err := repo.defaultWorkspace(ctx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		row, err := repo.scheduleByUID(ctx, workspace.ID, scheduleUID)
		if err != nil {
			return err
		}
		if row.ID == 0 {
			return apperror.New(http.StatusNotFound, 404402, "schedule not found")
		}
		nextFireAt := sql.NullTime{}
		if status == "active" {
			loc, err := time.LoadLocation(normalizeTimezone(row.Timezone))
			if err != nil {
				return err
			}
			next, err := nextCronTime(row.CronExpr.String, loc, time.Now())
			if err != nil {
				return err
			}
			nextFireAt = sql.NullTime{Time: next, Valid: true}
		}
		if err := tx.WithContext(ctx).Exec(
			`UPDATE schedules
			    SET status = ?, next_fire_at = CASE WHEN ? = 'active' THEN ? ELSE next_fire_at END
			  WHERE id = ?`,
			status, status, nextFireAt, row.ID,
		).Error; err != nil {
			return err
		}
		actor, err := repo.userByUID(ctx, auditCtx.ActorUID)
		if err != nil {
			return err
		}
		actorID := sql.NullInt64{}
		if actor.ID != 0 {
			actorID = sql.NullInt64{Int64: int64(actor.ID), Valid: true}
		}
		audit.Write(ctx, tx, audit.Event{
			WorkspaceID:   workspace.ID,
			ActorUserID:   actorID,
			Action:        action,
			ResourceType:  "schedule",
			ResourceID:    sql.NullInt64{Int64: int64(row.ID), Valid: true},
			IP:            auditCtx.IP,
			UserAgent:     auditCtx.UserAgent,
			TraceID:       auditCtx.TraceID,
			RequestMethod: auditCtx.RequestMethod,
			RequestPath:   auditCtx.RequestPath,
			Before:        summaryFromRecord(row),
		})
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return appErr
		}
		return apperror.Wrap(http.StatusInternalServerError, 500404, "update schedule failed", txErr)
	}
	return nil
}

func (s *Service) FireDue(ctx context.Context, now time.Time, limit int) (FireResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var result FireResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var due []dueScheduleRecord
		if err := tx.WithContext(ctx).Raw(
			`SELECT s.id, s.workspace_id, s.task_id, s.name, s.cron_expr, s.timezone,
			        DATE_FORMAT(s.next_fire_at, '%Y-%m-%d %H:%i:%s') AS next_fire_at,
			        s.created_by AS created_by_id
			   FROM schedules s
			  WHERE s.status = 'active'
			    AND s.schedule_type = 'cron'
			    AND s.next_fire_at IS NOT NULL
			    AND s.next_fire_at <= ?
			    AND s.deleted_at IS NULL
			  ORDER BY s.next_fire_at ASC, s.id ASC
			  LIMIT ?
			  FOR UPDATE SKIP LOCKED`,
			now, limit,
		).Scan(&due).Error; err != nil {
			return err
		}
		for _, schedule := range due {
			if s.fireSchedule(ctx, tx, schedule, now) {
				result.Fired++
			} else {
				result.Failed++
			}
		}
		return nil
	})
	return result, err
}

func (s *Service) fireSchedule(ctx context.Context, tx *gorm.DB, schedule dueScheduleRecord, now time.Time) bool {
	plannedAt := parseDBTime(schedule.NextFireAt)
	if !plannedAt.Valid {
		plannedAt = sql.NullTime{Time: now, Valid: true}
	}
	var triggerID uint64
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO schedule_triggers(schedule_id, planned_fire_at, actual_fire_at, status)
		 VALUES (?, ?, ?, 'pending')`,
		schedule.ID, plannedAt, now,
	).Error; err != nil {
		return false
	}
	if err := tx.WithContext(ctx).Raw("SELECT LAST_INSERT_ID()").Scan(&triggerID).Error; err != nil {
		return false
	}
	runID, _, err := execution.CreateRunFromTask(
		ctx,
		tx,
		schedule.WorkspaceID,
		schedule.TaskID,
		"schedule",
		sql.NullInt64{Int64: int64(triggerID), Valid: true},
		schedule.CreatedByID,
	)
	if err != nil {
		_ = tx.WithContext(ctx).Exec(
			"UPDATE schedule_triggers SET status = 'failed', error_message = ? WHERE id = ?",
			limitString(err.Error(), 1024), triggerID,
		).Error
		_ = s.advanceSchedule(ctx, tx, schedule, now)
		return false
	}
	if err := tx.WithContext(ctx).Exec(
		"UPDATE schedule_triggers SET status = 'fired', task_run_id = ?, actual_fire_at = ? WHERE id = ?",
		runID, now, triggerID,
	).Error; err != nil {
		return false
	}
	if err := s.advanceSchedule(ctx, tx, schedule, now); err != nil {
		return false
	}
	return true
}

func (s *Service) advanceSchedule(ctx context.Context, tx *gorm.DB, schedule dueScheduleRecord, now time.Time) error {
	loc, err := time.LoadLocation(normalizeTimezone(schedule.Timezone))
	if err != nil {
		loc = time.Local
	}
	next, err := nextCronTime(schedule.CronExpr, loc, now)
	if err != nil {
		return tx.WithContext(ctx).Exec(
			"UPDATE schedules SET status = 'paused', last_fire_at = ?, next_fire_at = NULL WHERE id = ?",
			now, schedule.ID,
		).Error
	}
	return tx.WithContext(ctx).Exec(
		"UPDATE schedules SET last_fire_at = ?, next_fire_at = ? WHERE id = ?",
		now, next, schedule.ID,
	).Error
}

func (s *Service) workspace(ctx context.Context) (workspaceRecord, *apperror.Error) {
	workspace, err := s.repo.defaultWorkspace(ctx, s.cfg.Bootstrap.WorkspaceSlug)
	if err != nil {
		return workspaceRecord{}, apperror.Wrap(http.StatusInternalServerError, 500402, "load workspace failed", err)
	}
	if workspace.ID == 0 {
		return workspaceRecord{}, apperror.New(http.StatusInternalServerError, 500402, "default workspace is not initialized")
	}
	return workspace, nil
}

func summaryFromRecord(row scheduleRecord) ScheduleSummary {
	return ScheduleSummary{
		ID:           row.UID,
		Name:         row.Name,
		TaskID:       row.TaskUID,
		TaskName:     row.TaskName,
		ScheduleType: row.ScheduleType,
		CronExpr:     row.CronExpr.String,
		Timezone:     row.Timezone,
		Status:       row.Status,
		NextFireAt:   row.NextFireAt.String,
		LastFireAt:   row.LastFireAt.String,
		CreatedBy:    row.CreatedBy.String,
		CreatedAt:    row.CreatedAt,
	}
}

func normalizePage(page, pageSize int) (int, int) {
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

func normalizeTimezone(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "Asia/Shanghai"
	}
	return value
}

func parseDBTime(value string) sql.NullTime {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullTime{}
	}
	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
	if err != nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: parsed, Valid: true}
}

func limitString(value string, maxLen int) string {
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}
