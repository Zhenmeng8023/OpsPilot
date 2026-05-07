package tasks

import (
	"context"
	"database/sql"
	"strings"

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
	ID       uint64
	UID      string
	Username string
}

type scriptRecord struct {
	TemplateID  uint64
	TemplateUID string
	VersionID   uint64
	VersionUID  string
	Name        string
	ScriptType  string
	Content     string
	Status      string
	VersionNo   uint
}

type targetCandidate struct {
	AgentID     uint64
	AgentUID    string
	AgentName   string
	AgentStatus string
	HostID      sql.NullInt64
	HostUID     sql.NullString
	HostName    sql.NullString
	Hostname    sql.NullString
	IP          sql.NullString
}

type runRecord struct {
	ID              uint64
	UID             string
	TaskID          uint64
	TaskUID         string
	TaskName        string
	Description     sql.NullString
	Status          string
	TimeoutSeconds  uint
	TotalTargets    uint
	SuccessTargets  uint
	FailedTargets   uint
	CanceledTargets uint
	RunningTargets  uint
	QueuedTargets   uint
	CreatedBy       sql.NullString
	CreatedByID     sql.NullInt64
	CreatedAt       string
	QueuedAt        sql.NullString
	StartedAt       sql.NullString
	FinishedAt      sql.NullString
	ErrorMessage    sql.NullString
}

type runTargetRecord struct {
	ID           uint64
	UID          string
	RunID        uint64
	TaskTargetID sql.NullInt64
	AgentID      sql.NullInt64
	AgentUID     sql.NullString
	AgentName    sql.NullString
	AgentStatus  sql.NullString
	HostID       sql.NullInt64
	HostUID      sql.NullString
	HostName     sql.NullString
	Status       string
	ExitCode     sql.NullInt64
	ErrorMessage sql.NullString
	StartedAt    sql.NullString
	FinishedAt   sql.NullString
	CreatedAt    string
}

type agentTaskRecord struct {
	TargetID       string
	RunID          string
	TaskName       string
	Description    sql.NullString
	ScriptType     string
	Command        string
	TimeoutSeconds uint
	Status         string
	AgentName      string
	HostName       sql.NullString
}

type attemptRecord struct {
	ID        uint64
	UID       string
	AttemptNo uint
	Status    string
}

type logRecord struct {
	ID        uint64
	TargetID  string
	RunID     string
	Sequence  uint64
	Stream    string
	Content   string
	CreatedAt string
	AgentName sql.NullString
	HostName  sql.NullString
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
	err := r.db.WithContext(ctx).Raw(
		"SELECT id, uid, username FROM users WHERE uid = ? AND status = 'active' LIMIT 1",
		uid,
	).Scan(&user).Error
	return user, err
}

func (r repository) activeScriptByUID(ctx context.Context, workspaceID uint64, uid string) (scriptRecord, error) {
	var row scriptRecord
	err := r.db.WithContext(ctx).Raw(
		`SELECT st.id AS template_id, st.uid AS template_uid, st.name, st.script_type, st.status,
		        sv.id AS version_id, sv.uid AS version_uid, sv.version_no, sv.content
		   FROM script_templates st
		   JOIN script_versions sv ON sv.id = (
		     SELECT sv2.id FROM script_versions sv2
		      WHERE sv2.template_id = st.id
		        AND sv2.status = 'active'
		      ORDER BY sv2.version_no DESC
		      LIMIT 1
		   )
		  WHERE st.workspace_id = ? AND st.uid = ? AND st.status = 'active' AND st.deleted_at IS NULL
		  LIMIT 1`,
		workspaceID, uid,
	).Scan(&row).Error
	return row, err
}

func (r repository) targetAgentsByUIDs(ctx context.Context, workspaceID uint64, uids []string) ([]targetCandidate, error) {
	if len(uids) == 0 {
		return []targetCandidate{}, nil
	}
	var rows []targetCandidate
	err := r.db.WithContext(ctx).Raw(
		`SELECT a.id AS agent_id, a.uid AS agent_uid, a.name AS agent_name, a.status AS agent_status,
		        a.host_id, h.uid AS host_uid, h.name AS host_name, h.hostname, COALESCE(a.ip, h.primary_ip) AS ip
		   FROM agents a
		   LEFT JOIN hosts h ON h.id = a.host_id
		  WHERE a.workspace_id = ?
		    AND a.uid IN ?
		    AND a.status <> 'disabled'
		    AND a.deleted_at IS NULL`,
		workspaceID, uids,
	).Scan(&rows).Error
	return rows, err
}

