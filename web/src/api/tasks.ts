import { streamSSE, request } from "./request";
import type { PageResult, TaskDetail, TaskLogEntry, TaskSummary, TaskTarget } from "./types";

export interface CreateTaskPayload {
  name: string;
  description?: string;
  scriptId?: string;
  command?: string;
  scriptType?: string;
  timeoutSeconds: number;
  targetAgentIds: string[];
  targetHostIds: string[];
}

export function listTasks(params: {
  keyword?: string;
  status?: string;
  creator?: string;
  createdFrom?: string;
  createdTo?: string;
  page?: number;
  pageSize?: number;
} = {}) {
  const search = new URLSearchParams();
  if (params.keyword) search.set("keyword", params.keyword);
  if (params.status) search.set("status", params.status);
  if (params.creator) search.set("creator", params.creator);
  if (params.createdFrom) search.set("createdFrom", params.createdFrom);
  if (params.createdTo) search.set("createdTo", params.createdTo);
  if (params.page) search.set("page", String(params.page));
  if (params.pageSize) search.set("pageSize", String(params.pageSize));
  return request<PageResult<TaskSummary>>(`/api/v1/tasks${search.toString() ? `?${search}` : ""}`);
}

export function createTask(payload: CreateTaskPayload) {
  return request<TaskDetail>("/api/v1/tasks", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function getTask(id: string) {
  return request<TaskDetail>(`/api/v1/tasks/${id}`);
}

export function cancelTask(id: string) {
  return request<{ ok: boolean }>(`/api/v1/tasks/${id}/cancel`, { method: "POST" });
}

export function getTaskTargets(id: string) {
  return request<TaskTarget[]>(`/api/v1/tasks/${id}/targets`);
}

export function getTaskLogs(id: string, targetId?: string, afterSequence?: number, stream?: string, limit?: number) {
  const search = new URLSearchParams();
  if (afterSequence) search.set("afterSequence", String(afterSequence));
  if (stream) search.set("stream", stream);
  if (limit) search.set("limit", String(limit));
  if (targetId) {
    return request<TaskLogEntry[]>(`/api/v1/tasks/${id}/targets/${targetId}/logs${search.toString() ? `?${search}` : ""}`);
  }
  return request<TaskLogEntry[]>(`/api/v1/tasks/${id}/logs${search.toString() ? `?${search}` : ""}`);
}

export function streamTaskLogs(
  id: string,
  options: {
    targetId?: string;
    afterSequence?: number;
    stream?: string;
    limit?: number;
    signal?: AbortSignal;
    onLog: (log: TaskLogEntry) => void;
    onError?: (error: Error) => void;
  }
) {
  const search = new URLSearchParams();
  if (options.targetId) search.set("targetId", options.targetId);
  if (options.afterSequence) search.set("afterSequence", String(options.afterSequence));
  if (options.stream) search.set("stream", options.stream);
  if (options.limit) search.set("limit", String(options.limit));
  return streamSSE(`/api/v1/tasks/${id}/logs/stream${search.toString() ? `?${search}` : ""}`, {
    signal: options.signal,
    onError: options.onError,
    onEvent: (event, data) => {
      if (event !== "log" || !data) return;
      try {
        options.onLog(JSON.parse(data) as TaskLogEntry);
      } catch (error) {
        options.onError?.(error instanceof Error ? error : new Error("Invalid log event"));
      }
    }
  });
}
