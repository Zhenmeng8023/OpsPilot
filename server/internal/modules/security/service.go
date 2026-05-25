package security

import (
	"context"
	"database/sql"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"gorm.io/gorm"

	"opspilot/server/internal/config"
	"opspilot/server/internal/shared/apperror"
)

type Service struct {
	db  *gorm.DB
	cfg config.Config
}

type PermissionDiffInput struct {
	RoleID          string
	PermissionCodes []string
}

type PermissionDiffResult struct {
	RoleID               string   `json:"roleId,omitempty"`
	RoleName             string   `json:"roleName,omitempty"`
	CurrentPermissions   []string `json:"currentPermissions"`
	RequestedPermissions []string `json:"requestedPermissions"`
	Added                []string `json:"added"`
	Removed              []string `json:"removed"`
	AffectedUsers        int      `json:"affectedUsers"`
	HighRisk             bool     `json:"highRisk"`
}

type SecretRotationEvent struct {
	ID           uint64 `json:"id"`
	SecretID     string `json:"secretId"`
	Action       string `json:"action"`
	Operator     string `json:"operator,omitempty"`
	ResourceType string `json:"resourceType,omitempty"`
	ResourceID   string `json:"resourceId,omitempty"`
	TraceID      string `json:"traceId,omitempty"`
	CreatedAt    string `json:"createdAt"`
}

type workspaceRecord struct {
	ID uint64
}

func NewService(db *gorm.DB, cfg config.Config) *Service {
	return &Service{db: db, cfg: cfg}
}

func (s *Service) PermissionDiff(ctx context.Context, input PermissionDiffInput) (PermissionDiffResult, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return PermissionDiffResult{}, appErr
	}
	roleID := strings.TrimSpace(input.RoleID)
	if roleID == "" {
		return PermissionDiffResult{}, apperror.New(http.StatusBadRequest, 400990, "roleId is required")
	}
	var role struct {
		ID   uint64 `gorm:"column:id"`
		UID  string `gorm:"column:uid"`
		Name string `gorm:"column:name"`
	}
	if err := s.db.WithContext(ctx).Raw(
		`SELECT id, uid, name FROM roles WHERE workspace_id = ? AND uid = ? AND deleted_at IS NULL LIMIT 1`,
		workspace.ID, roleID,
	).Scan(&role).Error; err != nil {
		return PermissionDiffResult{}, apperror.Wrap(http.StatusInternalServerError, 500990, "load role failed", err)
	}
	if role.ID == 0 {
		return PermissionDiffResult{}, apperror.New(http.StatusNotFound, 404990, "role not found")
	}
	current, err := s.rolePermissions(ctx, role.ID)
	if err != nil {
		return PermissionDiffResult{}, apperror.Wrap(http.StatusInternalServerError, 500991, "load role permissions failed", err)
	}
	requested := normalize(input.PermissionCodes)
	var affected int
	if err := s.db.WithContext(ctx).Raw(
		`SELECT COUNT(DISTINCT user_id) FROM user_roles WHERE workspace_id = ? AND role_id = ?`,
		workspace.ID, role.ID,
	).Scan(&affected).Error; err != nil {
		return PermissionDiffResult{}, apperror.Wrap(http.StatusInternalServerError, 500992, "count affected users failed", err)
	}
	added, removed := diff(current, requested)
	return PermissionDiffResult{
		RoleID:               role.UID,
		RoleName:             role.Name,
		CurrentPermissions:   current,
		RequestedPermissions: requested,
		Added:                added,
		Removed:              removed,
		AffectedUsers:        affected,
		HighRisk:             containsHighRisk(added),
	}, nil
}