func (r repository) targetAgentsByHostUIDs(ctx context.Context, workspaceID uint64, hostUIDs []string) ([]targetCandidate, error) {
	if len(hostUIDs) == 0 {
		return []targetCandidate{}, nil
	}
	var rows []targetCandidate
	err := r.db.WithContext(ctx).Raw(
		`SELECT a.id AS agent_id, a.uid AS agent_uid, a.name AS agent_name, a.status AS agent_status,
		        a.host_id, h.uid AS host_uid, h.name AS host_name, h.hostname, COALESCE(a.ip, h.primary_ip) AS ip
		   FROM hosts h
		   JOIN agents a ON a.host_id = h.id
		  WHERE h.workspace_id = ?
		    AND h.uid IN ?
		    AND h.deleted_at IS NULL
		    AND a.deleted_at IS NULL
		    AND a.status <> 'disabled'`,
		workspaceID, hostUIDs,
	).Scan(&rows).Error
	return rows, err
}

func (r repository) runByUID(ctx context.Context, workspaceID uint64, uid string) (runRecord, error) {
	var row runRecord
	err := r.db.WithContext(ctx).Raw(summarySQL()+`
		  WHERE tr.workspace_id = ? AND tr.uid = ?
		  GROUP BY tr.id, tr.uid, t.id, t.uid, t.name, t.description, tr.status, tr.timeout_seconds,
		           tr.total_targets, tr.success_targets, tr.failed_targets, tr.canceled_targets,
		           u.username, tr.created_by, tr.created_at, tr.queued_at, tr.started_at, tr.finished_at, tr.error_message
		  LIMIT 1`,
		workspaceID, uid,
	).Scan(&row).Error
	return row, err
}

func (r repository) listRuns(ctx context.Context, workspaceID uint64, keyword, status string) ([]runRecord, error) {
	args := []interface{}{workspaceID}
	where := "WHERE tr.workspace_id = ?"
	if keyword != "" {
		where += " AND (t.name LIKE ? OR t.description LIKE ?)"
		like := "%" + keyword + "%"
		args = append(args, like, like)
	}
	if status != "" {
		where += " AND tr.status = ?"
		args = append(args, status)
	}
	var rows []runRecord
	err := r.db.WithContext(ctx).Raw(summarySQL()+`
		  `+where+`
		  GROUP BY tr.id, tr.uid, t.id, t.uid, t.name, t.description, tr.status, tr.timeout_seconds,
		           tr.total_targets, tr.success_targets, tr.failed_targets, tr.canceled_targets,
		           u.username, tr.created_by, tr.created_at, tr.queued_at, tr.started_at, tr.finished_at, tr.error_message
		  ORDER BY tr.created_at DESC
		  LIMIT 200`,
		args...,
	).Scan(&rows).Error
	return rows, err
}

func (r repository) targetsByRunUID(ctx context.Context, workspaceID uint64, runUID string) ([]runTargetRecord, error) {
	var rows []runTargetRecord
	err := r.db.WithContext(ctx).Raw(
		`SELECT rt.id, rt.uid, rt.run_id, rt.task_target_id, rt.agent_id, a.uid AS agent_uid, a.name AS agent_name,
		        a.status AS agent_status, rt.host_id, h.uid AS host_uid, h.name AS host_name,
		        rt.status, rt.exit_code, rt.error_message,
		        DATE_FORMAT(rt.started_at, '%Y-%m-%d %H:%i:%s') AS started_at,
		        DATE_FORMAT(rt.finished_at, '%Y-%m-%d %H:%i:%s') AS finished_at,
		        DATE_FORMAT(rt.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM task_run_targets rt
		   JOIN task_runs tr ON tr.id = rt.run_id
		   LEFT JOIN agents a ON a.id = rt.agent_id
		   LEFT JOIN hosts h ON h.id = rt.host_id
		  WHERE tr.workspace_id = ? AND tr.uid = ?
		  ORDER BY rt.created_at, rt.id`,
		workspaceID, runUID,
	).Scan(&rows).Error
	return rows, err
}

