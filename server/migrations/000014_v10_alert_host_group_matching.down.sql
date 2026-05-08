ALTER TABLE alert_routing_policies
  DROP INDEX idx_alert_routing_policies_match,
  DROP COLUMN host_group_uid,
  ADD KEY idx_alert_routing_policies_match (workspace_id, status, alert_rule_uid, host_uid, severity);

ALTER TABLE alert_suppression_rules
  DROP INDEX idx_alert_suppression_rules_match,
  DROP COLUMN host_group_uid,
  ADD KEY idx_alert_suppression_rules_match (workspace_id, status, alert_rule_uid, host_uid, severity);
