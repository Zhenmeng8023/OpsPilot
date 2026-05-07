import { request } from "./request";
import type { NotificationChannel, NotificationChannelTestResult, NotificationDelivery, NotificationSummary } from "./types";

export function listNotificationChannels() {
  return request<NotificationChannel[]>("/api/v1/notification-channels");
}

export function createNotificationChannel(payload: { name: string; channelType?: string; config?: Record<string, unknown> }) {
  return request<NotificationChannel>("/api/v1/notification-channels", {
    method: "POST",
    body: JSON.stringify(payload)
  });
}

export function testNotificationChannel(id: string) {
  return request<NotificationChannelTestResult>(`/api/v1/notification-channels/${id}/test`, {
    method: "POST"
  });
}

export function listNotifications(params: { unread?: boolean } = {}) {
  const search = new URLSearchParams();
  if (params.unread) search.set("unread", "true");
  return request<NotificationSummary[]>(`/api/v1/notifications${search.toString() ? `?${search}` : ""}`);
}

export function markNotificationRead(id: string) {
  return request<{ ok: boolean }>(`/api/v1/notifications/${id}/read`, { method: "POST" });
}

export function listNotificationDeliveries(params: { status?: string; channelId?: string; notificationId?: string } = {}) {
  const search = new URLSearchParams();
  if (params.status) search.set("status", params.status);
  if (params.channelId) search.set("channelId", params.channelId);
  if (params.notificationId) search.set("notificationId", params.notificationId);
  return request<NotificationDelivery[]>(`/api/v1/notification-deliveries${search.toString() ? `?${search}` : ""}`);
}

export function retryNotificationDelivery(id: number) {
  return request<{ ok: boolean }>(`/api/v1/notification-deliveries/${id}/retry`, { method: "POST" });
}
