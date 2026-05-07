import type { UserProfile } from "../../api/types";

const permissionAliases: Record<string, string[]> = {
  "agent:read": ["agent:read", "agent.read"],
  "script:read": ["script:read", "script.read"],
  "script:write": ["script:write", "script.write"],
  "script:approve": ["script:approve", "script.approve"],
  "task:read": ["task:read", "task.read"],
  "task:execute": ["task:execute", "task.run"],
  "task:cancel": ["task:cancel", "task.cancel"],
  "task:log:read": ["task:log:read", "log.read"],
  "schedule:write": ["schedule:write", "schedule.write"],
  "metric:read": ["metric:read", "metric.read"],
  "alert:write": ["alert:write", "alert.write"],
  "webhook:manage": ["webhook:manage", "webhook.manage"],
  "notification:write": ["notification:write", "notification.write"]
};

const permissionCanonical = Object.fromEntries(
  Object.entries(permissionAliases).flatMap(([canonical, aliases]) => aliases.map((alias) => [alias, canonical]))
);

export function hasPermission(user: UserProfile | null | undefined, permission: string) {
  return hasPermissionCode(user?.permissions, permission);
}

export function hasPermissionCode(permissions: string[] | undefined, permission: string) {
  const permissionSet = new Set((permissions ?? []).map((value) => permissionCanonical[value] ?? value));
  return permissionSet.has(permissionCanonical[permission] ?? permission);
}
