-- T3/T4 task execution, live logs, Agent enrollment, and colon-style RBAC permissions.

SET NAMES utf8mb4;
SET time_zone = '+08:00';

START TRANSACTION;

ALTER TABLE script_templates DROP CHECK chk_script_templates_type;
ALTER TABLE script_templates
  ADD CONSTRAINT chk_script_templates_type CHECK (script_type IN ('shell', 'powershell', 'bash', 'python', 'custom'));

ALTER TABLE task_run_logs
  ADD COLUMN source_timestamp DATETIME(3) NULL AFTER content_hash,
  ADD KEY idx_task_run_logs_source_timestamp (source_timestamp);

CREATE TABLE IF NOT EXISTS agent_enrollment_tokens (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  uid CHAR(26) NOT NULL,
  workspace_id BIGINT UNSIGNED NOT NULL,
  token_hash CHAR(64) NOT NULL,
  token_prefix VARCHAR(16) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  max_uses INT UNSIGNED NOT NULL DEFAULT 1,
  used_count INT UNSIGNED NOT NULL DEFAULT 0,
  bind_workspace_slug VARCHAR(128) NULL,
  expires_at DATETIME(3) NOT NULL,
  created_by BIGINT UNSIGNED NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  revoked_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_agent_enrollment_tokens_uid (uid),
  UNIQUE KEY uk_agent_enrollment_tokens_hash (token_hash),
  KEY idx_agent_enrollment_tokens_workspace_status (workspace_id, status, expires_at),
  KEY idx_agent_enrollment_tokens_created_by (created_by),
  CONSTRAINT fk_agent_enrollment_tokens_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
  CONSTRAINT fk_agent_enrollment_tokens_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
  CONSTRAINT chk_agent_enrollment_tokens_status CHECK (status IN ('active', 'used', 'revoked', 'expired'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

INSERT INTO permissions(code, module, name, description)
VALUES
  ('agent:read', 'agent', 'Read agents', 'View agents'),
  ('agent:write', 'agent', 'Write agents', 'Disable agents and revoke tokens'),
  ('host:read', 'host', 'Read hosts', 'View hosts'),
  ('host:write', 'host', 'Write hosts', 'Manage hosts'),
  ('script:read', 'script', 'Read scripts', 'View script templates and versions'),
  ('script:write', 'script', 'Write scripts', 'Create and update scripts'),
  ('script:approve', 'script', 'Approve scripts', 'Approve high risk scripts'),
  ('task:read', 'task', 'Read tasks', 'View task runs'),
  ('task:write', 'task', 'Write tasks', 'Manage task definitions'),
  ('task:execute', 'task', 'Execute tasks', 'Create and run tasks'),
  ('task:cancel', 'task', 'Cancel tasks', 'Cancel queued tasks'),
  ('task:log:read', 'task', 'Read task logs', 'View and stream task logs')
  ,('schedule:read', 'schedule', 'Read schedules', 'View task schedules')
  ,('schedule:write', 'schedule', 'Write schedules', 'Create and manage task schedules')
  ,('webhook:read', 'webhook', 'Read webhooks', 'View webhook sources and rules')
  ,('webhook:manage', 'webhook', 'Manage webhooks', 'Manage webhook sources and trigger rules')
  ,('alert:read', 'alert', 'Read alerts', 'View alert rules and alert events')
  ,('alert:write', 'alert', 'Write alerts', 'Manage alert rules and alert state')
  ,('notification:read', 'notification', 'Read notifications', 'View notification channels and messages')
  ,('notification:write', 'notification', 'Write notifications', 'Manage notification channels')
ON DUPLICATE KEY UPDATE
  module = VALUES(module),
  name = VALUES(name),
  description = VALUES(description);

INSERT IGNORE INTO role_permissions(role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN (
  'agent:read', 'agent:write', 'host:read', 'host:write',
  'script:read', 'script:write', 'script:approve',
  'task:read', 'task:write', 'task:execute', 'task:cancel', 'task:log:read'
  , 'schedule:read', 'schedule:write', 'webhook:read', 'webhook:manage', 'alert:read', 'alert:write'
  , 'notification:read', 'notification:write'
)
WHERE r.code = 'admin';

INSERT IGNORE INTO role_permissions(role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('agent:read', 'host:read', 'script:read', 'task:read', 'task:log:read')
WHERE r.code = 'member';

COMMIT;
