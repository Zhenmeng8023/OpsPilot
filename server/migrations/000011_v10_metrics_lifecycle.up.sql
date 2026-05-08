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
