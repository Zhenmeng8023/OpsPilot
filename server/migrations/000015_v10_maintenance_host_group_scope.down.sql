ALTER TABLE maintenance_windows
  DROP CHECK chk_maintenance_windows_scope,
  ADD CONSTRAINT chk_maintenance_windows_scope CHECK (scope_type IN ('all', 'agent', 'host'));

ALTER TABLE maintenance_windows
  DROP FOREIGN KEY fk_maintenance_windows_host_group,
  DROP INDEX idx_maintenance_windows_host_group,
  DROP INDEX idx_maintenance_windows_scope,
  DROP COLUMN host_group_id,
  ADD KEY idx_maintenance_windows_scope (workspace_id, status, scope_type, starts_at, ends_at);
