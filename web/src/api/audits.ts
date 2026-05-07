import { request } from "./request";
import type { AuditLog, PageResult } from "./types";

export function listAuditLogs(params: {
  action?: string;
  actorType?: string;
  result?: string;
  resourceType?: string;
  traceId?: string;
  keyword?: string;
  createdFrom?: string;
  createdTo?: string;
  page?: number;
  pageSize?: number;
} = {}) {
  const search = new URLSearchParams();
  if (params.action) search.set("action", params.action);
  if (params.actorType) search.set("actorType", params.actorType);
  if (params.result) search.set("result", params.result);
  if (params.resourceType) search.set("resourceType", params.resourceType);
  if (params.traceId) search.set("traceId", params.traceId);
  if (params.keyword) search.set("keyword", params.keyword);
  if (params.createdFrom) search.set("createdFrom", params.createdFrom);
  if (params.createdTo) search.set("createdTo", params.createdTo);
  if (params.page) search.set("page", String(params.page));
  if (params.pageSize) search.set("pageSize", String(params.pageSize));
  return request<PageResult<AuditLog>>(`/api/v1/audit-logs${search.toString() ? `?${search}` : ""}`);
}
