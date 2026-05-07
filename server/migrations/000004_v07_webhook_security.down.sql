SET NAMES utf8mb4;
SET time_zone = '+08:00';

START TRANSACTION;

ALTER TABLE webhook_events
  DROP KEY idx_webhook_events_source_nonce_received,
  DROP COLUMN signature_header,
  DROP COLUMN nonce,
  DROP COLUMN source_timestamp;

ALTER TABLE webhook_sources
  DROP COLUMN timestamp_tolerance_seconds,
  CHANGE COLUMN signing_secret secret_hash CHAR(64) NULL AFTER token_hash;

COMMIT;
