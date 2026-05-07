-- V0.7 webhook signing secret split, timestamp validation, and nonce tracking.

SET NAMES utf8mb4;
SET time_zone = '+08:00';

START TRANSACTION;

ALTER TABLE webhook_sources
  CHANGE COLUMN secret_hash signing_secret VARCHAR(128) NULL AFTER token_hash,
  ADD COLUMN timestamp_tolerance_seconds INT UNSIGNED NOT NULL DEFAULT 300 AFTER signing_secret;

ALTER TABLE webhook_events
  ADD COLUMN source_timestamp DATETIME(3) NULL AFTER delivery_id,
  ADD COLUMN nonce VARCHAR(191) NULL AFTER source_timestamp,
  ADD COLUMN signature_header VARCHAR(64) NULL AFTER nonce,
  ADD KEY idx_webhook_events_source_nonce_received (source_id, nonce, received_at);

COMMIT;
