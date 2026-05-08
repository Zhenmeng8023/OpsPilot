import { request } from "./request";
import type { HostMetric, MetricDashboard, MetricRetentionResult, MetricRollupResult, MetricTrendSeries } from "./types";

export function listHostMetrics(params: {
  hostId?: string;
  agentId?: string;
  metricCode?: string;
  limit?: number;
} = {}) {
  const search = new URLSearchParams();
  if (params.hostId) search.set("hostId", params.hostId);
  if (params.agentId) search.set("agentId", params.agentId);
  if (params.metricCode) search.set("metricCode", params.metricCode);
  if (params.limit) search.set("limit", String(params.limit));
  return request<HostMetric[]>(`/api/v1/metrics/hosts${search.toString() ? `?${search}` : ""}`);
}

export function listMetricTrends(params: {
  hostId?: string;
  agentId?: string;
  metricCode: string;
  hours?: number;
  limit?: number;
  granularity?: string;
}) {
  const search = new URLSearchParams();
  if (params.hostId) search.set("hostId", params.hostId);
  if (params.agentId) search.set("agentId", params.agentId);
  search.set("metricCode", params.metricCode);
  if (params.hours) search.set("hours", String(params.hours));
  if (params.limit) search.set("limit", String(params.limit));
  if (params.granularity) search.set("granularity", params.granularity);
  return request<MetricTrendSeries[]>(`/api/v1/metrics/hosts/trends?${search}`);
}

export function listMetricDashboards() {
  return request<MetricDashboard[]>("/api/v1/metrics/dashboards");
}

export function createMetricDashboard(payload: {
  name: string;
  metricCode?: string;
  hostId?: string;
  agentId?: string;
  rangeHours?: number;
  pointLimit?: number;
  granularity?: string;
  status?: string;
}) {
  return request<MetricDashboard>("/api/v1/metrics/dashboards", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function updateMetricDashboard(id: string, payload: {
  name: string;
  metricCode?: string;
  hostId?: string;
  agentId?: string;
  rangeHours?: number;
  pointLimit?: number;
  granularity?: string;
  status?: string;
}) {
  return request<MetricDashboard>(`/api/v1/metrics/dashboards/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export function runMetricRollup(payload: { interval?: string; hours?: number }) {
  return request<MetricRollupResult>("/api/v1/metrics/rollups/run", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function runMetricRetention(payload: { detailDays?: number; rollupDays?: number; dryRun?: boolean }) {
  return request<MetricRetentionResult>("/api/v1/metrics/retention/run", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}
