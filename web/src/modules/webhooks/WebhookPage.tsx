import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { createWebhookRule, createWebhookSource, listWebhookRules, listWebhookSources } from "../../api/webhooks";
import { listTasks } from "../../api/tasks";
import { API_BASE_URL } from "../../api/request";
import { useLanguageStore } from "../../i18n/language";
import { hasPermission } from "../auth/permissions";
import { useAuthStore } from "../auth/store";

export function WebhookPage() {
  const user = useAuthStore((state) => state.user);
  const t = useLanguageStore((state) => state.t);
  const queryClient = useQueryClient();
  const canManage = hasPermission(user, "webhook:manage");
  const [sourceForm, setSourceForm] = useState({ name: "", sourceType: "custom" });
  const [ruleForm, setRuleForm] = useState({ sourceId: "", taskId: "", name: "", eventType: "" });
  const [issuedToken, setIssuedToken] = useState("");
  const sourcesQuery = useQuery({ queryKey: ["webhookSources"], queryFn: listWebhookSources });
  const rulesQuery = useQuery({ queryKey: ["webhookRules"], queryFn: listWebhookRules });
  const tasksQuery = useQuery({ queryKey: ["tasks", "webhook-options"], queryFn: () => listTasks({ pageSize: 100 }) });
  const createSourceMutation = useMutation({
    mutationFn: createWebhookSource,
    onSuccess: (source) => {
      setIssuedToken(source.token ?? "");
      setSourceForm({ name: "", sourceType: "custom" });
      queryClient.invalidateQueries({ queryKey: ["webhookSources"] });
    }
  });
  const createRuleMutation = useMutation({
    mutationFn: createWebhookRule,
    onSuccess: () => {
      setRuleForm((current) => ({ ...current, name: "" }));
      queryClient.invalidateQueries({ queryKey: ["webhookRules"] });
    }
  });
  const sources = useMemo(() => sourcesQuery.data ?? [], [sourcesQuery.data]);
  const rules = useMemo(() => rulesQuery.data ?? [], [rulesQuery.data]);
  const taskOptions = useMemo(() => {
    const map = new Map<string, NonNullable<typeof tasksQuery.data>["items"][number]>();
    for (const task of tasksQuery.data?.items ?? []) {
      if (!map.has(task.taskId)) {
        map.set(task.taskId, task);
      }
    }
    return Array.from(map.values());
  }, [tasksQuery.data]);
  const triggerURL = issuedToken ? `${API_BASE_URL || window.location.origin}/api/v1/webhooks/trigger/${issuedToken}` : "";

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">{t("automation.eyebrow")}</p>
          <h1>{t("webhooks.title")}</h1>
        </div>
      </section>

      {canManage ? (
        <section className="panel form-panel">
          <div className="panel-title"><h3>{t("webhooks.createSource")}</h3><span>{t("webhooks.sourceHint")}</span></div>
          <form className="form-grid" onSubmit={(event) => { event.preventDefault(); createSourceMutation.mutate(sourceForm); }}>
            <label>{t("common.name")}<input value={sourceForm.name} onChange={(event) => setSourceForm({ ...sourceForm, name: event.target.value })} required /></label>
            <label>
              {t("common.type")}
              <select value={sourceForm.sourceType} onChange={(event) => setSourceForm({ ...sourceForm, sourceType: event.target.value })}>
                <option value="custom">custom</option>
                <option value="github">github</option>
                <option value="gitlab">gitlab</option>
                <option value="gitee">gitee</option>
              </select>
            </label>
            <button type="submit" disabled={createSourceMutation.isPending}>{t("webhooks.createSourceAction")}</button>
          </form>
          {triggerURL ? <p className="empty-state">{triggerURL}</p> : null}
          {createSourceMutation.isError ? <p className="form-error">{createSourceMutation.error.message}</p> : null}
        </section>
      ) : null}

      {canManage ? (
        <section className="panel form-panel">
          <div className="panel-title"><h3>{t("webhooks.createRule")}</h3><span>{t("webhooks.ruleHint")}</span></div>
          <form className="form-grid" onSubmit={(event) => { event.preventDefault(); createRuleMutation.mutate(ruleForm); }}>
            <label>
              {t("common.source")}
              <select value={ruleForm.sourceId} onChange={(event) => setRuleForm({ ...ruleForm, sourceId: event.target.value })} required>
                <option value="">{t("webhooks.selectSource")}</option>
                {sources.map((source) => <option key={source.id} value={source.id}>{source.name}</option>)}
              </select>
            </label>
            <label>
              {t("common.task")}
              <select value={ruleForm.taskId} onChange={(event) => setRuleForm({ ...ruleForm, taskId: event.target.value })} required>
                <option value="">{t("webhooks.selectTask")}</option>
                {taskOptions.map((task) => <option key={task.taskId} value={task.taskId}>{task.name}</option>)}
              </select>
            </label>
            <label>{t("common.name")}<input value={ruleForm.name} onChange={(event) => setRuleForm({ ...ruleForm, name: event.target.value })} required /></label>
            <label>{t("webhooks.eventType")}<input value={ruleForm.eventType} onChange={(event) => setRuleForm({ ...ruleForm, eventType: event.target.value })} /></label>
            <button type="submit" disabled={createRuleMutation.isPending}>{t("webhooks.createRuleAction")}</button>
          </form>
          {createRuleMutation.isError ? <p className="form-error">{createRuleMutation.error.message}</p> : null}
        </section>
      ) : null}

      <section className="panel table-panel">
        <div className="panel-title"><h3>{t("webhooks.rules")}</h3><span>{rules.length} {t("common.total")}</span></div>
        <div className="data-table">
          <table>
            <thead><tr><th>{t("common.name")}</th><th>{t("common.source")}</th><th>{t("common.task")}</th><th>{t("webhooks.event")}</th><th>{t("common.status")}</th><th>{t("common.created")}</th></tr></thead>
            <tbody>
              {rules.map((rule) => (
                <tr key={rule.id}>
                  <td><strong>{rule.name}</strong><small>{rule.id}</small></td>
                  <td>{rule.sourceName}</td>
                  <td><strong>{rule.taskName}</strong><small>{rule.taskId}</small></td>
                  <td>{rule.eventType || "*"}</td>
                  <td><span className={`status-chip status-${rule.status}`}>{rule.status}</span></td>
                  <td>{rule.createdAt}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!rulesQuery.isLoading && rules.length === 0 ? <p className="empty-state">{t("webhooks.emptyRules")}</p> : null}
      </section>
    </main>
  );
}
