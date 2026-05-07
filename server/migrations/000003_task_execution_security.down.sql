-- Rollback T3/T4 task execution security additions.

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

START TRANSACTION;

DELETE rp FROM role_permissions rp
JOIN permissions p ON p.id = rp.permission_id
WHERE p.code IN (
  'agent:read', 'agent:write', 'host:read', 'host:write',
  'script:read', 'script:write', 'script:approve',
  'task:read', 'task:write', 'task:execute', 'task:cancel', 'task:log:read'
  , 'schedule:read', 'schedule:write', 'webhook:read', 'webhook:manage', 'alert:read', 'alert:write'
  , 'notification:read', 'notification:write'
);

DELETE FROM permissions
WHERE code IN (
  'agent:read', 'agent:write', 'host:read', 'host:write',
  'script:read', 'script:write', 'script:approve',
  'task:read', 'task:write', 'task:execute', 'task:cancel', 'task:log:read'
  , 'schedule:read', 'schedule:write', 'webhook:read', 'webhook:manage', 'alert:read', 'alert:write'
  , 'notification:read', 'notification:write'
);

DROP TABLE IF EXISTS agent_enrollment_tokens;

ALTER TABLE task_run_logs DROP INDEX idx_task_run_logs_source_timestamp;
ALTER TABLE task_run_logs DROP COLUMN source_timestamp;

ALTER TABLE script_templates DROP CHECK chk_script_templates_type;
ALTER TABLE script_templates
  ADD CONSTRAINT chk_script_templates_type CHECK (script_type IN ('shell', 'powershell', 'python', 'custom'));

COMMIT;

SET FOREIGN_KEY_CHECKS = 1;
