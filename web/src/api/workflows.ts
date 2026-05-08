import { request } from "./request";
import type { PageResult, WorkflowDefinitionDetail, WorkflowDefinitionSummary, WorkflowRunDetail, WorkflowRunSummary, WorkflowVersionSummary } from "./types";

export interface WorkflowDefinitionPayload {
  name: string;
  description?: string;
  definition: string;
}

export interface WorkflowRunPayload {
  triggerType?: string;
  idempotencyKey?: string;
  input?: string;
}

export function listWorkflows(params: { keyword?: string; status?: string; page?: number; pageSize?: number } = {}) {
  const search = new URLSearchParams();
  if (params.keyword) search.set("keyword", params.keyword);
  if (params.status) search.set("status", params.status);
  if (params.page) search.set("page", String(params.page));
  if (params.pageSize) search.set("pageSize", String(params.pageSize));
  return request<PageResult<WorkflowDefinitionSummary>>(`/api/v1/workflows${search.toString() ? `?${search}` : ""}`);
}

export function getWorkflow(id: string) {
  return request<WorkflowDefinitionDetail>(`/api/v1/workflows/${id}`);
}

export function createWorkflow(payload: WorkflowDefinitionPayload) {
  return request<WorkflowDefinitionDetail>("/api/v1/workflows", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function updateWorkflow(id: string, payload: WorkflowDefinitionPayload) {
  return request<WorkflowDefinitionDetail>(`/api/v1/workflows/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export function publishWorkflow(id: string) {
  return request<WorkflowDefinitionDetail>(`/api/v1/workflows/${id}/publish`, { method: "POST" });
}

export function disableWorkflow(id: string) {
  return request<WorkflowDefinitionDetail>(`/api/v1/workflows/${id}/disable`, { method: "POST" });
}

export function copyWorkflow(id: string, payload: { name?: string } = {}) {
  return request<WorkflowDefinitionDetail>(`/api/v1/workflows/${id}/copy`, {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function listWorkflowVersions(id: string) {
  return request<WorkflowVersionSummary[]>(`/api/v1/workflows/${id}/versions`);
}

export function runWorkflow(id: string, payload: WorkflowRunPayload = {}) {
  return request<WorkflowRunDetail>(`/api/v1/workflows/${id}/run`, {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function listWorkflowRuns(params: { keyword?: string; status?: string; page?: number; pageSize?: number } = {}) {
  const search = new URLSearchParams();
  if (params.keyword) search.set("keyword", params.keyword);
  if (params.status) search.set("status", params.status);
  if (params.page) search.set("page", String(params.page));
  if (params.pageSize) search.set("pageSize", String(params.pageSize));
  return request<PageResult<WorkflowRunSummary>>(`/api/v1/workflow-runs${search.toString() ? `?${search}` : ""}`);
}

export function getWorkflowRun(id: string) {
  return request<WorkflowRunDetail>(`/api/v1/workflow-runs/${id}`);
}

export function cancelWorkflowRun(id: string, reason?: string) {
  return request<WorkflowRunDetail>(`/api/v1/workflow-runs/${id}/cancel`, {
    method: "POST",
    body: JSON.stringify({ reason })
  });
}

export function retryWorkflowRun(id: string) {
  return request<WorkflowRunDetail>(`/api/v1/workflow-runs/${id}/retry`, {
    method: "POST"
  });
}

export function retryWorkflowNode(runId: string, nodeId: string) {
  return request<WorkflowRunDetail>(`/api/v1/workflow-runs/${runId}/nodes/${nodeId}/retry`, {
    method: "POST"
  });
}

export function approveWorkflowNode(runId: string, nodeId: string, comment?: string) {
  return request<WorkflowRunDetail>(`/api/v1/workflow-runs/${runId}/nodes/${nodeId}/approve`, {
    method: "POST",
    body: JSON.stringify({ comment })
  });
}

export function rejectWorkflowNode(runId: string, nodeId: string, comment?: string) {
  return request<WorkflowRunDetail>(`/api/v1/workflow-runs/${runId}/nodes/${nodeId}/reject`, {
    method: "POST",
    body: JSON.stringify({ comment })
  });
}
