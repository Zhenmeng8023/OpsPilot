import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  createNotificationChannel,
  listNotificationDeliveries,
  listNotificationChannels,
  listNotifications,
  markNotificationRead,
  retryNotificationDelivery,
  testNotificationChannel,
  updateNotificationChannel
} from "../../api/notifications";
import type { NotificationChannel } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";
import { DataTable } from "../../shared/components/DataTable";
import { FilterToolbar } from "../../shared/components/FilterToolbar";
import { hasPermission } from "../auth/permissions";
import { useAuthStore } from "../auth/store";

export function NotificationsPage() {
  const user = useAuthStore((state) => state.user);
  const t = useLanguageStore((state) => state.t);
  const queryClient = useQueryClient();
  const canWrite = hasPermission(user, "notification:write");
  const [unreadOnly, setUnreadOnly] = useState(false);
  const [form, setForm] = useState({ name: "", channelType: "site", target: "", signingSecret: "" });
  const [editingChannel, setEditingChannel] = useState<NotificationChannel | null>(null);
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
      resetChannelForm();
      queryClient.invalidateQueries({ queryKey: ["notificationChannels"] });
    }
  });
  const updateMutation = useMutation({
    mutationFn: (payload: { id: string; name: string; channelType: string; config?: Record<string, unknown> }) =>
      updateNotificationChannel(payload.id, {
        name: payload.name,
        channelType: payload.channelType,
        config: payload.config
      }),
    onSuccess: () => {
      resetChannelForm();
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

  function resetChannelForm() {
    setEditingChannel(null);
    setForm({ name: "", channelType: "site", target: "", signingSecret: "" });
  }

  function startEditingChannel(channel: NotificationChannel) {
    setEditingChannel(channel);
    setForm({
      name: channel.name,
      channelType: channel.channelType,
      target: "",
      signingSecret: ""
    });
  }

  function buildChannelConfig() {
    if (form.channelType === "site") {
      return undefined;
    }
    if (form.channelType === "email") {
      return form.target.trim() ? { email: form.target.trim() } : undefined;
    }
    const config: Record<string, unknown> = {};
    if (form.target.trim()) {
      config.url = form.target.trim();
    }
    if (form.signingSecret.trim()) {
      config.signingSecret = form.signingSecret.trim();
    }
    return Object.keys(config).length > 0 ? config : undefined;
  }

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
          <div className="panel-title">
            <h3>{editingChannel ? t("notifications.updateChannel") : t("notifications.createChannel")}</h3>
            <span>{editingChannel ? t("notifications.channelUpdateHint") : t("notifications.channelHint")}</span>
          </div>
          <form
            className="form-grid"
            onSubmit={(event) => {
              event.preventDefault();
              const payload = {
                name: form.name,
                channelType: form.channelType,
                config: buildChannelConfig()
              };
              if (editingChannel) {
                updateMutation.mutate({ id: editingChannel.id, ...payload });
                return;
              }
              createMutation.mutate(payload);
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
                  <input
                    value={form.target}
                    onChange={(event) => setForm({ ...form, target: event.target.value })}
                    required={!editingChannel}
                    placeholder={editingChannel?.targetSummary || ""}
                  />
                </label>
                {form.channelType !== "email" ? (
                  <label>
                    {t("notifications.signingSecret")}
                    <input
                      value={form.signingSecret}
                      onChange={(event) => setForm({ ...form, signingSecret: event.target.value })}
                      placeholder={editingChannel ? t("notifications.signingSecretRotateHint") : t("notifications.signingSecretOptional")}
                    />
                  </label>
                ) : null}
              </>
            ) : null}
            <button type="submit" disabled={createMutation.isPending || updateMutation.isPending}>
              {editingChannel ? t("notifications.updateChannelAction") : t("notifications.createChannelAction")}
            </button>
            {editingChannel ? <button type="button" className="ghost-button" onClick={resetChannelForm}>{t("common.cancel")}</button> : null}
          </form>
          {createMutation.isError ? <p className="form-error">{createMutation.error.message}</p> : null}
          {updateMutation.isError ? <p className="form-error">{updateMutation.error.message}</p> : null}
        </section>
      ) : null}

      <section className="panel table-panel">
        <div className="panel-title"><h3>{t("notifications.deliveries")}</h3><span>{deliveries.length} {t("common.items")}</span></div>
        <FilterToolbar>
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
        </FilterToolbar>
        <DataTable loading={deliveriesQuery.isLoading} empty={deliveries.length === 0} emptyMessage={t("notifications.emptyDeliveries")} error={deliveriesQuery.isError ? deliveriesQuery.error.message : null}>
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
        </DataTable>
        {retryMutation.isError ? <p className="form-error">{retryMutation.error.message}</p> : null}
      </section>

      <section className="panel table-panel">
        <div className="panel-title"><h3>{t("notifications.title")}</h3><span>{notifications.length} {t("common.items")}</span></div>
        <FilterToolbar>
          <label className="inline-check">
            <input type="checkbox" checked={unreadOnly} onChange={(event) => setUnreadOnly(event.target.checked)} />
            <span>{t("notifications.unreadOnly")}</span>
          </label>
          <button type="button" onClick={() => notificationsQuery.refetch()}>{t("common.refresh")}</button>
        </FilterToolbar>
        <DataTable loading={notificationsQuery.isLoading} empty={notifications.length === 0} emptyMessage={t("notifications.empty")} error={notificationsQuery.isError ? notificationsQuery.error.message : null}>
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
        </DataTable>
        {readMutation.isError ? <p className="form-error">{readMutation.error.message}</p> : null}
      </section>

      <section className="panel table-panel">
        <div className="panel-title"><h3>{t("notifications.channels")}</h3><span>{channels.length} {t("common.total")}</span></div>
        <DataTable loading={channelsQuery.isLoading} empty={channels.length === 0} emptyMessage={t("notifications.emptyChannels")} error={channelsQuery.isError ? channelsQuery.error.message : null}>
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
                    {canWrite && channel.status === "active" ? <button type="button" onClick={() => startEditingChannel(channel)}>{t("common.edit")}</button> : null}
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
        </DataTable>
        {testChannelMutation.isError ? <p className="form-error">{testChannelMutation.error.message}</p> : null}
      </section>
    </main>
  );
}
