import { request } from "./request";

export interface PermissionDiffResult {
  roleId: string;
  roleName: string;
  currentPermissions: string[];
  requestedPermissions: string[];
  added: string[];
  removed: string[];
  affectedUsers: number;
  highRisk: boolean;
}

export interface SecretRotationEvent {
  id: number;
  secretId: string;
  action: string;
  operator?: string;
  resourceType?: string;
  resourceId?: string;
  traceId?: string;
  createdAt: string;
}

export function getPermissionDiff(params: { roleId: string; permissionCodes?: string[] }) {
  const search = new URLSearchParams();
  search.set("roleId", params.roleId);
  if (params.permissionCodes?.length) search.set("permissionCodes", params.permissionCodes.join(","));
  return request<PermissionDiffResult>(`/api/v1/security/permissions/diff?${search.toString()}`);
}

export function listSecretRotationHistory(id: string) {
  return request<SecretRotationEvent[]>(`/api/v1/security/secrets/${id}/rotation-history`);
}
