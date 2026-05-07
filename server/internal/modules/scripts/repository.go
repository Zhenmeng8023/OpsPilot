package scripts

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

type scriptRecord struct {
	ID               uint64
	UID              string
	Name             string
	Description      sql.NullString
	ScriptType       string
	Status           string
	CreatedBy        sql.NullString
	CreatedByID      sql.NullInt64
	CreatedAt        string
	UpdatedAt        string
	LatestVersion    uint
	LatestVersionID  uint64
	LatestVersionUID sql.NullString
	Content          sql.NullString
	ChangeSummary    sql.NullString
}

type scriptVersionRecord struct {
	ID            uint64
	UID           string
	TemplateID    uint64
	VersionNo     uint
	Content       string
	Checksum      string
	Status        string
	ChangeSummary sql.NullString
}

type approvalRecord struct {
	ID           uint64
	VersionUID   string
	ScriptUID    string
	ScriptName   string
	VersionNo    uint
	Status       string
	Comment      sql.NullString
	Approver     sql.NullString
	ApprovedAt   sql.NullString
	CreatedAt    string
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

func (r repository) userIDByUID(ctx context.Context, uid string) (sql.NullInt64, error) {
	var id uint64
	err := r.db.WithContext(ctx).Raw("SELECT id FROM users WHERE uid = ? LIMIT 1", uid).Scan(&id).Error
	if err != nil || id == 0 {
		return sql.NullInt64{}, err
	}
	return sql.NullInt64{Int64: int64(id), Valid: true}, nil
}

func (r repository) listScripts(ctx context.Context, workspaceID uint64, keyword, status string) ([]scriptRecord, error) {
	args := []interface{}{workspaceID}
	where := "WHERE st.workspace_id = ? AND st.deleted_at IS NULL"
	if keyword != "" {
		where += " AND (st.name LIKE ? OR st.description LIKE ?)"
		like := "%" + keyword + "%"
		args = append(args, like, like)
	}
	if status != "" {
		where += " AND st.status = ?"
		args = append(args, status)
	}

	var rows []scriptRecord
	err := r.db.WithContext(ctx).Raw(
		`SELECT st.id, st.uid, st.name, st.description, st.script_type, st.status,
		        u.username AS created_by,
		        DATE_FORMAT(st.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
		        DATE_FORMAT(st.updated_at, '%Y-%m-%d %H:%i:%s') AS updated_at,
		        COALESCE(MAX(sv.version_no), 0) AS latest_version
		   FROM script_templates st
		   LEFT JOIN script_versions sv ON sv.template_id = st.id
		   LEFT JOIN users u ON u.id = st.created_by
		   `+where+`
		  GROUP BY st.id, st.uid, st.name, st.description, st.script_type, st.status, u.username, st.created_at, st.updated_at
		  ORDER BY st.updated_at DESC, st.created_at DESC`,
		args...,
	).Scan(&rows).Error
	return rows, err
}

func (r repository) scriptByUID(ctx context.Context, workspaceID uint64, uid string) (scriptRecord, error) {
	var row scriptRecord
	err := r.db.WithContext(ctx).Raw(
		`SELECT st.id, st.uid, st.name, st.description, st.script_type, st.status,
		        st.created_by AS created_by_id,
		        u.username AS created_by,
		        DATE_FORMAT(st.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
		        DATE_FORMAT(st.updated_at, '%Y-%m-%d %H:%i:%s') AS updated_at,
		        sv.id AS latest_version_id,
		        sv.uid AS latest_version_uid,
		        sv.version_no AS latest_version,
		        sv.content,
		        sv.change_summary
		   FROM script_templates st
		   LEFT JOIN users u ON u.id = st.created_by
		   LEFT JOIN script_versions sv ON sv.id = (
		     SELECT sv2.id FROM script_versions sv2
		      WHERE sv2.template_id = st.id
		      ORDER BY sv2.version_no DESC
		      LIMIT 1
		   )
		  WHERE st.workspace_id = ? AND st.uid = ? AND st.deleted_at IS NULL
		  LIMIT 1`,
		workspaceID, uid,
	).Scan(&row).Error
	return row, err
}

func (r repository) latestVersion(ctx context.Context, templateID uint64) (scriptVersionRecord, error) {
	var row scriptVersionRecord
	err := r.db.WithContext(ctx).Raw(
		`SELECT id, uid, template_id, version_no, content, checksum, status, change_summary
		   FROM script_versions
		  WHERE template_id = ?
		  ORDER BY version_no DESC
		  LIMIT 1`,
		templateID,
	).Scan(&row).Error
	return row, err
}

func (r repository) listApprovals(ctx context.Context, workspaceID uint64, status string) ([]approvalRecord, error) {
	args := []interface{}{workspaceID}
	where := "WHERE st.workspace_id = ?"
	if status != "" {
		where += " AND sa.status = ?"
		args = append(args, status)
	}
	var rows []approvalRecord
	err := r.db.WithContext(ctx).Raw(
		`SELECT sa.id, sv.uid AS version_uid, st.uid AS script_uid, st.name AS script_name, sv.version_no,
		        sa.status, sa.comment, u.username AS approver,
		        DATE_FORMAT(sa.approved_at, '%Y-%m-%d %H:%i:%s') AS approved_at,
		        DATE_FORMAT(sa.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM script_approvals sa
		   JOIN script_versions sv ON sv.id = sa.script_version_id
		   JOIN script_templates st ON st.id = sv.template_id
		   LEFT JOIN users u ON u.id = sa.approver_id
		  `+where+`
		  ORDER BY sa.created_at DESC`,
		args...,
	).Scan(&rows).Error
	return rows, err
}
