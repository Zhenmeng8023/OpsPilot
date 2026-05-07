package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"sort"
	"strings"

	"gorm.io/gorm"

	"opspilot/server/internal/shared/apperror"
)

type UserSummary struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	Email       string   `json:"email,omitempty"`
	Status      string   `json:"status"`
	Roles       []string `json:"roles"`
	LastLoginAt string   `json:"lastLoginAt,omitempty"`
	CreatedAt   string   `json:"createdAt"`
}

type RoleSummary struct {
	ID          string              `json:"id"`
	Code        string              `json:"code"`
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	BuiltIn     bool                `json:"builtIn"`
	Status      string              `json:"status"`
	Permissions []PermissionSummary `json:"permissions"`
}

type PermissionSummary struct {
	Code        string `json:"code"`
	Module      string `json:"module"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

type CreateUserInput struct {
	Username string
	Email    string
	Password string
	Roles    []string
}

type UpdateUserStatusInput struct {
	UserID string
	Status string
	Actor  string
}

type UpdateUserRolesInput struct {
	UserID string
	Roles  []string
}

type CreateRoleInput struct {
	Code        string
	Name        string
	Description string
	Permissions []string
}

type UpdateRolePermissionsInput struct {
	RoleID      string
	Permissions []string
}

func (s *Service) ListUsers(ctx context.Context, keyword, status string) ([]UserSummary, *apperror.Error) {
	workspace, appErr := s.defaultWorkspaceForAPI(ctx)
	if appErr != nil {
		return nil, appErr
	}

	keyword = strings.TrimSpace(keyword)
	status = strings.TrimSpace(status)
	args := []interface{}{workspace.ID}
	where := "WHERE wm.workspace_id = ?"
	if keyword != "" {
		where += " AND (u.username LIKE ? OR u.email LIKE ?)"
		like := "%" + keyword + "%"
		args = append(args, like, like)
	}
	if status != "" {
		where += " AND u.status = ?"
		args = append(args, status)
	}

	var rows []struct {
		UID         string
		Username    string
		Email       sql.NullString
		Status      string
		LastLoginAt sql.NullString
		CreatedAt   string
		Roles       sql.NullString
	}
	err := s.db.WithContext(ctx).Raw(
		`SELECT u.uid, u.username, u.email, u.status,
		        DATE_FORMAT(u.last_login_at, '%Y-%m-%d %H:%i:%s') AS last_login_at,
		        DATE_FORMAT(u.created_at, '%Y-%m-%d %H:%i:%s') AS created_at,
		        GROUP_CONCAT(DISTINCT r.code ORDER BY r.code SEPARATOR ',') AS roles
		   FROM users u
		   JOIN workspace_members wm ON wm.user_id = u.id
		   LEFT JOIN user_roles ur ON ur.user_id = u.id AND ur.workspace_id = wm.workspace_id
		   LEFT JOIN roles r ON r.id = ur.role_id AND r.status = 'active'
		   `+where+`
		  GROUP BY u.id, u.uid, u.username, u.email, u.status, u.last_login_at, u.created_at
		  ORDER BY u.created_at DESC`,
		args...,
	).Scan(&rows).Error
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500001, "list users failed", err)
	}

	users := make([]UserSummary, 0, len(rows))
	for _, row := range rows {
		user := UserSummary{
			ID:        row.UID,
			Username:  row.Username,
			Status:    row.Status,
			CreatedAt: row.CreatedAt,
			Roles:     splitCSV(row.Roles.String),
		}
		if row.Email.Valid {
			user.Email = row.Email.String
		}
		if row.LastLoginAt.Valid {
			user.LastLoginAt = row.LastLoginAt.String
		}
		users = append(users, user)
	}
	return users, nil
}

func (s *Service) CreateUser(ctx context.Context, input CreateUserInput) (UserSummary, *apperror.Error) {
	username := strings.TrimSpace(input.Username)
	email := strings.TrimSpace(input.Email)
	if err := validateUsername(username); err != nil {
		return UserSummary{}, err
	}
	if err := validatePassword(input.Password); err != nil {
		return UserSummary{}, err
	}
	roles := normalizeRoleCodes(input.Roles)
	if len(roles) == 0 {
		roles = []string{"member"}
	}

	var created UserSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		if workspace.ID == 0 {
			return missingSeed("workspace")
		}
		if exists, err := userExists(ctx, tx, username, email); err != nil {
			return err
		} else if exists {
			return apperror.New(http.StatusConflict, 409001, "username or email already exists")
		}

		passwordHash, err := hashPassword(input.Password)
		if err != nil {
			return err
		}
		uid, err := newUID()
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO users(uid, username, email, password_hash, status, password_changed_at)
			 VALUES (?, ?, ?, ?, 'active', NOW(3))`,
			uid, username, nullString(email), passwordHash,
		).Error; err != nil {
			return err
		}
		var user userRecord
		if err := tx.WithContext(ctx).Raw("SELECT id, uid, username, email, password_hash, status FROM users WHERE uid = ?", uid).Scan(&user).Error; err != nil {
			return err
		}
		if err := seedMembership(ctx, tx, workspace.ID, user.ID); err != nil {
			return err
		}
		if err := replaceUserRoles(ctx, tx, workspace.ID, user.ID, roles); err != nil {
			return err
		}
		created = UserSummary{
			ID:       user.UID,
			Username: user.Username,
			Status:   user.Status,
			Roles:    roles,
		}
		if user.Email.Valid {
			created.Email = user.Email.String
		}
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return UserSummary{}, appErr
		}
		return UserSummary{}, apperror.Wrap(http.StatusInternalServerError, 500001, "create user failed", txErr)
	}
	return created, nil
}

