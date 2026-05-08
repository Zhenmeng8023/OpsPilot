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
