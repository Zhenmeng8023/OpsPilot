import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { createAlertRule, listAlertRules, listAlerts, resolveAlert } from "../../api/alerts";
import { listHostMetrics } from "../../api/metrics";
import { useLanguageStore } from "../../i18n/language";
import { hasPermission } from "../auth/permissions";
import { useAuthStore } from "../auth/store";

export function MetricsPage() {
  const user = useAuthStore((state) => state.user);
  const t = useLanguageStore((state) => state.t);
  const queryClient = useQueryClient();
  const [metricCode, setMetricCode] = useState("");
  const [ruleForm, setRuleForm] = useState({
    name: "",
    metricCode: "agent.runtime.goroutines",
    operator: ">",
    threshold: 100,
    durationSeconds: 60,
    severity: "warning"
  });
  const canReadAlerts = hasPermission(user, "alert:read");
  const canWriteAlerts = hasPermission(user, "alert:write");
  const metricsQuery = useQuery({
    queryKey: ["hostMetrics", metricCode],
    queryFn: () => listHostMetrics({ metricCode, limit: 300 })
  });
  const rulesQuery = useQuery({
    queryKey: ["alertRules"],
    queryFn: listAlertRules,
    enabled: canReadAlerts
  });
  const alertsQuery = useQuery({
    queryKey: ["alerts", "firing"],
    queryFn: () => listAlerts({ status: "firing" }),
    enabled: canReadAlerts
  });
  const createRuleMutation = useMutation({
    mutationFn: createAlertRule,
    onSuccess: () => {
      setRuleForm((current) => ({ ...current, name: "" }));
      queryClient.invalidateQueries({ queryKey: ["alertRules"] });
    }
  });
  const resolveMutation = useMutation({
    mutationFn: (id: string) => resolveAlert(id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["alerts"] })
  });
  const metrics = useMemo(() => metricsQuery.data ?? [], [metricsQuery.data]);
  const rules = useMemo(() => rulesQuery.data ?? [], [rulesQuery.data]);
  const alerts = useMemo(() => alertsQuery.data ?? [], [alertsQuery.data]);
  const latest = useMemo(() => {
    const map = new Map<string, (typeof metrics)[number]>();
    for (const metric of metrics) {
      const key = `${metric.hostId}:${metric.metricCode}`;
      if (!map.has(key)) map.set(key, metric);
    }
    return Array.from(map.values());
  }, [metrics]);

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">{t("monitoring.eyebrow")}</p>
          <h1>{t("metrics.title")}</h1>
        </div>
      </section>
      {canWriteAlerts ? (
        <section className="panel form-panel">
          <div className="panel-title"><h3>{t("metrics.createRule")}</h3><span>{t("metrics.ruleHint")}</span></div>
          <form
            className="form-grid"
            onSubmit={(event) => {
              event.preventDefault();
              createRuleMutation.mutate(ruleForm);
            }}
          >
            <label>{t("common.name")}<input value={ruleForm.name} onChange={(event) => setRuleForm({ ...ruleForm, name: event.target.value })} required /></label>
            <label>
              {t("metrics.metric")}
              <select value={ruleForm.metricCode} onChange={(event) => setRuleForm({ ...ruleForm, metricCode: event.target.value })}>
                <option value="agent.running_tasks">agent.running_tasks</option>
                <option value="agent.cpu.logical">agent.cpu.logical</option>
                <option value="agent.runtime.goroutines">agent.runtime.goroutines</option>
                <option value="agent.runtime.alloc_bytes">agent.runtime.alloc_bytes</option>
                <option value="agent.runtime.sys_bytes">agent.runtime.sys_bytes</option>
              </select>
            </label>
            <label>
              {t("metrics.operator")}
              <select value={ruleForm.operator} onChange={(event) => setRuleForm({ ...ruleForm, operator: event.target.value })}>
                <option value=">">&gt;</option>
                <option value=">=">&gt;=</option>
                <option value="<">&lt;</option>
                <option value="<=">&lt;=</option>
                <option value="==">==</option>
                <option value="!=">!=</option>
              </select>
            </label>
            <label>{t("metrics.threshold")}<input type="number" value={ruleForm.threshold} onChange={(event) => setRuleForm({ ...ruleForm, threshold: Number(event.target.value) })} /></label>
            <label>{t("metrics.duration")}<input type="number" value={ruleForm.durationSeconds} onChange={(event) => setRuleForm({ ...ruleForm, durationSeconds: Number(event.target.value) })} /></label>
            <label>
              {t("common.severity")}
              <select value={ruleForm.severity} onChange={(event) => setRuleForm({ ...ruleForm, severity: event.target.value })}>
                <option value="info">info</option>
                <option value="warning">warning</option>
                <option value="critical">critical</option>
              </select>
            </label>
            <button type="submit" disabled={createRuleMutation.isPending}>{t("metrics.createRuleAction")}</button>
          </form>
          {createRuleMutation.isError ? <p className="form-error">{createRuleMutation.error.message}</p> : null}
        </section>
      ) : null}
      {canReadAlerts ? (
        <section className="panel table-panel">
          <div className="panel-title"><h3>{t("metrics.firingAlerts")}</h3><span>{t("metrics.firingCount").replace("{count}", String(alerts.length))}</span></div>
          <div className="data-table">
            <table>
              <thead><tr><th>{t("metrics.alert")}</th><th>{t("common.severity")}</th><th>{t("metrics.rule")}</th><th>{t("metrics.lastSeen")}</th><th>{t("common.action")}</th></tr></thead>
              <tbody>
                {alerts.map((alert) => (
                  <tr key={alert.id}>
                    <td><strong>{alert.title}</strong><small>{alert.message || alert.id}</small></td>
                    <td><span className={`status-chip status-${alert.severity}`}>{alert.severity}</span></td>
                    <td>{alert.ruleName || "-"}</td>
                    <td>{alert.lastSeenAt}</td>
                    <td className="action-cell">
                      {canWriteAlerts ? <button type="button" onClick={() => resolveMutation.mutate(alert.id)}>{t("metrics.resolve")}</button> : null}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {!alertsQuery.isLoading && alerts.length === 0 ? <p className="empty-state">{t("metrics.emptyAlerts")}</p> : null}
          {alertsQuery.isError ? <p className="form-error">{alertsQuery.error.message}</p> : null}
          {resolveMutation.isError ? <p className="form-error">{resolveMutation.error.message}</p> : null}
        </section>
      ) : null}
      {canReadAlerts ? (
        <section className="panel table-panel">
          <div className="panel-title"><h3>{t("metrics.alertRules")}</h3><span>{rules.length} {t("common.total")}</span></div>
          <div className="data-table">
            <table>
              <thead><tr><th>{t("common.name")}</th><th>{t("metrics.metric")}</th><th>{t("metrics.condition")}</th><th>{t("common.severity")}</th><th>{t("common.status")}</th></tr></thead>
              <tbody>
                {rules.map((rule) => (
                  <tr key={rule.id}>
                    <td><strong>{rule.name}</strong><small>{rule.id}</small></td>
                    <td>{rule.metricCode}</td>
                    <td>{rule.operator} {rule.threshold}</td>
                    <td>{rule.severity}</td>
                    <td><span className={`status-chip status-${rule.status}`}>{rule.status}</span></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {!rulesQuery.isLoading && rules.length === 0 ? <p className="empty-state">{t("metrics.emptyRules")}</p> : null}
          {rulesQuery.isError ? <p className="form-error">{rulesQuery.error.message}</p> : null}
        </section>
      ) : null}
      <section className="panel table-panel">
        <div className="toolbar-row">
          <select value={metricCode} onChange={(event) => setMetricCode(event.target.value)}>
            <option value="">{t("common.allMetrics")}</option>
            <option value="agent.running_tasks">agent.running_tasks</option>
            <option value="agent.cpu.logical">agent.cpu.logical</option>
            <option value="agent.runtime.goroutines">agent.runtime.goroutines</option>
            <option value="agent.runtime.alloc_bytes">agent.runtime.alloc_bytes</option>
            <option value="agent.runtime.sys_bytes">agent.runtime.sys_bytes</option>
          </select>
          <button type="button" onClick={() => metricsQuery.refetch()}>{t("common.refresh")}</button>
        </div>
        <div className="data-table">
          <table>
            <thead><tr><th>{t("metrics.host")}</th><th>{t("metrics.agent")}</th><th>{t("metrics.metric")}</th><th>{t("metrics.value")}</th><th>{t("metrics.collected")}</th></tr></thead>
            <tbody>
              {latest.map((metric) => (
                <tr key={`${metric.hostId}:${metric.metricCode}`}>
                  <td><strong>{metric.hostName}</strong><small>{metric.hostId}</small></td>
                  <td>{metric.agentName || "-"}</td>
                  <td>{metric.metricCode}</td>
                  <td><strong>{metric.value}</strong> <small>{metric.unit || ""}</small></td>
                  <td>{metric.collectedAt}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!metricsQuery.isLoading && latest.length === 0 ? <p className="empty-state">{t("metrics.emptyMetrics")}</p> : null}
        {metricsQuery.isError ? <p className="form-error">{metricsQuery.error.message}</p> : null}
      </section>
    </main>
  );
}
