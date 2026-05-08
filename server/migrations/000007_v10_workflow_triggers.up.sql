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
