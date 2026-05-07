import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import {
  createEnrollmentToken,
  disableAgent,
  listAgents,
  listEnrollmentTokens,
  listHosts,
  markOffline,
  revokeEnrollmentToken
} from "../../api/agents";
import type { Agent, EnrollmentTokenDetail } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";

const statusOrder = ["online", "offline", "disabled", "registered", "upgrading"];

export function AgentManagementPage() {
  const queryClient = useQueryClient();
  const t = useLanguageStore((state) => state.t);
  const [createForm, setCreateForm] = useState({
    maxUses: 1,
    expiresInSeconds: 3600,
    bindWorkspaceSlug: ""
  });
  const [issuedToken, setIssuedToken] = useState<EnrollmentTokenDetail | null>(null);
  const [copyMessage, setCopyMessage] = useState("");

  const agentsQuery = useQuery({
    queryKey: ["agents"],
    queryFn: () => listAgents()
  });
  const hostsQuery = useQuery({
    queryKey: ["hosts"],
    queryFn: () => listHosts()
  });
  const enrollmentQuery = useQuery({
    queryKey: ["agentEnrollmentTokens"],
    queryFn: () => listEnrollmentTokens()
  });

  const disableAgentMutation = useMutation({
    mutationFn: (agent: Agent) => disableAgent(agent.id),
    onSuccess: () => refreshLists(queryClient)
  });

  const scanOfflineMutation = useMutation({
    mutationFn: () => markOffline(),
    onSuccess: () => refreshLists(queryClient)
  });

  const createEnrollmentMutation = useMutation({
    mutationFn: () =>
      createEnrollmentToken({
        maxUses: Math.max(1, createForm.maxUses),
        expiresInSeconds: Math.max(60, createForm.expiresInSeconds),
        bindWorkspaceSlug: createForm.bindWorkspaceSlug.trim() || undefined
      }),
    onSuccess: (detail) => {
      setIssuedToken(detail);
      setCopyMessage("");
      void refreshLists(queryClient, true);
    }
  });

  const revokeEnrollmentMutation = useMutation({
    mutationFn: (id: string) => revokeEnrollmentToken(id),
    onSuccess: () => {
      if (issuedToken && issuedToken.status === "active") {
        setIssuedToken({ ...issuedToken, status: "revoked" });
      }
      void refreshLists(queryClient, true);
    }
  });

  const agents = agentsQuery.data?.items ?? [];
  const hosts = hostsQuery.data?.items ?? [];
  const enrollmentTokens = enrollmentQuery.data ?? [];
  const counts = useMemo(() => summarizeAgents(agents), [agents]);

  async function copyToken(token: string) {
    try {
      await navigator.clipboard.writeText(token);
      setCopyMessage("Copied");
    } catch {
      setCopyMessage("Copy failed");
    }
  }

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">Control Plane</p>
          <h1>{t("agents.title")}</h1>
        </div>
        <button className="ghost-button" type="button" onClick={() => scanOfflineMutation.mutate()} disabled={scanOfflineMutation.isPending}>
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
                        disabled={agent.status === "disabled" || disableAgentMutation.isPending}
                        onClick={() => disableAgentMutation.mutate(agent)}
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
          {disableAgentMutation.isError ? <p className="form-error">{String(disableAgentMutation.error.message)}</p> : null}
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
          {scanOfflineMutation.isError ? <p className="form-error">{String(scanOfflineMutation.error.message)}</p> : null}
        </section>

        <section className="panel table-panel">
          <div className="panel-title">
            <h3>Enrollment Tokens</h3>
            <span>{enrollmentTokens.length} total</span>
          </div>
          <div className="enrollment-form">
            <div className="form-grid">
              <label>
                Max uses
                <input
                  type="number"
                  min={1}
                  value={createForm.maxUses}
                  onChange={(event) =>
                    setCreateForm((current) => ({ ...current, maxUses: Number(event.target.value) || 1 }))
                  }
                />
              </label>
              <label>
                Expires in seconds
                <input
                  type="number"
                  min={60}
                  value={createForm.expiresInSeconds}
                  onChange={(event) =>
                    setCreateForm((current) => ({ ...current, expiresInSeconds: Number(event.target.value) || 60 }))
                  }
                />
              </label>
            </div>
            <label>
              Bind workspace slug (optional)
              <input
                value={createForm.bindWorkspaceSlug}
                onChange={(event) => setCreateForm((current) => ({ ...current, bindWorkspaceSlug: event.target.value }))}
                placeholder="default"
              />
            </label>
            <div className="enrollment-actions">
              <button
                className="ghost-button"
                type="button"
                disabled={createEnrollmentMutation.isPending}
                onClick={() => createEnrollmentMutation.mutate()}
              >
                {createEnrollmentMutation.isPending ? "Creating" : "Create token"}
              </button>
              {issuedToken?.token ? (
                <button className="ghost-button" type="button" onClick={() => copyToken(issuedToken.token ?? "")}>
                  Copy token
                </button>
              ) : null}
            </div>
            {issuedToken?.token ? (
              <div className="token-secret">
                <strong>{issuedToken.token}</strong>
                <span>
                  Save this now. It is only returned once. {copyMessage}
                </span>
              </div>
            ) : null}
          </div>
          <div className="data-table">
            <table>
              <thead>
                <tr>
                  <th>Prefix</th>
                  <th>Status</th>
                  <th>Usage</th>
                  <th>Expires</th>
                  <th>Created by</th>
                  <th>Action</th>
                </tr>
              </thead>
              <tbody>
                {enrollmentTokens.map((token) => (
                  <tr key={token.id}>
                    <td>
                      <strong>{token.tokenPrefix}</strong>
                      <small>{token.id}</small>
                    </td>
                    <td>
                      <StatusChip status={token.status} />
                    </td>
                    <td>
                      <strong>
                        {token.usedCount}/{token.maxUses}
                      </strong>
                      <small>{token.bindWorkspaceSlug || "any workspace"}</small>
                    </td>
                    <td>{token.expiresAt}</td>
                    <td>{token.createdBy || "-"}</td>
                    <td>
                      <button
                        className="danger-button"
                        type="button"
                        disabled={token.status !== "active" || revokeEnrollmentMutation.isPending}
                        onClick={() => revokeEnrollmentMutation.mutate(token.id)}
                      >
                        Revoke
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {!enrollmentQuery.isLoading && enrollmentTokens.length === 0 ? (
            <p className="empty-state">No enrollment tokens found.</p>
          ) : null}
          {enrollmentQuery.isError ? <p className="form-error">{String(enrollmentQuery.error.message)}</p> : null}
          {createEnrollmentMutation.isError ? (
            <p className="form-error">{String(createEnrollmentMutation.error.message)}</p>
          ) : null}
          {revokeEnrollmentMutation.isError ? (
            <p className="form-error">{String(revokeEnrollmentMutation.error.message)}</p>
          ) : null}
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

async function refreshLists(queryClient: QueryClient, includeEnrollment?: boolean) {
  const tasks = [
    queryClient.invalidateQueries({ queryKey: ["agents"] }),
    queryClient.invalidateQueries({ queryKey: ["hosts"] })
  ];
  if (includeEnrollment) {
    tasks.push(queryClient.invalidateQueries({ queryKey: ["agentEnrollmentTokens"] }));
  }
  await Promise.all(tasks);
}
