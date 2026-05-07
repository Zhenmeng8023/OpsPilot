import { request } from "./request";
import type { WebhookRule, WebhookSource } from "./types";

export function listWebhookSources() {
  return request<WebhookSource[]>("/api/v1/webhooks/sources");
}

export function createWebhookSource(payload: { name: string; sourceType?: string }) {
  return request<WebhookSource>("/api/v1/webhooks/sources", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function listWebhookRules() {
  return request<WebhookRule[]>("/api/v1/webhooks/rules");
}

export function createWebhookRule(payload: { sourceId: string; taskId: string; name: string; eventType?: string }) {
  return request<WebhookRule>("/api/v1/webhooks/rules", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}
