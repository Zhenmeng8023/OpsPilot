-- Migration governance baseline: track executed root migrations.

SET NAMES utf8mb4;
SET time_zone = '+08:00';

CREATE TABLE IF NOT EXISTS schema_migrations (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  version VARCHAR(128) NOT NULL,
  checksum CHAR(64) NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'applied',
  execution_ms INT UNSIGNED NOT NULL DEFAULT 0,
  notes VARCHAR(255) NULL,
  applied_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_schema_migrations_version (version),
  KEY idx_schema_migrations_applied_at (applied_at),
  CONSTRAINT chk_schema_migrations_status CHECK (status IN ('applied', 'legacy', 'skipped', 'failed'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

INSERT IGNORE INTO schema_migrations(version, status, notes)
VALUES
  ('000001_init_mysql_schema.up.sql', 'legacy', 'backfilled by 000006'),
  ('000002_seed_initial_auth_data.up.sql', 'legacy', 'backfilled by 000006'),
  ('000003_task_execution_security.up.sql', 'legacy', 'backfilled by 000006'),
  ('000004_v07_webhook_security.up.sql', 'legacy', 'backfilled by 000006'),
  ('000005_v10_productization_bundle.up.sql', 'legacy', 'backfilled by 000006'),
  ('000006_schema_migrations_governance.up.sql', 'applied', 'governance baseline');
