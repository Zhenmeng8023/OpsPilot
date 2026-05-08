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
