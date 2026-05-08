ALTER TABLE maintenance_windows
  ADD COLUMN host_group_id BIGINT UNSIGNED NULL AFTER host_id,
  DROP INDEX idx_maintenance_windows_scope,
  ADD KEY idx_maintenance_windows_scope (workspace_id, status, scope_type, starts_at, ends_at),
  ADD KEY idx_maintenance_windows_host_group (host_group_id),
  ADD CONSTRAINT fk_maintenance_windows_host_group FOREIGN KEY (host_group_id) REFERENCES host_groups(id) ON DELETE SET NULL;

ALTER TABLE maintenance_windows
  DROP CHECK chk_maintenance_windows_scope,
  ADD CONSTRAINT chk_maintenance_windows_scope CHECK (scope_type IN ('all', 'agent', 'host', 'group'));
