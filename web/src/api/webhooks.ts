import { request } from "./request";
import type { PageResult, WebhookEvent, WebhookEventDetail, WebhookMatcher, WebhookMatcherSimulationResult, WebhookRule, WebhookSource } from "./types";

export function listWebhookSources() {
  return request<WebhookSource[]>("/api/v1/webhooks/sources");
}

export function createWebhookSource(payload: { name: string; sourceType?: string }) {
  return request<WebhookSource>("/api/v1/webhooks/sources", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function pauseWebhookSource(id: string) {
  return request<{ ok: boolean }>(`/api/v1/webhooks/sources/${id}/pause`, { method: "POST" });
}

export function resumeWebhookSource(id: string) {
  return request<{ ok: boolean }>(`/api/v1/webhooks/sources/${id}/resume`, { method: "POST" });
}

export function disableWebhookSource(id: string) {
  return request<{ ok: boolean }>(`/api/v1/webhooks/sources/${id}/disable`, { method: "POST" });
}

export function rotateWebhookSourceSecret(id: string, payload: { rotateToken?: boolean } = {}) {
  return request<WebhookSource>(`/api/v1/webhooks/sources/${id}/rotate-secret`, {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function listWebhookRules() {
  return request<WebhookRule[]>("/api/v1/webhooks/rules");
}

export function listWebhookEvents(params: {
  sourceId?: string;
  status?: string;
  deliveryId?: string;
  receivedFrom?: string;
  receivedTo?: string;
  page?: number;
  pageSize?: number;
} = {}) {
  const search = new URLSearchParams();
  if (params.sourceId) search.set("sourceId", params.sourceId);
  if (params.status) search.set("status", params.status);
  if (params.deliveryId) search.set("deliveryId", params.deliveryId);
  if (params.receivedFrom) search.set("receivedFrom", params.receivedFrom);
  if (params.receivedTo) search.set("receivedTo", params.receivedTo);
  if (params.page) search.set("page", String(params.page));
  if (params.pageSize) search.set("pageSize", String(params.pageSize));
  return request<PageResult<WebhookEvent>>(`/api/v1/webhook-events${search.toString() ? `?${search}` : ""}`);
}

export function getWebhookEvent(id: string) {
  return request<WebhookEventDetail>(`/api/v1/webhooks/events/${id}`);
}

export function createWebhookRule(payload: { sourceId: string; targetType?: string; taskId?: string; workflowId?: string; name: string; eventType?: string; matcher?: WebhookMatcher }) {
  return request<WebhookRule>("/api/v1/webhooks/rules", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function updateWebhookRule(id: string, payload: { name: string; eventType?: string; matcher?: WebhookMatcher }) {
  return request<WebhookRule>(`/api/v1/webhooks/rules/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload)
  });
}

export function pauseWebhookRule(id: string) {
  return request<{ ok: boolean }>(`/api/v1/webhooks/rules/${id}/pause`, { method: "POST" });
}

export function resumeWebhookRule(id: string) {
  return request<{ ok: boolean }>(`/api/v1/webhooks/rules/${id}/resume`, { method: "POST" });
}

export function disableWebhookRule(id: string) {
  return request<{ ok: boolean }>(`/api/v1/webhooks/rules/${id}/disable`, { method: "POST" });
}

export function simulateWebhookMatcher(payload: {
  ruleId?: string;
  eventType?: string;
  matcher?: WebhookMatcher;
  headers?: Record<string, string>;
  payload?: string;
}) {
  return request<WebhookMatcherSimulationResult>("/api/v1/webhooks/matcher/simulate", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function replayWebhookEvent(id: string, payload: { simulateOnly?: boolean; idempotencyKey?: string } = {}) {
  return request<{ eventId: string; status: string; matchedRules: number; triggeredRuns: string[] }>(`/api/v1/webhooks/events/${id}/replay`, {
    method: "POST",
    body: JSON.stringify(payload)
  });
}
