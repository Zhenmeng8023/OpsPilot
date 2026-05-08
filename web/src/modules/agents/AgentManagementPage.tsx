import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient, type QueryClient } from "@tanstack/react-query";

import {
  createHostGroup,
  createMaintenanceWindow,
  createEnrollmentToken,
  createTag,
  disableAgent,
  disableHostGroupAgents,
  listAgentDiagnostics,
  listHostGroupDiagnostics,
  listAgents,
  listEnrollmentTokens,
  listHostGroups,
  listHosts,
  listMaintenanceWindows,
  listTags,
  markOffline,
  revokeEnrollmentToken,
  setAgentTags,
  setHostTags,
  updateHostGroup,
  updateMaintenanceWindow
} from "../../api/agents";
import type { Agent, EnrollmentTokenDetail, HostGroup, MaintenanceWindow, TagSummary } from "../../api/types";
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
  const [maintenanceForm, setMaintenanceForm] = useState({
    name: "",
    scopeType: "all",
    agentId: "",
    hostId: "",
    hostGroupId: "",
    reason: "",
    startsAt: "",
    endsAt: "",
    status: "active"
  });
  const [editingMaintenanceId, setEditingMaintenanceId] = useState("");
  const [tagForm, setTagForm] = useState({ name: "", color: "#2f7d99" });
  const [tagAssignment, setTagAssignment] = useState({
    resourceType: "agent",
    resourceId: "",
    tagIds: [] as string[]
  });
  const [hostGroupForm, setHostGroupForm] = useState({
    name: "",
    description: "",
    hostIds: [] as string[]
  });
  const [editingHostGroupId, setEditingHostGroupId] = useState("");
  const [selectedBatchGroupId, setSelectedBatchGroupId] = useState("");

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
  const diagnosticsQuery = useQuery({
    queryKey: ["agentDiagnostics"],
    queryFn: listAgentDiagnostics
  });
  const maintenanceQuery = useQuery({
    queryKey: ["maintenanceWindows"],
    queryFn: listMaintenanceWindows
  });
  const tagsQuery = useQuery({
    queryKey: ["tags"],
    queryFn: listTags
  });
  const hostGroupsQuery = useQuery({
    queryKey: ["hostGroups"],
    queryFn: listHostGroups
  });
  const hostGroupDiagnosticsQuery = useQuery({
    queryKey: ["hostGroupDiagnostics", selectedBatchGroupId],
    queryFn: () => listHostGroupDiagnostics(selectedBatchGroupId),
    enabled: selectedBatchGroupId !== ""
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
  const createMaintenanceMutation = useMutation({
    mutationFn: createMaintenanceWindow,
    onSuccess: () => {
      resetMaintenanceForm();
      void queryClient.invalidateQueries({ queryKey: ["maintenanceWindows"] });
    }
  });
  const updateMaintenanceMutation = useMutation({
    mutationFn: (payload: typeof maintenanceForm) => updateMaintenanceWindow(editingMaintenanceId, payload),
    onSuccess: () => {
      resetMaintenanceForm();
      void queryClient.invalidateQueries({ queryKey: ["maintenanceWindows"] });
    }
  });
  const createTagMutation = useMutation({
    mutationFn: createTag,
    onSuccess: () => {
      setTagForm({ name: "", color: "#2f7d99" });
      void queryClient.invalidateQueries({ queryKey: ["tags"] });
    }
  });
  const setResourceTagsMutation = useMutation({
    mutationFn: () =>
      tagAssignment.resourceType === "host"
        ? setHostTags(tagAssignment.resourceId, tagAssignment.tagIds)
        : setAgentTags(tagAssignment.resourceId, tagAssignment.tagIds),
    onSuccess: () => {
      void Promise.all([
        queryClient.invalidateQueries({ queryKey: ["agents"] }),
        queryClient.invalidateQueries({ queryKey: ["hosts"] }),
        queryClient.invalidateQueries({ queryKey: ["tags"] })
      ]);
    }
  });
  const createHostGroupMutation = useMutation({
    mutationFn: createHostGroup,
    onSuccess: () => {
      resetHostGroupForm();
      void queryClient.invalidateQueries({ queryKey: ["hostGroups"] });
    }
  });
  const updateHostGroupMutation = useMutation({
    mutationFn: (payload: typeof hostGroupForm) => updateHostGroup(editingHostGroupId, payload),
    onSuccess: () => {
      resetHostGroupForm();
      void queryClient.invalidateQueries({ queryKey: ["hostGroups"] });
    }
  });
  const disableHostGroupAgentsMutation = useMutation({
    mutationFn: (id: string) => disableHostGroupAgents(id),
    onSuccess: () => {
      void refreshLists(queryClient);
      void queryClient.invalidateQueries({ queryKey: ["hostGroupDiagnostics"] });
    }
  });

  const agents = agentsQuery.data?.items ?? [];
  const hosts = hostsQuery.data?.items ?? [];
  const enrollmentTokens = enrollmentQuery.data ?? [];
  const diagnostics = diagnosticsQuery.data ?? [];
  const maintenanceWindows = maintenanceQuery.data ?? [];
  const tags = tagsQuery.data ?? [];
  const hostGroups = hostGroupsQuery.data ?? [];
  const hostGroupDiagnostics = hostGroupDiagnosticsQuery.data ?? [];
  const counts = useMemo(() => summarizeAgents(agents), [agents]);

  async function copyToken(token: string) {
    try {
      await navigator.clipboard.writeText(token);
      setCopyMessage("Copied");
    } catch {
      setCopyMessage("Copy failed");
    }
  }

  function resetMaintenanceForm() {
    setMaintenanceForm({ name: "", scopeType: "all", agentId: "", hostId: "", hostGroupId: "", reason: "", startsAt: "", endsAt: "", status: "active" });
    setEditingMaintenanceId("");
  }

  function startEditingMaintenance(item: MaintenanceWindow) {
    setEditingMaintenanceId(item.id);
    setMaintenanceForm({
      name: item.name,
      scopeType: item.scopeType || "all",
      agentId: item.agentId || "",
      hostId: item.hostId || "",
      hostGroupId: item.hostGroupId || "",
      reason: item.reason || "",
      startsAt: item.startsAt,
      endsAt: item.endsAt,
      status: item.status || "active"
    });
  }

  function resetHostGroupForm() {
    setHostGroupForm({ name: "", description: "", hostIds: [] });
    setEditingHostGroupId("");
  }

  function startEditingHostGroup(item: HostGroup) {
    setEditingHostGroupId(item.id);
    setHostGroupForm({
      name: item.name,
      description: item.description || "",
      hostIds: (item.hosts ?? []).map((host) => host.id)
    });
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
            <h3>{t("agents.tags")}</h3>
            <span>{tags.length} {t("common.total")}</span>
          </div>
          <form
            className="form-grid"
            onSubmit={(event) => {
              event.preventDefault();
              createTagMutation.mutate({ name: tagForm.name, color: tagForm.color });
            }}
          >
            <label>{t("common.name")}<input value={tagForm.name} onChange={(event) => setTagForm({ ...tagForm, name: event.target.value })} required /></label>
            <label>{t("agents.color")}<input type="color" value={tagForm.color} onChange={(event) => setTagForm({ ...tagForm, color: event.target.value })} /></label>
            <button type="submit" disabled={createTagMutation.isPending}>{t("common.create")}</button>
          </form>
          <div className="tag-list">
            {tags.map((tag) => <TagBadge key={tag.id} tag={tag} />)}
          </div>
          {!tagsQuery.isLoading && tags.length === 0 ? <p className="empty-state">{t("agents.emptyTags")}</p> : null}
          {tagsQuery.isError ? <p className="form-error">{String(tagsQuery.error.message)}</p> : null}
          {createTagMutation.isError ? <p className="form-error">{String(createTagMutation.error.message)}</p> : null}

          <form
            className="form-grid"
            onSubmit={(event) => {
              event.preventDefault();
              setResourceTagsMutation.mutate();
            }}
          >
            <label>{t("agents.resourceType")}<select value={tagAssignment.resourceType} onChange={(event) => setTagAssignment({ resourceType: event.target.value, resourceId: "", tagIds: [] })}><option value="agent">{t("agents.agent")}</option><option value="host">{t("agents.host")}</option></select></label>
            <label>{t("agents.resource")}<select value={tagAssignment.resourceId} onChange={(event) => setTagAssignment({ ...tagAssignment, resourceId: event.target.value })} required><option value="">{t("agents.resource")}</option>{(tagAssignment.resourceType === "host" ? hosts : agents).map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
            <label>{t("agents.tags")}<select multiple value={tagAssignment.tagIds} onChange={(event) => setTagAssignment({ ...tagAssignment, tagIds: selectedValues(event.currentTarget) })}>{tags.map((tag) => <option key={tag.id} value={tag.id}>{tag.name}</option>)}</select></label>
            <button type="submit" disabled={setResourceTagsMutation.isPending || !tagAssignment.resourceId}>{t("agents.assignTags")}</button>
          </form>
          {setResourceTagsMutation.isError ? <p className="form-error">{String(setResourceTagsMutation.error.message)}</p> : null}
        </section>

        <section className="panel table-panel">
          <div className="panel-title">
            <h3>{t("agents.hostGroups")}</h3>
            <span>{hostGroups.length} {t("common.total")}</span>
          </div>
          <form
            className="form-grid"
            onSubmit={(event) => {
              event.preventDefault();
              const payload = { ...hostGroupForm };
              if (editingHostGroupId) updateHostGroupMutation.mutate(payload);
              else createHostGroupMutation.mutate(payload);
            }}
          >
            <label>{t("common.name")}<input value={hostGroupForm.name} onChange={(event) => setHostGroupForm({ ...hostGroupForm, name: event.target.value })} required /></label>
            <label>{t("common.description")}<input value={hostGroupForm.description} onChange={(event) => setHostGroupForm({ ...hostGroupForm, description: event.target.value })} /></label>
            <label>{t("agents.hosts")}<select multiple value={hostGroupForm.hostIds} onChange={(event) => setHostGroupForm({ ...hostGroupForm, hostIds: selectedValues(event.currentTarget) })}>{hosts.map((host) => <option key={host.id} value={host.id}>{host.name}</option>)}</select></label>
            <button type="submit" disabled={createHostGroupMutation.isPending || updateHostGroupMutation.isPending}>{editingHostGroupId ? t("common.save") : t("common.create")}</button>
            {editingHostGroupId ? <button type="button" onClick={resetHostGroupForm}>{t("common.cancel")}</button> : null}
          </form>
          <div className="inline-form">
            <select value={selectedBatchGroupId} onChange={(event) => setSelectedBatchGroupId(event.target.value)}>
              <option value="">{t("agents.hostGroups")}</option>
              {hostGroups.map((group) => <option key={group.id} value={group.id}>{group.name}</option>)}
            </select>
            <button type="button" disabled={!selectedBatchGroupId || disableHostGroupAgentsMutation.isPending} onClick={() => disableHostGroupAgentsMutation.mutate(selectedBatchGroupId)}>
              {t("agents.disable")}
            </button>
          </div>
          {disableHostGroupAgentsMutation.data ? <p className="empty-state">{disableHostGroupAgentsMutation.data.groupName}: {disableHostGroupAgentsMutation.data.affectedAgents} Agents / {disableHostGroupAgentsMutation.data.affectedHosts} Hosts</p> : null}
          {selectedBatchGroupId ? (
            <div className="data-table">
              <table>
                <thead><tr><th>{t("agents.agent")}</th><th>{t("agents.host")}</th><th>{t("common.version")}</th><th>{t("agents.reportedAt")}</th></tr></thead>
                <tbody>
                  {hostGroupDiagnostics.map((item) => (
                    <tr key={item.id}>
                      <td><strong>{item.agentName}</strong><small>{item.agentId}</small></td>
                      <td><strong>{item.hostName || "-"}</strong><small>{item.hostId || item.ip || "-"}</small></td>
                      <td><strong>{item.version || "-"}</strong><small>{[item.os, item.arch].filter(Boolean).join(" / ") || "-"}</small></td>
                      <td>{item.reportedAt}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : null}
          <div className="data-table">
            <table>
              <thead><tr><th>{t("common.name")}</th><th>{t("agents.hosts")}</th><th>{t("common.createdBy")}</th><th>{t("common.action")}</th></tr></thead>
              <tbody>
                {hostGroups.map((group) => (
                  <tr key={group.id}>
                    <td><strong>{group.name}</strong><small>{group.description || group.id}</small></td>
                    <td><strong>{group.hostCount}</strong><small>{(group.hosts ?? []).map((host) => host.name).join(", ") || "-"}</small></td>
                    <td>{group.createdBy || "-"}</td>
                    <td><button type="button" onClick={() => startEditingHostGroup(group)}>{t("common.edit")}</button></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {!hostGroupsQuery.isLoading && hostGroups.length === 0 ? <p className="empty-state">{t("agents.emptyHostGroups")}</p> : null}
          {hostGroupsQuery.isError ? <p className="form-error">{String(hostGroupsQuery.error.message)}</p> : null}
          {hostGroupDiagnosticsQuery.isError ? <p className="form-error">{String(hostGroupDiagnosticsQuery.error.message)}</p> : null}
          {createHostGroupMutation.isError ? <p className="form-error">{String(createHostGroupMutation.error.message)}</p> : null}
          {updateHostGroupMutation.isError ? <p className="form-error">{String(updateHostGroupMutation.error.message)}</p> : null}
          {disableHostGroupAgentsMutation.isError ? <p className="form-error">{String(disableHostGroupAgentsMutation.error.message)}</p> : null}
        </section>
      </section>

      <section className="agent-grid">
        <section className="panel table-panel">
          <div className="panel-title">
            <h3>{t("agents.diagnostics")}</h3>
            <span>{diagnostics.length} {t("common.total")}</span>
          </div>
          <div className="data-table">
            <table>
              <thead>
                <tr>
                  <th>{t("agents.agent")}</th>
                  <th>{t("agents.host")}</th>
                  <th>{t("common.version")}</th>
                  <th>{t("common.system")}</th>
                  <th>{t("agents.runningTasks")}</th>
                  <th>{t("agents.reportedAt")}</th>
                </tr>
              </thead>
              <tbody>
                {diagnostics.map((item) => (
                  <tr key={item.id}>
                    <td><strong>{item.agentName}</strong><small>{item.agentId}</small></td>
                    <td><strong>{item.hostName || "-"}</strong><small>{item.ip || item.hostId || "-"}</small></td>
                    <td>{item.version || "-"}</td>
                    <td><strong>{item.os || "-"}</strong><small>{[item.osVersion, item.arch].filter(Boolean).join(" / ") || "-"}</small></td>
                    <td>{item.runningTasks ?? 0}</td>
                    <td>{item.reportedAt}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {!diagnosticsQuery.isLoading && diagnostics.length === 0 ? <p className="empty-state">{t("agents.emptyDiagnostics")}</p> : null}
          {diagnosticsQuery.isError ? <p className="form-error">{String(diagnosticsQuery.error.message)}</p> : null}
        </section>

        <section className="panel table-panel">
          <div className="panel-title">
            <h3>{t("agents.maintenance")}</h3>
            <span>{maintenanceWindows.length} {t("common.total")}</span>
          </div>
          <form
            className="form-grid"
            onSubmit={(event) => {
              event.preventDefault();
              const payload = { ...maintenanceForm };
              if (editingMaintenanceId) updateMaintenanceMutation.mutate(payload);
              else createMaintenanceMutation.mutate(payload);
            }}
          >
            <label>{t("common.name")}<input value={maintenanceForm.name} onChange={(event) => setMaintenanceForm({ ...maintenanceForm, name: event.target.value })} required /></label>
            <label>{t("agents.scope")}<select value={maintenanceForm.scopeType} onChange={(event) => setMaintenanceForm({ ...maintenanceForm, scopeType: event.target.value })}><option value="all">{t("agents.allFleet")}</option><option value="agent">{t("agents.agent")}</option><option value="host">{t("agents.host")}</option><option value="group">{t("agents.hostGroups")}</option></select></label>
            {maintenanceForm.scopeType === "agent" ? <label>{t("agents.agent")}<select value={maintenanceForm.agentId} onChange={(event) => setMaintenanceForm({ ...maintenanceForm, agentId: event.target.value })} required><option value="">{t("agents.agent")}</option>{agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}</select></label> : null}
            {maintenanceForm.scopeType === "host" ? <label>{t("agents.host")}<select value={maintenanceForm.hostId} onChange={(event) => setMaintenanceForm({ ...maintenanceForm, hostId: event.target.value })} required><option value="">{t("agents.host")}</option>{hosts.map((host) => <option key={host.id} value={host.id}>{host.name}</option>)}</select></label> : null}
            {maintenanceForm.scopeType === "group" ? <label>{t("agents.hostGroups")}<select value={maintenanceForm.hostGroupId} onChange={(event) => setMaintenanceForm({ ...maintenanceForm, hostGroupId: event.target.value })} required><option value="">{t("agents.hostGroups")}</option>{hostGroups.map((group) => <option key={group.id} value={group.id}>{group.name}</option>)}</select></label> : null}
            <label>{t("common.reason")}<input value={maintenanceForm.reason} onChange={(event) => setMaintenanceForm({ ...maintenanceForm, reason: event.target.value })} /></label>
            <label>{t("agents.startsAt")}<input value={maintenanceForm.startsAt} onChange={(event) => setMaintenanceForm({ ...maintenanceForm, startsAt: event.target.value })} placeholder="2026-05-08 10:00:00" required /></label>
            <label>{t("agents.endsAt")}<input value={maintenanceForm.endsAt} onChange={(event) => setMaintenanceForm({ ...maintenanceForm, endsAt: event.target.value })} placeholder="2026-05-08 12:00:00" required /></label>
            <label>{t("common.status")}<select value={maintenanceForm.status} onChange={(event) => setMaintenanceForm({ ...maintenanceForm, status: event.target.value })}><option value="active">{t("common.status.active")}</option><option value="disabled">{t("common.status.disabled")}</option><option value="archived">{t("common.status.archived")}</option></select></label>
            <button type="submit" disabled={createMaintenanceMutation.isPending || updateMaintenanceMutation.isPending}>{editingMaintenanceId ? t("common.save") : t("common.create")}</button>
            {editingMaintenanceId ? <button type="button" onClick={resetMaintenanceForm}>{t("common.cancel")}</button> : null}
          </form>
          <div className="data-table">
            <table>
              <thead><tr><th>{t("common.name")}</th><th>{t("agents.scope")}</th><th>{t("agents.window")}</th><th>{t("common.status")}</th><th>{t("common.action")}</th></tr></thead>
              <tbody>
                {maintenanceWindows.map((item) => (
                  <tr key={item.id}>
                    <td><strong>{item.name}</strong><small>{item.reason || item.id}</small></td>
                    <td><strong>{item.scopeType}</strong><small>{item.agentName || item.hostName || item.hostGroupName || "-"}</small></td>
                    <td><strong>{item.startsAt}</strong><small>{item.endsAt}</small></td>
                    <td><StatusChip status={item.status} /></td>
                    <td><button type="button" onClick={() => startEditingMaintenance(item)}>{t("common.edit")}</button></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {!maintenanceQuery.isLoading && maintenanceWindows.length === 0 ? <p className="empty-state">{t("agents.emptyMaintenance")}</p> : null}
          {maintenanceQuery.isError ? <p className="form-error">{String(maintenanceQuery.error.message)}</p> : null}
          {createMaintenanceMutation.isError ? <p className="form-error">{String(createMaintenanceMutation.error.message)}</p> : null}
          {updateMaintenanceMutation.isError ? <p className="form-error">{String(updateMaintenanceMutation.error.message)}</p> : null}
        </section>
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
                      <TagRow tags={agent.tags ?? []} />
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
                      <TagRow tags={host.tags ?? []} />
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
                {createEnrollmentMutation.isPending ? t("agents.creating") : t("agents.createToken")}
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
                  <th>{t("common.status")}</th>
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

function TagRow({ tags }: { tags: TagSummary[] }) {
  if (tags.length === 0) return null;
  return <span className="tag-row">{tags.map((tag) => <TagBadge key={tag.id} tag={tag} />)}</span>;
}

function TagBadge({ tag }: { tag: TagSummary }) {
  return <span className="tag-badge" style={tag.color ? { borderColor: tag.color, color: tag.color } : undefined}>{tag.name}</span>;
}

function selectedValues(select: HTMLSelectElement) {
  return Array.from(select.selectedOptions).map((option) => option.value);
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
