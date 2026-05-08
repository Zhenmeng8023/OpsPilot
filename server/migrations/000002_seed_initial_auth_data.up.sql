-- OpsPilot initial auth/RBAC seed data for MySQL 8.0.
-- Default login after applying this seed to an empty database:
--   username: admin
--   password: Admin@123456
--
-- This seed is idempotent. Re-running it restores the default workspace,
-- permissions, built-in roles, admin membership, and admin role binding.
-- It does not reset an existing admin user's password.

SET NAMES utf8mb4;
SET time_zone = '+08:00';

START TRANSACTION;

SET @workspace_uid := '01H00000000000000000000001';
SET @workspace_name := 'Default Workspace';
SET @workspace_slug := 'default';

SET @admin_role_uid := '01H00000000000000000000002';
SET @member_role_uid := '01H00000000000000000000003';

SET @admin_user_uid := '01H00000000000000000000004';
SET @admin_username := 'admin';
SET @admin_email := 'admin@opspilot.local';
SET @admin_display_name := 'System Administrator';
SET @admin_password_hash := '$2a$10$zw441Avx32eIlyBtT8G4c.Hw0OlyG/T6CeAKUqoCDHcGuVQFSKd7O';

INSERT INTO workspaces(uid, name, slug, status)
VALUES (@workspace_uid, @workspace_name, @workspace_slug, 'active')
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  status = 'active',
  deleted_at = NULL;

SET @workspace_id := (SELECT id FROM workspaces WHERE slug = @workspace_slug LIMIT 1);

INSERT INTO permissions(code, module, name, description)
VALUES
  ('workspace.read', 'workspace', 'Read workspace', 'View workspace metadata'),
  ('user.read', 'user', 'Read users', 'View users and members'),
  ('user.write', 'user', 'Write users', 'Create and update users'),
  ('role.read', 'role', 'Read roles', 'View roles and permissions'),
  ('role.write', 'role', 'Write roles', 'Manage roles and permissions'),
  ('agent.read', 'agent', 'Read agents', 'View agents and hosts'),
  ('agent.disable', 'agent', 'Disable agents', 'Disable or revoke agents'),
  ('agent:read', 'agent', 'Read agents', 'View agents'),
  ('agent:write', 'agent', 'Write agents', 'Disable agents and revoke tokens'),
  ('host:read', 'host', 'Read hosts', 'View hosts'),
  ('host:write', 'host', 'Write hosts', 'Manage hosts'),
  ('script.read', 'script', 'Read scripts', 'View script templates and versions'),
  ('script.write', 'script', 'Write scripts', 'Create and update scripts'),
  ('script.approve', 'script', 'Approve scripts', 'Approve high risk scripts'),
  ('script:read', 'script', 'Read scripts', 'View script templates and versions'),
  ('script:write', 'script', 'Write scripts', 'Create and update scripts'),
  ('script:approve', 'script', 'Approve scripts', 'Approve high risk scripts'),
  ('task.read', 'task', 'Read tasks', 'View task definitions and runs'),
  ('task.run', 'task', 'Run tasks', 'Run automation tasks'),
  ('task.cancel', 'task', 'Cancel tasks', 'Cancel running tasks'),
  ('log.read', 'log', 'Read logs', 'View task execution logs'),
  ('task:read', 'task', 'Read tasks', 'View task runs'),
  ('task:write', 'task', 'Write tasks', 'Manage task definitions'),
  ('task:execute', 'task', 'Execute tasks', 'Create and run tasks'),
  ('task:cancel', 'task', 'Cancel tasks', 'Cancel queued tasks'),
  ('task:log:read', 'task', 'Read task logs', 'View and stream task logs'),
  ('schedule.write', 'schedule', 'Write schedules', 'Manage schedules'),
  ('schedule:read', 'schedule', 'Read schedules', 'View task schedules'),
  ('schedule:write', 'schedule', 'Write schedules', 'Create and manage task schedules'),
  ('workflow:read', 'workflow', 'Read workflows', 'View workflow definitions and runs'),
  ('workflow:manage', 'workflow', 'Manage workflows', 'Create, update, publish, and cancel workflows'),
  ('workflow:write', 'workflow', 'Write workflows', 'Create, update, publish, and disable workflows'),
  ('workflow:execute', 'workflow', 'Execute workflows', 'Start workflow runs'),
  ('workflow:cancel', 'workflow', 'Cancel workflows', 'Cancel running workflow runs'),
  ('metric.read', 'metric', 'Read metrics', 'View metrics and service checks'),
  ('alert.write', 'alert', 'Write alerts', 'Manage alert rules and alert state'),
  ('webhook.manage', 'webhook', 'Manage webhooks', 'Manage webhook sources and rules'),
  ('webhook:read', 'webhook', 'Read webhooks', 'View webhook sources and rules'),
  ('webhook:manage', 'webhook', 'Manage webhooks', 'Manage webhook sources and trigger rules'),
  ('notification.write', 'notification', 'Write notifications', 'Manage notification channels'),
  ('audit.read', 'audit', 'Read audit logs', 'View audit log entries')
ON DUPLICATE KEY UPDATE
  module = VALUES(module),
  name = VALUES(name),
  description = VALUES(description);