func (s *Service) SecretRotationHistory(ctx context.Context, secretID string) ([]SecretRotationEvent, *apperror.Error) {
	workspace, appErr := s.workspace(ctx)
	if appErr != nil {
		return nil, appErr
	}
	secretID = strings.TrimSpace(secretID)
	var rows []struct {
		ID           uint64         `gorm:"column:id"`
		Action       string         `gorm:"column:action"`
		Operator     sql.NullString `gorm:"column:operator"`
		ResourceType sql.NullString `gorm:"column:resource_type"`
		ResourceID   sql.NullInt64  `gorm:"column:resource_id"`
		TraceID      sql.NullString `gorm:"column:trace_id"`
		CreatedAt    string         `gorm:"column:created_at"`
	}
	args := []interface{}{workspace.ID}
	where := "al.workspace_id = ? AND al.action LIKE '%rotate_secret%'"
	if secretID != "" {
		where += " AND (al.trace_id = ? OR CAST(al.resource_id AS CHAR) = ? OR al.resource_type = ?)"
		args = append(args, secretID, secretID, secretID)
	}
	if err := s.db.WithContext(ctx).Raw(
		`SELECT al.id, al.action, u.username AS operator, al.resource_type, al.resource_id, al.trace_id,
		        DATE_FORMAT(al.created_at, '%Y-%m-%d %H:%i:%s') AS created_at
		   FROM audit_logs al
		   LEFT JOIN users u ON u.id = al.actor_user_id
		  WHERE `+where+`
		  ORDER BY al.created_at DESC, al.id DESC
		  LIMIT 200`,
		args...,
	).Scan(&rows).Error; err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500993, "load secret rotation history failed", err)
	}
	out := make([]SecretRotationEvent, 0, len(rows))
	for _, row := range rows {
		resourceID := ""
		if row.ResourceID.Valid {
			resourceID = sqlID(row.ResourceID.Int64)
		}
		out = append(out, SecretRotationEvent{
			ID:           row.ID,
			SecretID:     secretID,
			Action:       row.Action,
			Operator:     row.Operator.String,
			ResourceType: row.ResourceType.String,
			ResourceID:   resourceID,
			TraceID:      row.TraceID.String,
			CreatedAt:    row.CreatedAt,
		})
	}
	return out, nil
}

func (s *Service) workspace(ctx context.Context) (workspaceRecord, *apperror.Error) {
	var workspace workspaceRecord
	err := s.db.WithContext(ctx).Raw("SELECT id FROM workspaces WHERE slug = ? AND status = 'active' LIMIT 1", s.cfg.Bootstrap.WorkspaceSlug).Scan(&workspace).Error
	if err != nil {
		return workspaceRecord{}, apperror.Wrap(http.StatusInternalServerError, 500994, "load workspace failed", err)
	}
	if workspace.ID == 0 {
		return workspaceRecord{}, apperror.New(http.StatusInternalServerError, 500995, "default workspace is not initialized")
	}
	return workspace, nil
}

func (s *Service) rolePermissions(ctx context.Context, roleID uint64) ([]string, error) {
	var rows []string
	err := s.db.WithContext(ctx).Raw(
		`SELECT p.code
		   FROM role_permissions rp
		   JOIN permissions p ON p.id = rp.permission_id
		  WHERE rp.role_id = ?
		  ORDER BY p.code ASC`,
		roleID,
	).Scan(&rows).Error
	return normalize(rows), err
}

func normalize(values []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func diff(current, requested []string) ([]string, []string) {
	currentSet := map[string]bool{}
	requestedSet := map[string]bool{}
	for _, value := range current {
		currentSet[value] = true
	}
	for _, value := range requested {
		requestedSet[value] = true
	}
	added := []string{}
	removed := []string{}
	for _, value := range requested {
		if !currentSet[value] {
			added = append(added, value)
		}
	}
	for _, value := range current {
		if !requestedSet[value] {
			removed = append(removed, value)
		}
	}
	return added, removed
}

func containsHighRisk(values []string) bool {
	for _, value := range values {
		if strings.Contains(value, "execute") || strings.Contains(value, "manage") || strings.Contains(value, "write") || strings.Contains(value, "cancel") || strings.Contains(value, "security") {
			return true
		}
	}
	return false
}

func sqlID(value int64) string {
	if value == 0 {
		return ""
	}
	return strconv.FormatInt(value, 10)
}
