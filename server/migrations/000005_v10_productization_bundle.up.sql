-- Source: archive/v1.0-incremental/000005_v10_incidents.up.sql
-- V1.0 incident projection over alerts.

SET NAMES utf8mb4;
SET time_zone = '+08:00';

START TRANSACTION;

CREATE TABLE IF NOT EXISTS incidents (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  uid VARCHAR(40) NOT NULL,
  workspace_id BIGINT UNSIGNED NOT NULL,
  alert_rule_id BIGINT UNSIGNED NULL,
  resource_type VARCHAR(64) NOT NULL,
  resource_id BIGINT UNSIGNED NULL,
  title VARCHAR(255) NOT NULL,
  message VARCHAR(2048) NULL,
  severity VARCHAR(32) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'open',
  first_seen_at DATETIME(3) NOT NULL,
  last_seen_at DATETIME(3) NOT NULL,
  resolved_at DATETIME(3) NULL,
  metadata JSON NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_incidents_uid (uid),
  KEY idx_incidents_workspace_status_severity (workspace_id, status, severity),
  KEY idx_incidents_rule (alert_rule_id),
  KEY idx_incidents_resource (workspace_id, resource_type, resource_id),
  CONSTRAINT fk_incidents_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
  CONSTRAINT fk_incidents_rule FOREIGN KEY (alert_rule_id) REFERENCES alert_rules(id) ON DELETE SET NULL,
  CONSTRAINT chk_incidents_severity CHECK (severity IN ('info', 'warning', 'critical')),
  CONSTRAINT chk_incidents_status CHECK (status IN ('open', 'acknowledged', 'silenced', 'resolved'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS incident_alerts (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  incident_id BIGINT UNSIGNED NOT NULL,
  alert_id BIGINT UNSIGNED NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_incident_alerts_alert (alert_id),
  KEY idx_incident_alerts_incident (incident_id),
  CONSTRAINT fk_incident_alerts_incident FOREIGN KEY (incident_id) REFERENCES incidents(id) ON DELETE CASCADE,
  CONSTRAINT fk_incident_alerts_alert FOREIGN KEY (alert_id) REFERENCES alerts(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS incident_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  incident_id BIGINT UNSIGNED NOT NULL,
  alert_id BIGINT UNSIGNED NULL,
  event_type VARCHAR(64) NOT NULL,
  message VARCHAR(1024) NULL,
  actor_id BIGINT UNSIGNED NULL,
  payload JSON NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_incident_events_incident_created (incident_id, created_at),
  KEY idx_incident_events_alert (alert_id),
  KEY idx_incident_events_actor (actor_id),
  CONSTRAINT fk_incident_events_incident FOREIGN KEY (incident_id) REFERENCES incidents(id) ON DELETE CASCADE,
  CONSTRAINT fk_incident_events_alert FOREIGN KEY (alert_id) REFERENCES alerts(id) ON DELETE SET NULL,
  CONSTRAINT fk_incident_events_actor FOREIGN KEY (actor_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

INSERT INTO incidents(uid, workspace_id, alert_rule_id, resource_type, resource_id, title, message, severity, status,
                      first_seen_at, last_seen_at, resolved_at, metadata, created_at, updated_at)
SELECT CONCAT('inc_', a.uid), a.workspace_id, a.alert_rule_id, a.resource_type, a.resource_id, a.title, a.message,
       a.severity,
       CASE a.status
         WHEN 'firing' THEN 'open'
         WHEN 'acknowledged' THEN 'acknowledged'
         WHEN 'silenced' THEN 'silenced'
         ELSE 'resolved'
       END,
       a.first_seen_at, a.last_seen_at, a.resolved_at, a.metadata, a.created_at, a.updated_at
  FROM alerts a
  LEFT JOIN incident_alerts ia ON ia.alert_id = a.id
 WHERE ia.id IS NULL;

INSERT INTO incident_alerts(incident_id, alert_id, created_at)
SELECT i.id, a.id, a.created_at
  FROM alerts a
  JOIN incidents i ON i.uid = CONCAT('inc_', a.uid)
  LEFT JOIN incident_alerts ia ON ia.alert_id = a.id
 WHERE ia.id IS NULL;

INSERT INTO incident_events(incident_id, alert_id, event_type, message, actor_id, payload, created_at)
SELECT ia.incident_id, ae.alert_id, ae.event_type, ae.message, ae.actor_id, ae.payload, ae.created_at
  FROM alert_events ae
  JOIN incident_alerts ia ON ia.alert_id = ae.alert_id
  LEFT JOIN incident_events ie ON ie.incident_id = ia.incident_id
                              AND ie.alert_id = ae.alert_id
                              AND ie.event_type = ae.event_type
                              AND ie.created_at = ae.created_at
 WHERE ie.id IS NULL;

COMMIT;

-- Source: archive/v1.0-incremental/000006_v10_workflows.up.sql
-- V1.0 workflow definitions, run records, and RBAC permissions.

SET NAMES utf8mb4;
SET time_zone = '+08:00';

START TRANSACTION;

CREATE TABLE IF NOT EXISTS workflow_definitions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  uid CHAR(26) NOT NULL,
  workspace_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(128) NOT NULL,
  description VARCHAR(1024) NULL,
  definition JSON NOT NULL,
  version INT UNSIGNED NOT NULL DEFAULT 1,
  status VARCHAR(32) NOT NULL DEFAULT 'draft',
  created_by BIGINT UNSIGNED NULL,
  published_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_workflow_definitions_uid (uid),
  UNIQUE KEY uk_workflow_definitions_workspace_name (workspace_id, name),
  KEY idx_workflow_definitions_workspace_status (workspace_id, status),
  KEY idx_workflow_definitions_created_by (created_by),
  CONSTRAINT fk_workflow_definitions_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
  CONSTRAINT fk_workflow_definitions_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
  CONSTRAINT chk_workflow_definitions_status CHECK (status IN ('draft', 'active', 'disabled', 'archived'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS workflow_runs (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  uid CHAR(26) NOT NULL,
  workspace_id BIGINT UNSIGNED NOT NULL,
  workflow_id BIGINT UNSIGNED NOT NULL,
  workflow_version INT UNSIGNED NOT NULL,
  definition_snapshot JSON NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  trigger_type VARCHAR(32) NOT NULL DEFAULT 'manual',
  trigger_id BIGINT UNSIGNED NULL,
  idempotency_key VARCHAR(128) NULL,
  input JSON NULL,
  output JSON NULL,
  total_nodes INT UNSIGNED NOT NULL DEFAULT 0,
  success_nodes INT UNSIGNED NOT NULL DEFAULT 0,
  failed_nodes INT UNSIGNED NOT NULL DEFAULT 0,
  skipped_nodes INT UNSIGNED NOT NULL DEFAULT 0,
  error_message VARCHAR(2048) NULL,
  created_by BIGINT UNSIGNED NULL,
  cancel_requested_by BIGINT UNSIGNED NULL,
  cancel_reason VARCHAR(512) NULL,
  queued_at DATETIME(3) NULL,
  started_at DATETIME(3) NULL,
  finished_at DATETIME(3) NULL,
  duration_ms BIGINT UNSIGNED NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_workflow_runs_uid (uid),
  UNIQUE KEY uk_workflow_runs_idempotency (workspace_id, idempotency_key),
  KEY idx_workflow_runs_workspace_status_created (workspace_id, status, created_at),
  KEY idx_workflow_runs_workflow_created (workflow_id, created_at),
  KEY idx_workflow_runs_created_by (created_by),
  KEY idx_workflow_runs_cancel_requested_by (cancel_requested_by),
  CONSTRAINT fk_workflow_runs_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
  CONSTRAINT fk_workflow_runs_workflow FOREIGN KEY (workflow_id) REFERENCES workflow_definitions(id) ON DELETE RESTRICT,
  CONSTRAINT fk_workflow_runs_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
  CONSTRAINT fk_workflow_runs_cancel_by FOREIGN KEY (cancel_requested_by) REFERENCES users(id) ON DELETE SET NULL,
  CONSTRAINT chk_workflow_runs_status CHECK (status IN ('pending', 'queued', 'running', 'canceling', 'success', 'failed', 'canceled')),
  CONSTRAINT chk_workflow_runs_trigger CHECK (trigger_type IN ('manual', 'schedule', 'webhook', 'incident', 'api', 'retry'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS workflow_run_nodes (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  run_id BIGINT UNSIGNED NOT NULL,
  node_id VARCHAR(64) NOT NULL,
  node_type VARCHAR(64) NOT NULL,
  node_name VARCHAR(128) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  task_run_id BIGINT UNSIGNED NULL,
  input JSON NULL,
  output JSON NULL,
  error_message VARCHAR(2048) NULL,
  attempts INT UNSIGNED NOT NULL DEFAULT 0,
  queued_at DATETIME(3) NULL,
  started_at DATETIME(3) NULL,
  finished_at DATETIME(3) NULL,
  duration_ms BIGINT UNSIGNED NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_workflow_run_nodes_run_node (run_id, node_id),
  KEY idx_workflow_run_nodes_status (run_id, status),
  KEY idx_workflow_run_nodes_task_run (task_run_id),
  CONSTRAINT fk_workflow_run_nodes_run FOREIGN KEY (run_id) REFERENCES workflow_runs(id) ON DELETE CASCADE,
  CONSTRAINT fk_workflow_run_nodes_task_run FOREIGN KEY (task_run_id) REFERENCES task_runs(id) ON DELETE SET NULL,
  CONSTRAINT chk_workflow_run_nodes_status CHECK (status IN ('pending', 'queued', 'running', 'success', 'failed', 'skipped', 'canceled'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS workflow_run_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  run_id BIGINT UNSIGNED NOT NULL,
  node_id VARCHAR(64) NULL,
  event_type VARCHAR(64) NOT NULL,
  message VARCHAR(1024) NULL,
  actor_id BIGINT UNSIGNED NULL,
  payload JSON NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_workflow_run_events_run_created (run_id, created_at),
  KEY idx_workflow_run_events_actor (actor_id),
  CONSTRAINT fk_workflow_run_events_run FOREIGN KEY (run_id) REFERENCES workflow_runs(id) ON DELETE CASCADE,
  CONSTRAINT fk_workflow_run_events_actor FOREIGN KEY (actor_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

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

-- Source: archive/v1.0-incremental/000007_v10_workflow_triggers.up.sql
SET NAMES utf8mb4;

ALTER TABLE schedules
  MODIFY COLUMN task_id BIGINT UNSIGNED NULL;

ALTER TABLE schedules
  ADD COLUMN workflow_id BIGINT UNSIGNED NULL AFTER task_id,
  ADD COLUMN target_type VARCHAR(32) NOT NULL DEFAULT 'task' AFTER workflow_id,
  ADD KEY idx_schedules_workflow (workflow_id),
  ADD CONSTRAINT fk_schedules_workflow FOREIGN KEY (workflow_id) REFERENCES workflow_definitions(id) ON DELETE CASCADE;

UPDATE schedules
   SET target_type = 'task'
 WHERE target_type <> 'task' OR workflow_id IS NULL;

ALTER TABLE schedule_triggers
  ADD COLUMN workflow_run_id BIGINT UNSIGNED NULL AFTER task_run_id,
  ADD KEY idx_schedule_triggers_workflow_run (workflow_run_id),
  ADD CONSTRAINT fk_schedule_triggers_workflow_run FOREIGN KEY (workflow_run_id) REFERENCES workflow_runs(id) ON DELETE SET NULL;

ALTER TABLE webhook_rules
  MODIFY COLUMN task_id BIGINT UNSIGNED NULL;

ALTER TABLE webhook_rules
  ADD COLUMN workflow_id BIGINT UNSIGNED NULL AFTER task_id,
  ADD COLUMN target_type VARCHAR(32) NOT NULL DEFAULT 'task' AFTER workflow_id,
  ADD KEY idx_webhook_rules_workflow (workflow_id),
  ADD CONSTRAINT fk_webhook_rules_workflow FOREIGN KEY (workflow_id) REFERENCES workflow_definitions(id) ON DELETE RESTRICT;

UPDATE webhook_rules
   SET target_type = 'task'
 WHERE target_type <> 'task' OR workflow_id IS NULL;

ALTER TABLE webhook_event_matches
  ADD COLUMN workflow_run_id BIGINT UNSIGNED NULL AFTER task_run_id,
  ADD KEY idx_webhook_event_matches_workflow_run (workflow_run_id),
  ADD CONSTRAINT fk_webhook_event_matches_workflow_run FOREIGN KEY (workflow_run_id) REFERENCES workflow_runs(id) ON DELETE SET NULL;

-- Source: archive/v1.0-incremental/000008_v10_secret_audit_webhook_hardening.up.sql
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

-- Source: archive/v1.0-incremental/000009_v10_workflow_versions.up.sql
SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS workflow_versions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  uid CHAR(26) NOT NULL,
  workflow_id BIGINT UNSIGNED NOT NULL,
  version_no INT UNSIGNED NOT NULL,
  definition JSON NOT NULL,
  definition_hash CHAR(64) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'draft',
  created_by BIGINT UNSIGNED NULL,
  published_at DATETIME(3) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_workflow_versions_uid (uid),
  UNIQUE KEY uk_workflow_versions_workflow_version (workflow_id, version_no),
  KEY idx_workflow_versions_status (workflow_id, status),
  KEY idx_workflow_versions_created_by (created_by),
  CONSTRAINT fk_workflow_versions_workflow FOREIGN KEY (workflow_id) REFERENCES workflow_definitions(id) ON DELETE CASCADE,
  CONSTRAINT fk_workflow_versions_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
  CONSTRAINT chk_workflow_versions_status CHECK (status IN ('draft', 'published', 'deprecated', 'archived'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

INSERT IGNORE INTO workflow_versions(uid, workflow_id, version_no, definition, definition_hash, status, created_by, published_at, created_at)
SELECT
  CONCAT('wfv', SUBSTRING(wd.uid, 4, 23)),
  wd.id,
  wd.version,
  wd.definition,
  SHA2(CAST(wd.definition AS CHAR), 256),
  CASE WHEN wd.status = 'active' THEN 'published' ELSE 'draft' END,
  wd.created_by,
  wd.published_at,
  wd.created_at
FROM workflow_definitions wd
WHERE wd.deleted_at IS NULL;

-- Source: archive/v1.0-incremental/000010_v10_notification_templates.up.sql
SET NAMES utf8mb4;

CREATE TABLE IF NOT EXISTS notification_templates (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  uid CHAR(26) NOT NULL,
  workspace_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(128) NOT NULL,
  category VARCHAR(64) NOT NULL,
  channel_type VARCHAR(32) NOT NULL DEFAULT 'any',
  title_template VARCHAR(255) NOT NULL,
  content_template VARCHAR(2048) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_by BIGINT UNSIGNED NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_notification_templates_uid (uid),
  UNIQUE KEY uk_notification_templates_workspace_name (workspace_id, name),
  KEY idx_notification_templates_match (workspace_id, category, channel_type, status),
  KEY idx_notification_templates_created_by (created_by),
  CONSTRAINT fk_notification_templates_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
  CONSTRAINT fk_notification_templates_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
  CONSTRAINT chk_notification_templates_channel CHECK (channel_type IN ('any', 'site', 'email', 'webhook', 'dingtalk', 'wechat', 'slack')),
  CONSTRAINT chk_notification_templates_status CHECK (status IN ('active', 'disabled', 'archived'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

-- Source: archive/v1.0-incremental/000011_v10_metrics_lifecycle.up.sql
CREATE TABLE IF NOT EXISTS host_metric_rollups (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  workspace_id BIGINT UNSIGNED NOT NULL,
  host_id BIGINT UNSIGNED NOT NULL,
  agent_id BIGINT UNSIGNED NULL,
  metric_code VARCHAR(128) NOT NULL,
  interval_type VARCHAR(16) NOT NULL,
  bucket_start DATETIME(3) NOT NULL,
  avg_value DECIMAL(20, 6) NOT NULL,
  min_value DECIMAL(20, 6) NOT NULL,
  max_value DECIMAL(20, 6) NOT NULL,
  sample_count INT UNSIGNED NOT NULL,
  unit VARCHAR(32) NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_host_metric_rollups_bucket (workspace_id, host_id, metric_code, interval_type, bucket_start),
  KEY idx_host_metric_rollups_lookup (workspace_id, metric_code, interval_type, bucket_start),
  KEY idx_host_metric_rollups_host_metric_time (host_id, metric_code, interval_type, bucket_start),
  CONSTRAINT fk_host_metric_rollups_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
  CONSTRAINT fk_host_metric_rollups_host FOREIGN KEY (host_id) REFERENCES hosts(id) ON DELETE CASCADE,
  CONSTRAINT fk_host_metric_rollups_agent FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE SET NULL,
  CONSTRAINT chk_host_metric_rollups_interval CHECK (interval_type IN ('5m', '1h'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS metric_dashboards (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  uid CHAR(26) NOT NULL,
  workspace_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(128) NOT NULL,
  metric_code VARCHAR(128) NULL,
  host_uid CHAR(26) NULL,
  agent_uid CHAR(26) NULL,
  range_hours INT UNSIGNED NOT NULL DEFAULT 24,
  point_limit INT UNSIGNED NOT NULL DEFAULT 120,
  granularity VARCHAR(16) NOT NULL DEFAULT 'auto',
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_by BIGINT UNSIGNED NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_metric_dashboards_uid (uid),
  UNIQUE KEY uk_metric_dashboards_workspace_name (workspace_id, name),
  KEY idx_metric_dashboards_workspace_status (workspace_id, status),
  CONSTRAINT fk_metric_dashboards_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
  CONSTRAINT fk_metric_dashboards_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
  CONSTRAINT chk_metric_dashboards_granularity CHECK (granularity IN ('auto', 'raw', '5m', '1h')),
  CONSTRAINT chk_metric_dashboards_status CHECK (status IN ('active', 'disabled', 'archived'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

INSERT INTO permissions(code, module, name, description)
VALUES ('metric:write', 'metric', 'Write metrics', 'Manage metric dashboards, rollups, and retention')
ON DUPLICATE KEY UPDATE
  module = VALUES(module),
  name = VALUES(name),
  description = VALUES(description);

INSERT IGNORE INTO role_permissions(role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code = 'metric:write'
WHERE r.code = 'admin';

-- Source: archive/v1.0-incremental/000012_v10_alert_routing_suppression.up.sql
CREATE TABLE IF NOT EXISTS alert_suppression_rules (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  uid CHAR(26) NOT NULL,
  workspace_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(128) NOT NULL,
  alert_rule_uid CHAR(26) NULL,
  host_uid CHAR(26) NULL,
  severity VARCHAR(32) NULL,
  starts_at DATETIME(3) NULL,
  ends_at DATETIME(3) NULL,
  reason VARCHAR(512) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_by BIGINT UNSIGNED NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_alert_suppression_rules_uid (uid),
  UNIQUE KEY uk_alert_suppression_rules_workspace_name (workspace_id, name),
  KEY idx_alert_suppression_rules_match (workspace_id, status, alert_rule_uid, host_uid, severity),
  CONSTRAINT fk_alert_suppression_rules_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
  CONSTRAINT fk_alert_suppression_rules_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
  CONSTRAINT chk_alert_suppression_rules_severity CHECK (severity IS NULL OR severity IN ('info', 'warning', 'critical')),
  CONSTRAINT chk_alert_suppression_rules_status CHECK (status IN ('active', 'disabled', 'archived'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS alert_routing_policies (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  uid CHAR(26) NOT NULL,
  workspace_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(128) NOT NULL,
  alert_rule_uid CHAR(26) NULL,
  host_uid CHAR(26) NULL,
  severity VARCHAR(32) NULL,
  channel_uid CHAR(26) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_by BIGINT UNSIGNED NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_alert_routing_policies_uid (uid),
  UNIQUE KEY uk_alert_routing_policies_workspace_name (workspace_id, name),
  KEY idx_alert_routing_policies_match (workspace_id, status, alert_rule_uid, host_uid, severity),
  CONSTRAINT fk_alert_routing_policies_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
  CONSTRAINT fk_alert_routing_policies_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
  CONSTRAINT chk_alert_routing_policies_severity CHECK (severity IS NULL OR severity IN ('info', 'warning', 'critical')),
  CONSTRAINT chk_alert_routing_policies_status CHECK (status IN ('active', 'disabled', 'archived'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

-- Source: archive/v1.0-incremental/000013_v10_agent_fleet_operations.up.sql
CREATE TABLE IF NOT EXISTS agent_diagnostics (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  uid CHAR(26) NOT NULL,
  workspace_id BIGINT UNSIGNED NOT NULL,
  agent_id BIGINT UNSIGNED NOT NULL,
  host_id BIGINT UNSIGNED NULL,
  version VARCHAR(64) NULL,
  os_name VARCHAR(128) NULL,
  os_version VARCHAR(128) NULL,
  arch VARCHAR(64) NULL,
  ip VARCHAR(45) NULL,
  running_tasks INT UNSIGNED NULL,
  payload JSON NULL,
  reported_at DATETIME(3) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_agent_diagnostics_uid (uid),
  KEY idx_agent_diagnostics_workspace_reported (workspace_id, reported_at),
  KEY idx_agent_diagnostics_agent_reported (agent_id, reported_at),
  CONSTRAINT fk_agent_diagnostics_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
  CONSTRAINT fk_agent_diagnostics_agent FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE CASCADE,
  CONSTRAINT fk_agent_diagnostics_host FOREIGN KEY (host_id) REFERENCES hosts(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS maintenance_windows (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  uid CHAR(26) NOT NULL,
  workspace_id BIGINT UNSIGNED NOT NULL,
  name VARCHAR(128) NOT NULL,
  scope_type VARCHAR(32) NOT NULL DEFAULT 'all',
  agent_id BIGINT UNSIGNED NULL,
  host_id BIGINT UNSIGNED NULL,
  reason VARCHAR(512) NULL,
  starts_at DATETIME(3) NOT NULL,
  ends_at DATETIME(3) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_by BIGINT UNSIGNED NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_maintenance_windows_uid (uid),
  UNIQUE KEY uk_maintenance_windows_workspace_name (workspace_id, name),
  KEY idx_maintenance_windows_scope (workspace_id, status, scope_type, starts_at, ends_at),
  KEY idx_maintenance_windows_agent (agent_id),
  KEY idx_maintenance_windows_host (host_id),
  CONSTRAINT fk_maintenance_windows_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
  CONSTRAINT fk_maintenance_windows_agent FOREIGN KEY (agent_id) REFERENCES agents(id) ON DELETE SET NULL,
  CONSTRAINT fk_maintenance_windows_host FOREIGN KEY (host_id) REFERENCES hosts(id) ON DELETE SET NULL,
  CONSTRAINT fk_maintenance_windows_created_by FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
  CONSTRAINT chk_maintenance_windows_scope CHECK (scope_type IN ('all', 'agent', 'host')),
  CONSTRAINT chk_maintenance_windows_status CHECK (status IN ('active', 'disabled', 'archived'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

-- Source: archive/v1.0-incremental/000014_v10_alert_host_group_matching.up.sql
ALTER TABLE alert_suppression_rules
  ADD COLUMN host_group_uid CHAR(26) NULL AFTER host_uid,
  DROP INDEX idx_alert_suppression_rules_match,
  ADD KEY idx_alert_suppression_rules_match (workspace_id, status, alert_rule_uid, host_uid, host_group_uid, severity);

ALTER TABLE alert_routing_policies
  ADD COLUMN host_group_uid CHAR(26) NULL AFTER host_uid,
  DROP INDEX idx_alert_routing_policies_match,
  ADD KEY idx_alert_routing_policies_match (workspace_id, status, alert_rule_uid, host_uid, host_group_uid, severity);

-- Source: archive/v1.0-incremental/000015_v10_maintenance_host_group_scope.up.sql
ALTER TABLE maintenance_windows
  ADD COLUMN host_group_id BIGINT UNSIGNED NULL AFTER host_id,
  DROP INDEX idx_maintenance_windows_scope,
  ADD KEY idx_maintenance_windows_scope (workspace_id, status, scope_type, starts_at, ends_at),
  ADD KEY idx_maintenance_windows_host_group (host_group_id),
  ADD CONSTRAINT fk_maintenance_windows_host_group FOREIGN KEY (host_group_id) REFERENCES host_groups(id) ON DELETE SET NULL;

ALTER TABLE maintenance_windows
  DROP CHECK chk_maintenance_windows_scope,
  ADD CONSTRAINT chk_maintenance_windows_scope CHECK (scope_type IN ('all', 'agent', 'host', 'group'));


