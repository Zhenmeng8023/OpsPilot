ALTER TABLE alert_suppression_rules
  ADD COLUMN host_group_uid CHAR(26) NULL AFTER host_uid,
  DROP INDEX idx_alert_suppression_rules_match,
  ADD KEY idx_alert_suppression_rules_match (workspace_id, status, alert_rule_uid, host_uid, host_group_uid, severity);

ALTER TABLE alert_routing_policies
  ADD COLUMN host_group_uid CHAR(26) NULL AFTER host_uid,
  DROP INDEX idx_alert_routing_policies_match,
  ADD KEY idx_alert_routing_policies_match (workspace_id, status, alert_rule_uid, host_uid, host_group_uid, severity);