INSERT INTO metric_definitions(code, name, unit, description)
VALUES
  ('agent.running_tasks', 'Running Tasks', 'count', 'Current number of tasks running on the Agent.'),
  ('agent.cpu.logical', 'Logical CPU', 'count', 'Logical CPU count visible to the Agent runtime.'),
  ('agent.runtime.goroutines', 'Goroutines', 'count', 'Current Go runtime goroutine count for the Agent.'),
  ('agent.runtime.alloc_bytes', 'Runtime Alloc', 'bytes', 'Bytes allocated and still in use by the Agent runtime.'),
  ('agent.runtime.sys_bytes', 'Runtime Sys', 'bytes', 'Bytes obtained from the OS by the Agent runtime.'),
  ('agent.os.cpu.percent', 'CPU Usage', 'percent', 'Host CPU usage percent collected by the Agent.'),
  ('agent.os.memory.used_bytes', 'Memory Used', 'bytes', 'Host memory bytes used.'),
  ('agent.os.memory.total_bytes', 'Memory Total', 'bytes', 'Host total memory bytes.'),
  ('agent.os.memory.used_percent', 'Memory Usage', 'percent', 'Host memory usage percent.'),
  ('agent.os.disk.used_bytes', 'Disk Used', 'bytes', 'Host root filesystem used bytes.'),
  ('agent.os.disk.total_bytes', 'Disk Total', 'bytes', 'Host root filesystem total bytes.'),
  ('agent.os.disk.used_percent', 'Disk Usage', 'percent', 'Host root filesystem usage percent.'),
  ('agent.os.network.bytes_sent', 'Network Sent', 'bytes', 'Host network bytes sent counter.'),
  ('agent.os.network.bytes_recv', 'Network Received', 'bytes', 'Host network bytes received counter.')
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  unit = VALUES(unit),
  description = VALUES(description);

INSERT INTO roles(uid, workspace_id, code, name, description, built_in, status)
VALUES
  (@admin_role_uid, @workspace_id, 'admin', 'Administrator', 'Full workspace administrator', 1, 'active'),
  (@member_role_uid, @workspace_id, 'member', 'Member', 'Default workspace member', 1, 'active')
ON DUPLICATE KEY UPDATE
  name = VALUES(name),
  description = VALUES(description),
  built_in = VALUES(built_in),
  status = 'active',
  deleted_at = NULL;

SET @admin_role_id := (
  SELECT id FROM roles
  WHERE workspace_id = @workspace_id AND code = 'admin'
  LIMIT 1
);

SET @member_role_id := (
  SELECT id FROM roles
  WHERE workspace_id = @workspace_id AND code = 'member'
  LIMIT 1
);

INSERT IGNORE INTO role_permissions(role_id, permission_id)
SELECT @admin_role_id, p.id
FROM permissions p
WHERE @admin_role_id IS NOT NULL
  AND p.code IN (
    'workspace.read',
    'user.read',
    'user.write',
    'role.read',
    'role.write',
    'agent.read',
    'agent.disable',
    'agent:read',
    'agent:write',
    'host:read',
    'host:write',
    'script.read',
    'script.write',
    'script.approve',
    'script:read',
    'script:write',
    'script:approve',
    'task.read',
    'task.run',
    'task.cancel',
    'log.read',
    'task:read',
    'task:write',
    'task:execute',
    'task:cancel',
    'task:log:read',
    'schedule.write',
    'schedule:read',
    'schedule:write',
    'workflow:read',
    'workflow:manage',
    'workflow:write',
    'workflow:execute',
    'workflow:cancel',
    'metric.read',
    'alert.write',
    'webhook.manage',
    'webhook:read',
    'webhook:manage',
    'notification.write',
    'audit.read'
  );

INSERT IGNORE INTO role_permissions(role_id, permission_id)
SELECT @member_role_id, p.id
FROM permissions p
WHERE @member_role_id IS NOT NULL
  AND p.code IN (
    'workspace.read',
    'agent.read',
    'agent:read',
    'host:read',
    'script.read',
    'script:read',
    'task.read',
    'task:read',
    'log.read',
    'task:log:read',
    'metric.read'
  );

INSERT INTO users(uid, username, email, display_name, password_hash, status, password_changed_at)
VALUES (
  @admin_user_uid,
  @admin_username,
  @admin_email,
  @admin_display_name,
  @admin_password_hash,
  'active',
  NOW(3)
)
ON DUPLICATE KEY UPDATE
  email = IF(email IS NULL, VALUES(email), email),
  display_name = IF(display_name IS NULL, VALUES(display_name), display_name),
  status = 'active',
  deleted_at = NULL;

SET @admin_user_id := (
  SELECT id FROM users
  WHERE username = @admin_username
  LIMIT 1
);

INSERT INTO workspace_members(workspace_id, user_id, member_type, status)
SELECT @workspace_id, @admin_user_id, 'human', 'active'
WHERE @workspace_id IS NOT NULL AND @admin_user_id IS NOT NULL
ON DUPLICATE KEY UPDATE
  status = 'active';

INSERT IGNORE INTO user_roles(workspace_id, user_id, role_id, granted_by)
SELECT @workspace_id, @admin_user_id, @admin_role_id, @admin_user_id
WHERE @workspace_id IS NOT NULL
  AND @admin_user_id IS NOT NULL
  AND @admin_role_id IS NOT NULL;

COMMIT;
