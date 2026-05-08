SET NAMES utf8mb4;
SET time_zone = '+08:00';

START TRANSACTION;

DELETE rp
FROM role_permissions rp
JOIN permissions p ON p.id = rp.permission_id
JOIN roles r ON r.id = rp.role_id
WHERE r.code = 'admin'
  AND p.code IN ('workflow:read', 'workflow:manage', 'workflow:write', 'workflow:execute', 'workflow:cancel');

DELETE FROM permissions
WHERE code IN ('workflow:read', 'workflow:manage', 'workflow:write', 'workflow:execute', 'workflow:cancel');

ALTER TABLE audit_logs
  DROP INDEX idx_audit_logs_actor_type_created,
  DROP INDEX idx_audit_logs_trace_actor_resource;

ALTER TABLE webhook_rules
  DROP CHECK chk_webhook_rules_status,
  ADD CONSTRAINT chk_webhook_rules_status CHECK (status IN ('active', 'disabled', 'archived'));

ALTER TABLE webhook_sources
  DROP CHECK chk_webhook_sources_status,
  ADD CONSTRAINT chk_webhook_sources_status CHECK (status IN ('active', 'disabled', 'archived'));

ALTER TABLE webhook_sources
  MODIFY COLUMN signing_secret VARCHAR(128) NULL AFTER token_hash;

COMMIT;
