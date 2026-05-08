import { request } from "./request";
import type { AlertEventSummary, AlertGroupSummary, AlertHistoryPoint, AlertRoutingPolicy, AlertRule, AlertSummary, AlertSuppressionRule } from "./types";

export function listAlertRules() {
  return request<AlertRule[]>("/api/v1/alert-rules");
}

export function createAlertRule(payload: {
  name: string;
  metricCode: string;
  operator: string;
  threshold: number;
  durationSeconds?: number;
  cooldownSeconds?: number;
  severity?: string;
}) {
  return request<AlertRule>("/api/v1/alert-rules", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function updateAlertRule(id: string, payload: {
  name: string;
  metricCode: string;
  operator: string;
  threshold: number;
  durationSeconds?: number;
  cooldownSeconds?: number;
  severity?: string;
}) {
  return request<AlertRule>(`/api/v1/alert-rules/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export function pauseAlertRule(id: string) {
  return request<{ ok: boolean }>(`/api/v1/alert-rules/${id}/pause`, { method: "POST" });
}

export function resumeAlertRule(id: string) {
  return request<{ ok: boolean }>(`/api/v1/alert-rules/${id}/resume`, { method: "POST" });
}

export function disableAlertRule(id: string) {
  return request<{ ok: boolean }>(`/api/v1/alert-rules/${id}/disable`, { method: "POST" });
}

export function listAlertSuppressionRules() {
  return request<AlertSuppressionRule[]>("/api/v1/alert-suppression-rules");
}

export function createAlertSuppressionRule(payload: {
  name: string;
  ruleId?: string;
  hostId?: string;
  hostGroupId?: string;
  severity?: string;
  startsAt?: string;
  endsAt?: string;
  reason?: string;
  status?: string;
}) {
  return request<AlertSuppressionRule>("/api/v1/alert-suppression-rules", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function updateAlertSuppressionRule(id: string, payload: {
  name: string;
  ruleId?: string;
  hostId?: string;
  hostGroupId?: string;
  severity?: string;
  startsAt?: string;
  endsAt?: string;
  reason?: string;
  status?: string;
}) {
  return request<AlertSuppressionRule>(`/api/v1/alert-suppression-rules/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export function listAlertRoutingPolicies() {
  return request<AlertRoutingPolicy[]>("/api/v1/alert-routing-policies");
}

export function createAlertRoutingPolicy(payload: {
  name: string;
  ruleId?: string;
  hostId?: string;
  hostGroupId?: string;
  severity?: string;
  channelId: string;
  status?: string;
}) {
  return request<AlertRoutingPolicy>("/api/v1/alert-routing-policies", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function updateAlertRoutingPolicy(id: string, payload: {
  name: string;
  ruleId?: string;
  hostId?: string;
  hostGroupId?: string;
  severity?: string;
  channelId: string;
  status?: string;
}) {
  return request<AlertRoutingPolicy>(`/api/v1/alert-routing-policies/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export function listAlerts(params: { status?: string; severity?: string; ruleId?: string; hostId?: string } = {}) {
  const search = new URLSearchParams();
  if (params.status) search.set("status", params.status);
  if (params.severity) search.set("severity", params.severity);
  if (params.ruleId) search.set("ruleId", params.ruleId);
  if (params.hostId) search.set("hostId", params.hostId);
  return request<AlertSummary[]>(`/api/v1/alerts${search.toString() ? `?${search}` : ""}`);
}

export function listAlertGroups(params: { status?: string; severity?: string; ruleId?: string; hostGroupId?: string } = {}) {
  const search = new URLSearchParams();
  if (params.status) search.set("status", params.status);
  if (params.severity) search.set("severity", params.severity);
  if (params.ruleId) search.set("ruleId", params.ruleId);
  if (params.hostGroupId) search.set("hostGroupId", params.hostGroupId);
  return request<AlertGroupSummary[]>(`/api/v1/alert-groups${search.toString() ? `?${search}` : ""}`);
}

export function listAlertHistory(params: { hours?: number; bucketMinutes?: number; severity?: string; ruleId?: string; hostId?: string } = {}) {
  const search = new URLSearchParams();
  if (params.hours) search.set("hours", String(params.hours));
  if (params.bucketMinutes) search.set("bucketMinutes", String(params.bucketMinutes));
  if (params.severity) search.set("severity", params.severity);
  if (params.ruleId) search.set("ruleId", params.ruleId);
  if (params.hostId) search.set("hostId", params.hostId);
  return request<AlertHistoryPoint[]>(`/api/v1/alerts/history${search.toString() ? `?${search}` : ""}`);
}

export function listAlertEvents(id: string, params: { eventType?: string } = {}) {
  const search = new URLSearchParams();
  if (params.eventType) search.set("eventType", params.eventType);
  return request<AlertEventSummary[]>(`/api/v1/alerts/${id}/events${search.toString() ? `?${search}` : ""}`);
}

export function resolveAlert(id: string) {
  return request<{ ok: boolean }>(`/api/v1/alerts/${id}/resolve`, { method: "POST" });
}

export function acknowledgeAlert(id: string) {
  return request<{ ok: boolean }>(`/api/v1/alerts/${id}/ack`, { method: "POST" });
}

export function silenceAlert(id: string, payload: { durationSeconds?: number; reason?: string }) {
  return request<{ ok: boolean }>(`/api/v1/alerts/${id}/silence`, {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function unsilenceAlert(id: string) {
  return request<{ ok: boolean }>(`/api/v1/alerts/${id}/unsilence`, { method: "POST" });
}
