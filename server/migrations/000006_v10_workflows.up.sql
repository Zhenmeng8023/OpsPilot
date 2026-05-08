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
JOIN permissions p ON p.code IN ('workflow:read', 'workflow:write', 'workflow:execute', 'workflow:cancel')
WHERE r.code = 'admin';

COMMIT;
