import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { closeIncident, getIncident, listIncidents, mergeIncident, updateIncidentLifecycle } from "../../api/incidents";
import type { IncidentDetail } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";
import { Button } from "../../shared/components/Button";
import { DataTable } from "../../shared/components/DataTable";
import { FormGrid } from "../../shared/components/FormGrid";
import { JsonViewer } from "../../shared/components/JsonViewer";
import { PagePanel } from "../../shared/components/PagePanel";
import { SectionToolbar } from "../../shared/components/SectionToolbar";
import { SelectField } from "../../shared/components/SelectField";
import { TextField } from "../../shared/components/TextField";
import { Timeline, type TimelineItem } from "../../shared/components/Timeline";
import { useToast } from "../../shared/components/ToastProvider";
import { hasPermission } from "../auth/permissions";
import { useAuthStore } from "../auth/store";

export function IncidentPage() {
  const t = useLanguageStore((state) => state.t);
  const user = useAuthStore((state) => state.user);
  const { notify } = useToast();
  const queryClient = useQueryClient();
  const [filters, setFilters] = useState({ status: "", severity: "" });
  const [selectedId, setSelectedId] = useState("");
  const [lifecycleForm, setLifecycleForm] = useState({
    owner: "",
    impactScope: "",
    rootCauseClass: "",
    postmortem: ""
  });
  const [mergeForm, setMergeForm] = useState({ targetIncidentId: "", reason: "" });
  const [closeReason, setCloseReason] = useState("");
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

  useEffect(() => {
    if (!detail) {
      return;
    }
    setLifecycleForm({
      owner: detail.owner ?? "",
      impactScope: detail.impactScope ?? "",
      rootCauseClass: detail.rootCause ?? "",
      postmortem: detail.postmortem ?? ""
    });
  }, [detail]);

  useEffect(() => {
    setMergeForm({ targetIncidentId: "", reason: "" });
    setCloseReason("");
  }, [selectedId]);

  const refreshIncidentData = () => {
    queryClient.invalidateQueries({ queryKey: ["incidents"] });
    if (selectedId) {
      queryClient.invalidateQueries({ queryKey: ["incident", selectedId] });
    }
  };

  const lifecycleMutation = useMutation({
    mutationFn: async () => {
      if (!selectedId) throw new Error("incident id is required");
      return updateIncidentLifecycle(selectedId, lifecycleForm);
    },
    onSuccess: () => {
      notify(t("incidents.updatedToast"), "success");
      refreshIncidentData();
    }
  });

  const mergeMutation = useMutation({
    mutationFn: async () => {
      if (!selectedId) throw new Error("incident id is required");
      const targetIncidentId = mergeForm.targetIncidentId.trim();
      if (!targetIncidentId) throw new Error(t("incidents.mergeTargetRequired"));
      return mergeIncident(selectedId, {
        targetIncidentId,
        reason: mergeForm.reason.trim()
      });
    },
    onSuccess: () => {
      notify(t("incidents.mergeToast"), "success");
      refreshIncidentData();
    }
  });

  const closeMutation = useMutation({
    mutationFn: async () => {
      if (!selectedId) throw new Error("incident id is required");
      return closeIncident(selectedId, { reason: closeReason.trim() });
    },
    onSuccess: () => {
      notify(t("incidents.closeToast"), "success");
      refreshIncidentData();
    }
  });

  const statusOptions = [
    { value: "", label: t("common.allStatus") },
    { value: "open", label: statusText("open") },
    { value: "acknowledged", label: statusText("acknowledged") },
    { value: "silenced", label: statusText("silenced") },
    { value: "resolved", label: statusText("resolved") }
  ];

  const severityOptions = [
    { value: "", label: t("metrics.allSeverities") },
    { value: "critical", label: "critical" },
    { value: "warning", label: "warning" },
    { value: "info", label: "info" }
  ];

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">{t("incidents.eyebrow")}</p>
          <h1>{t("incidents.title")}</h1>
        </div>
      </section>

      <PagePanel title={t("incidents.active")} subtitle={`${incidents.length} ${t("common.total")}`} className="table-panel">
        <SectionToolbar>
          <SelectField
            value={filters.status}
            options={statusOptions}
            onChange={(event) => setFilters((current) => ({ ...current, status: event.target.value }))}
          />
          <SelectField
            value={filters.severity}
            options={severityOptions}
            onChange={(event) => setFilters((current) => ({ ...current, severity: event.target.value }))}
          />
          <Button type="button" onClick={() => incidentsQuery.refetch()}>{t("common.refresh")}</Button>
        </SectionToolbar>
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
                    <Button type="button" onClick={() => setSelectedId((current) => current === incident.id ? "" : incident.id)}>
                      {selectedId === incident.id ? t("common.hide") : t("common.view")}
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </DataTable>
      </PagePanel>

      {selectedId ? (
        <PagePanel title={detail?.title ?? t("incidents.timeline")} subtitle={detail?.status ? statusText(detail.status) : t("common.loading")}>
          {detailQuery.isLoading ? <p className="empty-state">{t("common.loading")}</p> : null}
          {detail ? (
            <>
              <div className="event-detail-grid">
                <div><strong>{t("common.severity")}</strong><span>{detail.severity}</span></div>
                <div><strong>{t("common.status")}</strong><span>{statusText(detail.status)}</span></div>
                <div><strong>{t("metrics.rule")}</strong><span>{detail.ruleName || "-"}</span></div>
                <div><strong>{t("metrics.host")}</strong><span>{detail.hostName || "-"}</span></div>
                <div><strong>{t("incidents.owner")}</strong><span>{detail.owner || "-"}</span></div>
                <div><strong>{t("incidents.impactScope")}</strong><span>{detail.impactScope || "-"}</span></div>
                <div><strong>{t("incidents.rootCause")}</strong><span>{detail.rootCause || "-"}</span></div>
                <div><strong>{t("incidents.mergedInto")}</strong><span>{detail.mergedInto || "-"}</span></div>
              </div>
              {detail.postmortem ? (
                <div className="event-payload">
                  <strong>{t("incidents.postmortem")}</strong>
                  <pre className="json-block">{detail.postmortem}</pre>
                </div>
              ) : null}
              <Timeline
                items={detail.events.map((event): TimelineItem => ({
                  id: event.id,
                  title: eventLabel(event.eventType, t),
                  time: event.createdAt,
                  description: event.message || event.actor || event.eventType,
                  tone: eventTone(event.eventType)
                }))}
              />
              {detail.events.map((event) => event.payload ? (
                <div className="event-payload" key={`payload-${event.id}`}>
                  <strong>{eventLabel(event.eventType, t)}</strong>
                  <JsonViewer value={event.payload} />
                </div>
              ) : null)}
            </>
          ) : null}
          {detailQuery.isError ? <p className="form-error">{detailQuery.error.message}</p> : null}

          {detail && canWrite ? (
            <div className="incident-action-grid">
              <form
                className="editor-form"
                onSubmit={(event) => {
                  event.preventDefault();
                  lifecycleMutation.mutate();
                }}
              >
                <h4>{t("incidents.lifecycle")}</h4>
                <p>{t("incidents.lifecycleHint")}</p>
                <FormGrid columns={2} className="incident-lifecycle-grid">
                  <TextField
                    label={t("incidents.owner")}
                    value={lifecycleForm.owner}
                    onChange={(event) => setLifecycleForm((current) => ({ ...current, owner: event.target.value }))}
                  />
                  <TextField
                    label={t("incidents.rootCause")}
                    value={lifecycleForm.rootCauseClass}
                    onChange={(event) => setLifecycleForm((current) => ({ ...current, rootCauseClass: event.target.value }))}
                  />
                  <TextField
                    label={t("incidents.impactScope")}
                    value={lifecycleForm.impactScope}
                    onChange={(event) => setLifecycleForm((current) => ({ ...current, impactScope: event.target.value }))}
                    className="span-2"
                  />
                  <TextField
                    label={t("incidents.postmortem")}
                    value={lifecycleForm.postmortem}
                    onChange={(event) => setLifecycleForm((current) => ({ ...current, postmortem: event.target.value }))}
                    multiline
                    rows={4}
                    className="span-2"
                  />
                </FormGrid>
                <SectionToolbar className="incident-toolbar">
                  <Button type="submit" disabled={lifecycleMutation.isPending}>{t("incidents.saveLifecycle")}</Button>
                </SectionToolbar>
              </form>

              <form
                className="editor-form"
                onSubmit={(event) => {
                  event.preventDefault();
                  mergeMutation.mutate();
                }}
              >
                <h4>{t("incidents.merge")}</h4>
                <p>{t("incidents.mergeHint")}</p>
                <TextField
                  label={t("incidents.mergeTarget")}
                  value={mergeForm.targetIncidentId}
                  onChange={(event) => setMergeForm((current) => ({ ...current, targetIncidentId: event.target.value }))}
                />
                <TextField
                  label={t("incidents.mergeReason")}
                  value={mergeForm.reason}
                  onChange={(event) => setMergeForm((current) => ({ ...current, reason: event.target.value }))}
                />
                <SectionToolbar className="incident-toolbar">
                  <Button type="submit" disabled={mergeMutation.isPending}>{t("incidents.mergeAction")}</Button>
                </SectionToolbar>
              </form>

              <form
                className="editor-form"
                onSubmit={(event) => {
                  event.preventDefault();
                  closeMutation.mutate();
                }}
              >
                <h4>{t("incidents.close")}</h4>
                <p>{t("incidents.closeHint")}</p>
                <TextField
                  label={t("incidents.closeReason")}
                  value={closeReason}
                  onChange={(event) => setCloseReason(event.target.value)}
                />
                <SectionToolbar className="incident-toolbar">
                  <Button type="submit" tone="danger" disabled={closeMutation.isPending}>{t("incidents.closeAction")}</Button>
                </SectionToolbar>
              </form>
            </div>
          ) : null}
          {lifecycleMutation.isError ? <p className="form-error">{lifecycleMutation.error.message}</p> : null}
          {mergeMutation.isError ? <p className="form-error">{mergeMutation.error.message}</p> : null}
          {closeMutation.isError ? <p className="form-error">{closeMutation.error.message}</p> : null}
        </PagePanel>
      ) : null}
    </main>
  );
}

function eventTone(eventType: string): TimelineItem["tone"] {
  if (eventType === "resolved" || eventType === "closed" || eventType === "acknowledged") return "success";
  if (eventType === "silenced" || eventType === "cooldown_suppressed" || eventType === "lifecycle_updated") return "warning";
  if (eventType === "firing") return "danger";
  return "default";
}

function eventLabel(eventType: string, t: (key: string) => string) {
  switch (eventType) {
    case "lifecycle_updated":
      return t("incidents.lifecycle");
    case "merged":
      return t("incidents.merge");
    case "merged_from":
      return t("incidents.mergedFrom");
    case "closed":
      return t("incidents.close");
    default:
      return t(`common.status.${eventType}`);
  }
}

