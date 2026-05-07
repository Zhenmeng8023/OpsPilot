import { useQuery } from "@tanstack/react-query";

import { request } from "../../api/request";
import type { HealthData, VersionInfo } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";
import { StatusBadge } from "../../shared/components/StatusBadge";

export function DashboardPage() {
  const t = useLanguageStore((state) => state.t);
  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: () => request<HealthData>("/health", { skipAuth: true })
  });
  const versionQuery = useQuery({
    queryKey: ["version"],
    queryFn: () => request<VersionInfo>("/api/v1/version", { skipAuth: true })
  });

  const cards = [
    { label: t("dashboard.onlineAgents"), value: "0", hint: t("dashboard.agentHint") },
    { label: t("dashboard.todayTasks"), value: "0", hint: t("dashboard.taskHint") },
    { label: t("dashboard.failedTasks"), value: "0", hint: t("dashboard.failedHint") },
    { label: t("dashboard.recentAlerts"), value: "0", hint: t("dashboard.alertHint") }
  ];

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">Overview</p>
          <h1>{t("dashboard.title")}</h1>
        </div>
        <StatusBadge status={healthQuery.data?.status ?? "checking"} />
      </section>

      <section className="metrics-grid">
        {cards.map((card) => (
          <article className="metric-card" key={card.label}>
            <span>{card.label}</span>
            <strong>{card.value}</strong>
            <small>{card.hint}</small>
          </article>
        ))}
      </section>

      <section className="panel-grid">
        <article className="panel">
          <div className="panel-title">
            <h3>{t("dashboard.apiHealth")}</h3>
            <span>{healthQuery.isFetching ? t("common.loading") : t("dashboard.realtime")}</span>
          </div>
          {healthQuery.isError ? (
            <p className="error-text">{t("dashboard.connectError")}</p>
          ) : (
            <dl className="health-list">
              <div>
                <dt>{t("dashboard.service")}</dt>
                <dd>{healthQuery.data?.service ?? "-"}</dd>
              </div>
              <div>
                <dt>{t("dashboard.env")}</dt>
                <dd>{healthQuery.data?.env ?? "-"}</dd>
              </div>
              <div>
                <dt>{t("common.version")}</dt>
                <dd>{versionQuery.data?.version ?? healthQuery.data?.version ?? "-"}</dd>
              </div>
              <div>
                <dt>Commit</dt>
                <dd>{versionQuery.data?.commit || "-"}</dd>
              </div>
              <div>
                <dt>Build time</dt>
                <dd>{versionQuery.data?.buildTime || "-"}</dd>
              </div>
              <div>
                <dt>Go</dt>
                <dd>{versionQuery.data?.goVersion || "-"}</dd>
              </div>
              <div>
                <dt>{t("dashboard.time")}</dt>
                <dd>{healthQuery.data?.time ?? "-"}</dd>
              </div>
            </dl>
          )}
        </article>

        <article className="panel terminal-panel">
          <div className="panel-title">
            <h3>{t("dashboard.devFlow")}</h3>
            <span>MVP</span>
          </div>
          <pre>{`login -> agent register -> script template
task create -> run -> stream logs -> result audit`}</pre>
        </article>
      </section>
    </main>
  );
}
