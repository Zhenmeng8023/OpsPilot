import { request } from "./request";
import type { IncidentDetail, IncidentEvent, IncidentSummary } from "./types";

export function listIncidents(params: { status?: string; severity?: string } = {}) {
  const search = new URLSearchParams();
  if (params.status) search.set("status", params.status);
  if (params.severity) search.set("severity", params.severity);
  return request<IncidentSummary[]>(`/api/v1/incidents${search.toString() ? `?${search}` : ""}`);
}

export function getIncident(id: string) {
  return request<IncidentDetail>(`/api/v1/incidents/${id}`);
}

export function updateIncidentLifecycle(id: string, payload: {
  owner?: string;
  impactScope?: string;
  rootCauseClass?: string;
  postmortem?: string;
}) {
  return request<IncidentDetail>(`/api/v1/incidents/${id}/lifecycle`, {
    method: "PATCH",
    body: JSON.stringify(payload)
  });
}

export function listIncidentTimeline(id: string) {
  return request<IncidentEvent[]>(`/api/v1/incidents/${id}/timeline`);
}

export function linkIncidentAlertGroup(id: string, payload: { alertGroupId?: string; alertIds?: string[]; reason?: string }) {
  return request<IncidentDetail>(`/api/v1/incidents/${id}/alert-groups`, {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function mergeIncident(id: string, payload: { targetIncidentId: string; reason?: string }) {
  return request<IncidentDetail>(`/api/v1/incidents/${id}/merge`, {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function closeIncident(id: string, payload: { reason?: string } = {}) {
  return request<IncidentDetail>(`/api/v1/incidents/${id}/close`, {
    method: "POST",
    body: JSON.stringify(payload)
  });
}
