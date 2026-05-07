import { useMemo } from "react";
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import { request } from "../../api/request";
import type { Agent, Host, OfflineScanResult } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";

const statusOrder = ["online", "offline", "disabled", "registered", "upgrading"];

export function AgentManagementPage() {
  const queryClient = useQueryClient();
  const t = useLanguageStore((state) => state.t);
  const agentsQuery = useQuery({
    queryKey: ["agents"],
    queryFn: () => request<Agent[]>("/api/v1/agents")
  });
  const hostsQuery = useQuery({
    queryKey: ["hosts"],
    queryFn: () => request<Host[]>("/api/v1/hosts")
  });

  const disableAgent = useMutation({
    mutationFn: (agent: Agent) =>
      request<{ ok: boolean }>(`/api/v1/agents/${agent.id}/disable`, {
        method: "POST"
      }),
    onSuccess: () => refreshLists(queryClient)
  });

  const scanOffline = useMutation({
    mutationFn: () =>
      request<OfflineScanResult>("/api/v1/agents/offline-scan", {
        method: "POST"
      }),
    onSuccess: () => refreshLists(queryClient)
  });

  const agents = agentsQuery.data ?? [];
  const hosts = hostsQuery.data ?? [];
  const counts = useMemo(() => summarizeAgents(agents), [agents]);

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">Control Plane</p>
          <h1>{t("agents.title")}</h1>
        </div>
        <button className="ghost-button" type="button" onClick={() => scanOffline.mutate()} disabled={scanOffline.isPending}>
          {t("agents.scanOffline")}
        </button>
      </section>

      <section className="agent-summary">
        {statusOrder.slice(0, 3).map((status) => (
          <div className="panel agent-stat" key={status}>
            <span>{status === "online" ? t("common.online") : status === "offline" ? t("common.offline") : t("common.disabled")}</span>
            <strong>{counts[status] ?? 0}</strong>
          </div>
        ))}
      </section>

      <section className="agent-grid">
        <section className="panel table-panel">
          <div className="panel-title">
            <h3>Agents</h3>
            <span>
              {agents.length} {t("common.total")}
            </span>
          </div>
          <div className="data-table">
            <table>
              <thead>
                <tr>
                  <th>{t("agents.agent")}</th>
                  <th>{t("common.status")}</th>
                  <th>{t("agents.host")}</th>
                  <th>{t("common.version")}</th>
                  <th>{t("common.lastHeartbeat")}</th>
                  <th>{t("common.action")}</th>
                </tr>
              </thead>
              <tbody>
                {agents.map((agent) => (
                  <tr key={agent.id}>
                    <td>
                      <strong>{agent.name}</strong>
                      <small>{agent.tokenPrefix || agent.id}</small>
                    </td>
                    <td>
                      <StatusChip status={agent.status} />
                    </td>
                    <td>
                      <strong>{agent.host?.name ?? "-"}</strong>
                      <small>{agent.ip || agent.host?.ip || "-"}</small>
                    </td>
                    <td>
                      <strong>{agent.version || "-"}</strong>
                      <small>{[agent.os, agent.arch].filter(Boolean).join(" / ") || "-"}</small>
                    </td>
                    <td>{agent.lastHeartbeatAt || "-"}</td>
                    <td>
                      <button
                        className="danger-button"
                        type="button"
                        disabled={agent.status === "disabled" || disableAgent.isPending}
                        onClick={() => disableAgent.mutate(agent)}
                      >
                        {t("agents.disable")}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {agentsQuery.isError ? <p className="form-error">{String(agentsQuery.error.message)}</p> : null}
          {disableAgent.isError ? <p className="form-error">{String(disableAgent.error.message)}</p> : null}
        </section>

        <section className="panel table-panel">
          <div className="panel-title">
            <h3>{t("agents.hosts")}</h3>
            <span>
              {hosts.length} {t("common.total")}
            </span>
          </div>
          <div className="data-table">
            <table>
              <thead>
                <tr>
                  <th>{t("agents.host")}</th>
                  <th>{t("common.status")}</th>
                  <th>Agents</th>
                  <th>{t("common.system")}</th>
                  <th>{t("common.lastHeartbeat")}</th>
                </tr>
              </thead>
              <tbody>
                {hosts.map((host) => (
                  <tr key={host.id}>
                    <td>
                      <strong>{host.name}</strong>
                      <small>{host.hostname || host.ip || host.id}</small>
                    </td>
                    <td>
                      <StatusChip status={host.status} />
                    </td>
                    <td>
                      <strong>
                        {host.onlineAgentCount ?? 0} {t("agents.onlineCount")}
                      </strong>
                      <small>
                        {host.agentCount ?? 0} {t("agents.totalCount")}
                      </small>
                    </td>
                    <td>
                      <strong>{host.os || "-"}</strong>
                      <small>{host.arch || "-"}</small>
                    </td>
                    <td>{host.lastHeartbeatAt || "-"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {hostsQuery.isError ? <p className="form-error">{String(hostsQuery.error.message)}</p> : null}
          {scanOffline.isError ? <p className="form-error">{String(scanOffline.error.message)}</p> : null}
        </section>
      </section>
    </main>
  );
}

function StatusChip({ status }: { status: string }) {
  return <span className={`status-chip status-${status}`}>{status}</span>;
}

function summarizeAgents(agents: Agent[]) {
  return agents.reduce<Record<string, number>>((acc, agent) => {
    acc[agent.status] = (acc[agent.status] ?? 0) + 1;
    return acc;
  }, {});
}

async function refreshLists(queryClient: QueryClient) {
  await Promise.all([
    queryClient.invalidateQueries({ queryKey: ["agents"] }),
    queryClient.invalidateQueries({ queryKey: ["hosts"] })
  ]);
}
