import { useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { lookupTrace } from "../../api/tracecenter";
import type { TraceCenterResult } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";
import { DataTable } from "../../shared/components/DataTable";
import { FilterToolbar } from "../../shared/components/FilterToolbar";
import { JsonViewer } from "../../shared/components/JsonViewer";
import { Timeline, type TimelineItem } from "../../shared/components/Timeline";

type QueryState = {
  traceId: string;
  taskRunId: string;
  workflowRunId: string;
  webhookEventId: string;
};

const emptyQuery: QueryState = {
  traceId: "",
  taskRunId: "",
  workflowRunId: "",
  webhookEventId: ""
};

export function TraceCenterPage() {
  const t = useLanguageStore((state) => state.t);
  const [filters, setFilters] = useState<QueryState>(emptyQuery);
  const [submitted, setSubmitted] = useState<QueryState>(emptyQuery);
  const [selectedPayload, setSelectedPayload] = useState("");

  const hasSubmittedQuery = useMemo(() => {
    return Object.values(submitted).some((value) => value.trim() !== "");
  }, [submitted]);

  const resultQuery = useQuery({
    queryKey: ["traceCenter", submitted],
    enabled: hasSubmittedQuery,
    queryFn: () => lookupTrace(submitted)
  });
  const result = resultQuery.data;

  const timelineItems = useMemo<TimelineItem[]>(() => {
    return (result?.timeline ?? []).map((item, index) => ({
      id: `${item.time}-${item.category}-${item.title}-${index}`,
      title: `${item.category}: ${item.title}`,
      time: item.time,
      description: [item.status, item.reference, item.detail].filter(Boolean).join(" / "),
      tone: timelineTone(item.status)
    }));
  }, [result?.timeline]);

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">{t("traceCenter.eyebrow")}</p>
          <h1>{t("traceCenter.title")}</h1>
        </div>
      </section>

      <section className="panel table-panel">
        <div className="panel-title">
          <h3>{t("traceCenter.search")}</h3>
          <span>{t("traceCenter.searchHint")}</span>
        </div>
        <FilterToolbar>
          <input
            placeholder={t("traceCenter.traceId")}
            value={filters.traceId}
            onChange={(event) => setFilters((current) => ({ ...current, traceId: event.target.value }))}
          />
          <input
            placeholder={t("traceCenter.workflowRunId")}
            value={filters.workflowRunId}
            onChange={(event) => setFilters((current) => ({ ...current, workflowRunId: event.target.value }))}
          />
          <input
            placeholder={t("traceCenter.taskRunId")}
            value={filters.taskRunId}
            onChange={(event) => setFilters((current) => ({ ...current, taskRunId: event.target.value }))}
          />
          <input
            placeholder={t("traceCenter.webhookEventId")}
            value={filters.webhookEventId}
            onChange={(event) => setFilters((current) => ({ ...current, webhookEventId: event.target.value }))}
          />
          <button type="button" onClick={() => setSubmitted(filters)}>{t("traceCenter.search")}</button>
          <button
            type="button"
            onClick={() => {
              setFilters(emptyQuery);
              setSubmitted(emptyQuery);
              setSelectedPayload("");
            }}
          >
            {t("common.clear")}
          </button>
        </FilterToolbar>
        {resultQuery.isError ? <p className="form-error">{resultQuery.error.message}</p> : null}
      </section>

      {result ? <TraceResultCard result={result} /> : null}

      <section className="panel">
        <div className="panel-title">
          <h3>{t("traceCenter.timeline")}</h3>
          <span>{result?.timeline.length ?? 0} {t("common.items")}</span>
        </div>
        <DataTable
          loading={resultQuery.isFetching}
          empty={(result?.timeline.length ?? 0) === 0}
          emptyMessage={hasSubmittedQuery ? t("traceCenter.empty") : t("traceCenter.searchHint")}
          error={resultQuery.isError ? resultQuery.error.message : null}
        >
          <table>
            <thead>
              <tr>
                <th>{t("dashboard.time")}</th>
                <th>{t("traceCenter.category")}</th>
                <th>{t("common.status")}</th>
                <th>{t("traceCenter.reference")}</th>
                <th>{t("traceCenter.detail")}</th>
                <th>{t("traceCenter.payload")}</th>
              </tr>
            </thead>
            <tbody>
              {(result?.timeline ?? []).map((item, index) => (
                <tr key={`${item.time}-${item.category}-${item.title}-${index}`}>
                  <td>{item.time}</td>
                  <td><strong>{item.category}</strong><small>{item.title}</small></td>
                  <td>{item.status || "-"}</td>
                  <td>{item.reference || "-"}</td>
                  <td>{item.detail || "-"}</td>
                  <td>
                    {item.payload ? (
                      <button type="button" onClick={() => setSelectedPayload(item.payload ?? "")}>JSON</button>
                    ) : "-"}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </DataTable>
        {timelineItems.length > 0 ? <Timeline items={timelineItems} /> : null}
      </section>

      {selectedPayload ? (
        <section className="panel">
          <div className="panel-title">
            <h3>{t("traceCenter.payload")}</h3>
          </div>
          <JsonViewer value={selectedPayload} />
        </section>
      ) : null}
    </main>
  );
}

function TraceResultCard({ result }: { result: TraceCenterResult }) {
  const t = useLanguageStore((state) => state.t);
  return (
    <section className="panel">
      <div className="panel-title">
        <h3>{t("traceCenter.result")}</h3>
      </div>
      <div className="event-detail-grid">
        <div><strong>{t("traceCenter.traceId")}</strong><span>{result.traceId || "-"}</span></div>
        <div><strong>{t("traceCenter.workflowRunId")}</strong><span>{result.workflowRunId || "-"}</span></div>
        <div><strong>{t("traceCenter.taskRunId")}</strong><span>{result.taskRunId || "-"}</span></div>
        <div><strong>{t("traceCenter.webhookEventId")}</strong><span>{result.webhookEventId || "-"}</span></div>
      </div>
    </section>
  );
}

function timelineTone(status?: string): TimelineItem["tone"] {
  if (!status) return "default";
  if (status === "success" || status === "matched" || status === "resolved") return "success";
  if (status === "failed" || status === "denied" || status === "canceled") return "danger";
  if (status === "warning" || status === "pending" || status === "queued") return "warning";
  return "default";
}
