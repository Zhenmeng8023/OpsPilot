SET NAMES utf8mb4;
SET time_zone = '+08:00';

START TRANSACTION;

DROP TABLE IF EXISTS workflow_run_events;
DROP TABLE IF EXISTS workflow_run_nodes;
DROP TABLE IF EXISTS workflow_runs;
DROP TABLE IF EXISTS workflow_definitions;

DELETE rp FROM role_permissions rp
JOIN permissions p ON p.id = rp.permission_id
WHERE p.code IN ('workflow:read', 'workflow:manage', 'workflow:write', 'workflow:execute', 'workflow:cancel');

DELETE FROM permissions
WHERE code IN ('workflow:read', 'workflow:manage', 'workflow:write', 'workflow:execute', 'workflow:cancel');

COMMIT;
