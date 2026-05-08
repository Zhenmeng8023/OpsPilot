import { API_BASE_URL, TOKEN_KEY, request } from "./request";
import type { AuditLog, AuditRetentionResult, PageResult } from "./types";

export function listAuditLogs(params: {
  action?: string;
  actor?: string;
  actorType?: string;
  result?: string;
  resourceType?: string;
  resourceId?: string;
  traceId?: string;
  keyword?: string;
  createdFrom?: string;
  createdTo?: string;
  page?: number;
  pageSize?: number;
} = {}) {
  const search = new URLSearchParams();
  if (params.action) search.set("action", params.action);
  if (params.actor) search.set("actor", params.actor);
  if (params.actorType) search.set("actorType", params.actorType);
  if (params.result) search.set("result", params.result);
  if (params.resourceType) search.set("resourceType", params.resourceType);
  if (params.resourceId) search.set("resourceId", params.resourceId);
  if (params.traceId) search.set("traceId", params.traceId);
  if (params.keyword) search.set("keyword", params.keyword);
  if (params.createdFrom) search.set("createdFrom", params.createdFrom);
  if (params.createdTo) search.set("createdTo", params.createdTo);
  if (params.page) search.set("page", String(params.page));
  if (params.pageSize) search.set("pageSize", String(params.pageSize));
  return request<PageResult<AuditLog>>(`/api/v1/audit-logs${search.toString() ? `?${search}` : ""}`);
}

export function exportAuditLogs(params: {
  format?: "csv" | "json";
  action?: string;
  actor?: string;
  actorType?: string;
  result?: string;
  resourceType?: string;
  resourceId?: string;
  traceId?: string;
  keyword?: string;
  createdFrom?: string;
  createdTo?: string;
} = {}) {
  const search = new URLSearchParams();
  if (params.format) search.set("format", params.format);
  if (params.action) search.set("action", params.action);
  if (params.actor) search.set("actor", params.actor);
  if (params.actorType) search.set("actorType", params.actorType);
  if (params.result) search.set("result", params.result);
  if (params.resourceType) search.set("resourceType", params.resourceType);
  if (params.resourceId) search.set("resourceId", params.resourceId);
  if (params.traceId) search.set("traceId", params.traceId);
  if (params.keyword) search.set("keyword", params.keyword);
  if (params.createdFrom) search.set("createdFrom", params.createdFrom);
  if (params.createdTo) search.set("createdTo", params.createdTo);
  return fetch(`${API_BASE_URL}/api/v1/audit-logs/export${search.toString() ? `?${search}` : ""}`, {
    method: "GET",
    headers: authHeaders()
  });
}

export function runAuditRetention(payload: { days?: number; dryRun?: boolean }) {
  return request<AuditRetentionResult>("/api/v1/audit-logs/retention/run", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

function authHeaders(): Record<string, string> {
  const token = window.localStorage.getItem(TOKEN_KEY);
  if (!token) {
    return {};
  }
  return { Authorization: `Bearer ${token}` };
}
