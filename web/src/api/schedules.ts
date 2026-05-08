import { request } from "./request";
import type { PageResult, ScheduleSummary, ScheduleTrigger } from "./types";

export interface CreateSchedulePayload {
  name: string;
  targetType?: string;
  taskId?: string;
  workflowId?: string;
  cronExpr: string;
  timezone?: string;
  misfirePolicy?: string;
}

export function listSchedules(params: {
  keyword?: string;
  status?: string;
  taskId?: string;
  page?: number;
  pageSize?: number;
} = {}) {
  const search = new URLSearchParams();
  if (params.keyword) search.set("keyword", params.keyword);
  if (params.status) search.set("status", params.status);
  if (params.taskId) search.set("taskId", params.taskId);
  if (params.page) search.set("page", String(params.page));
  if (params.pageSize) search.set("pageSize", String(params.pageSize));
  return request<PageResult<ScheduleSummary>>(`/api/v1/schedules${search.toString() ? `?${search}` : ""}`);
}

export function createSchedule(payload: CreateSchedulePayload) {
  return request<ScheduleSummary>("/api/v1/schedules", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function previewSchedule(payload: { cronExpr: string; timezone?: string; count?: number }) {
  return request<{ times: string[] }>("/api/v1/schedules/preview", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function listScheduleTriggers(id: string, limit = 20) {
  return request<ScheduleTrigger[]>(`/api/v1/schedules/${id}/triggers?limit=${limit}`);
}

export function pauseSchedule(id: string) {
  return request<{ ok: boolean }>(`/api/v1/schedules/${id}/pause`, { method: "POST" });
}

export function resumeSchedule(id: string) {
  return request<{ ok: boolean }>(`/api/v1/schedules/${id}/resume`, { method: "POST" });
}

export function disableSchedule(id: string) {
  return request<{ ok: boolean }>(`/api/v1/schedules/${id}/disable`, { method: "POST" });
}
