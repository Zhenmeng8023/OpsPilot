SET NAMES utf8mb4;
SET time_zone = '+08:00';

START TRANSACTION;

CREATE TABLE IF NOT EXISTS workflow_run_actions (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  uid CHAR(26) NOT NULL,
  workflow_run_id BIGINT UNSIGNED NOT NULL,
  trace_id VARCHAR(128) NULL,
  action_type VARCHAR(64) NOT NULL,
  actor_id BIGINT UNSIGNED NULL,
  node_id VARCHAR(64) NULL,
  detail VARCHAR(1024) NULL,
  result VARCHAR(32) NOT NULL DEFAULT 'success',
  payload LONGTEXT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_workflow_run_actions_uid (uid),
  KEY idx_workflow_run_actions_run_created (workflow_run_id, created_at),
  KEY idx_workflow_run_actions_trace_created (trace_id, created_at),
  KEY idx_workflow_run_actions_actor (actor_id),
  CONSTRAINT fk_workflow_run_actions_run FOREIGN KEY (workflow_run_id) REFERENCES workflow_runs(id) ON DELETE CASCADE,
  CONSTRAINT fk_workflow_run_actions_actor FOREIGN KEY (actor_id) REFERENCES users(id) ON DELETE SET NULL,
  CONSTRAINT chk_workflow_run_actions_result CHECK (result IN ('success', 'queued', 'running', 'failed', 'warning'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS trace_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  uid CHAR(26) NOT NULL,
  workspace_id BIGINT UNSIGNED NOT NULL,
  trace_id VARCHAR(128) NOT NULL,
  workflow_run_id BIGINT UNSIGNED NULL,
  task_run_id BIGINT UNSIGNED NULL,
  webhook_event_id BIGINT UNSIGNED NULL,
  source_type VARCHAR(64) NOT NULL,
  source_id BIGINT UNSIGNED NULL,
  category VARCHAR(64) NOT NULL,
  title VARCHAR(128) NOT NULL,
  status VARCHAR(32) NULL,
  reference_value VARCHAR(255) NULL,
  detail LONGTEXT NULL,
  payload LONGTEXT NULL,
  occurred_at DATETIME(3) NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_trace_events_uid (uid),
  KEY idx_trace_events_workspace_trace_time (workspace_id, trace_id, occurred_at),
  KEY idx_trace_events_trace_time (trace_id, occurred_at),
  KEY idx_trace_events_workflow_time (workflow_run_id, occurred_at),
  KEY idx_trace_events_task_time (task_run_id, occurred_at),
  KEY idx_trace_events_webhook_time (webhook_event_id, occurred_at),
  CONSTRAINT fk_trace_events_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
  CONSTRAINT fk_trace_events_workflow_run FOREIGN KEY (workflow_run_id) REFERENCES workflow_runs(id) ON DELETE SET NULL,
  CONSTRAINT fk_trace_events_task_run FOREIGN KEY (task_run_id) REFERENCES task_runs(id) ON DELETE SET NULL,
  CONSTRAINT fk_trace_events_webhook_event FOREIGN KEY (webhook_event_id) REFERENCES webhook_events(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

COMMIT;
