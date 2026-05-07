import { request } from "./request";
import type {
  Agent,
  EnrollmentTokenDetail,
  EnrollmentTokenSummary,
  Host,
  OfflineScanResult,
  PageResult
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