func (r repository) agentPoll(ctx context.Context, agentID uint64, limit int) ([]agentTaskRecord, error) {
	var rows []agentTaskRecord
	err := r.db.WithContext(ctx).Raw(
		`SELECT rt.uid AS target_id, tr.uid AS run_id, t.name AS task_name, t.description,
		        st.script_type, sv.content AS command, tr.timeout_seconds, rt.status,
		        a.name AS agent_name, h.name AS host_name
		   FROM task_run_targets rt
		   JOIN task_runs tr ON tr.id = rt.run_id
		   JOIN tasks t ON t.id = tr.task_id
		   JOIN script_versions sv ON sv.id = tr.script_version_id
		   JOIN script_templates st ON st.id = t.script_template_id
		   JOIN agents a ON a.id = rt.agent_id
		   LEFT JOIN hosts h ON h.id = rt.host_id
		  WHERE rt.agent_id = ?
		    AND rt.status = 'queued'
		    AND tr.status IN ('queued', 'running')
		  ORDER BY rt.created_at, rt.id
		  LIMIT ?`,
		agentID, limit,
	).Scan(&rows).Error
	return rows, err
}

func (r repository) agentTaskByTargetUID(ctx context.Context, targetUID string) (agentTaskRecord, error) {
	var row agentTaskRecord
	err := r.db.WithContext(ctx).Raw(
		`SELECT rt.uid AS target_id, tr.uid AS run_id, t.name AS task_name, t.description,
		        st.script_type, sv.content AS command, tr.timeout_seconds, rt.status,
		        a.name AS agent_name, h.name AS host_name
		   FROM task_run_targets rt
		   JOIN task_runs tr ON tr.id = rt.run_id
		   JOIN tasks t ON t.id = tr.task_id
		   JOIN script_versions sv ON sv.id = tr.script_version_id
		   JOIN script_templates st ON st.id = t.script_template_id
		   JOIN agents a ON a.id = rt.agent_id
		   LEFT JOIN hosts h ON h.id = rt.host_id
		  WHERE rt.uid = ?
		  LIMIT 1`,
		targetUID,
	).Scan(&row).Error
	return row, err
}

func (r repository) latestAttempt(ctx context.Context, runTargetID uint64) (attemptRecord, error) {
	var row attemptRecord
	err := r.db.WithContext(ctx).Raw(
		`SELECT id, uid, attempt_no, status
		   FROM task_run_attempts
		  WHERE run_target_id = ?
		  ORDER BY attempt_no DESC
		  LIMIT 1`,
		runTargetID,
	).Scan(&row).Error
	return row, err
}

func (r repository) logs(ctx context.Context, workspaceID uint64, runUID, targetUID string, afterID uint64) ([]logRecord, error) {
	args := []interface{}{workspaceID, runUID, afterID}
	where := "WHERE tr.workspace_id = ? AND tr.uid = ? AND l.id > ?"
	if strings.TrimSpace(targetUID) != "" {
		where += " AND rt.uid = ?"
		args = append(args, targetUID)
	}
	var rows []logRecord
	err := r.db.WithContext(ctx).Raw(
		`SELECT l.id, tr.uid AS run_id, rt.uid AS target_id, l.sequence, l.stream, l.content,
		        DATE_FORMAT(l.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
		        a.name AS agent_name, h.name AS host_name
		   FROM task_run_logs l
		   JOIN task_runs tr ON tr.id = l.run_id
		   JOIN task_run_targets rt ON rt.id = l.run_target_id
		   LEFT JOIN agents a ON a.id = rt.agent_id
		   LEFT JOIN hosts h ON h.id = rt.host_id
		  `+where+`
		  ORDER BY l.id ASC
		  LIMIT 500`,
		args...,
	).Scan(&rows).Error
	return rows, err
}

func summarySQL() string {
	return `SELECT tr.id, tr.uid, tr.task_id, t.uid AS task_uid, t.name AS task_name, t.description,
	        tr.status, tr.timeout_seconds, tr.total_targets, tr.success_targets,
	        tr.failed_targets, tr.canceled_targets,
	        COALESCE(SUM(CASE WHEN rt.status = 'running' THEN 1 ELSE 0 END), 0) AS running_targets,
	        COALESCE(SUM(CASE WHEN rt.status = 'queued' THEN 1 ELSE 0 END), 0) AS queued_targets,
	        u.username AS created_by, tr.created_by AS created_by_id,
	        DATE_FORMAT(tr.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
	        DATE_FORMAT(tr.queued_at, '%Y-%m-%d %H:%i:%s') AS queued_at,
	        DATE_FORMAT(tr.started_at, '%Y-%m-%d %H:%i:%s') AS started_at,
	        DATE_FORMAT(tr.finished_at, '%Y-%m-%d %H:%i:%s') AS finished_at,
	        tr.error_message
	   FROM task_runs tr
	   JOIN tasks t ON t.id = tr.task_id
	   LEFT JOIN task_run_targets rt ON rt.run_id = tr.id
	   LEFT JOIN users u ON u.id = tr.created_by`
}