func (s *Service) UpdateUserStatus(ctx context.Context, input UpdateUserStatusInput) *apperror.Error {
	status := strings.TrimSpace(input.Status)
	if status != "active" && status != "disabled" && status != "locked" {
		return apperror.New(http.StatusBadRequest, 400004, "unsupported user status")
	}
	userID := strings.TrimSpace(input.UserID)
	if userID == "" {
		return apperror.New(http.StatusBadRequest, 400001, "user id is required")
	}
	if userID == input.Actor && status != "active" {
		return apperror.New(http.StatusBadRequest, 400005, "cannot disable current user")
	}
	res := s.db.WithContext(ctx).Exec("UPDATE users SET status = ? WHERE uid = ?", status, userID)
	if res.Error != nil {
		return apperror.Wrap(http.StatusInternalServerError, 500001, "update user status failed", res.Error)
	}
	if res.RowsAffected == 0 {
		return apperror.New(http.StatusNotFound, 404002, "user not found")
	}
	return nil
}

func (s *Service) UpdateUserRoles(ctx context.Context, input UpdateUserRolesInput) (UserSummary, *apperror.Error) {
	roles := normalizeRoleCodes(input.Roles)
	if len(roles) == 0 {
		return UserSummary{}, apperror.New(http.StatusBadRequest, 400006, "at least one role is required")
	}

	var updated UserSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		user, err := userByUID(ctx, tx, input.UserID)
		if err != nil {
			return err
		}
		if user.ID == 0 {
			return apperror.New(http.StatusNotFound, 404002, "user not found")
		}
		if err := replaceUserRoles(ctx, tx, workspace.ID, user.ID, roles); err != nil {
			return err
		}
		updated = UserSummary{
			ID:       user.UID,
			Username: user.Username,
			Status:   user.Status,
			Roles:    roles,
		}
		if user.Email.Valid {
			updated.Email = user.Email.String
		}
		return nil
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return UserSummary{}, appErr
		}
		return UserSummary{}, apperror.Wrap(http.StatusInternalServerError, 500001, "update user roles failed", txErr)
	}
	return updated, nil
}

func (s *Service) ListPermissions(ctx context.Context) ([]PermissionSummary, *apperror.Error) {
	var permissions []PermissionSummary
	err := s.db.WithContext(ctx).Raw(
		`SELECT code, module, name, COALESCE(description, '') AS description
		   FROM permissions
		  ORDER BY module, code`,
	).Scan(&permissions).Error
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500001, "list permissions failed", err)
	}
	return permissions, nil
}

func (s *Service) ListRoles(ctx context.Context) ([]RoleSummary, *apperror.Error) {
	workspace, appErr := s.defaultWorkspaceForAPI(ctx)
	if appErr != nil {
		return nil, appErr
	}
	roles, err := s.rolesByWorkspace(ctx, workspace.ID)
	if err != nil {
		return nil, apperror.Wrap(http.StatusInternalServerError, 500001, "list roles failed", err)
	}
	return roles, nil
}

