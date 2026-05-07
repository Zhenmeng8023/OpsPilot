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
	Name          string
	TaskID        string
	CronExpr      string
	Timezone      string
	MisfirePolicy string
	Audit         AuditContext
}

type PreviewInput struct {
	CronExpr string
	Timezone string
	Count    int
}

type ScheduleSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	TaskID       string `json:"taskId"`
	TaskName     string `json:"taskName"`
	ScheduleType string `json:"scheduleType"`
	CronExpr     string `json:"cronExpr"`
	Timezone     string `json:"timezone"`
	MisfirePolicy string `json:"misfirePolicy"`
	Status       string `json:"status"`
	NextFireAt   string `json:"nextFireAt,omitempty"`
	LastFireAt   string `json:"lastFireAt,omitempty"`
	CreatedBy    string `json:"createdBy,omitempty"`
	CreatedAt    string `json:"createdAt"`
}

type PreviewResult struct {
	Times []string `json:"times"`
}

type TriggerSummary struct {
	ID            uint64 `json:"id"`
	TaskRunID     string `json:"taskRunId,omitempty"`
	PlannedFireAt string `json:"plannedFireAt"`
	ActualFireAt  string `json:"actualFireAt,omitempty"`
	Status        string `json:"status"`
	ErrorMessage  string `json:"errorMessage,omitempty"`
	CreatedAt     string `json:"createdAt"`
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
	misfirePolicy := normalizeMisfirePolicy(input.MisfirePolicy)
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
			`INSERT INTO schedules(uid, workspace_id, task_id, name, schedule_type, cron_expr, timezone, misfire_policy, status, next_fire_at, created_by)
			 VALUES (?, ?, ?, ?, 'cron', ?, ?, ?, 'active', ?, ?)`,
			scheduleUID, workspace.ID, task.ID, name, cronExpr, timezone, misfirePolicy, nextFireAt, actorID,
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

func (s *Service) Preview(ctx context.Context, input PreviewInput) (PreviewResult, *apperror.Error) {
	cronExpr := strings.TrimSpace(input.CronExpr)
	timezone := normalizeTimezone(input.Timezone)
	if cronExpr == "" {
		return PreviewResult{}, apperror.New(http.StatusBadRequest, 400403, "cronExpr is required")
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return PreviewResult{}, apperror.New(http.StatusBadRequest, 400404, "invalid timezone")
	}
	count := input.Count
	if count <= 0 {
		count = 5
	}
	if count > 20 {
		count = 20
	}
	times, err := nextCronTimes(cronExpr, loc, time.Now(), count)
	if err != nil {
		return PreviewResult{}, apperror.New(http.StatusBadRequest, 400405, "invalid cronExpr")
	}
	out := make([]string, 0, len(times))
	for _, item := range times {
		out = append(out, item.In(loc).Format("2006-01-02 15:04:05"))
	}
	return PreviewResult{Times: out}, nil
}

func (s *Service) ListTriggers(ctx context.Context, scheduleUID string, limit int) ([]TriggerSummary, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	scheduleUID = strings.TrimSpace(scheduleUID)
	if scheduleUID == "" {
		return nil, apperror.New(http.StatusBadRequest, 400001, "schedule id is required")
	}
	row, err := s.repo.scheduleByUID(ctx, workspace.ID, scheduleUID)
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500405, "load schedule failed", err)
	}
	if row.ID == 0 {
		return nil, apperror.New(http.StatusNotFound, 404402, "schedule not found")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.repo.listTriggers(ctx, row.ID, limit)
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500406, "list schedule triggers failed", err)
	}
	out := make([]TriggerSummary, 0, len(rows))
	for _, trigger := range rows {
		out = append(out, TriggerSummary{
			ID:            trigger.ID,
			TaskRunID:     trigger.TaskRunUID.String,
			PlannedFireAt: trigger.PlannedFireAt,
			ActualFireAt:  trigger.ActualFireAt.String,
			Status:        trigger.Status,
			ErrorMessage:  trigger.ErrorMessage.String,
			CreatedAt:     trigger.CreatedAt,
		})
	}
	return out, nil
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
			        s.misfire_policy,
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
			fired, failed := s.processDueSchedule(ctx, tx, schedule, now)
			result.Fired += fired
			result.Failed += failed
		}
		return nil
	})
	return result, err
}

