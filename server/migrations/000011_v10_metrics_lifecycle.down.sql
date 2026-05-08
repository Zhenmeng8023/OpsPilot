DELETE rp FROM role_permissions rp
JOIN permissions p ON p.id = rp.permission_id
WHERE p.code = 'metric:write';

DELETE FROM permissions WHERE code = 'metric:write';

DROP TABLE IF EXISTS metric_dashboards;
DROP TABLE IF EXISTS host_metric_rollups;
