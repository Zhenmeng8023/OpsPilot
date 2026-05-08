-- Source: archive/v1.0-incremental/000015_v10_maintenance_host_group_scope.down.sql
ALTER TABLE maintenance_windows
  DROP CHECK chk_maintenance_windows_scope,
  ADD CONSTRAINT chk_maintenance_windows_scope CHECK (scope_type IN ('all', 'agent', 'host'));

ALTER TABLE maintenance_windows
  DROP FOREIGN KEY fk_maintenance_windows_host_group,
  DROP INDEX idx_maintenance_windows_host_group,
  DROP INDEX idx_maintenance_windows_scope,
  DROP COLUMN host_group_id,
  ADD KEY idx_maintenance_windows_scope (workspace_id, status, scope_type, starts_at, ends_at);

-- Source: archive/v1.0-incremental/000014_v10_alert_host_group_matching.down.sql
ALTER TABLE alert_routing_policies
  DROP INDEX idx_alert_routing_policies_match,
  DROP COLUMN host_group_uid,
  ADD KEY idx_alert_routing_policies_match (workspace_id, status, alert_rule_uid, host_uid, severity);

ALTER TABLE alert_suppression_rules
  DROP INDEX idx_alert_suppression_rules_match,
  DROP COLUMN host_group_uid,
  ADD KEY idx_alert_suppression_rules_match (workspace_id, status, alert_rule_uid, host_uid, severity);

-- Source: archive/v1.0-incremental/000013_v10_agent_fleet_operations.down.sql
DROP TABLE IF EXISTS maintenance_windows;
DROP TABLE IF EXISTS agent_diagnostics;

-- Source: archive/v1.0-incremental/000012_v10_alert_routing_suppression.down.sql
DROP TABLE IF EXISTS alert_routing_policies;
DROP TABLE IF EXISTS alert_suppression_rules;

-- Source: archive/v1.0-incremental/000011_v10_metrics_lifecycle.down.sql
DELETE rp FROM role_permissions rp
JOIN permissions p ON p.id = rp.permission_id
WHERE p.code = 'metric:write';

DELETE FROM permissions WHERE code = 'metric:write';

DROP TABLE IF EXISTS metric_dashboards;
DROP TABLE IF EXISTS host_metric_rollups;

-- Source: archive/v1.0-incremental/000010_v10_notification_templates.down.sql
SET NAMES utf8mb4;

DROP TABLE IF EXISTS notification_templates;

-- Source: archive/v1.0-incremental/000009_v10_workflow_versions.down.sql
SET NAMES utf8mb4;

DROP TABLE IF EXISTS workflow_versions;

-- Source: archive/v1.0-incremental/000008_v10_secret_audit_webhook_hardening.down.sql
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

-- Source: archive/v1.0-incremental/000007_v10_workflow_triggers.down.sql
SET NAMES utf8mb4;

DELETE FROM webhook_event_matches WHERE workflow_run_id IS NOT NULL;
DELETE FROM schedule_triggers WHERE workflow_run_id IS NOT NULL;
DELETE FROM webhook_rules WHERE workflow_id IS NOT NULL;
DELETE FROM schedules WHERE workflow_id IS NOT NULL;

ALTER TABLE webhook_event_matches
  DROP FOREIGN KEY fk_webhook_event_matches_workflow_run,
  DROP INDEX idx_webhook_event_matches_workflow_run,
  DROP COLUMN workflow_run_id;

ALTER TABLE webhook_rules
  DROP FOREIGN KEY fk_webhook_rules_workflow,
  DROP INDEX idx_webhook_rules_workflow,
  DROP COLUMN target_type,
  DROP COLUMN workflow_id;

ALTER TABLE webhook_rules
  MODIFY COLUMN task_id BIGINT UNSIGNED NOT NULL;

ALTER TABLE schedule_triggers
  DROP FOREIGN KEY fk_schedule_triggers_workflow_run,
  DROP INDEX idx_schedule_triggers_workflow_run,
  DROP COLUMN workflow_run_id;

ALTER TABLE schedules
  DROP FOREIGN KEY fk_schedules_workflow,
  DROP INDEX idx_schedules_workflow,
  DROP COLUMN target_type,
  DROP COLUMN workflow_id;

ALTER TABLE schedules
  MODIFY COLUMN task_id BIGINT UNSIGNED NOT NULL;

-- Source: archive/v1.0-incremental/000006_v10_workflows.down.sql
SET NAMES utf8mb4;
SET time_zone = '+08:00';

START TRANSACTION;

DROP TABLE IF EXISTS workflow_run_events;
DROP TABLE IF EXISTS workflow_run_nodes;
DROP TABLE IF EXISTS workflow_runs;
DROP TABLE IF EXISTS workflow_definitions;

DELETE rp FROM role_permissions rp
JOIN permissions p ON p.id = rp.permission_id
WHERE p.code IN ('workflow:read', 'workflow:manage', 'workflow:write', 'workflow:execute', 'workflow:cancel');

DELETE FROM permissions
WHERE code IN ('workflow:read', 'workflow:manage', 'workflow:write', 'workflow:execute', 'workflow:cancel');

COMMIT;

-- Source: archive/v1.0-incremental/000005_v10_incidents.down.sql
SET NAMES utf8mb4;
SET time_zone = '+08:00';

START TRANSACTION;

DROP TABLE IF EXISTS incident_events;
DROP TABLE IF EXISTS incident_alerts;
DROP TABLE IF EXISTS incidents;

COMMIT;