func (s *Service) processDueSchedule(ctx context.Context, tx *gorm.DB, schedule dueScheduleRecord, now time.Time) (int, int) {
	switch normalizeMisfirePolicy(schedule.MisfirePolicy) {
	case "skip":
		if s.skipSchedule(ctx, tx, schedule, now) {
			return 0, 0
		}
		return 0, 1
	case "fire_all":
		return s.fireAllMissed(ctx, tx, schedule, now)
	default:
		if s.fireSchedule(ctx, tx, schedule, now) {
			return 1, 0
		}
		return 0, 1
	}
}

func (s *Service) fireAllMissed(ctx context.Context, tx *gorm.DB, schedule dueScheduleRecord, now time.Time) (int, int) {
	loc, err := time.LoadLocation(normalizeTimezone(schedule.Timezone))
	if err != nil {
		loc = time.Local
	}
	fired, failed := 0, 0
	current := schedule
	for attempts := 0; attempts < 20; attempts++ {
		plannedAt := parseDBTime(current.NextFireAt)
		if !plannedAt.Valid || plannedAt.Time.After(now) {
			break
		}
		if s.fireScheduleWithoutAdvance(ctx, tx, current, plannedAt.Time, now) {
			fired++
		} else {
			failed++
		}
		next, err := nextCronTime(schedule.CronExpr, loc, plannedAt.Time)
		if err != nil || next.After(now) {
			break
		}
		current.NextFireAt = next.Format("2006-01-02 15:04:05")
	}
	if err := s.advanceSchedule(ctx, tx, schedule, now); err != nil {
		failed++
	}
	return fired, failed
}

func (s *Service) skipSchedule(ctx context.Context, tx *gorm.DB, schedule dueScheduleRecord, now time.Time) bool {
	plannedAt := parseDBTime(schedule.NextFireAt)
	if !plannedAt.Valid {
		plannedAt = sql.NullTime{Time: now, Valid: true}
	}
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO schedule_triggers(schedule_id, planned_fire_at, actual_fire_at, status, error_message)
		 VALUES (?, ?, ?, 'skipped', 'misfire skipped')`,
		schedule.ID, plannedAt, now,
	).Error; err != nil {
		return false
	}
	return s.advanceSchedule(ctx, tx, schedule, now) == nil
}

func (s *Service) fireSchedule(ctx context.Context, tx *gorm.DB, schedule dueScheduleRecord, now time.Time) bool {
	plannedAt := parseDBTime(schedule.NextFireAt)
	if !plannedAt.Valid {
		plannedAt = sql.NullTime{Time: now, Valid: true}
	}
	if ok := s.fireScheduleWithoutAdvance(ctx, tx, schedule, plannedAt.Time, now); !ok {
		_ = s.advanceSchedule(ctx, tx, schedule, now)
		return false
	}
	if err := s.advanceSchedule(ctx, tx, schedule, now); err != nil {
		return false
	}
	return true
}

func (s *Service) fireScheduleWithoutAdvance(ctx context.Context, tx *gorm.DB, schedule dueScheduleRecord, plannedAt time.Time, now time.Time) bool {
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
		return false
	}
	if err := tx.WithContext(ctx).Exec(
		"UPDATE schedule_triggers SET status = 'fired', task_run_id = ?, actual_fire_at = ? WHERE id = ?",
		runID, now, triggerID,
	).Error; err != nil {
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
		MisfirePolicy: row.MisfirePolicy,
		Status:       row.Status,
		NextFireAt:   row.NextFireAt.String,
		LastFireAt:   row.LastFireAt.String,
		CreatedBy:    row.CreatedBy.String,
		CreatedAt:    row.CreatedAt,
	}
}

func normalizeMisfirePolicy(value string) string {
	switch strings.TrimSpace(value) {
	case "fire_once", "fire_all":
		return strings.TrimSpace(value)
	default:
		return "skip"
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
