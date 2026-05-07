import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { createAlertRule, listAlertRules, listAlerts, resolveAlert } from "../../api/alerts";
import { listHostMetrics } from "../../api/metrics";
import { hasPermission } from "../auth/permissions";
import { useAuthStore } from "../auth/store";

export function MetricsPage() {
  const user = useAuthStore((state) => state.user);
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
          <p className="eyebrow">Monitoring</p>
          <h1>Host Metrics</h1>
        </div>
      </section>
      {canWriteAlerts ? (
        <section className="panel form-panel">
          <div className="panel-title"><h3>Create Alert Rule</h3><span>metric threshold</span></div>
          <form
            className="form-grid"
            onSubmit={(event) => {
              event.preventDefault();
              createRuleMutation.mutate(ruleForm);
            }}
          >
            <label>Name<input value={ruleForm.name} onChange={(event) => setRuleForm({ ...ruleForm, name: event.target.value })} required /></label>
            <label>
              Metric
              <select value={ruleForm.metricCode} onChange={(event) => setRuleForm({ ...ruleForm, metricCode: event.target.value })}>
                <option value="agent.running_tasks">agent.running_tasks</option>
                <option value="agent.cpu.logical">agent.cpu.logical</option>
                <option value="agent.runtime.goroutines">agent.runtime.goroutines</option>
                <option value="agent.runtime.alloc_bytes">agent.runtime.alloc_bytes</option>
                <option value="agent.runtime.sys_bytes">agent.runtime.sys_bytes</option>
              </select>
            </label>
            <label>
              Operator
              <select value={ruleForm.operator} onChange={(event) => setRuleForm({ ...ruleForm, operator: event.target.value })}>
                <option value=">">&gt;</option>
                <option value=">=">&gt;=</option>
                <option value="<">&lt;</option>
                <option value="<=">&lt;=</option>
                <option value="==">==</option>
                <option value="!=">!=</option>
              </select>
            </label>
            <label>Threshold<input type="number" value={ruleForm.threshold} onChange={(event) => setRuleForm({ ...ruleForm, threshold: Number(event.target.value) })} /></label>
            <label>Duration<input type="number" value={ruleForm.durationSeconds} onChange={(event) => setRuleForm({ ...ruleForm, durationSeconds: Number(event.target.value) })} /></label>
            <label>
              Severity
              <select value={ruleForm.severity} onChange={(event) => setRuleForm({ ...ruleForm, severity: event.target.value })}>
                <option value="info">info</option>
                <option value="warning">warning</option>
                <option value="critical">critical</option>
              </select>
            </label>
            <button type="submit" disabled={createRuleMutation.isPending}>Create rule</button>
          </form>
          {createRuleMutation.isError ? <p className="form-error">{createRuleMutation.error.message}</p> : null}
        </section>
      ) : null}
      {canReadAlerts ? (
        <section className="panel table-panel">
          <div className="panel-title"><h3>Firing Alerts</h3><span>{alerts.length} firing</span></div>
          <div className="data-table">
            <table>
              <thead><tr><th>Alert</th><th>Severity</th><th>Rule</th><th>Last seen</th><th>Action</th></tr></thead>
              <tbody>
                {alerts.map((alert) => (
                  <tr key={alert.id}>
                    <td><strong>{alert.title}</strong><small>{alert.message || alert.id}</small></td>
                    <td><span className={`status-chip status-${alert.severity}`}>{alert.severity}</span></td>
                    <td>{alert.ruleName || "-"}</td>
                    <td>{alert.lastSeenAt}</td>
                    <td className="action-cell">
                      {canWriteAlerts ? <button type="button" onClick={() => resolveMutation.mutate(alert.id)}>Resolve</button> : null}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {!alertsQuery.isLoading && alerts.length === 0 ? <p className="empty-state">No firing alerts.</p> : null}
          {alertsQuery.isError ? <p className="form-error">{alertsQuery.error.message}</p> : null}
          {resolveMutation.isError ? <p className="form-error">{resolveMutation.error.message}</p> : null}
        </section>
      ) : null}
      {canReadAlerts ? (
        <section className="panel table-panel">
          <div className="panel-title"><h3>Alert Rules</h3><span>{rules.length} total</span></div>
          <div className="data-table">
            <table>
              <thead><tr><th>Name</th><th>Metric</th><th>Condition</th><th>Severity</th><th>Status</th></tr></thead>
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
          {!rulesQuery.isLoading && rules.length === 0 ? <p className="empty-state">No alert rules found.</p> : null}
          {rulesQuery.isError ? <p className="form-error">{rulesQuery.error.message}</p> : null}
        </section>
      ) : null}
      <section className="panel table-panel">
        <div className="toolbar-row">
          <select value={metricCode} onChange={(event) => setMetricCode(event.target.value)}>
            <option value="">All metrics</option>
            <option value="agent.running_tasks">agent.running_tasks</option>
            <option value="agent.cpu.logical">agent.cpu.logical</option>
            <option value="agent.runtime.goroutines">agent.runtime.goroutines</option>
            <option value="agent.runtime.alloc_bytes">agent.runtime.alloc_bytes</option>
            <option value="agent.runtime.sys_bytes">agent.runtime.sys_bytes</option>
          </select>
          <button type="button" onClick={() => metricsQuery.refetch()}>Refresh</button>
        </div>
        <div className="data-table">
          <table>
            <thead><tr><th>Host</th><th>Agent</th><th>Metric</th><th>Value</th><th>Collected</th></tr></thead>
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
        {!metricsQuery.isLoading && latest.length === 0 ? <p className="empty-state">No metrics found.</p> : null}
        {metricsQuery.isError ? <p className="form-error">{metricsQuery.error.message}</p> : null}
      </section>
    </main>
  );
}
