import { request } from "./request";
import type {
  Agent,
  AgentDiagnostic,
  EnrollmentTokenDetail,
  EnrollmentTokenSummary,
  Host,
  HostGroup,
  MaintenanceWindow,
  OfflineScanResult,
  PageResult,
  TagSummary
} from "./types";

export interface ListQuery {
  keyword?: string;
  status?: string;
  page?: number;
  pageSize?: number;
}

export interface CreateEnrollmentTokenPayload {
  maxUses?: number;
  expiresInSeconds?: number;
  bindWorkspaceSlug?: string;
}

function toSearch(params: ListQuery = {}) {
  const search = new URLSearchParams();
  if (params.keyword) search.set("keyword", params.keyword);
  if (params.status) search.set("status", params.status);
  if (params.page) search.set("page", String(params.page));
  if (params.pageSize) search.set("pageSize", String(params.pageSize));
  return search.toString() ? `?${search}` : "";
}

export function listAgents(params: ListQuery = {}) {
  return request<PageResult<Agent>>(`/api/v1/agents${toSearch(params)}`);
}

export function listHosts(params: ListQuery = {}) {
  return request<PageResult<Host>>(`/api/v1/hosts${toSearch(params)}`);
}

export function listAgentDiagnostics() {
  return request<AgentDiagnostic[]>("/api/v1/agents/diagnostics");
}

export function listMaintenanceWindows() {
  return request<MaintenanceWindow[]>("/api/v1/maintenance-windows");
}

export function listTags() {
  return request<TagSummary[]>("/api/v1/tags");
}

export function createTag(payload: { name: string; color?: string }) {
  return request<TagSummary>("/api/v1/tags", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function updateTag(id: string, payload: { name: string; color?: string }) {
  return request<TagSummary>(`/api/v1/tags/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export function setAgentTags(id: string, tagIds: string[]) {
  return request<TagSummary[]>(`/api/v1/agents/${id}/tags`, {
    method: "PUT",
    body: JSON.stringify({ tagIds })
  });
}

export function setHostTags(id: string, tagIds: string[]) {
  return request<TagSummary[]>(`/api/v1/hosts/${id}/tags`, {
    method: "PUT",
    body: JSON.stringify({ tagIds })
  });
}

export function listHostGroups() {
  return request<HostGroup[]>("/api/v1/host-groups");
}

export function createHostGroup(payload: { name: string; description?: string; hostIds?: string[] }) {
  return request<HostGroup>("/api/v1/host-groups", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function updateHostGroup(id: string, payload: { name: string; description?: string; hostIds?: string[] }) {
  return request<HostGroup>(`/api/v1/host-groups/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export function setHostGroupMembers(id: string, hostIds: string[]) {
  return request<HostGroup>(`/api/v1/host-groups/${id}/members`, {
    method: "PUT",
    body: JSON.stringify({ hostIds })
  });
}

export function createMaintenanceWindow(payload: {
  name: string;
  scopeType?: string;
  agentId?: string;
  hostId?: string;
  reason?: string;
  startsAt: string;
  endsAt: string;
  status?: string;
}) {
  return request<MaintenanceWindow>("/api/v1/maintenance-windows", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function updateMaintenanceWindow(id: string, payload: {
  name: string;
  scopeType?: string;
  agentId?: string;
  hostId?: string;
  reason?: string;
  startsAt: string;
  endsAt: string;
  status?: string;
}) {
  return request<MaintenanceWindow>(`/api/v1/maintenance-windows/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export function markOffline() {
  return request<OfflineScanResult>("/api/v1/agents/offline-scan", { method: "POST" });
}

export function disableAgent(id: string) {
  return request<{ ok: boolean }>(`/api/v1/agents/${id}/disable`, { method: "POST" });
}

export function revokeAgentToken(id: string) {
  return request<{ ok: boolean }>(`/api/v1/agents/${id}/revoke-token`, { method: "POST" });
}

export function listEnrollmentTokens() {
  return request<EnrollmentTokenSummary[]>("/api/v1/agent-enrollment-tokens");
}

export function createEnrollmentToken(payload: CreateEnrollmentTokenPayload) {
  return request<EnrollmentTokenDetail>("/api/v1/agent-enrollment-tokens", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function revokeEnrollmentToken(id: string) {
  return request<{ ok: boolean }>(`/api/v1/agent-enrollment-tokens/${id}/revoke`, { method: "POST" });
}
