import type { UserProfile } from "../../api/types";

export function hasPermission(user: UserProfile | null | undefined, permission: string) {
  return hasPermissionCode(user?.permissions, permission);
}

export function hasPermissionCode(permissions: string[] | undefined, permission: string) {
  return (permissions ?? []).includes(permission);
}
