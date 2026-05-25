import { request } from "./request";
import type { TraceCenterResult } from "./types";

export interface TraceCenterQuery {
  traceId?: string;
  taskRunId?: string;
  workflowRunId?: string;
  webhookEventId?: string;
}

export function searchTraceCenter(params: TraceCenterQuery) {
  const search = new URLSearchParams();
  if (params.traceId) search.set("traceId", params.traceId);
  if (params.taskRunId) search.set("taskRunId", params.taskRunId);
  if (params.workflowRunId) search.set("workflowRunId", params.workflowRunId);
  if (params.webhookEventId) search.set("webhookEventId", params.webhookEventId);
  return request<TraceCenterResult>(`/api/v1/trace-center${search.toString() ? `?${search}` : ""}`);
}

export function getTrace(id: string) {
  return request<TraceCenterResult>(`/api/v1/traces/${id}`);
}

export function listTraceEvents(id: string, params: { page?: number; pageSize?: number } = {}) {
  const search = new URLSearchParams();
  if (params.page) search.set("page", String(params.page));
  if (params.pageSize) search.set("pageSize", String(params.pageSize));
  return request<{ traceId: string; items: TraceCenterResult["timeline"]; total: number; page: number; pageSize: number }>(
    `/api/v1/traces/${id}/events${search.toString() ? `?${search}` : ""}`
  );
}

export function lookupTrace(params: TraceCenterQuery & { entityType?: string; entityId?: string }) {
  const search = new URLSearchParams();
  if (params.entityType) search.set("entityType", params.entityType);
  if (params.entityId) search.set("entityId", params.entityId);
  if (params.traceId) search.set("traceId", params.traceId);
  if (params.taskRunId) search.set("taskRunId", params.taskRunId);
  if (params.workflowRunId) search.set("workflowRunId", params.workflowRunId);
  if (params.webhookEventId) search.set("webhookEventId", params.webhookEventId);
  return request<TraceCenterResult>(`/api/v1/traces/lookup${search.toString() ? `?${search}` : ""}`);
}
