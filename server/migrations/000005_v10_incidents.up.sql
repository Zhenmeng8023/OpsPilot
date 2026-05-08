-- V1.0 incident projection over alerts.

SET NAMES utf8mb4;
SET time_zone = '+08:00';

START TRANSACTION;

CREATE TABLE IF NOT EXISTS incidents (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  uid VARCHAR(40) NOT NULL,
  workspace_id BIGINT UNSIGNED NOT NULL,
  alert_rule_id BIGINT UNSIGNED NULL,
  resource_type VARCHAR(64) NOT NULL,
  resource_id BIGINT UNSIGNED NULL,
  title VARCHAR(255) NOT NULL,
  message VARCHAR(2048) NULL,
  severity VARCHAR(32) NOT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'open',
  first_seen_at DATETIME(3) NOT NULL,
  last_seen_at DATETIME(3) NOT NULL,
  resolved_at DATETIME(3) NULL,
  metadata JSON NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  deleted_at DATETIME(3) NULL,
  PRIMARY KEY (id),
  UNIQUE KEY uk_incidents_uid (uid),
  KEY idx_incidents_workspace_status_severity (workspace_id, status, severity),
  KEY idx_incidents_rule (alert_rule_id),
  KEY idx_incidents_resource (workspace_id, resource_type, resource_id),
  CONSTRAINT fk_incidents_workspace FOREIGN KEY (workspace_id) REFERENCES workspaces(id) ON DELETE CASCADE,
  CONSTRAINT fk_incidents_rule FOREIGN KEY (alert_rule_id) REFERENCES alert_rules(id) ON DELETE SET NULL,
  CONSTRAINT chk_incidents_severity CHECK (severity IN ('info', 'warning', 'critical')),
  CONSTRAINT chk_incidents_status CHECK (status IN ('open', 'acknowledged', 'silenced', 'resolved'))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS incident_alerts (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  incident_id BIGINT UNSIGNED NOT NULL,
  alert_id BIGINT UNSIGNED NOT NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_incident_alerts_alert (alert_id),
  KEY idx_incident_alerts_incident (incident_id),
  CONSTRAINT fk_incident_alerts_incident FOREIGN KEY (incident_id) REFERENCES incidents(id) ON DELETE CASCADE,
  CONSTRAINT fk_incident_alerts_alert FOREIGN KEY (alert_id) REFERENCES alerts(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

CREATE TABLE IF NOT EXISTS incident_events (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  incident_id BIGINT UNSIGNED NOT NULL,
  alert_id BIGINT UNSIGNED NULL,
  event_type VARCHAR(64) NOT NULL,
  message VARCHAR(1024) NULL,
  actor_id BIGINT UNSIGNED NULL,
  payload JSON NULL,
  created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_incident_events_incident_created (incident_id, created_at),
  KEY idx_incident_events_alert (alert_id),
  KEY idx_incident_events_actor (actor_id),
  CONSTRAINT fk_incident_events_incident FOREIGN KEY (incident_id) REFERENCES incidents(id) ON DELETE CASCADE,
  CONSTRAINT fk_incident_events_alert FOREIGN KEY (alert_id) REFERENCES alerts(id) ON DELETE SET NULL,
  CONSTRAINT fk_incident_events_actor FOREIGN KEY (actor_id) REFERENCES users(id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;

INSERT INTO incidents(uid, workspace_id, alert_rule_id, resource_type, resource_id, title, message, severity, status,
                      first_seen_at, last_seen_at, resolved_at, metadata, created_at, updated_at)
SELECT CONCAT('inc_', a.uid), a.workspace_id, a.alert_rule_id, a.resource_type, a.resource_id, a.title, a.message,
       a.severity,
       CASE a.status
         WHEN 'firing' THEN 'open'
         WHEN 'acknowledged' THEN 'acknowledged'
         WHEN 'silenced' THEN 'silenced'
         ELSE 'resolved'
       END,
       a.first_seen_at, a.last_seen_at, a.resolved_at, a.metadata, a.created_at, a.updated_at
  FROM alerts a
  LEFT JOIN incident_alerts ia ON ia.alert_id = a.id
 WHERE ia.id IS NULL;

INSERT INTO incident_alerts(incident_id, alert_id, created_at)
SELECT i.id, a.id, a.created_at
  FROM alerts a
  JOIN incidents i ON i.uid = CONCAT('inc_', a.uid)
  LEFT JOIN incident_alerts ia ON ia.alert_id = a.id
 WHERE ia.id IS NULL;

INSERT INTO incident_events(incident_id, alert_id, event_type, message, actor_id, payload, created_at)
SELECT ia.incident_id, ae.alert_id, ae.event_type, ae.message, ae.actor_id, ae.payload, ae.created_at
  FROM alert_events ae
  JOIN incident_alerts ia ON ia.alert_id = ae.alert_id
  LEFT JOIN incident_events ie ON ie.incident_id = ia.incident_id
                              AND ie.alert_id = ae.alert_id
                              AND ie.event_type = ae.event_type
                              AND ie.created_at = ae.created_at
 WHERE ie.id IS NULL;

COMMIT;
