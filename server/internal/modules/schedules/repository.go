package schedules

import (
	"context"
	"database/sql"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

type workspaceRecord struct {
	ID  uint64
	UID string
}

type userRecord struct {
	ID uint64
}

type taskRecord struct {
	ID   uint64
	UID  string
	Name string
}

type scheduleRecord struct {
	ID           uint64
	UID          string
	WorkspaceID  uint64
	TaskID       uint64
	TaskUID      string
	TaskName     string
	Name         string
	ScheduleType string
	CronExpr     sql.NullString
	Timezone     string
	Status       string
	NextFireAt   sql.NullString
	LastFireAt   sql.NullString
	CreatedBy    sql.NullString
	CreatedByID  sql.NullInt64
	CreatedAt    string
}

type listFilter struct {
	Keyword  string
	Status   string
	TaskUID  string
	Page     int
	PageSize int
}

type dueScheduleRecord struct {
	ID          uint64
	WorkspaceID uint64
	TaskID      uint64
	Name        string
	CronExpr    string
	Timezone    string
	NextFireAt  string
	CreatedByID sql.NullInt64
}

func newRepository(db *gorm.DB) repository {
	return repository{db: db}
}

func (r repository) defaultWorkspace(ctx context.Context, slug string) (workspaceRecord, error) {
	var workspace workspaceRecord
	err := r.db.WithContext(ctx).Raw(
		"SELECT id, uid FROM workspaces WHERE slug = ? AND status = 'active' LIMIT 1",
		slug,
	).Scan(&workspace).Error
	return workspace, err
}

func (r repository) userByUID(ctx context.Context, uid string) (userRecord, error) {
	var user userRecord
	err := r.db.WithContext(ctx).Raw("SELECT id FROM users WHERE uid = ? AND status = 'active' LIMIT 1", uid).Scan(&user).Error
	return user, err
}

func (r repository) taskByUID(ctx context.Context, workspaceID uint64, uid string) (taskRecord, error) {
	var task taskRecord
	err := r.db.WithContext(ctx).Raw(
		`SELECT id, uid, name
		   FROM tasks
		  WHERE workspace_id = ? AND uid = ? AND status = 'active' AND deleted_at IS NULL
		  LIMIT 1`,
		workspaceID, uid,
	).Scan(&task).Error
	return task, err
}

func (r repository) scheduleByUID(ctx context.Context, workspaceID uint64, uid string) (scheduleRecord, error) {
	var row scheduleRecord
	err := r.db.WithContext(ctx).Raw(scheduleSelectSQL()+`
		  WHERE s.workspace_id = ? AND s.uid = ? AND s.deleted_at IS NULL
		  LIMIT 1`,
		workspaceID, uid,
	).Scan(&row).Error
	return row, err
}

func (r repository) listSchedules(ctx context.Context, workspaceID uint64, filter listFilter) ([]scheduleRecord, int64, error) {
	args := []interface{}{workspaceID}
	where := "WHERE s.workspace_id = ? AND s.deleted_at IS NULL"
	if filter.Keyword != "" {
		where += " AND (s.name LIKE ? OR t.name LIKE ?)"
		like := "%" + filter.Keyword + "%"
		args = append(args, like, like)
	}
	if filter.Status != "" {
		where += " AND s.status = ?"
		args = append(args, filter.Status)
	}
	if filter.TaskUID != "" {
		where += " AND t.uid = ?"
		args = append(args, filter.TaskUID)
	}
	var total int64
	if err := r.db.WithContext(ctx).Raw(
		`SELECT COUNT(*)
		   FROM schedules s
		   JOIN tasks t ON t.id = s.task_id
		   LEFT JOIN users u ON u.id = s.created_by
		  `+where,
		args...,
	).Scan(&total).Error; err != nil {
		return nil, 0, err
	}

	queryArgs := append([]interface{}{}, args...)
	queryArgs = append(queryArgs, (filter.Page-1)*filter.PageSize, filter.PageSize)
	var rows []scheduleRecord
	err := r.db.WithContext(ctx).Raw(scheduleSelectSQL()+`
		  `+where+`
		  ORDER BY s.next_fire_at IS NULL, s.next_fire_at ASC, s.created_at DESC
		  LIMIT ?, ?`,
		queryArgs...,
	).Scan(&rows).Error
	return rows, total, err
}

func scheduleSelectSQL() string {
	return `SELECT s.id, s.uid, s.workspace_id, s.task_id, t.uid AS task_uid, t.name AS task_name,
	        s.name, s.schedule_type, s.cron_expr, s.timezone, s.status,
	        DATE_FORMAT(s.next_fire_at, '%Y-%m-%d %H:%i:%s') AS next_fire_at,
	        DATE_FORMAT(s.last_fire_at, '%Y-%m-%d %H:%i:%s') AS last_fire_at,
	        u.username AS created_by, s.created_by AS created_by_id,
	        DATE_FORMAT(s.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
	   FROM schedules s
	   JOIN tasks t ON t.id = s.task_id
	   LEFT JOIN users u ON u.id = s.created_by`
}
