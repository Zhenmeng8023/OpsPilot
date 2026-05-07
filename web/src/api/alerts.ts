import { request } from "./request";
import type { AlertRule, AlertSummary } from "./types";

export function listAlertRules() {
  return request<AlertRule[]>("/api/v1/alert-rules");
}

export function createAlertRule(payload: {
  name: string;
  metricCode: string;
  operator: string;
  threshold: number;
  durationSeconds?: number;
  severity?: string;
}) {
  return request<AlertRule>("/api/v1/alert-rules", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function listAlerts(params: { status?: string } = {}) {
  const search = new URLSearchParams();
  if (params.status) search.set("status", params.status);
  return request<AlertSummary[]>(`/api/v1/alerts${search.toString() ? `?${search}` : ""}`);
}

export function resolveAlert(id: string) {
  return request<{ ok: boolean }>(`/api/v1/alerts/${id}/resolve`, { method: "POST" });
}
