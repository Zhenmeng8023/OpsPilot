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
