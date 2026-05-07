import { request } from "./request";
import type { NotificationChannel, NotificationSummary } from "./types";

export function listNotificationChannels() {
  return request<NotificationChannel[]>("/api/v1/notification-channels");
}

export function createNotificationChannel(payload: { name: string; channelType?: string; config?: Record<string, unknown> }) {
  return request<NotificationChannel>("/api/v1/notification-channels", {
    method: "POST",
    body: JSON.stringify(payload)
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
