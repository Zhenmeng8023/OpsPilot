import { request } from "./request";
import type { PageResult, WebhookEvent, WebhookEventDetail, WebhookMatcher, WebhookRule, WebhookSource } from "./types";

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

export function listWebhookRules() {
  return request<WebhookRule[]>("/api/v1/webhooks/rules");
}

export function listWebhookEvents(params: {
  sourceId?: string;
  status?: string;
  deliveryId?: string;
  page?: number;
  pageSize?: number;
} = {}) {
  const search = new URLSearchParams();
  if (params.sourceId) search.set("sourceId", params.sourceId);
  if (params.status) search.set("status", params.status);
  if (params.deliveryId) search.set("deliveryId", params.deliveryId);
  if (params.page) search.set("page", String(params.page));
  if (params.pageSize) search.set("pageSize", String(params.pageSize));
  return request<PageResult<WebhookEvent>>(`/api/v1/webhook-events${search.toString() ? `?${search}` : ""}`);
}

export function getWebhookEvent(id: string) {
  return request<WebhookEventDetail>(`/api/v1/webhooks/events/${id}`);
}

export function createWebhookRule(payload: { sourceId: string; taskId: string; name: string; eventType?: string; matcher?: WebhookMatcher }) {
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
