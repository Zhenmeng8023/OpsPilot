-- V1.0 hardening for secret storage, webhook operations, and audit governance.

SET NAMES utf8mb4;
SET time_zone = '+08:00';

START TRANSACTION;

ALTER TABLE webhook_sources
  MODIFY COLUMN signing_secret TEXT NULL AFTER token_hash;

ALTER TABLE webhook_sources
  DROP CHECK chk_webhook_sources_status,
  ADD CONSTRAINT chk_webhook_sources_status CHECK (status IN ('active', 'paused', 'disabled', 'archived'));

ALTER TABLE webhook_rules
  DROP CHECK chk_webhook_rules_status,
  ADD CONSTRAINT chk_webhook_rules_status CHECK (status IN ('active', 'paused', 'disabled', 'archived'));

ALTER TABLE audit_logs
  ADD KEY idx_audit_logs_actor_type_created (actor_type, created_at),
  ADD KEY idx_audit_logs_trace_actor_resource (trace_id, actor_type, resource_type, resource_id, created_at);

INSERT INTO permissions(code, module, name, description)
VALUES
  ('workflow:read', 'workflow', 'Read workflows', 'View workflow definitions and runs'),
  ('workflow:manage', 'workflow', 'Manage workflows', 'Create, update, publish, and cancel workflows'),
  ('workflow:write', 'workflow', 'Write workflows', 'Create, update, publish, and disable workflows'),
  ('workflow:execute', 'workflow', 'Execute workflows', 'Start workflow runs'),
  ('workflow:cancel', 'workflow', 'Cancel workflows', 'Cancel running workflow runs')
ON DUPLICATE KEY UPDATE
  module = VALUES(module),
  name = VALUES(name),
  description = VALUES(description);

INSERT IGNORE INTO role_permissions(role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('workflow:read', 'workflow:manage', 'workflow:write', 'workflow:execute', 'workflow:cancel')
WHERE r.code = 'admin';

COMMIT;
