import { FormEvent, useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";

import { request } from "../../api/request";
import { listScripts } from "../../api/scripts";
import { createTask } from "../../api/tasks";
import type { Agent, Host, PageResult, Script } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";

export function TaskCreatePage() {
  const navigate = useNavigate();
  const t = useLanguageStore((state) => state.t);
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
          <p className="eyebrow">{t("tasks.eyebrow")}</p>
          <h1>{t("tasks.createTitle")}</h1>
        </div>
      </section>
      <form className="panel editor-form" onSubmit={submit}>
        <div className="form-grid">
          <label>{t("common.name")}<input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required /></label>
          <label>{t("tasks.timeoutSeconds")}<input type="number" min={1} value={form.timeoutSeconds} onChange={(event) => setForm({ ...form, timeoutSeconds: Number(event.target.value) })} required /></label>
        </div>
        <label>{t("tasks.description")}<input value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} /></label>
        <div className="segmented">
          <button type="button" className={mode === "script" ? "active" : ""} onClick={() => setMode("script")}>{t("tasks.modeScript")}</button>
          <button type="button" className={mode === "command" ? "active" : ""} onClick={() => setMode("command")}>{t("tasks.modeCommand")}</button>
        </div>
        {mode === "script" ? (
          <label>
            {t("tasks.scriptTemplate")}
            <select value={form.scriptId} onChange={(event) => setForm({ ...form, scriptId: event.target.value })}>
              <option value="">{t("tasks.selectScript")}</option>
              {(scriptsQuery.data?.items ?? []).map((script) => (
                <option key={script.id} value={script.id}>
                  {script.name} v{script.version}
                  {script.approvalRequired ? ` [${t(`common.status.${script.approvalStatus || "required"}`)}]` : ""}
                </option>
              ))}
            </select>
          </label>
        ) : (
          <>
            <label>{t("tasks.commandType")}<select value={form.scriptType} onChange={(event) => setForm({ ...form, scriptType: event.target.value })}><option value="shell">shell</option><option value="bash">bash</option><option value="powershell">powershell</option></select></label>
            <label>{t("tasks.command")}<textarea className="code-input compact" value={form.command} onChange={(event) => setForm({ ...form, command: event.target.value })} /></label>
          </>
        )}
        <section className="target-picker">
          <div>
            <h3>{t("tasks.agents")}</h3>
            {(agentsQuery.data?.items ?? []).map((agent) => (
              <label key={agent.id}>
                <input type="checkbox" checked={form.targetAgentIds.includes(agent.id)} onChange={() => toggleValue(form.targetAgentIds, agent.id, (next) => setForm({ ...form, targetAgentIds: next }))} />
                <span>{agent.name} <small>{agent.status}</small></span>
              </label>
            ))}
          </div>
          <div>
            <h3>{t("tasks.hosts")}</h3>
            {(hostsQuery.data?.items ?? []).map((host) => (
              <label key={host.id}>
                <input type="checkbox" checked={form.targetHostIds.includes(host.id)} onChange={() => toggleValue(form.targetHostIds, host.id, (next) => setForm({ ...form, targetHostIds: next }))} />
                <span>{host.name} <small>{host.onlineAgentCount ?? 0} {t("common.online")}</small></span>
              </label>
            ))}
          </div>
        </section>
        {form.timeoutSeconds <= 0 ? <p className="form-error">{t("tasks.timeoutPositive")}</p> : null}
        {!hasTargets ? <p className="form-error">{t("tasks.selectTarget")}</p> : null}
        {scriptApprovalBlocked ? (
          <p className="form-error">{approvalBlockedMessage(t, selectedScript)}</p>
        ) : null}
        {createMutation.isError ? <p className="form-error">{createMutation.error.message}</p> : null}
        <button className="ghost-button" type="submit" disabled={createMutation.isPending || !isValid}>{t("tasks.createAction")}</button>
      </form>
    </main>
  );
}

function toggleValue(values: string[], value: string, setValues: (values: string[]) => void) {
  setValues(values.includes(value) ? values.filter((item) => item !== value) : [...values, value]);
}

function approvalBlockedMessage(t: (key: string) => string, script?: Script) {
  const status = script?.approvalStatus || "unapproved";
  if (status === "pending") {
    return t("tasks.approvalPending");
  }
  if (status === "rejected" || status === "canceled") {
    return t("tasks.approvalRejected");
  }
  return t("tasks.approvalRequired");
}
