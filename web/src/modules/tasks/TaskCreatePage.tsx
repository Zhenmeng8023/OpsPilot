import { FormEvent, useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";

import { request } from "../../api/request";
import { listScripts } from "../../api/scripts";
import { createTask } from "../../api/tasks";
import type { Agent, Host, PageResult, Script } from "../../api/types";

export function TaskCreatePage() {
  const navigate = useNavigate();
  const [mode, setMode] = useState<"script" | "command">("script");
  const [form, setForm] = useState({
    name: "",
    description: "",
    scriptId: "",
    command: "",
    scriptType: "shell",
    timeoutSeconds: 60,
    targetAgentIds: [] as string[],
    targetHostIds: [] as string[]
  });
  const scriptsQuery = useQuery({ queryKey: ["scripts", "active"], queryFn: () => listScripts({ status: "active", pageSize: 100 }) });
  const agentsQuery = useQuery({ queryKey: ["agents", "task-create"], queryFn: () => request<PageResult<Agent>>("/api/v1/agents?pageSize=100") });
  const hostsQuery = useQuery({ queryKey: ["hosts", "task-create"], queryFn: () => request<PageResult<Host>>("/api/v1/hosts?pageSize=100") });
  const selectedScript = useMemo(
    () => (scriptsQuery.data?.items ?? []).find((script) => script.id === form.scriptId),
    [scriptsQuery.data, form.scriptId]
  );
  const scriptApprovalBlocked =
    mode === "script" &&
    Boolean(selectedScript?.approvalRequired) &&
    selectedScript?.approvalStatus !== "approved";
  const createMutation = useMutation({
    mutationFn: () =>
      createTask({
        name: form.name,
        description: form.description || undefined,
        scriptId: mode === "script" ? form.scriptId : undefined,
        command: mode === "command" ? form.command : undefined,
        scriptType: form.scriptType,
        timeoutSeconds: form.timeoutSeconds,
        targetAgentIds: form.targetAgentIds,
        targetHostIds: form.targetHostIds
      }),
    onSuccess: (task) => navigate(`/tasks/${task.id}`)
  });
  const hasTargets = form.targetAgentIds.length + form.targetHostIds.length > 0;
  const isValid =
    form.name.trim() &&
    form.timeoutSeconds > 0 &&
    hasTargets &&
    (mode === "script" ? form.scriptId : form.command.trim()) &&
    !scriptApprovalBlocked;

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (isValid) createMutation.mutate();
  }

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">Execution</p>
          <h1>Create Task</h1>
        </div>
      </section>
      <form className="panel editor-form" onSubmit={submit}>
        <div className="form-grid">
          <label>Name<input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required /></label>
          <label>Timeout seconds<input type="number" min={1} value={form.timeoutSeconds} onChange={(event) => setForm({ ...form, timeoutSeconds: Number(event.target.value) })} required /></label>
        </div>
        <label>Description<input value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} /></label>
        <div className="segmented">
          <button type="button" className={mode === "script" ? "active" : ""} onClick={() => setMode("script")}>Script</button>
          <button type="button" className={mode === "command" ? "active" : ""} onClick={() => setMode("command")}>Command</button>
        </div>
        {mode === "script" ? (
          <label>
            Script template
            <select value={form.scriptId} onChange={(event) => setForm({ ...form, scriptId: event.target.value })}>
              <option value="">Select script</option>
              {(scriptsQuery.data?.items ?? []).map((script) => (
                <option key={script.id} value={script.id}>
                  {script.name} v{script.version}
                  {script.approvalRequired ? ` [${script.approvalStatus || "unapproved"}]` : ""}
                </option>
              ))}
            </select>
          </label>
        ) : (
          <>
            <label>Command type<select value={form.scriptType} onChange={(event) => setForm({ ...form, scriptType: event.target.value })}><option value="shell">shell</option><option value="bash">bash</option><option value="powershell">powershell</option></select></label>
            <label>Command<textarea className="code-input compact" value={form.command} onChange={(event) => setForm({ ...form, command: event.target.value })} /></label>
          </>
        )}
        <section className="target-picker">
          <div>
            <h3>Agents</h3>
            {(agentsQuery.data?.items ?? []).map((agent) => (
              <label key={agent.id}>
                <input type="checkbox" checked={form.targetAgentIds.includes(agent.id)} onChange={() => toggleValue(form.targetAgentIds, agent.id, (next) => setForm({ ...form, targetAgentIds: next }))} />
                <span>{agent.name} <small>{agent.status}</small></span>
              </label>
            ))}
          </div>
          <div>
            <h3>Hosts</h3>
            {(hostsQuery.data?.items ?? []).map((host) => (
              <label key={host.id}>
                <input type="checkbox" checked={form.targetHostIds.includes(host.id)} onChange={() => toggleValue(form.targetHostIds, host.id, (next) => setForm({ ...form, targetHostIds: next }))} />
                <span>{host.name} <small>{host.onlineAgentCount ?? 0} online</small></span>
              </label>
            ))}
          </div>
        </section>
        {form.timeoutSeconds <= 0 ? <p className="form-error">Timeout must be greater than 0.</p> : null}
        {!hasTargets ? <p className="form-error">Select at least one target Agent or Host.</p> : null}
        {scriptApprovalBlocked ? (
          <p className="form-error">{approvalBlockedMessage(selectedScript)}</p>
        ) : null}
        {createMutation.isError ? <p className="form-error">{createMutation.error.message}</p> : null}
        <button className="ghost-button" type="submit" disabled={createMutation.isPending || !isValid}>Create task</button>
      </form>
    </main>
  );
}

function toggleValue(values: string[], value: string, setValues: (values: string[]) => void) {
  setValues(values.includes(value) ? values.filter((item) => item !== value) : [...values, value]);
}

function approvalBlockedMessage(script?: Script) {
  const status = script?.approvalStatus || "unapproved";
  if (status === "pending") {
    return "Selected script is awaiting approval and cannot be executed yet.";
  }
  if (status === "rejected" || status === "canceled") {
    return "Selected script approval is not accepted. Please update and re-request approval.";
  }
  return "Selected script requires approval before execution.";
}