func (s *Service) CreateRole(ctx context.Context, input CreateRoleInput) (RoleSummary, *apperror.Error) {
	code := strings.TrimSpace(input.Code)
	name := strings.TrimSpace(input.Name)
	if code == "" || name == "" {
		return RoleSummary{}, apperror.New(http.StatusBadRequest, 400001, "role code and name are required")
	}
	if code == "admin" || code == "member" {
		return RoleSummary{}, apperror.New(http.StatusConflict, 409002, "built-in role code is reserved")
	}
	permissions := normalizePermissionCodes(input.Permissions)

	var created RoleSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspace, err := defaultWorkspace(ctx, tx, s.cfg.Bootstrap.WorkspaceSlug)
		if err != nil {
			return err
		}
		roleUID, err := newUID()
		if err != nil {
			return err
		}
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO roles(uid, workspace_id, code, name, description, built_in, status)
			 VALUES (?, ?, ?, ?, ?, 0, 'active')`,
			roleUID, workspace.ID, code, name, nullString(input.Description),
		).Error; err != nil {
			return err
		}
		role, err := roleByUID(ctx, tx, roleUID)
		if err != nil {
			return err
		}
		if err := replaceRolePermissions(ctx, tx, role.ID, permissions); err != nil {
			return err
		}
		created = RoleSummary{
			ID:          role.UID,
			Code:        role.Code,
			Name:        role.Name,
			Description: role.Description.String,
			BuiltIn:     role.BuiltIn,
			Status:      role.Status,
		}
		created.Permissions, err = permissionsByRole(ctx, tx, role.ID)
		return err
	})
	if txErr != nil {
		if isDuplicateError(txErr) {
			return RoleSummary{}, apperror.New(http.StatusConflict, 409003, "role code already exists")
		}
		return RoleSummary{}, apperror.Wrap(http.StatusInternalServerError, 500001, "create role failed", txErr)
	}
	return created, nil
}

func (s *Service) UpdateRolePermissions(ctx context.Context, input UpdateRolePermissionsInput) (RoleSummary, *apperror.Error) {
	permissions := normalizePermissionCodes(input.Permissions)
	var updated RoleSummary
	txErr := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		role, err := roleByUID(ctx, tx, input.RoleID)
		if err != nil {
			return err
		}
		if role.ID == 0 {
			return apperror.New(http.StatusNotFound, 404003, "role not found")
		}
		if role.Code == "admin" {
			return apperror.New(http.StatusBadRequest, 400007, "admin role permissions cannot be changed")
		}
		if err := replaceRolePermissions(ctx, tx, role.ID, permissions); err != nil {
			return err
		}
		updated = RoleSummary{
			ID:          role.UID,
			Code:        role.Code,
			Name:        role.Name,
			Description: role.Description.String,
			BuiltIn:     role.BuiltIn,
			Status:      role.Status,
		}
		updated.Permissions, err = permissionsByRole(ctx, tx, role.ID)
		return err
	})
	if txErr != nil {
		if appErr, ok := txErr.(*apperror.Error); ok {
			return RoleSummary{}, appErr
		}
		return RoleSummary{}, apperror.Wrap(http.StatusInternalServerError, 500001, "update role permissions failed", txErr)
	}
	return updated, nil
}

func (s *Service) HasPermission(ctx context.Context, userUID, permissionCode string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Raw(
		`SELECT COUNT(*)
		   FROM users u
		   JOIN workspace_members wm ON wm.user_id = u.id AND wm.status = 'active'
		   JOIN user_roles ur ON ur.user_id = u.id AND ur.workspace_id = wm.workspace_id
		   JOIN roles r ON r.id = ur.role_id AND r.status = 'active'
		   JOIN role_permissions rp ON rp.role_id = r.id
		   JOIN permissions p ON p.id = rp.permission_id
		  WHERE u.uid = ?
		    AND u.status = 'active'
		    AND p.code = ?
		    AND (ur.expires_at IS NULL OR ur.expires_at > NOW(3))`,
		userUID, permissionCode,
	).Scan(&count).Error
	return count > 0, err
}

func (s *Service) defaultWorkspaceForAPI(ctx context.Context) (workspaceRecord, *apperror.Error) {
	workspace, err := defaultWorkspace(ctx, s.db, s.cfg.Bootstrap.WorkspaceSlug)
	if err != nil {
		return workspaceRecord{}, apperror.Wrap(http.StatusInternalServerError, 500001, "load workspace failed", err)
	}
	if workspace.ID == 0 {
		return workspaceRecord{}, apperror.New(http.StatusInternalServerError, 500002, "default workspace is not initialized")
	}
	return workspace, nil
}

func (s *Service) rolesByWorkspace(ctx context.Context, workspaceID uint64) ([]RoleSummary, error) {
	var rows []roleRecord
	if err := s.db.WithContext(ctx).Raw(
		`SELECT id, uid, code, name, description, built_in, status
		   FROM roles
		  WHERE workspace_id = ?
		  ORDER BY built_in DESC, code`,
		workspaceID,
	).Scan(&rows).Error; err != nil {
		return nil, err
	}
	roles := make([]RoleSummary, 0, len(rows))
	for _, row := range rows {
		permissions, err := permissionsByRole(ctx, s.db, row.ID)
		if err != nil {
			return nil, err
		}
		role := RoleSummary{
			ID:          row.UID,
			Code:        row.Code,
			Name:        row.Name,
			Description: row.Description.String,
			BuiltIn:     row.BuiltIn,
			Status:      row.Status,
			Permissions: permissions,
		}
		roles = append(roles, role)
	}
	return roles, nil
}

type roleRecord struct {
	ID          uint64
	UID         string
	Code        string
	Name        string
	Description sql.NullString
	BuiltIn     bool
	Status      string
}

func userExists(ctx context.Context, tx *gorm.DB, username, email string) (bool, error) {
	var count int64
	if err := tx.WithContext(ctx).Raw("SELECT COUNT(*) FROM users WHERE username = ? OR (? <> '' AND email = ?)", username, email, email).Scan(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func userByUID(ctx context.Context, db *gorm.DB, uid string) (userRecord, error) {
	var user userRecord
	err := db.WithContext(ctx).Raw("SELECT id, uid, username, email, password_hash, status FROM users WHERE uid = ? LIMIT 1", uid).Scan(&user).Error
	return user, err
}

func roleByUID(ctx context.Context, db *gorm.DB, uid string) (roleRecord, error) {
	var role roleRecord
	err := db.WithContext(ctx).Raw(
		"SELECT id, uid, code, name, description, built_in, status FROM roles WHERE uid = ? LIMIT 1",
		uid,
	).Scan(&role).Error
	return role, err
}

func replaceUserRoles(ctx context.Context, tx *gorm.DB, workspaceID, userID uint64, roleCodes []string) error {
	roleIDs := make([]uint64, 0, len(roleCodes))
	for _, code := range roleCodes {
		id, err := roleID(ctx, tx, workspaceID, code)
		if err != nil {
			return err
		}
		roleIDs = append(roleIDs, id)
	}
	if err := tx.WithContext(ctx).Exec("DELETE FROM user_roles WHERE workspace_id = ? AND user_id = ?", workspaceID, userID).Error; err != nil {
		return err
	}
	for _, roleID := range roleIDs {
		if err := seedUserRole(ctx, tx, workspaceID, userID, roleID); err != nil {
			return err
		}
	}
	return nil
}

func replaceRolePermissions(ctx context.Context, tx *gorm.DB, roleID uint64, permissionCodes []string) error {
	if err := tx.WithContext(ctx).Exec("DELETE FROM role_permissions WHERE role_id = ?", roleID).Error; err != nil {
		return err
	}
	for _, code := range permissionCodes {
		var permissionID uint64
		if err := tx.WithContext(ctx).Raw("SELECT id FROM permissions WHERE code = ? LIMIT 1", code).Scan(&permissionID).Error; err != nil {
			return err
		}
		if permissionID == 0 {
			return errors.New("permission not found: " + code)
		}
		if err := tx.WithContext(ctx).Exec("INSERT IGNORE INTO role_permissions(role_id, permission_id) VALUES (?, ?)", roleID, permissionID).Error; err != nil {
			return err
		}
	}
	return nil
}

func permissionsByRole(ctx context.Context, db *gorm.DB, roleID uint64) ([]PermissionSummary, error) {
	var permissions []PermissionSummary
	err := db.WithContext(ctx).Raw(
		`SELECT p.code, p.module, p.name, COALESCE(p.description, '') AS description
		   FROM role_permissions rp
		   JOIN permissions p ON p.id = rp.permission_id
		  WHERE rp.role_id = ?
		  ORDER BY p.module, p.code`,
		roleID,
	).Scan(&permissions).Error
	return permissions, err
}

func normalizeRoleCodes(values []string) []string {
	seen := map[string]bool{}
	roles := make([]string, 0, len(values))
	for _, value := range values {
		code := strings.TrimSpace(value)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		roles = append(roles, code)
	}
	sort.Strings(roles)
	return roles
}

func normalizePermissionCodes(values []string) []string {
	seen := map[string]bool{}
	permissions := make([]string, 0, len(values))
	for _, value := range values {
		code := strings.TrimSpace(value)
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		permissions = append(permissions, code)
	}
	sort.Strings(permissions)
	return permissions
}

func splitCSV(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func isDuplicateError(err error) bool {
	return strings.Contains(strings.ToLower(err.Error()), "duplicate")
}
