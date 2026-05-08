import { request } from "./request";
import type { IncidentDetail, IncidentSummary } from "./types";

export function listIncidents(params: { status?: string; severity?: string } = {}) {
  const search = new URLSearchParams();
  if (params.status) search.set("status", params.status);
  if (params.severity) search.set("severity", params.severity);
  return request<IncidentSummary[]>(`/api/v1/incidents${search.toString() ? `?${search}` : ""}`);
}

export function getIncident(id: string) {
  return request<IncidentDetail>(`/api/v1/incidents/${id}`);
}
