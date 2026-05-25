import { useQueries } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";

import { request } from "../../api/request";
import type { HealthData, VersionInfo } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";
import { StatusBadge } from "../../shared/components/StatusBadge";
import { IconAgents, IconTasks, IconSchedules, IconWorkflows, IconScripts, IconWebhooks } from "../../shared/components/Icons";

interface CountResponse { total: number }

export function DashboardPage() {
  const t = useLanguageStore((state) => state.t);
  const navigate = useNavigate();

  const results = useQueries({
    queries: [
      { queryKey: ["health"], queryFn: () => request<HealthData>("/health", { skipAuth: true }), staleTime: 30000 },
      { queryKey: ["version"], queryFn: () => request<VersionInfo>("/api/v1/version", { skipAuth: true }), staleTime: 60000 },
      { queryKey: ["dashboard-agents"], queryFn: () => request<CountResponse>("/api/v1/agents"), staleTime: 15000 },
      { queryKey: ["dashboard-tasks"], queryFn: () => request<CountResponse>("/api/v1/tasks?limit=1"), staleTime: 15000 },
      { queryKey: ["dashboard-schedules"], queryFn: () => request<CountResponse>("/api/v1/schedules?limit=1"), staleTime: 15000 },
      { queryKey: ["dashboard-incidents"], queryFn: () => request<CountResponse>("/api/v1/incidents?limit=1"), staleTime: 15000 },
      { queryKey: ["dashboard-workflows"], queryFn: () => request<CountResponse>("/api/v1/workflows?limit=1"), staleTime: 15000 },
    ]
  });

  const [healthQuery, versionQuery, agentsQuery, tasksQuery, schedulesQuery, incidentsQuery, workflowsQuery] = results;
  const isLoading = results.some((r) => r.isLoading);

  const statCards = [
    {
      label: t("dashboard.onlineAgents"),
      value: agentsQuery.data?.total ?? (agentsQuery.data as any)?.length ?? "-",
      hint: t("dashboard.agentHint"),
      icon: <IconAgents />,
      className: "stat-card-info",
      onClick: () => navigate("/agents")
    },
    {
      label: t("dashboard.todayTasks"),
      value: tasksQuery.data?.total ?? "-",
      hint: t("dashboard.taskHint"),
      icon: <IconTasks />,
      className: "stat-card-accent",
      onClick: () => navigate("/tasks")
    },
    {
      label: t("dashboard.schedules"),
      value: schedulesQuery.data?.total ?? "-",
      hint: t("dashboard.scheduleHint"),
      icon: <IconSchedules />,
      className: "stat-card-accent",
      onClick: () => navigate("/schedules")
    },
    {
      label: t("dashboard.workflows"),
      value: workflowsQuery.data?.total ?? "-",
      hint: t("dashboard.workflowHint"),
      icon: <IconWorkflows />,
      className: "stat-card-info",
      onClick: () => navigate("/workflows")
    },
  ];

  const quickActions = [
    { label: t("tasks.createAction"), to: "/tasks/new", icon: <IconTasks /> },
    { label: t("scripts.createAction"), to: "/scripts/new", icon: <IconScripts /> },
    { label: t("schedules.createAction"), to: "/schedules", icon: <IconSchedules /> },
    { label: t("webhooks.createSource"), to: "/webhooks", icon: <IconWebhooks /> },
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
        {statCards.map((card) => (
          <article
            className={`stat-card ${card.className}`}
            key={card.label}
            onClick={card.onClick}
            role="button"
            tabIndex={0}
            onKeyDown={(e) => {
              if (e.key === "Enter" || e.key === " ") {
                e.preventDefault();
                card.onClick();
              }
            }}
          >
            <div className="stat-card-label">
              <span className="stat-card-icon">{card.icon}</span>
              {card.label}
            </div>
            <div className="stat-card-value">
              {isLoading ? <span className="skeleton" style={{ display: "inline-block", width: "3rem", height: "1.5em" }} /> : card.value}
            </div>
            <span className="stat-card-hint">{card.hint}</span>
          </article>
        ))}
      </section>

      <section className="panel-grid">
        <article className="panel">
          <div className="panel-title">
            <h3>{t("dashboard.quickActions")}</h3>
          </div>
          <div className="quick-actions-grid">
            {quickActions.map((action) => (
              <button
                key={action.to}
                type="button"
                className="quick-action"
                onClick={() => navigate(action.to)}
              >
                <span className="quick-action-icon">{action.icon}</span>
                <span className="quick-action-label">{action.label}</span>
              </button>
            ))}
          </div>
        </article>

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
                <dd style={{ fontSize: "0.7rem", fontFamily: "var(--font-mono)" }}>{versionQuery.data?.commit?.substring(0, 8) || "-"}</dd>
              </div>
              <div>
                <dt>Go</dt>
                <dd>{versionQuery.data?.goVersion || "-"}</dd>
              </div>
              <div>
                <dt>{t("dashboard.time")}</dt>
                <dd style={{ fontSize: "0.72rem" }}>{healthQuery.data?.time ?? "-"}</dd>
              </div>
            </dl>
          )}
        </article>
      </section>
    </main>
  );
}
