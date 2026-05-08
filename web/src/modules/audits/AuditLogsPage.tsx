import { useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";

import { exportAuditLogs, listAuditLogs, runAuditRetention } from "../../api/audits";
import type { AuditLog } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";
import { DataTable } from "../../shared/components/DataTable";
import { FilterToolbar } from "../../shared/components/FilterToolbar";
import { JsonViewer } from "../../shared/components/JsonViewer";
import { PaginationBar } from "../../shared/components/PaginationBar";

const pageSize = 20;

export function AuditLogsPage() {
  const t = useLanguageStore((state) => state.t);
  const [page, setPage] = useState(1);
  const [selected, setSelected] = useState<AuditLog | null>(null);
  const [retentionDays, setRetentionDays] = useState("90");
  const [filters, setFilters] = useState({
    keyword: "",
    action: "",
    actor: "",
    actorType: "",
    result: "",
    resourceType: "",
    resourceId: "",
    traceId: "",
    createdFrom: "",
    createdTo: ""
  });
  const exportMutation = useMutation({
    mutationFn: async (format: "csv" | "json") => {
      const response = await exportAuditLogs({ ...filters, format });
      if (!response.ok) {
        throw new Error(`Export failed with status ${response.status}`);
      }
      const blob = await response.blob();
      const filename = response.headers.get("Content-Disposition")?.match(/filename="(.+)"/)?.[1] ?? `audit-logs.${format}`;
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = filename;
      link.click();
      URL.revokeObjectURL(url);
    }
  });
  const retentionMutation = useMutation({
    mutationFn: (dryRun: boolean) => runAuditRetention({ dryRun, days: Number(retentionDays) || undefined })
  });
  const query = useQuery({
    queryKey: ["auditLogs", filters, page],
    queryFn: () => listAuditLogs({ ...filters, page, pageSize })
  });
  const logs = useMemo(() => query.data?.items ?? [], [query.data]);
  const total = query.data?.total ?? 0;

  function updateFilter(key: keyof typeof filters, value: string) {
    setPage(1);
    setFilters((current) => ({ ...current, [key]: value }));
  }

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">{t("audit.eyebrow")}</p>
          <h1>{t("audit.title")}</h1>
        </div>
      </section>

      <section className="panel table-panel">
        <div className="panel-title">
          <h3>{t("audit.events")}</h3>
          <span>{total} {t("common.total")}</span>
        </div>
        <FilterToolbar>
          <input placeholder={t("audit.keyword")} value={filters.keyword} onChange={(event) => updateFilter("keyword", event.target.value)} />
          <input placeholder={t("audit.action")} value={filters.action} onChange={(event) => updateFilter("action", event.target.value)} />
          <input placeholder={t("audit.actor")} value={filters.actor} onChange={(event) => updateFilter("actor", event.target.value)} />
          <select value={filters.actorType} onChange={(event) => updateFilter("actorType", event.target.value)}>
            <option value="">{t("audit.allActorTypes")}</option>
            <option value="user">user</option>
            <option value="agent">agent</option>
            <option value="webhook">webhook</option>
            <option value="system">system</option>
          </select>
          <select value={filters.result} onChange={(event) => updateFilter("result", event.target.value)}>
            <option value="">{t("audit.allResults")}</option>
            <option value="success">success</option>
            <option value="failed">failed</option>
            <option value="denied">denied</option>
          </select>
          <input placeholder={t("audit.resourceType")} value={filters.resourceType} onChange={(event) => updateFilter("resourceType", event.target.value)} />
          <input placeholder={t("audit.resourceId")} value={filters.resourceId} onChange={(event) => updateFilter("resourceId", event.target.value)} />
          <input placeholder={t("audit.traceId")} value={filters.traceId} onChange={(event) => updateFilter("traceId", event.target.value)} />
          <input type="datetime-local" value={filters.createdFrom} onChange={(event) => updateFilter("createdFrom", event.target.value)} />
          <input type="datetime-local" value={filters.createdTo} onChange={(event) => updateFilter("createdTo", event.target.value)} />
          <input placeholder={t("audit.retentionDays")} value={retentionDays} onChange={(event) => setRetentionDays(event.target.value)} />
          <button type="button" onClick={() => query.refetch()}>{t("common.refresh")}</button>
          <button type="button" onClick={() => exportMutation.mutate("csv")}>{t("audit.exportCsv")}</button>
          <button type="button" onClick={() => exportMutation.mutate("json")}>{t("audit.exportJson")}</button>
          <button type="button" onClick={() => retentionMutation.mutate(true)}>{t("audit.retentionDryRun")}</button>
          <button type="button" className="danger-button" onClick={() => retentionMutation.mutate(false)}>{t("audit.retentionExecute")}</button>
        </FilterToolbar>
        <DataTable loading={query.isLoading} empty={logs.length === 0} emptyMessage={t("audit.empty")} error={query.isError ? query.error.message : null}>
          <table>
            <thead>
              <tr>
                <th>{t("audit.action")}</th>
                <th>{t("audit.actor")}</th>
                <th>{t("audit.resource")}</th>
                <th>{t("audit.request")}</th>
                <th>{t("common.status")}</th>
                <th>{t("common.created")}</th>
                <th>{t("common.action")}</th>
              </tr>
            </thead>
            <tbody>
              {logs.map((item) => (
                <tr key={item.id}>
                  <td><strong>{item.action}</strong><small>{item.traceId || item.id}</small></td>
                  <td><strong>{actorName(item)}</strong><small>{item.actorType}</small></td>
                  <td><strong>{item.resourceType || "-"}</strong><small>{item.resourceId || "-"}</small></td>
                  <td><strong>{requestLine(item)}</strong><small>{item.ip || "-"}</small></td>
                  <td><span className={`status-chip status-${item.result}`}>{item.result}</span></td>
                  <td>{item.createdAt}</td>
                  <td className="action-cell"><button type="button" onClick={() => setSelected(item)}>{t("common.view")}</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </DataTable>
        <PaginationBar total={total} page={page} pageSize={pageSize} onPageChange={setPage} />
        {retentionMutation.data ? (
          <p className="empty-state">
            {t("audit.retentionResult", {
              mode: retentionMutation.data.dryRun ? t("audit.retentionDryRun") : t("audit.retentionExecute"),
              matched: retentionMutation.data.matched,
              deleted: retentionMutation.data.deleted,
              cutoff: retentionMutation.data.cutoffAt
            })}
          </p>
        ) : null}
        {exportMutation.isError ? <p className="form-error">{exportMutation.error.message}</p> : null}
        {retentionMutation.isError ? <p className="form-error">{retentionMutation.error.message}</p> : null}
      </section>

      {selected ? (
        <section className="panel">
          <div className="panel-title">
            <h3>{t("audit.detail")}</h3>
            <button type="button" onClick={() => setSelected(null)}>{t("common.hide")}</button>
          </div>
          <div className="event-detail-grid">
            <div><strong>{t("audit.action")}</strong><span>{selected.action}</span></div>
            <div><strong>{t("audit.actor")}</strong><span>{actorName(selected)} ({selected.actorType})</span></div>
            <div><strong>{t("audit.resource")}</strong><span>{selected.resourceType || "-"} #{selected.resourceId || "-"}</span></div>
            <div><strong>{t("audit.request")}</strong><span>{requestLine(selected)}</span></div>
            <div><strong>{t("audit.traceId")}</strong><span>{selected.traceId || "-"}</span></div>
            <div><strong>IP</strong><span>{selected.ip || "-"}</span></div>
            <div><strong>User Agent</strong><span>{selected.userAgent || "-"}</span></div>
            <div><strong>{t("common.created")}</strong><span>{selected.createdAt}</span></div>
          </div>
          <AuditJsonBlock title={t("audit.before")} value={selected.before} />
          <AuditJsonBlock title={t("audit.after")} value={selected.after} />
          <AuditJsonBlock title={t("audit.metadata")} value={selected.metadata} />
        </section>
      ) : null}
    </main>
  );
}

function AuditJsonBlock({ title, value }: { title: string; value?: string }) {
  return (
    <div className="event-payload">
      <strong>{title}</strong>
      <JsonViewer value={value} emptyLabel="-" />
    </div>
  );
}

function actorName(item: AuditLog) {
  return item.actorUser || item.actorAgent || item.actorType;
}

function requestLine(item: AuditLog) {
  const method = item.requestMethod || "-";
  const path = item.requestPath || "-";
  return `${method} ${path}`;
}
