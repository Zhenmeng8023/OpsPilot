import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { acknowledgeAlert, resolveAlert, silenceAlert, unsilenceAlert } from "../../api/alerts";
import { getIncident, listIncidents } from "../../api/incidents";
import type { IncidentSummary } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";
import { ConfirmDialog } from "../../shared/components/ConfirmDialog";
import { DataTable } from "../../shared/components/DataTable";
import { FilterToolbar } from "../../shared/components/FilterToolbar";
import { JsonViewer } from "../../shared/components/JsonViewer";
import { Timeline, type TimelineItem } from "../../shared/components/Timeline";
import { useToast } from "../../shared/components/ToastProvider";
import { hasPermission } from "../auth/permissions";
import { useAuthStore } from "../auth/store";

type IncidentAction = "acknowledge" | "silence" | "unsilence" | "resolve";
type PendingAction = { action: IncidentAction; incident: IncidentSummary };

export function IncidentPage() {
  const t = useLanguageStore((state) => state.t);
  const user = useAuthStore((state) => state.user);
  const { notify } = useToast();
  const queryClient = useQueryClient();
  const [filters, setFilters] = useState({ status: "", severity: "" });
  const [selectedId, setSelectedId] = useState("");
  const [pendingAction, setPendingAction] = useState<PendingAction | null>(null);
  const canWrite = hasPermission(user, "alert:write");
  const incidentsQuery = useQuery({
    queryKey: ["incidents", filters],
    queryFn: () => listIncidents(filters)
  });
  const detailQuery = useQuery({
    queryKey: ["incident", selectedId],
    enabled: Boolean(selectedId),
    queryFn: () => getIncident(selectedId)
  });
  const incidents = useMemo(() => incidentsQuery.data ?? [], [incidentsQuery.data]);
  const detail = detailQuery.data;
  const statusText = (value: string) => t(`common.status.${value}`);
  const actionMutation = useMutation({
    mutationFn: async ({ action, incident }: PendingAction) => {
      if (!incident.alertId) throw new Error("incident alert link is missing");
      if (action === "acknowledge") return acknowledgeAlert(incident.alertId);
      if (action === "silence") return silenceAlert(incident.alertId, { durationSeconds: 3600 });
      if (action === "unsilence") return unsilenceAlert(incident.alertId);
      return resolveAlert(incident.alertId);
    },
    onSuccess: () => {
      notify(t("incidents.updatedToast"), "success");
      queryClient.invalidateQueries({ queryKey: ["incidents"] });
      if (selectedId) queryClient.invalidateQueries({ queryKey: ["incident", selectedId] });
    }
  });
  const confirmTitle = pendingAction ? t(`incidents.confirm${capitalize(pendingAction.action)}`) : "";

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">{t("incidents.eyebrow")}</p>
          <h1>{t("incidents.title")}</h1>
        </div>
      </section>

      <section className="panel table-panel">
        <div className="panel-title">
          <h3>{t("incidents.active")}</h3>
          <span>{incidents.length} {t("common.total")}</span>
        </div>
        <FilterToolbar>
          <select value={filters.status} onChange={(event) => setFilters((current) => ({ ...current, status: event.target.value }))}>
            <option value="">{t("common.allStatus")}</option>
            <option value="open">{statusText("open")}</option>
            <option value="acknowledged">{statusText("acknowledged")}</option>
            <option value="silenced">{statusText("silenced")}</option>
            <option value="resolved">{statusText("resolved")}</option>
          </select>
          <select value={filters.severity} onChange={(event) => setFilters((current) => ({ ...current, severity: event.target.value }))}>
            <option value="">{t("metrics.allSeverities")}</option>
            <option value="critical">critical</option>
            <option value="warning">warning</option>
            <option value="info">info</option>
          </select>
          <button type="button" onClick={() => incidentsQuery.refetch()}>{t("common.refresh")}</button>
        </FilterToolbar>
        <DataTable loading={incidentsQuery.isLoading} empty={incidents.length === 0} emptyMessage={t("incidents.empty")} error={incidentsQuery.isError ? incidentsQuery.error.message : null}>
          <table>
            <thead>
              <tr>
                <th>{t("incidents.incident")}</th>
                <th>{t("common.severity")}</th>
                <th>{t("common.status")}</th>
                <th>{t("metrics.rule")}</th>
                <th>{t("metrics.host")}</th>
                <th>{t("metrics.lastSeen")}</th>
                <th>{t("common.action")}</th>
              </tr>
            </thead>
            <tbody>
              {incidents.map((incident) => (
                <tr key={incident.id}>
                  <td><strong>{incident.title}</strong><small>{incident.message || incident.id}</small></td>
                  <td>{incident.severity}</td>
                  <td><span className={`status-chip status-${incident.status}`}>{statusText(incident.status)}</span></td>
                  <td><strong>{incident.ruleName || "-"}</strong><small>{incident.ruleId || "-"}</small></td>
                  <td><strong>{incident.hostName || "-"}</strong><small>{incident.hostId || "-"}</small></td>
                  <td><strong>{incident.lastSeenAt}</strong><small>{incident.firstSeenAt}</small></td>
                  <td className="action-cell">
                    <button type="button" onClick={() => setSelectedId((current) => current === incident.id ? "" : incident.id)}>{selectedId === incident.id ? t("common.hide") : t("common.view")}</button>
                    {canWrite ? incidentActions(incident).map((action) => (
                      <button key={action} type="button" disabled={!incident.alertId || actionMutation.isPending} onClick={() => setPendingAction({ action, incident })}>
                        {t(`incidents.${action}`)}
                      </button>
                    )) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </DataTable>
      </section>

      {selectedId ? (
        <section className="panel">
          <div className="panel-title">
            <h3>{detail?.title ?? t("incidents.timeline")}</h3>
            <span>{detail?.status ? statusText(detail.status) : t("common.loading")}</span>
          </div>
          {detailQuery.isLoading ? <p className="empty-state">{t("common.loading")}</p> : null}
          {detail ? (
            <>
              <div className="event-detail-grid">
                <div><strong>{t("common.severity")}</strong><span>{detail.severity}</span></div>
                <div><strong>{t("common.status")}</strong><span>{statusText(detail.status)}</span></div>
                <div><strong>{t("metrics.rule")}</strong><span>{detail.ruleName || "-"}</span></div>
                <div><strong>{t("metrics.host")}</strong><span>{detail.hostName || "-"}</span></div>
              </div>
              <Timeline
                items={detail.events.map((event): TimelineItem => ({
                  id: event.id,
                  title: statusText(event.eventType),
                  time: event.createdAt,
                  description: event.message || event.actor || event.eventType,
                  tone: eventTone(event.eventType)
                }))}
              />
              {detail.events.map((event) => event.payload ? (
                <div className="event-payload" key={`payload-${event.id}`}>
                  <strong>{event.eventType}</strong>
                  <JsonViewer value={event.payload} />
                </div>
              ) : null)}
            </>
          ) : null}
          {detailQuery.isError ? <p className="form-error">{detailQuery.error.message}</p> : null}
          {actionMutation.isError ? <p className="form-error">{actionMutation.error.message}</p> : null}
        </section>
      ) : null}
      <ConfirmDialog
        open={Boolean(pendingAction)}
        title={confirmTitle}
        confirmLabel={t("common.confirm")}
        cancelLabel={t("common.cancel")}
        danger={pendingAction?.action === "resolve"}
        onCancel={() => setPendingAction(null)}
        onConfirm={() => {
          if (pendingAction) actionMutation.mutate(pendingAction);
          setPendingAction(null);
        }}
      />
    </main>
  );
}

function eventTone(eventType: string): TimelineItem["tone"] {
  if (eventType === "resolved" || eventType === "acknowledged") return "success";
  if (eventType === "silenced" || eventType === "cooldown_suppressed") return "warning";
  if (eventType === "firing") return "danger";
  return "default";
}

function incidentActions(incident: IncidentSummary): IncidentAction[] {
  if (incident.status === "resolved") return [];
  const actions: IncidentAction[] = [];
  if (incident.status === "open") actions.push("acknowledge", "silence");
  if (incident.status === "acknowledged") actions.push("silence");
  if (incident.status === "silenced") actions.push("unsilence");
  actions.push("resolve");
  return actions;
}

function capitalize(value: string) {
  return `${value.slice(0, 1).toUpperCase()}${value.slice(1)}`;
}
