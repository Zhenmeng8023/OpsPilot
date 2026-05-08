package auth

import (
	"context"
	"database/sql"
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"opspilot/server/internal/config"
)

type permissionSeed struct {
	Code        string
	Module      string
	Name        string
	Description string
}

var basePermissions = []permissionSeed{
	{"workspace.read", "workspace", "Read workspace", "View workspace metadata"},
	{"user.read", "user", "Read users", "View users and members"},
	{"user.write", "user", "Write users", "Create and update users"},
	{"role.read", "role", "Read roles", "View roles and permissions"},
	{"role.write", "role", "Write roles", "Manage roles and permissions"},
	{"agent.read", "agent", "Read agents", "View agents and hosts"},
	{"agent.disable", "agent", "Disable agents", "Disable or revoke agents"},
	{"agent:read", "agent", "Read agents", "View agents"},
	{"agent:write", "agent", "Write agents", "Disable agents and revoke tokens"},
	{"host:read", "host", "Read hosts", "View hosts"},
	{"host:write", "host", "Write hosts", "Manage hosts"},
	{"script.read", "script", "Read scripts", "View script templates and versions"},
	{"script.write", "script", "Write scripts", "Create and update scripts"},
	{"script.approve", "script", "Approve scripts", "Approve high risk scripts"},
	{"script:read", "script", "Read scripts", "View script templates and versions"},
	{"script:write", "script", "Write scripts", "Create and update scripts"},
	{"script:approve", "script", "Approve scripts", "Approve high risk scripts"},
	{"task.read", "task", "Read tasks", "View task definitions and runs"},
	{"task.run", "task", "Run tasks", "Run automation tasks"},
	{"task.cancel", "task", "Cancel tasks", "Cancel running tasks"},
	{"log.read", "log", "Read logs", "View task execution logs"},
	{"task:read", "task", "Read tasks", "View task runs"},
	{"task:write", "task", "Write tasks", "Manage task definitions"},
	{"task:execute", "task", "Execute tasks", "Create and run tasks"},
	{"task:cancel", "task", "Cancel tasks", "Cancel queued tasks"},
	{"task:log:read", "task", "Read task logs", "View and stream task logs"},
	{"schedule.write", "schedule", "Write schedules", "Manage schedules"},
	{"schedule:read", "schedule", "Read schedules", "View task schedules"},
	{"schedule:write", "schedule", "Write schedules", "Create and manage task schedules"},
	{"metric.read", "metric", "Read metrics", "View metrics and service checks"},
	{"metric:read", "metric", "Read metrics", "View metrics and service checks"},
	{"alert:read", "alert", "Read alerts", "View alert rules and alert events"},
	{"alert:write", "alert", "Write alerts", "Manage alert rules and alert state"},
	{"alert.write", "alert", "Write alerts", "Manage alert rules and alert state"},
	{"webhook.manage", "webhook", "Manage webhooks", "Manage webhook sources and rules"},
	{"webhook:read", "webhook", "Read webhooks", "View webhook sources and rules"},
	{"webhook:manage", "webhook", "Manage webhooks", "Manage webhook sources and trigger rules"},
	{"notification:read", "notification", "Read notifications", "View notification channels and messages"},
	{"notification:write", "notification", "Write notifications", "Manage notification channels"},
	{"notification.write", "notification", "Write notifications", "Manage notification channels"},
	{"audit.read", "audit", "Read audit logs", "View audit log entries"},
	{"workflow:read", "workflow", "Read workflows", "View workflow definitions and runs"},
	{"workflow:write", "workflow", "Write workflows", "Create, update, publish, and disable workflows"},
	{"workflow:execute", "workflow", "Execute workflows", "Start workflow runs"},
	{"workflow:cancel", "workflow", "Cancel workflows", "Cancel running workflow runs"},
}

func Seed(ctx context.Context, db *gorm.DB, cfg config.Config) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		workspaceID, err := seedWorkspace(ctx, tx, cfg)
		if err != nil {
			return err
		}
		permissionIDs, err := seedPermissions(ctx, tx)
		if err != nil {
			return err
		}
		adminRoleID, err := seedRole(ctx, tx, workspaceID, "admin", "Administrator", "Full workspace administrator", true)
		if err != nil {
			return err
		}
		memberRoleID, err := seedRole(ctx, tx, workspaceID, "member", "Member", "Default workspace member", true)
		if err != nil {
			return err
		}
		if err := assignPermissions(ctx, tx, adminRoleID, permissionIDs); err != nil {
			return err
		}
		memberPermissions := filterPermissions(permissionIDs, "workspace.read", "agent:read", "host:read", "script:read", "task:read", "task:log:read", "metric:read")
		if err := assignPermissions(ctx, tx, memberRoleID, memberPermissions); err != nil {
			return err
		}
		adminUserID, err := seedAdminUser(ctx, tx, cfg)
		if err != nil {
			return err
		}
		if err := seedMembership(ctx, tx, workspaceID, adminUserID); err != nil {
			return err
		}
		return seedUserRole(ctx, tx, workspaceID, adminUserID, adminRoleID)
	})
}

