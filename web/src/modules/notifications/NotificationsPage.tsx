import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  createNotificationChannel,
  listNotificationChannels,
  listNotifications,
  markNotificationRead
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
  const [form, setForm] = useState({ name: "", channelType: "site", url: "" });
  const channelsQuery = useQuery({ queryKey: ["notificationChannels"], queryFn: listNotificationChannels });
  const notificationsQuery = useQuery({
    queryKey: ["notifications", unreadOnly],
    queryFn: () => listNotifications({ unread: unreadOnly })
  });
  const createMutation = useMutation({
    mutationFn: createNotificationChannel,
    onSuccess: () => {
      setForm({ name: "", channelType: "site", url: "" });
      queryClient.invalidateQueries({ queryKey: ["notificationChannels"] });
    }
  });
  const readMutation = useMutation({
    mutationFn: (id: string) => markNotificationRead(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["notifications"] })
  });
  const channels = useMemo(() => channelsQuery.data ?? [], [channelsQuery.data]);
  const notifications = useMemo(() => notificationsQuery.data ?? [], [notificationsQuery.data]);

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
                config: form.channelType === "site" ? undefined : { url: form.url }
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
              <label>{t("notifications.webhookUrl")}<input value={form.url} onChange={(event) => setForm({ ...form, url: event.target.value })} required /></label>
            ) : null}
            <button type="submit" disabled={createMutation.isPending}>{t("notifications.createChannelAction")}</button>
          </form>
          {createMutation.isError ? <p className="form-error">{createMutation.error.message}</p> : null}
        </section>
      ) : null}

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
            <thead><tr><th>{t("common.name")}</th><th>{t("common.type")}</th><th>{t("common.status")}</th><th>{t("common.created")}</th></tr></thead>
            <tbody>
              {channels.map((channel) => (
                <tr key={channel.id}>
                  <td><strong>{channel.name}</strong><small>{channel.id}</small></td>
                  <td>{channel.channelType}</td>
                  <td><span className={`status-chip status-${channel.status}`}>{channel.status}</span></td>
                  <td>{channel.createdAt}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!channelsQuery.isLoading && channels.length === 0 ? <p className="empty-state">{t("notifications.emptyChannels")}</p> : null}
        {channelsQuery.isError ? <p className="form-error">{channelsQuery.error.message}</p> : null}
      </section>
    </main>
  );
}
