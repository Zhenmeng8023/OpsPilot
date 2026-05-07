import { request } from "./request";
import type { HostMetric } from "./types";

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
