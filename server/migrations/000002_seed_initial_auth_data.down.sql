-- Rollback OpsPilot initial auth/RBAC seed data.

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

SET @workspace_uid := '01H00000000000000000000001';
SET @admin_role_uid := '01H00000000000000000000002';
SET @member_role_uid := '01H00000000000000000000003';
SET @admin_user_uid := '01H00000000000000000000004';

DELETE FROM user_roles
WHERE user_id IN (SELECT id FROM users WHERE uid = @admin_user_uid)
   OR role_id IN (SELECT id FROM roles WHERE uid IN (@admin_role_uid, @member_role_uid));

DELETE FROM workspace_members
WHERE workspace_id IN (SELECT id FROM workspaces WHERE uid = @workspace_uid)
  AND user_id IN (SELECT id FROM users WHERE uid = @admin_user_uid);

DELETE FROM role_permissions
WHERE role_id IN (SELECT id FROM roles WHERE uid IN (@admin_role_uid, @member_role_uid));

DELETE FROM users WHERE uid = @admin_user_uid;
DELETE FROM roles WHERE uid IN (@admin_role_uid, @member_role_uid);
DELETE FROM workspaces WHERE uid = @workspace_uid;

SET FOREIGN_KEY_CHECKS = 1;