func seedWorkspace(ctx context.Context, tx *gorm.DB, cfg config.Config) (uint64, error) {
	uid, err := newUID()
	if err != nil {
		return 0, err
	}
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO workspaces(uid, name, slug, status)
		 VALUES (?, ?, ?, 'active')
		 ON DUPLICATE KEY UPDATE name = VALUES(name), status = 'active'`,
		uid, cfg.Bootstrap.WorkspaceName, cfg.Bootstrap.WorkspaceSlug,
	).Error; err != nil {
		return 0, err
	}
	var id uint64
	err = tx.WithContext(ctx).Raw("SELECT id FROM workspaces WHERE slug = ? LIMIT 1", cfg.Bootstrap.WorkspaceSlug).Scan(&id).Error
	return id, err
}

func seedPermissions(ctx context.Context, tx *gorm.DB) (map[string]uint64, error) {
	ids := make(map[string]uint64, len(basePermissions))
	for _, permission := range basePermissions {
		if err := tx.WithContext(ctx).Exec(
			`INSERT INTO permissions(code, module, name, description)
			 VALUES (?, ?, ?, ?)
			 ON DUPLICATE KEY UPDATE module = VALUES(module), name = VALUES(name), description = VALUES(description)`,
			permission.Code, permission.Module, permission.Name, permission.Description,
		).Error; err != nil {
			return nil, err
		}
		var id uint64
		if err := tx.WithContext(ctx).Raw("SELECT id FROM permissions WHERE code = ? LIMIT 1", permission.Code).Scan(&id).Error; err != nil {
			return nil, err
		}
		ids[permission.Code] = id
	}
	return ids, nil
}

func seedRole(ctx context.Context, tx *gorm.DB, workspaceID uint64, code, name, description string, builtIn bool) (uint64, error) {
	uid, err := newUID()
	if err != nil {
		return 0, err
	}
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO roles(uid, workspace_id, code, name, description, built_in, status)
		 VALUES (?, ?, ?, ?, ?, ?, 'active')
		 ON DUPLICATE KEY UPDATE name = VALUES(name), description = VALUES(description), built_in = VALUES(built_in), status = 'active'`,
		uid, workspaceID, code, name, description, builtIn,
	).Error; err != nil {
		return 0, err
	}
	var id uint64
	err = tx.WithContext(ctx).Raw("SELECT id FROM roles WHERE workspace_id = ? AND code = ? LIMIT 1", workspaceID, code).Scan(&id).Error
	return id, err
}

func assignPermissions(ctx context.Context, tx *gorm.DB, roleID uint64, permissionIDs map[string]uint64) error {
	for _, permissionID := range permissionIDs {
		if err := tx.WithContext(ctx).Exec(
			"INSERT IGNORE INTO role_permissions(role_id, permission_id) VALUES (?, ?)",
			roleID, permissionID,
		).Error; err != nil {
			return err
		}
	}
	return nil
}

func seedAdminUser(ctx context.Context, tx *gorm.DB, cfg config.Config) (uint64, error) {
	uid, err := newUID()
	if err != nil {
		return 0, err
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(cfg.Bootstrap.AdminPassword), bcrypt.DefaultCost)
	if err != nil {
		return 0, err
	}
	if err := tx.WithContext(ctx).Exec(
		`INSERT INTO users(uid, username, email, password_hash, status, password_changed_at)
		 VALUES (?, ?, ?, ?, 'active', NOW(3))
		 ON DUPLICATE KEY UPDATE status = 'active'`,
		uid, cfg.Bootstrap.AdminUsername, nullString(cfg.Bootstrap.AdminEmail), string(passwordHash),
	).Error; err != nil {
		return 0, err
	}
	var id uint64
	err = tx.WithContext(ctx).Raw("SELECT id FROM users WHERE username = ? LIMIT 1", cfg.Bootstrap.AdminUsername).Scan(&id).Error
	return id, err
}

func seedMembership(ctx context.Context, tx *gorm.DB, workspaceID, userID uint64) error {
	return tx.WithContext(ctx).Exec(
		`INSERT INTO workspace_members(workspace_id, user_id, member_type, status)
		 VALUES (?, ?, 'human', 'active')
		 ON DUPLICATE KEY UPDATE status = 'active'`,
		workspaceID, userID,
	).Error
}

func seedUserRole(ctx context.Context, tx *gorm.DB, workspaceID, userID, roleID uint64) error {
	return tx.WithContext(ctx).Exec(
		"INSERT IGNORE INTO user_roles(workspace_id, user_id, role_id) VALUES (?, ?, ?)",
		workspaceID, userID, roleID,
	).Error
}

func filterPermissions(permissionIDs map[string]uint64, codes ...string) map[string]uint64 {
	filtered := make(map[string]uint64, len(codes))
	for _, code := range codes {
		if id, ok := permissionIDs[code]; ok {
			filtered[code] = id
		}
	}
	return filtered
}

func nullableUint64(value *uint64) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*value), Valid: true}
}

func missingSeed(name string) error {
	return fmt.Errorf("missing bootstrap seed: %s", name)
}
