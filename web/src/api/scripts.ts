import { request } from "./request";
import type { Script, ScriptApprovalSummary } from "./types";

export interface ScriptPayload {
  name: string;
  description?: string;
  scriptType: string;
  content: string;
  changeSummary?: string;
}

export function listScripts(params: { keyword?: string; status?: string } = {}) {
  const search = new URLSearchParams();
  if (params.keyword) search.set("keyword", params.keyword);
  if (params.status) search.set("status", params.status);
  return request<Script[]>(`/api/v1/scripts${search.toString() ? `?${search}` : ""}`);
}

export function getScript(id: string) {
  return request<Script>(`/api/v1/scripts/${id}`);
}

export function createScript(payload: ScriptPayload) {
  return request<Script>("/api/v1/scripts", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function updateScript(id: string, payload: ScriptPayload) {
  return request<Script>(`/api/v1/scripts/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export function disableScript(id: string) {
  return request<{ ok: boolean }>(`/api/v1/scripts/${id}/disable`, { method: "POST" });
}

export function requestScriptApproval(id: string) {
  return request<ScriptApprovalSummary>(`/api/v1/scripts/${id}/approval`, {
    method: "POST"
  });
}

export function listScriptApprovals(params: { status?: string } = {}) {
  const search = new URLSearchParams();
  if (params.status) search.set("status", params.status);
  return request<ScriptApprovalSummary[]>(`/api/v1/script-approvals${search.toString() ? `?${search}` : ""}`);
}

export function approveScriptApproval(id: number, comment?: string) {
  return request<ScriptApprovalSummary>(`/api/v1/script-approvals/${id}/approve`, {
    method: "POST",
    body: JSON.stringify({ comment: comment ?? "" })
  });
}

export function rejectScriptApproval(id: number, comment?: string) {
  return request<ScriptApprovalSummary>(`/api/v1/script-approvals/${id}/reject`, {
    method: "POST",
    body: JSON.stringify({ comment: comment ?? "" })
  });
}
