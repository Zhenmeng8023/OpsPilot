import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  createNotificationChannel,
  listNotificationDeliveries,
  listNotificationChannels,
  listNotifications,
  markNotificationRead,
  retryNotificationDelivery,
  testNotificationChannel
} from "../../api/notifications";
import { useLanguageStore } from "../../i18n/language";
import { hasPermission } from "../auth/permissions";
import { useAuthStore } from "../auth/store";

export function NotificationsPage() {
  const user = useAuthStore((state) => state.user);
  const t = useLanguageStore((state) => state.t);
  const queryClient = useQueryClient();
  const canWrite = hasPermission(user, "notification:write");
  const [unreadOnly, setUnreadOnly] = useState(false);
  const [form, setForm] = useState({ name: "", channelType: "site", target: "", signingSecret: "" });
  const [deliveryFilters, setDeliveryFilters] = useState({ status: "", channelId: "", notificationId: "" });
  const channelsQuery = useQuery({ queryKey: ["notificationChannels"], queryFn: listNotificationChannels });
  const notificationsQuery = useQuery({
    queryKey: ["notifications", unreadOnly],
    queryFn: () => listNotifications({ unread: unreadOnly })
  });
  const deliveriesQuery = useQuery({
    queryKey: ["notificationDeliveries", deliveryFilters],
    queryFn: () => listNotificationDeliveries(deliveryFilters)
  });
  const createMutation = useMutation({
    mutationFn: createNotificationChannel,
    onSuccess: () => {
      setForm({ name: "", channelType: "site", target: "", signingSecret: "" });
      queryClient.invalidateQueries({ queryKey: ["notificationChannels"] });
    }
  });
  const readMutation = useMutation({
    mutationFn: (id: string) => markNotificationRead(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["notifications"] })
  });
  const retryMutation = useMutation({
    mutationFn: (id: number) => retryNotificationDelivery(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["notificationDeliveries"] })
  });
  const testChannelMutation = useMutation({
    mutationFn: (id: string) => testNotificationChannel(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["notifications"] });
      queryClient.invalidateQueries({ queryKey: ["notificationDeliveries"] });
    }
  });
  const channels = useMemo(() => channelsQuery.data ?? [], [channelsQuery.data]);
  const notifications = useMemo(() => notificationsQuery.data ?? [], [notificationsQuery.data]);
  const deliveries = useMemo(() => deliveriesQuery.data ?? [], [deliveriesQuery.data]);

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">{t("monitoring.eyebrow")}</p>
          <h1>{t("notifications.title")}</h1>
        </div>
      </section>

      {canWrite ? (
        <section className="panel form-panel">
          <div className="panel-title"><h3>{t("notifications.createChannel")}</h3><span>{t("notifications.channelHint")}</span></div>
          <form
            className="form-grid"
            onSubmit={(event) => {
              event.preventDefault();
              createMutation.mutate({
                name: form.name,
                channelType: form.channelType,
                config: form.channelType === "site"
                  ? undefined
                  : form.channelType === "email"
                    ? { email: form.target }
                    : {
                        url: form.target,
                        ...(form.signingSecret ? { signingSecret: form.signingSecret } : {})
                      }
              });
            }}
          >
            <label>{t("common.name")}<input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required /></label>
            <label>
              {t("common.type")}
              <select value={form.channelType} onChange={(event) => setForm({ ...form, channelType: event.target.value })}>
                <option value="site">site</option>
                <option value="email">email</option>
                <option value="webhook">webhook</option>
                <option value="dingtalk">dingtalk</option>
                <option value="wechat">wechat</option>
                <option value="slack">slack</option>
              </select>
            </label>
            {form.channelType !== "site" ? (
              <>
                <label>
                  {form.channelType === "email" ? t("notifications.emailTo") : t("notifications.webhookUrl")}
                  <input value={form.target} onChange={(event) => setForm({ ...form, target: event.target.value })} required />
                </label>
                {form.channelType !== "email" ? (
                  <label>
                    {t("notifications.signingSecret")}
                    <input value={form.signingSecret} onChange={(event) => setForm({ ...form, signingSecret: event.target.value })} placeholder={t("notifications.signingSecretOptional")} />
                  </label>
                ) : null}
              </>
            ) : null}
            <button type="submit" disabled={createMutation.isPending}>{t("notifications.createChannelAction")}</button>
          </form>
          {createMutation.isError ? <p className="form-error">{createMutation.error.message}</p> : null}
        </section>
      ) : null}

      <section className="panel table-panel">
        <div className="panel-title"><h3>{t("notifications.deliveries")}</h3><span>{deliveries.length} {t("common.items")}</span></div>
        <div className="toolbar-row">
          <select value={deliveryFilters.status} onChange={(event) => setDeliveryFilters((current) => ({ ...current, status: event.target.value }))}>
            <option value="">{t("notifications.allDeliveryStatuses")}</option>
            <option value="pending">pending</option>
            <option value="sending">sending</option>
            <option value="success">success</option>
            <option value="failed">failed</option>
            <option value="canceled">canceled</option>
          </select>
          <select value={deliveryFilters.channelId} onChange={(event) => setDeliveryFilters((current) => ({ ...current, channelId: event.target.value }))}>
            <option value="">{t("notifications.allChannels")}</option>
            {channels.map((channel) => <option key={channel.id} value={channel.id}>{channel.name}</option>)}
          </select>
          <input
            placeholder={t("notifications.notificationId")}
            value={deliveryFilters.notificationId}
            onChange={(event) => setDeliveryFilters((current) => ({ ...current, notificationId: event.target.value }))}
          />
          <button type="button" onClick={() => deliveriesQuery.refetch()}>{t("common.refresh")}</button>
        </div>
        <div className="data-table">
          <table>
            <thead><tr><th>{t("notifications.notification")}</th><th>{t("notifications.channels")}</th><th>{t("common.status")}</th><th>{t("notifications.attempts")}</th><th>{t("common.reason")}</th><th>{t("common.created")}</th><th>{t("common.action")}</th></tr></thead>
            <tbody>
              {deliveries.map((item) => (
                <tr key={item.id}>
                  <td><strong>{item.title || "-"}</strong><small>{item.notificationId || item.id}</small></td>
                  <td><strong>{item.channelName || "-"}</strong><small>{item.channelType || "-"}</small></td>
                  <td><span className={`status-chip status-${item.status}`}>{item.status}</span></td>
                  <td><strong>{item.attempts}</strong><small>{item.nextRetryAt || item.deliveredAt || "-"}</small></td>
                  <td>{item.errorMessage || "-"}</td>
                  <td>{item.createdAt}</td>
                  <td className="action-cell">
                    {canWrite && item.status === "failed" ? <button type="button" onClick={() => retryMutation.mutate(item.id)}>{t("notifications.retryDelivery")}</button> : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!deliveriesQuery.isLoading && deliveries.length === 0 ? <p className="empty-state">{t("notifications.emptyDeliveries")}</p> : null}
        {deliveriesQuery.isError ? <p className="form-error">{deliveriesQuery.error.message}</p> : null}
        {retryMutation.isError ? <p className="form-error">{retryMutation.error.message}</p> : null}
      </section>

      <section className="panel table-panel">
        <div className="panel-title"><h3>{t("notifications.title")}</h3><span>{notifications.length} {t("common.items")}</span></div>
        <div className="toolbar-row">
          <label className="inline-check">
            <input type="checkbox" checked={unreadOnly} onChange={(event) => setUnreadOnly(event.target.checked)} />
            <span>{t("notifications.unreadOnly")}</span>
          </label>
          <button type="button" onClick={() => notificationsQuery.refetch()}>{t("common.refresh")}</button>
        </div>
        <div className="data-table">
          <table>
            <thead><tr><th>{t("notifications.notification")}</th><th>{t("common.severity")}</th><th>{t("common.category")}</th><th>{t("common.status")}</th><th>{t("common.created")}</th><th>{t("common.action")}</th></tr></thead>
            <tbody>
              {notifications.map((item) => (
                <tr key={item.id}>
                  <td><strong>{item.title}</strong><small>{item.content || item.id}</small></td>
                  <td><span className={`status-chip status-${item.severity}`}>{item.severity}</span></td>
                  <td>{item.category}</td>
                  <td>{item.readAt ? t("common.status.read") : t("common.status.unread")}</td>
                  <td>{item.createdAt}</td>
                  <td className="action-cell">
                    {!item.readAt ? <button type="button" onClick={() => readMutation.mutate(item.id)}>{t("notifications.markRead")}</button> : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!notificationsQuery.isLoading && notifications.length === 0 ? <p className="empty-state">{t("notifications.empty")}</p> : null}
        {notificationsQuery.isError ? <p className="form-error">{notificationsQuery.error.message}</p> : null}
        {readMutation.isError ? <p className="form-error">{readMutation.error.message}</p> : null}
      </section>

      <section className="panel table-panel">
        <div className="panel-title"><h3>{t("notifications.channels")}</h3><span>{channels.length} {t("common.total")}</span></div>
        <div className="data-table">
          <table>
            <thead><tr><th>{t("common.name")}</th><th>{t("common.type")}</th><th>{t("notifications.target")}</th><th>{t("common.status")}</th><th>{t("common.created")}</th><th>{t("common.action")}</th></tr></thead>
            <tbody>
              {channels.map((channel) => (
                <tr key={channel.id}>
                  <td><strong>{channel.name}</strong><small>{channel.id}</small></td>
                  <td>{channel.channelType}</td>
                  <td>{channel.targetSummary || "-"}</td>
                  <td><span className={`status-chip status-${channel.status}`}>{channel.status}</span></td>
                  <td>{channel.createdAt}</td>
                  <td className="action-cell">
                    {canWrite && channel.status === "active" ? (
                      <button type="button" disabled={testChannelMutation.isPending} onClick={() => testChannelMutation.mutate(channel.id)}>
                        {t("notifications.testChannel")}
                      </button>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!channelsQuery.isLoading && channels.length === 0 ? <p className="empty-state">{t("notifications.emptyChannels")}</p> : null}
        {channelsQuery.isError ? <p className="form-error">{channelsQuery.error.message}</p> : null}
        {testChannelMutation.isError ? <p className="form-error">{testChannelMutation.error.message}</p> : null}
      </section>
    </main>
  );
}
