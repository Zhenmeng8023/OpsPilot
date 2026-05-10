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

