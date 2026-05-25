import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { acknowledgeAlert, createAlertRoutingPolicy, createAlertRule, createAlertSuppressionRule, disableAlertRule, listAlertEvents, listAlertGroups, listAlertHistory, listAlertRoutingPolicies, listAlertRules, listAlertSuppressionRules, listAlerts, pauseAlertRule, resolveAlert, resumeAlertRule, silenceAlert, unsilenceAlert, updateAlertRoutingPolicy, updateAlertRule, updateAlertSuppressionRule } from "../../api/alerts";
import { listHostGroups } from "../../api/agents";
import { createMetricDashboard, listHostMetrics, listMetricDashboards, listMetricTrends, runMetricRetention, runMetricRollup, updateMetricDashboard } from "../../api/metrics";
import { listNotificationChannels } from "../../api/notifications";
import type { AlertGroupSummary, AlertHistoryPoint, AlertRoutingPolicy, AlertSuppressionRule, MetricDashboard, MetricRetentionResult, MetricRollupResult, MetricTrendSeries } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";
import { hasPermission } from "../auth/permissions";
import { useAuthStore } from "../auth/store";
import { alertEventTypes, alertHistoryRangeOptions, alertRuleTemplates, metricCatalog, overviewMetrics, quickTrendMetrics, trendRangeOptions } from "./catalog";
import { AlertNoiseGovernancePanel } from "./AlertNoiseGovernancePanel";

export function MetricsPage() {
  const user = useAuthStore((state) => state.user);
  const t = useLanguageStore((state) => state.t);
  const queryClient = useQueryClient();
  const [tab, setTab] = useState<"metrics" | "alerts" | "rules" | "ops">("metrics");
  const [metricCode, setMetricCode] = useState("");
  const [trendHours, setTrendHours] = useState(24);
  const [trendGranularity, setTrendGranularity] = useState("auto");
  const [selectedTrendSeries, setSelectedTrendSeries] = useState("");
  const [dashboardForm, setDashboardForm] = useState({
    name: "",
    metricCode: "agent.os.cpu.percent",
    hostId: "",
    agentId: "",
    rangeHours: 24,
    pointLimit: 120,
    granularity: "auto",
    status: "active"
  });
  const [editingDashboardId, setEditingDashboardId] = useState("");
  const [rollupForm, setRollupForm] = useState({ interval: "5m", hours: 24 });
  const [retentionForm, setRetentionForm] = useState({ detailDays: 7, rollupDays: 90, dryRun: true });
  const [rollupResult, setRollupResult] = useState<MetricRollupResult | null>(null);
  const [retentionResult, setRetentionResult] = useState<MetricRetentionResult | null>(null);
  const [suppressionForm, setSuppressionForm] = useState({ name: "", ruleId: "", hostId: "", hostGroupId: "", severity: "", startsAt: "", endsAt: "", reason: "", status: "active" });
  const [routingForm, setRoutingForm] = useState({ name: "", ruleId: "", hostId: "", hostGroupId: "", severity: "", channelId: "", status: "active" });
  const [editingSuppressionId, setEditingSuppressionId] = useState("");
  const [editingRoutingId, setEditingRoutingId] = useState("");
  const [ruleForm, setRuleForm] = useState({
    name: "",
    metricCode: "agent.os.cpu.percent",
    operator: ">",
    threshold: 85,
    durationSeconds: 60,
    cooldownSeconds: 300,
    severity: "warning"
  });
  const [editingRuleId, setEditingRuleId] = useState("");
  const [alertStatus, setAlertStatus] = useState("");
  const [alertSeverity, setAlertSeverity] = useState("");
  const [alertRuleId, setAlertRuleId] = useState("");
  const [alertHostId, setAlertHostId] = useState("");
  const [alertGroupHostGroupId, setAlertGroupHostGroupId] = useState("");
  const [alertHistoryHours, setAlertHistoryHours] = useState(24);
  const [alertEventType, setAlertEventType] = useState("");
  const [silenceForm, setSilenceForm] = useState({ durationSeconds: 3600, reason: "" });
  const [expandedAlertId, setExpandedAlertId] = useState("");
  const canReadAlerts = hasPermission(user, "alert:read");
  const canWriteAlerts = hasPermission(user, "alert:write");
  const canReadHosts = hasPermission(user, "host:read");
  const canWriteMetrics = hasPermission(user, "metric:write");

  const metricsQuery = useQuery({
    queryKey: ["hostMetrics", metricCode],
    queryFn: () => listHostMetrics({ metricCode, limit: 300 })
  });
  const trendQuery = useQuery({
    queryKey: ["metricTrends", metricCode, trendHours, trendGranularity],
    queryFn: () => listMetricTrends({ metricCode, hours: trendHours, limit: 120, granularity: trendGranularity }),
    enabled: metricCode !== ""
  });
  const dashboardsQuery = useQuery({
    queryKey: ["metricDashboards"],
    queryFn: listMetricDashboards
  });
  const rulesQuery = useQuery({
    queryKey: ["alertRules"],
    queryFn: listAlertRules,
    enabled: canReadAlerts
  });
  const alertsQuery = useQuery({
    queryKey: ["alerts", alertStatus, alertSeverity, alertRuleId, alertHostId],
    queryFn: () => listAlerts({ status: alertStatus, severity: alertSeverity, ruleId: alertRuleId, hostId: alertHostId }),
    enabled: canReadAlerts
  });
  const alertGroupsQuery = useQuery({
    queryKey: ["alertGroups", alertStatus, alertSeverity, alertRuleId, alertGroupHostGroupId],
    queryFn: () => listAlertGroups({
      status: alertStatus,
      severity: alertSeverity,
      ruleId: alertRuleId,
      hostGroupId: alertGroupHostGroupId
    }),
    enabled: canReadAlerts
  });
  const alertHistoryQuery = useQuery({
    queryKey: ["alertHistory", alertHistoryHours, alertSeverity, alertRuleId, alertHostId],
    queryFn: () => listAlertHistory({
      hours: alertHistoryHours,
      bucketMinutes: alertHistoryHours <= 24 ? 30 : 60,
      severity: alertSeverity,
      ruleId: alertRuleId,
      hostId: alertHostId
    }),
    enabled: canReadAlerts
  });
  const eventsQuery = useQuery({
    queryKey: ["alertEvents", expandedAlertId, alertEventType],
    queryFn: () => listAlertEvents(expandedAlertId, { eventType: alertEventType }),
    enabled: canReadAlerts && expandedAlertId !== ""
  });
  const suppressionRulesQuery = useQuery({
    queryKey: ["alertSuppressionRules"],
    queryFn: listAlertSuppressionRules,
    enabled: canReadAlerts
  });
  const routingPoliciesQuery = useQuery({
    queryKey: ["alertRoutingPolicies"],
    queryFn: listAlertRoutingPolicies,
    enabled: canReadAlerts
  });
  const channelsQuery = useQuery({
    queryKey: ["notificationChannels"],
    queryFn: listNotificationChannels,
    enabled: canWriteAlerts
  });
  const hostGroupsQuery = useQuery({
    queryKey: ["hostGroups"],
    queryFn: listHostGroups,
    enabled: canReadAlerts && canReadHosts
  });

  const createRuleMutation = useMutation({
    mutationFn: createAlertRule,
    onSuccess: () => {
      resetRuleForm();
      queryClient.invalidateQueries({ queryKey: ["alertRules"] });
    }
  });
  const updateRuleMutation = useMutation({
    mutationFn: (payload: typeof ruleForm) => updateAlertRule(editingRuleId, payload),
    onSuccess: () => {
      resetRuleForm();
      queryClient.invalidateQueries({ queryKey: ["alertRules"] });
    }
  });
  const resolveMutation = useMutation({
    mutationFn: (id: string) => resolveAlert(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alerts"] });
      queryClient.invalidateQueries({ queryKey: ["alertEvents"] });
    }
  });
  const acknowledgeMutation = useMutation({
    mutationFn: (id: string) => acknowledgeAlert(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alerts"] });
      queryClient.invalidateQueries({ queryKey: ["alertEvents"] });
    }
  });
  const silenceMutation = useMutation({
    mutationFn: (id: string) => silenceAlert(id, silenceForm),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alerts"] });
      queryClient.invalidateQueries({ queryKey: ["alertEvents"] });
    }
  });
  const unsilenceMutation = useMutation({
    mutationFn: (id: string) => unsilenceAlert(id),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["alerts"] });
      queryClient.invalidateQueries({ queryKey: ["alertEvents"] });
    }
  });
  const ruleStatusMutation = useMutation({
    mutationFn: ({ id, action }: { id: string; action: "pause" | "resume" | "disable" }) => {
      switch (action) {
        case "pause":
          return pauseAlertRule(id);
        case "resume":
          return resumeAlertRule(id);
        default:
          return disableAlertRule(id);
      }
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["alertRules"] })
  });
  const createSuppressionMutation = useMutation({
    mutationFn: createAlertSuppressionRule,
    onSuccess: () => {
      resetSuppressionForm();
      queryClient.invalidateQueries({ queryKey: ["alertSuppressionRules"] });
    }
  });
  const updateSuppressionMutation = useMutation({
    mutationFn: (payload: typeof suppressionForm) => updateAlertSuppressionRule(editingSuppressionId, payload),
    onSuccess: () => {
      resetSuppressionForm();
      queryClient.invalidateQueries({ queryKey: ["alertSuppressionRules"] });
    }
  });
  const createRoutingMutation = useMutation({
    mutationFn: createAlertRoutingPolicy,
    onSuccess: () => {
      resetRoutingForm();
      queryClient.invalidateQueries({ queryKey: ["alertRoutingPolicies"] });
    }
  });
  const updateRoutingMutation = useMutation({
    mutationFn: (payload: typeof routingForm) => updateAlertRoutingPolicy(editingRoutingId, payload),
    onSuccess: () => {
      resetRoutingForm();
      queryClient.invalidateQueries({ queryKey: ["alertRoutingPolicies"] });
    }
  });
  const createDashboardMutation = useMutation({
    mutationFn: createMetricDashboard,
    onSuccess: () => {
      resetDashboardForm();
      queryClient.invalidateQueries({ queryKey: ["metricDashboards"] });
    }
  });
  const updateDashboardMutation = useMutation({
    mutationFn: (payload: typeof dashboardForm) => updateMetricDashboard(editingDashboardId, payload),
    onSuccess: () => {
      resetDashboardForm();
      queryClient.invalidateQueries({ queryKey: ["metricDashboards"] });
    }
  });
  const rollupMutation = useMutation({
    mutationFn: runMetricRollup,
    onSuccess: (result) => {
      setRollupResult(result);
      queryClient.invalidateQueries({ queryKey: ["metricTrends"] });
    }
  });
  const retentionMutation = useMutation({
    mutationFn: runMetricRetention,
    onSuccess: (result) => {
      setRetentionResult(result);
      queryClient.invalidateQueries({ queryKey: ["hostMetrics"] });
      queryClient.invalidateQueries({ queryKey: ["metricTrends"] });
    }
  });

  const metrics = useMemo(() => metricsQuery.data ?? [], [metricsQuery.data]);
  const trendSeries = useMemo(() => trendQuery.data ?? [], [trendQuery.data]);
  const dashboards = useMemo(() => dashboardsQuery.data ?? [], [dashboardsQuery.data]);
  const rules = useMemo(() => rulesQuery.data ?? [], [rulesQuery.data]);
  const alerts = useMemo(() => alertsQuery.data ?? [], [alertsQuery.data]);
  const alertGroups = useMemo(() => alertGroupsQuery.data ?? [], [alertGroupsQuery.data]);
  const alertHistory = useMemo(() => alertHistoryQuery.data ?? [], [alertHistoryQuery.data]);
  const events = useMemo(() => eventsQuery.data ?? [], [eventsQuery.data]);
  const suppressionRules = useMemo(() => suppressionRulesQuery.data ?? [], [suppressionRulesQuery.data]);
  const routingPolicies = useMemo(() => routingPoliciesQuery.data ?? [], [routingPoliciesQuery.data]);
  const channels = useMemo(() => channelsQuery.data ?? [], [channelsQuery.data]);
  const hostGroups = useMemo(() => hostGroupsQuery.data ?? [], [hostGroupsQuery.data]);

  const latest = useMemo(() => {
    const map = new Map<string, (typeof metrics)[number]>();
    for (const metric of metrics) {
      const key = `${metric.hostId}:${metric.metricCode}`;
      if (!map.has(key)) map.set(key, metric);
    }
    return Array.from(map.values());
  }, [metrics]);
  const overviewHosts = useMemo(() => buildHostOverview(latest), [latest]);

  useEffect(() => {
    if (expandedAlertId === "") {
      setAlertEventType("");
    }
  }, [expandedAlertId]);

  useEffect(() => {
    if (trendSeries.length === 0) {
      if (selectedTrendSeries !== "") {
        setSelectedTrendSeries("");
      }
      return;
    }
    if (!trendSeries.some((series) => trendSeriesKey(series) === selectedTrendSeries)) {
      setSelectedTrendSeries(trendSeriesKey(trendSeries[0]));
    }
  }, [selectedTrendSeries, trendSeries]);

  const activeTrend = useMemo(
    () => trendSeries.find((series) => trendSeriesKey(series) === selectedTrendSeries) ?? trendSeries[0],
    [selectedTrendSeries, trendSeries]
  );
  const alertSummary = useMemo(() => summarizeAlerts(alerts), [alerts]);

  const resetRuleForm = () => {
    setRuleForm({
      name: "",
      metricCode: "agent.os.cpu.percent",
      operator: ">",
      threshold: 85,
      durationSeconds: 60,
      cooldownSeconds: 300,
      severity: "warning"
    });
    setEditingRuleId("");
  };

  const resetDashboardForm = () => {
    setDashboardForm({
      name: "",
      metricCode: "agent.os.cpu.percent",
      hostId: "",
      agentId: "",
      rangeHours: 24,
      pointLimit: 120,
      granularity: "auto",
      status: "active"
    });
    setEditingDashboardId("");
  };

  const startEditingDashboard = (dashboard: MetricDashboard) => {
    setEditingDashboardId(dashboard.id);
    setDashboardForm({
      name: dashboard.name,
      metricCode: dashboard.metricCode || "agent.os.cpu.percent",
      hostId: dashboard.hostId || "",
      agentId: dashboard.agentId || "",
      rangeHours: dashboard.rangeHours,
      pointLimit: dashboard.pointLimit,
      granularity: dashboard.granularity || "auto",
      status: dashboard.status || "active"
    });
  };

  const applyDashboard = (dashboard: MetricDashboard) => {
    setMetricCode(dashboard.metricCode || "");
    setTrendHours(dashboard.rangeHours || 24);
    setTrendGranularity(dashboard.granularity || "auto");
    setSelectedTrendSeries(dashboard.hostId && dashboard.metricCode ? `${dashboard.hostId}:${dashboard.metricCode}` : "");
  };

  const resetSuppressionForm = () => {
    setSuppressionForm({ name: "", ruleId: "", hostId: "", hostGroupId: "", severity: "", startsAt: "", endsAt: "", reason: "", status: "active" });
    setEditingSuppressionId("");
  };

  const resetRoutingForm = () => {
    setRoutingForm({ name: "", ruleId: "", hostId: "", hostGroupId: "", severity: "", channelId: "", status: "active" });
    setEditingRoutingId("");
  };

  const startEditingSuppression = (item: AlertSuppressionRule) => {
    setEditingSuppressionId(item.id);
    setSuppressionForm({
      name: item.name,
      ruleId: item.ruleId || "",
      hostId: item.hostId || "",
      hostGroupId: item.hostGroupId || "",
      severity: item.severity || "",
      startsAt: item.startsAt || "",
      endsAt: item.endsAt || "",
      reason: item.reason || "",
      status: item.status || "active"
    });
  };

  const startEditingRouting = (item: AlertRoutingPolicy) => {
    setEditingRoutingId(item.id);
    setRoutingForm({
      name: item.name,
      ruleId: item.ruleId || "",
      hostId: item.hostId || "",
      hostGroupId: item.hostGroupId || "",
      severity: item.severity || "",
      channelId: item.channelId,
      status: item.status || "active"
    });
  };

  const alertStatusLabel = (status: string) => {
    switch (status) {
      case "firing":
        return t("common.status.firing");
      case "acknowledged":
        return t("common.status.acknowledged");
      case "silenced":
        return t("common.status.silenced");
      case "resolved":
        return t("common.status.resolved");
      case "active":
        return t("common.status.active");
      case "paused":
        return t("common.status.paused");
      case "disabled":
        return t("common.status.disabled");
      default:
        return status;
    }
  };

  const alertEventLabel = (eventType: string) => {
    switch (eventType) {
      case "firing":
        return t("metrics.eventFiring");
      case "acknowledged":
        return t("metrics.eventAcknowledged");
      case "silenced":
        return t("metrics.eventSilenced");
      case "unsilenced":
        return t("metrics.eventUnsilenced");
      case "resolved":
        return t("metrics.eventResolved");
      case "cooldown_suppressed":
        return t("metrics.eventCooldownSuppressed");
      default:
        return eventType;
    }
  };

  const openTrend = (hostId: string, code: string) => {
    setMetricCode(code);
    setSelectedTrendSeries(`${hostId}:${code}`);
  };

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">{t("monitoring.eyebrow")}</p>
          <h1>{t("metrics.title")}</h1>
        </div>
        <div className="segmented">
          <button type="button" className={tab === "metrics" ? "active" : ""} onClick={() => setTab("metrics")}>
            {t("metrics.overview")}
          </button>
          {canReadAlerts ? (
            <button type="button" className={tab === "alerts" ? "active" : ""} onClick={() => setTab("alerts")}>
              {t("metrics.alertActivity")}
            </button>
          ) : null}
          {canReadAlerts ? (
            <button type="button" className={tab === "rules" ? "active" : ""} onClick={() => setTab("rules")}>
              {t("metrics.alertRules")}
            </button>
          ) : null}
          {canWriteMetrics ? (
            <button type="button" className={tab === "ops" ? "active" : ""} onClick={() => setTab("ops")}>
              {t("metrics.lifecycle")}
            </button>
          ) : null}
        </div>
      </section>

      {tab === "metrics" ? (
      <section className="panel">
        <div className="panel-title">
          <h3>{t("metrics.savedDashboards")}</h3>
          <span>{t("metrics.savedDashboardsHint")}</span>
        </div>
        {canWriteMetrics ? (
          <form
            className="inline-form"
            onSubmit={(event) => {
              event.preventDefault();
              const payload = {
                ...dashboardForm,
                hostId: dashboardForm.hostId.trim(),
                agentId: dashboardForm.agentId.trim()
              };
              if (editingDashboardId) {
                updateDashboardMutation.mutate(payload);
              } else {
                createDashboardMutation.mutate(payload);
              }
            }}
          >
            <input value={dashboardForm.name} onChange={(event) => setDashboardForm({ ...dashboardForm, name: event.target.value })} placeholder={t("common.name")} />
            <select value={dashboardForm.metricCode} onChange={(event) => setDashboardForm({ ...dashboardForm, metricCode: event.target.value })}>
              {metricCatalog.map((option) => <option key={option.code} value={option.code}>{option.code}</option>)}
            </select>
            <input value={dashboardForm.hostId} onChange={(event) => setDashboardForm({ ...dashboardForm, hostId: event.target.value })} placeholder={t("metrics.host")} />
            <select value={String(dashboardForm.rangeHours)} onChange={(event) => setDashboardForm({ ...dashboardForm, rangeHours: Number(event.target.value) })}>
              {trendRangeOptions.map((hours) => (
                <option key={hours} value={hours}>{t("metrics.rangeHours").replace("{hours}", String(hours))}</option>
              ))}
            </select>
            <select value={dashboardForm.granularity} onChange={(event) => setDashboardForm({ ...dashboardForm, granularity: event.target.value })}>
              <option value="auto">{t("metrics.granularityAuto")}</option>
              <option value="raw">{t("metrics.granularityRaw")}</option>
              <option value="5m">5m</option>
              <option value="1h">1h</option>
            </select>
            <select value={dashboardForm.status} onChange={(event) => setDashboardForm({ ...dashboardForm, status: event.target.value })}>
              <option value="active">{t("common.status.active")}</option>
              <option value="disabled">{t("common.status.disabled")}</option>
              <option value="archived">{t("common.status.archived")}</option>
            </select>
            <button type="submit" disabled={createDashboardMutation.isPending || updateDashboardMutation.isPending}>{editingDashboardId ? t("common.save") : t("metrics.saveDashboard")}</button>
            {editingDashboardId ? <button type="button" onClick={resetDashboardForm}>{t("common.cancel")}</button> : null}
          </form>
        ) : null}
        <div className="data-table">
          <table>
            <thead>
              <tr>
                <th>{t("common.name")}</th>
                <th>{t("metrics.metric")}</th>
                <th>{t("metrics.range")}</th>
                <th>{t("metrics.granularity")}</th>
                <th>{t("common.status")}</th>
                <th>{t("common.action")}</th>
              </tr>
            </thead>
            <tbody>
              {dashboards.map((dashboard) => (
                <tr key={dashboard.id}>
                  <td><strong>{dashboard.name}</strong><small>{dashboard.id}</small></td>
                  <td>{dashboard.metricCode || "-"}</td>
                  <td>{t("metrics.rangeHours").replace("{hours}", String(dashboard.rangeHours))}</td>
                  <td>{dashboard.granularity}</td>
                  <td>{alertStatusLabel(dashboard.status)}</td>
                  <td className="action-cell">
                    <button type="button" onClick={() => applyDashboard(dashboard)}>{t("metrics.applyDashboard")}</button>
                    {canWriteMetrics ? <button type="button" onClick={() => startEditingDashboard(dashboard)}>{t("common.edit")}</button> : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!dashboardsQuery.isLoading && dashboards.length === 0 ? <p className="empty-state">{t("metrics.emptyDashboards")}</p> : null}
        {dashboardsQuery.isError ? <p className="form-error">{dashboardsQuery.error.message}</p> : null}
        {createDashboardMutation.isError ? <p className="form-error">{createDashboardMutation.error.message}</p> : null}
        {updateDashboardMutation.isError ? <p className="form-error">{updateDashboardMutation.error.message}</p> : null}
      </section>
      ) : null}

      {tab === "ops" && canWriteMetrics ? (
        <section className="panel">
          <div className="panel-title">
            <h3>{t("metrics.lifecycle")}</h3>
            <span>{t("metrics.lifecycleHint")}</span>
          </div>
          <div className="inline-form">
            <select value={rollupForm.interval} onChange={(event) => setRollupForm({ ...rollupForm, interval: event.target.value })}>
              <option value="5m">5m</option>
              <option value="1h">1h</option>
            </select>
            <input type="number" value={rollupForm.hours} onChange={(event) => setRollupForm({ ...rollupForm, hours: Number(event.target.value) })} />
            <button type="button" disabled={rollupMutation.isPending} onClick={() => rollupMutation.mutate(rollupForm)}>{t("metrics.runRollup")}</button>
            <input type="number" value={retentionForm.detailDays} onChange={(event) => setRetentionForm({ ...retentionForm, detailDays: Number(event.target.value) })} />
            <input type="number" value={retentionForm.rollupDays} onChange={(event) => setRetentionForm({ ...retentionForm, rollupDays: Number(event.target.value) })} />
            <label className="checkbox-label">
              <input type="checkbox" checked={retentionForm.dryRun} onChange={(event) => setRetentionForm({ ...retentionForm, dryRun: event.target.checked })} />
              {t("metrics.dryRun")}
            </label>
            <button type="button" disabled={retentionMutation.isPending} onClick={() => retentionMutation.mutate(retentionForm)}>{t("metrics.runRetention")}</button>
          </div>
          {rollupResult ? <p className="empty-state">{t("metrics.rollupResult").replace("{matched}", String(rollupResult.matched)).replace("{upserted}", String(rollupResult.upserted))}</p> : null}
          {retentionResult ? <p className="empty-state">{t("metrics.retentionResult").replace("{detail}", String(retentionResult.detailMatched)).replace("{rollup}", String(retentionResult.rollupMatched)).replace("{deleted}", String(retentionResult.detailDeleted + retentionResult.rollupDeleted))}</p> : null}
          {rollupMutation.isError ? <p className="form-error">{rollupMutation.error.message}</p> : null}
          {retentionMutation.isError ? <p className="form-error">{retentionMutation.error.message}</p> : null}
        </section>
      ) : null}

      {tab === "rules" && canWriteAlerts ? (
        <section className="panel form-panel">
          <div className="panel-title">
            <h3>{t("metrics.createRule")}</h3>
            <span>{t("metrics.ruleHint")}</span>
          </div>
          <div className="template-grid">
            {alertRuleTemplates.map((template) => (
              <button
                key={template.key}
                type="button"
                className="template-card"
                onClick={() => {
                  setEditingRuleId("");
                  setRuleForm({ ...template.payload });
                }}
              >
                <strong>{t(template.titleKey)}</strong>
                <span>{t(template.descriptionKey)}</span>
              </button>
            ))}
          </div>
          <form
            className="form-grid"
            onSubmit={(event) => {
              event.preventDefault();
              if (editingRuleId) {
                updateRuleMutation.mutate(ruleForm);
              } else {
                createRuleMutation.mutate(ruleForm);
              }
            }}
          >
            <label>{t("common.name")}<input value={ruleForm.name} onChange={(event) => setRuleForm({ ...ruleForm, name: event.target.value })} required /></label>
            <label>
              {t("metrics.metric")}
              <select value={ruleForm.metricCode} onChange={(event) => setRuleForm({ ...ruleForm, metricCode: event.target.value })}>
                {metricCatalog.map((option) => <option key={option.code} value={option.code}>{option.code}</option>)}
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
            <label>{t("metrics.cooldown")}<input type="number" value={ruleForm.cooldownSeconds} onChange={(event) => setRuleForm({ ...ruleForm, cooldownSeconds: Number(event.target.value) })} /></label>
            <label>
              {t("common.severity")}
              <select value={ruleForm.severity} onChange={(event) => setRuleForm({ ...ruleForm, severity: event.target.value })}>
                <option value="info">info</option>
                <option value="warning">warning</option>
                <option value="critical">critical</option>
              </select>
            </label>
            <button type="submit" disabled={createRuleMutation.isPending || updateRuleMutation.isPending}>
              {editingRuleId ? t("common.save") : t("metrics.createRuleAction")}
            </button>
            {editingRuleId ? <button type="button" onClick={resetRuleForm}>{t("common.cancel")}</button> : null}
          </form>
          {createRuleMutation.isError ? <p className="form-error">{createRuleMutation.error.message}</p> : null}
          {updateRuleMutation.isError ? <p className="form-error">{updateRuleMutation.error.message}</p> : null}
        </section>
      ) : null}

      {tab === "rules" && canReadAlerts ? (
        <section className="panel">
          <div className="panel-title">
            <h3>{t("metrics.alertOperations")}</h3>
            <span>{t("metrics.alertOperationsHint")}</span>
          </div>
          {canWriteAlerts ? (
            <div className="split-grid">
              <form className="form-grid" onSubmit={(event) => {
                event.preventDefault();
                const payload = { ...suppressionForm };
                if (editingSuppressionId) updateSuppressionMutation.mutate(payload);
                else createSuppressionMutation.mutate(payload);
              }}>
                <label>{t("metrics.suppressionRules")}<input value={suppressionForm.name} onChange={(event) => setSuppressionForm({ ...suppressionForm, name: event.target.value })} required /></label>
                <label>{t("metrics.rule")}<select value={suppressionForm.ruleId} onChange={(event) => setSuppressionForm({ ...suppressionForm, ruleId: event.target.value })}><option value="">{t("metrics.allRules")}</option>{rules.map((rule) => <option key={rule.id} value={rule.id}>{rule.name}</option>)}</select></label>
                <label>{t("metrics.host")}<input value={suppressionForm.hostId} onChange={(event) => setSuppressionForm({ ...suppressionForm, hostId: event.target.value })} /></label>
                <label>{t("agents.hostGroups")}<select value={suppressionForm.hostGroupId} onChange={(event) => setSuppressionForm({ ...suppressionForm, hostGroupId: event.target.value })}><option value="">{t("agents.hostGroups")}</option>{hostGroups.map((group) => <option key={group.id} value={group.id}>{group.name}</option>)}</select></label>
                <label>{t("common.severity")}<select value={suppressionForm.severity} onChange={(event) => setSuppressionForm({ ...suppressionForm, severity: event.target.value })}><option value="">{t("metrics.allSeverities")}</option><option value="info">info</option><option value="warning">warning</option><option value="critical">critical</option></select></label>
                <label>{t("metrics.reason")}<input value={suppressionForm.reason} onChange={(event) => setSuppressionForm({ ...suppressionForm, reason: event.target.value })} /></label>
                <label>{t("common.status")}<select value={suppressionForm.status} onChange={(event) => setSuppressionForm({ ...suppressionForm, status: event.target.value })}><option value="active">{t("common.status.active")}</option><option value="disabled">{t("common.status.disabled")}</option><option value="archived">{t("common.status.archived")}</option></select></label>
                <button type="submit" disabled={createSuppressionMutation.isPending || updateSuppressionMutation.isPending}>{editingSuppressionId ? t("common.save") : t("common.create")}</button>
                {editingSuppressionId ? <button type="button" onClick={resetSuppressionForm}>{t("common.cancel")}</button> : null}
              </form>
              <form className="form-grid" onSubmit={(event) => {
                event.preventDefault();
                const payload = { ...routingForm };
                if (editingRoutingId) updateRoutingMutation.mutate(payload);
                else createRoutingMutation.mutate(payload);
              }}>
                <label>{t("metrics.routingPolicies")}<input value={routingForm.name} onChange={(event) => setRoutingForm({ ...routingForm, name: event.target.value })} required /></label>
                <label>{t("metrics.rule")}<select value={routingForm.ruleId} onChange={(event) => setRoutingForm({ ...routingForm, ruleId: event.target.value })}><option value="">{t("metrics.allRules")}</option>{rules.map((rule) => <option key={rule.id} value={rule.id}>{rule.name}</option>)}</select></label>
                <label>{t("metrics.host")}<input value={routingForm.hostId} onChange={(event) => setRoutingForm({ ...routingForm, hostId: event.target.value })} /></label>
                <label>{t("agents.hostGroups")}<select value={routingForm.hostGroupId} onChange={(event) => setRoutingForm({ ...routingForm, hostGroupId: event.target.value })}><option value="">{t("agents.hostGroups")}</option>{hostGroups.map((group) => <option key={group.id} value={group.id}>{group.name}</option>)}</select></label>
                <label>{t("common.severity")}<select value={routingForm.severity} onChange={(event) => setRoutingForm({ ...routingForm, severity: event.target.value })}><option value="">{t("metrics.allSeverities")}</option><option value="info">info</option><option value="warning">warning</option><option value="critical">critical</option></select></label>
                <label>{t("notifications.channel")}<select value={routingForm.channelId} onChange={(event) => setRoutingForm({ ...routingForm, channelId: event.target.value })} required><option value="">{t("notifications.channel")}</option>{channels.map((channel) => <option key={channel.id} value={channel.id}>{channel.name}</option>)}</select></label>
                <label>{t("common.status")}<select value={routingForm.status} onChange={(event) => setRoutingForm({ ...routingForm, status: event.target.value })}><option value="active">{t("common.status.active")}</option><option value="disabled">{t("common.status.disabled")}</option><option value="archived">{t("common.status.archived")}</option></select></label>
                <button type="submit" disabled={createRoutingMutation.isPending || updateRoutingMutation.isPending}>{editingRoutingId ? t("common.save") : t("common.create")}</button>
                {editingRoutingId ? <button type="button" onClick={resetRoutingForm}>{t("common.cancel")}</button> : null}
              </form>
            </div>
          ) : null}
          <div className="split-grid">
            <div className="data-table"><table><thead><tr><th>{t("common.name")}</th><th>{t("common.severity")}</th><th>{t("metrics.rule")}</th><th>{t("common.status")}</th><th>{t("common.action")}</th></tr></thead><tbody>{suppressionRules.map((item) => <tr key={item.id}><td><strong>{item.name}</strong><small>{item.reason || item.hostGroupId || item.hostId || item.id}</small></td><td>{item.severity || "-"}</td><td>{item.ruleId || "-"}</td><td>{alertStatusLabel(item.status)}</td><td className="action-cell">{canWriteAlerts ? <button type="button" onClick={() => startEditingSuppression(item)}>{t("common.edit")}</button> : null}</td></tr>)}</tbody></table></div>
            <div className="data-table"><table><thead><tr><th>{t("common.name")}</th><th>{t("common.severity")}</th><th>{t("notifications.channel")}</th><th>{t("common.status")}</th><th>{t("common.action")}</th></tr></thead><tbody>{routingPolicies.map((item) => <tr key={item.id}><td><strong>{item.name}</strong><small>{item.ruleId || item.hostGroupId || item.hostId || item.id}</small></td><td>{item.severity || "-"}</td><td>{item.channelId}</td><td>{alertStatusLabel(item.status)}</td><td className="action-cell">{canWriteAlerts ? <button type="button" onClick={() => startEditingRouting(item)}>{t("common.edit")}</button> : null}</td></tr>)}</tbody></table></div>
          </div>
          {suppressionRulesQuery.isError ? <p className="form-error">{suppressionRulesQuery.error.message}</p> : null}
          {routingPoliciesQuery.isError ? <p className="form-error">{routingPoliciesQuery.error.message}</p> : null}
          {createSuppressionMutation.isError ? <p className="form-error">{createSuppressionMutation.error.message}</p> : null}
          {updateSuppressionMutation.isError ? <p className="form-error">{updateSuppressionMutation.error.message}</p> : null}
          {createRoutingMutation.isError ? <p className="form-error">{createRoutingMutation.error.message}</p> : null}
          {updateRoutingMutation.isError ? <p className="form-error">{updateRoutingMutation.error.message}</p> : null}
        </section>
      ) : null}

      {tab === "metrics" ? (
      <section className="panel">
        <div className="panel-title">
          <h3>{t("metrics.overview")}</h3>
          <span>{overviewHosts.length} {t("common.items")}</span>
        </div>
        {overviewHosts.length === 0 ? <p className="empty-state">{t("metrics.emptyMetrics")}</p> : null}
        {overviewHosts.length > 0 ? (
          <div className="overview-grid">
            {overviewHosts.map((host) => (
              <section key={host.hostId} className="overview-card">
                <div className="overview-card-header">
                  <div>
                    <strong>{host.hostName}</strong>
                    <small>{host.hostId}</small>
                  </div>
                  <button type="button" onClick={() => openTrend(host.hostId, "agent.os.cpu.percent")}>{t("metrics.viewTrend")}</button>
                </div>
                <div className="overview-stat-list">
                  {overviewMetrics.map((metricCode) => {
                    const metric = host.metrics[metricCode];
                    return (
                      <div key={metricCode} className="overview-stat">
                        <div className="overview-stat-top">
                          <span>{metricLabel(metricCode, t)}</span>
                          <strong>{metric ? formatMetricValue(metric.value, metric.unit) : "-"}</strong>
                        </div>
                        {metric ? (
                          metric.unit === "percent" ? (
                            <div className="meter">
                              <div className="meter-fill" style={{ width: `${Math.max(0, Math.min(100, metric.value))}%` }} />
                            </div>
                          ) : (
                            <small>{metric.collectedAt}</small>
                          )
                        ) : (
                          <small>{t("metrics.notAvailable")}</small>
                        )}
                      </div>
                    );
                  })}
                </div>
                <div className="overview-actions">
                  {quickTrendMetrics.map((metricCode) => (
                    <button key={metricCode} type="button" onClick={() => openTrend(host.hostId, metricCode)}>
                      {metricLabel(metricCode, t)}
                    </button>
                  ))}
                </div>
              </section>
            ))}
          </div>
        ) : null}
      </section>
      ) : null}

      {tab === "alerts" && canReadAlerts ? (
        <section className="panel">
          <div className="panel-title">
            <h3>{t("metrics.alertActivity")}</h3>
            <span>{t("metrics.activityHint")}</span>
          </div>
          <div className="toolbar-row">
            <select value={String(alertHistoryHours)} onChange={(event) => setAlertHistoryHours(Number(event.target.value))}>
              {alertHistoryRangeOptions.map((hours) => (
                <option key={hours} value={hours}>{t("metrics.rangeHours").replace("{hours}", String(hours))}</option>
              ))}
            </select>
            <select value={alertSeverity} onChange={(event) => setAlertSeverity(event.target.value)}>
              <option value="">{t("metrics.allSeverities")}</option>
              <option value="info">info</option>
              <option value="warning">warning</option>
              <option value="critical">critical</option>
            </select>
            <select value={alertRuleId} onChange={(event) => setAlertRuleId(event.target.value)}>
              <option value="">{t("metrics.allRules")}</option>
              {rules.map((rule) => <option key={rule.id} value={rule.id}>{rule.name}</option>)}
            </select>
            <select value={alertHostId} onChange={(event) => setAlertHostId(event.target.value)}>
              <option value="">{t("metrics.allHosts")}</option>
              {overviewHosts.map((host) => <option key={host.hostId} value={host.hostId}>{host.hostName}</option>)}
            </select>
            <button type="button" onClick={() => alertHistoryQuery.refetch()}>{t("common.refresh")}</button>
          </div>
          <div className="summary-grid">
            <div className="summary-card">
              <span>{t("common.status.firing")}</span>
              <strong>{alertSummary.firing}</strong>
            </div>
            <div className="summary-card">
              <span>{t("common.status.acknowledged")}</span>
              <strong>{alertSummary.acknowledged}</strong>
            </div>
            <div className="summary-card">
              <span>{t("common.status.silenced")}</span>
              <strong>{alertSummary.silenced}</strong>
            </div>
            <div className="summary-card">
              <span>{t("common.status.resolved")}</span>
              <strong>{alertSummary.resolved}</strong>
            </div>
          </div>
          {alertHistoryQuery.isLoading ? <p>{t("common.loading")}</p> : null}
          {!alertHistoryQuery.isLoading && alertHistory.length === 0 ? <p className="empty-state">{t("metrics.emptyAlertHistory")}</p> : null}
          {!alertHistoryQuery.isLoading && alertHistory.length > 0 ? <AlertHistoryChart points={alertHistory} t={t} /> : null}
          {alertHistoryQuery.isError ? <p className="form-error">{alertHistoryQuery.error.message}</p> : null}
        </section>
      ) : null}

      {tab === "alerts" && canReadAlerts ? (
        <section className="panel table-panel">
          <div className="panel-title">
            <h3>{t("metrics.alertGroups")}</h3>
            <span>{alertGroups.length} {t("common.total")}</span>
          </div>
          <div className="toolbar-row">
            <select value={alertStatus} onChange={(event) => setAlertStatus(event.target.value)}>
              <option value="">{t("common.allStatuses")}</option>
              <option value="firing">{t("common.status.firing")}</option>
              <option value="acknowledged">{t("common.status.acknowledged")}</option>
              <option value="silenced">{t("common.status.silenced")}</option>
              <option value="resolved">{t("common.status.resolved")}</option>
            </select>
            <select value={alertSeverity} onChange={(event) => setAlertSeverity(event.target.value)}>
              <option value="">{t("metrics.allSeverities")}</option>
              <option value="info">info</option>
              <option value="warning">warning</option>
              <option value="critical">critical</option>
            </select>
            <select value={alertRuleId} onChange={(event) => setAlertRuleId(event.target.value)}>
              <option value="">{t("metrics.allRules")}</option>
              {rules.map((rule) => <option key={rule.id} value={rule.id}>{rule.name}</option>)}
            </select>
            <select value={alertGroupHostGroupId} onChange={(event) => setAlertGroupHostGroupId(event.target.value)}>
              <option value="">{t("agents.hostGroups")}</option>
              {hostGroups.map((group) => <option key={group.id} value={group.id}>{group.name}</option>)}
            </select>
            <button type="button" onClick={() => alertGroupsQuery.refetch()}>{t("common.refresh")}</button>
          </div>
          <div className="data-table">
            <table>
              <thead>
                <tr>
                  <th>{t("metrics.alertGroup")}</th>
                  <th>{t("common.severity")}</th>
                  <th>{t("metrics.rule")}</th>
                  <th>{t("agents.hostGroups")}</th>
                  <th>{t("common.status")}</th>
                  <th>{t("metrics.meta")}</th>
                  <th>{t("metrics.lastSeen")}</th>
                </tr>
              </thead>
              <tbody>
                {alertGroups.map((group) => (
                  <tr key={group.id}>
                    <td>
                      <strong>{group.title}</strong>
                      <small>{formatAlertGroupHosts(group)}</small>
                    </td>
                    <td><span className={`status-chip status-${group.severity}`}>{group.severity}</span></td>
                    <td><strong>{group.ruleName || "-"}</strong><small>{group.ruleId || "-"}</small></td>
                    <td><strong>{group.hostGroupName || "-"}</strong><small>{group.hostGroupId || group.fingerprint.slice(0, 12)}</small></td>
                    <td><span className={`status-chip status-${group.status}`}>{alertStatusLabel(group.status)}</span></td>
                    <td>
                      <strong>{group.alertCount} / {group.hostCount}</strong>
                      <small>{formatAlertGroupCounts(group, t)}</small>
                    </td>
                    <td><strong>{group.lastSeenAt}</strong><small>{group.firstSeenAt}</small></td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {!alertGroupsQuery.isLoading && alertGroups.length === 0 ? <p className="empty-state">{t("metrics.emptyAlertGroups")}</p> : null}
          {alertGroupsQuery.isError ? <p className="form-error">{alertGroupsQuery.error.message}</p> : null}
        </section>
      ) : null}

      {tab === "alerts" && canReadAlerts ? (
        <section className="panel table-panel">
          <div className="panel-title">
            <h3>{t("metrics.firingAlerts")}</h3>
            <span>{t("metrics.firingCount").replace("{count}", String(alerts.length))}</span>
          </div>
          <div className="toolbar-row">
            <select value={alertStatus} onChange={(event) => setAlertStatus(event.target.value)}>
              <option value="">{t("common.allStatuses")}</option>
              <option value="firing">{t("common.status.firing")}</option>
              <option value="acknowledged">{t("common.status.acknowledged")}</option>
              <option value="silenced">{t("common.status.silenced")}</option>
              <option value="resolved">{t("common.status.resolved")}</option>
            </select>
            <select value={alertSeverity} onChange={(event) => setAlertSeverity(event.target.value)}>
              <option value="">{t("metrics.allSeverities")}</option>
              <option value="info">info</option>
              <option value="warning">warning</option>
              <option value="critical">critical</option>
            </select>
            <select value={alertRuleId} onChange={(event) => setAlertRuleId(event.target.value)}>
              <option value="">{t("metrics.allRules")}</option>
              {rules.map((rule) => <option key={rule.id} value={rule.id}>{rule.name}</option>)}
            </select>
            <select value={alertHostId} onChange={(event) => setAlertHostId(event.target.value)}>
              <option value="">{t("metrics.allHosts")}</option>
              {overviewHosts.map((host) => <option key={host.hostId} value={host.hostId}>{host.hostName}</option>)}
            </select>
            <input
              type="number"
              min={60}
              value={silenceForm.durationSeconds}
              onChange={(event) => setSilenceForm((current) => ({ ...current, durationSeconds: Number(event.target.value) }))}
              placeholder={t("metrics.silenceSeconds")}
            />
            <input
              value={silenceForm.reason}
              onChange={(event) => setSilenceForm((current) => ({ ...current, reason: event.target.value }))}
              placeholder={t("metrics.silenceReason")}
            />
            <button type="button" onClick={() => alertsQuery.refetch()}>{t("common.refresh")}</button>
          </div>
          <div className="data-table">
            <table>
              <thead>
                <tr>
                  <th>{t("metrics.alert")}</th>
                  <th>{t("common.severity")}</th>
                  <th>{t("metrics.rule")}</th>
                  <th>{t("common.status")}</th>
                  <th>{t("metrics.lastSeen")}</th>
                  <th>{t("metrics.meta")}</th>
                  <th>{t("common.action")}</th>
                </tr>
              </thead>
              <tbody>
                {alerts.flatMap((alert) => {
                  const rows = [
                    <tr key={alert.id}>
                      <td><strong>{alert.title}</strong><small>{alert.hostName || alert.message || alert.id}</small></td>
                      <td><span className={`status-chip status-${alert.severity}`}>{alert.severity}</span></td>
                      <td>{alert.ruleName || "-"}</td>
                      <td><span className={`status-chip status-${alert.status}`}>{alertStatusLabel(alert.status)}</span></td>
                      <td>{alert.lastSeenAt}</td>
                      <td>
                        <strong>{alert.acknowledgedBy || alert.silencedUntil || alert.cooldownUntil || "-"}</strong>
                        <small>
                          {alert.silencedUntil
                            ? `${t("metrics.silencedUntil")}: ${alert.silencedUntil}`
                            : alert.cooldownUntil
                              ? `${t("metrics.cooldownUntil")}: ${alert.cooldownUntil}`
                              : alert.acknowledgedAt
                                ? `${t("metrics.acknowledgedAt")}: ${alert.acknowledgedAt}`
                                : "-"}
                        </small>
                      </td>
                      <td className="action-cell">
                        <button type="button" onClick={() => setExpandedAlertId((current) => current === alert.id ? "" : alert.id)}>
                          {expandedAlertId === alert.id ? t("common.hide") : t("common.view")}
                        </button>
                        {canWriteAlerts && alert.status === "firing" ? <button type="button" onClick={() => acknowledgeMutation.mutate(alert.id)}>{t("metrics.acknowledge")}</button> : null}
                        {canWriteAlerts && alert.status !== "resolved" && alert.status !== "silenced" ? <button type="button" onClick={() => silenceMutation.mutate(alert.id)}>{t("metrics.silence")}</button> : null}
                        {canWriteAlerts && alert.status === "silenced" ? <button type="button" onClick={() => unsilenceMutation.mutate(alert.id)}>{t("metrics.unsilence")}</button> : null}
                        {canWriteAlerts && alert.status !== "resolved" ? <button type="button" onClick={() => resolveMutation.mutate(alert.id)}>{t("metrics.resolve")}</button> : null}
                      </td>
                    </tr>
                  ];
                  if (expandedAlertId === alert.id) {
                    rows.push(
                      <tr key={`${alert.id}:events`}>
                        <td colSpan={7}>
                          <div className="inline-detail">
                            <div className="toolbar-row compact-toolbar">
                              <select value={alertEventType} onChange={(event) => setAlertEventType(event.target.value)}>
                                <option value="">{t("metrics.allEventTypes")}</option>
                                {alertEventTypes.map((eventType) => (
                                  <option key={eventType} value={eventType}>{alertEventLabel(eventType)}</option>
                                ))}
                              </select>
                            </div>
                            {eventsQuery.isLoading ? <p>{t("common.loading")}</p> : null}
                            {!eventsQuery.isLoading && events.length === 0 ? <p>{t("metrics.emptyEvents")}</p> : null}
                            {events.map((event) => (
                              <div key={event.id} className="detail-item">
                                <strong>{alertEventLabel(event.eventType)}</strong>
                                <small>{event.eventType}</small>
                                <small>{event.actor || "-"} | {event.createdAt}</small>
                                <small>{event.message || event.payload || "-"}</small>
                              </div>
                            ))}
                          </div>
                        </td>
                      </tr>
                    );
                  }
                  return rows;
                })}
              </tbody>
            </table>
          </div>
          {!alertsQuery.isLoading && alerts.length === 0 ? <p className="empty-state">{t("metrics.emptyAlerts")}</p> : null}
          {alertsQuery.isError ? <p className="form-error">{alertsQuery.error.message}</p> : null}
          {acknowledgeMutation.isError ? <p className="form-error">{acknowledgeMutation.error.message}</p> : null}
          {silenceMutation.isError ? <p className="form-error">{silenceMutation.error.message}</p> : null}
          {unsilenceMutation.isError ? <p className="form-error">{unsilenceMutation.error.message}</p> : null}
          {resolveMutation.isError ? <p className="form-error">{resolveMutation.error.message}</p> : null}
          {eventsQuery.isError ? <p className="form-error">{eventsQuery.error.message}</p> : null}
        </section>
      ) : null}

      {tab === "rules" && canReadAlerts ? (
        <section className="panel table-panel">
          <div className="panel-title">
            <h3>{t("metrics.alertRules")}</h3>
            <span>{rules.length} {t("common.total")}</span>
          </div>
          <div className="data-table">
            <table>
              <thead>
                <tr>
                  <th>{t("common.name")}</th>
                  <th>{t("metrics.metric")}</th>
                  <th>{t("metrics.condition")}</th>
                  <th>{t("metrics.cooldown")}</th>
                  <th>{t("common.severity")}</th>
                  <th>{t("common.status")}</th>
                  <th>{t("common.action")}</th>
                </tr>
              </thead>
              <tbody>
                {rules.map((rule) => (
                  <tr key={rule.id}>
                    <td><strong>{rule.name}</strong><small>{rule.id}</small></td>
                    <td>{rule.metricCode}</td>
                    <td>{formatRuleCondition(rule.operator || ">", rule.threshold || 0, rule.durationSeconds, t("metrics.forDuration"))}</td>
                    <td>{rule.cooldownSeconds || 0}s</td>
                    <td>{rule.severity}</td>
                    <td><span className={`status-chip status-${rule.status}`}>{alertStatusLabel(rule.status)}</span></td>
                    <td className="action-cell">
                      {canWriteAlerts ? <button type="button" onClick={() => {
                        setEditingRuleId(rule.id);
                        setRuleForm({
                          name: rule.name,
                          metricCode: rule.metricCode || "agent.os.cpu.percent",
                          operator: rule.operator || ">",
                          threshold: rule.threshold || 0,
                          durationSeconds: rule.durationSeconds,
                          cooldownSeconds: rule.cooldownSeconds || 0,
                          severity: rule.severity
                        });
                      }}>{t("common.edit")}</button> : null}
                      {canWriteAlerts && rule.status === "active" ? <button type="button" onClick={() => ruleStatusMutation.mutate({ id: rule.id, action: "pause" })}>{t("common.pause")}</button> : null}
                      {canWriteAlerts && rule.status === "paused" ? <button type="button" onClick={() => ruleStatusMutation.mutate({ id: rule.id, action: "resume" })}>{t("common.resume")}</button> : null}
                      {canWriteAlerts && rule.status !== "disabled" ? <button type="button" onClick={() => ruleStatusMutation.mutate({ id: rule.id, action: "disable" })}>{t("common.disable")}</button> : null}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {!rulesQuery.isLoading && rules.length === 0 ? <p className="empty-state">{t("metrics.emptyRules")}</p> : null}
          {rulesQuery.isError ? <p className="form-error">{rulesQuery.error.message}</p> : null}
          {ruleStatusMutation.isError ? <p className="form-error">{ruleStatusMutation.error.message}</p> : null}
        </section>
      ) : null}

      {tab === "rules" && canReadAlerts ? <AlertNoiseGovernancePanel /> : null}

      {tab === "metrics" ? (
      <section className="panel">
        <div className="panel-title">
          <h3>{t("metrics.trends")}</h3>
          <span>{t("metrics.trendHint")}</span>
        </div>
        <div className="toolbar-row">
          <select value={metricCode} onChange={(event) => setMetricCode(event.target.value)}>
            <option value="">{t("common.allMetrics")}</option>
            {metricCatalog.map((option) => <option key={option.code} value={option.code}>{option.code}</option>)}
          </select>
          <select value={String(trendHours)} onChange={(event) => setTrendHours(Number(event.target.value))}>
            {trendRangeOptions.map((hours) => (
              <option key={hours} value={hours}>{t("metrics.rangeHours").replace("{hours}", String(hours))}</option>
            ))}
          </select>
          <select value={trendGranularity} onChange={(event) => setTrendGranularity(event.target.value)}>
            <option value="auto">{t("metrics.granularityAuto")}</option>
            <option value="raw">{t("metrics.granularityRaw")}</option>
            <option value="5m">5m</option>
            <option value="1h">1h</option>
          </select>
          <button type="button" onClick={() => {
            metricsQuery.refetch();
            trendQuery.refetch();
          }}>{t("common.refresh")}</button>
        </div>
        <div className="chip-row">
          {quickTrendMetrics.map((code) => (
            <button
              key={code}
              type="button"
              className={`metric-chip${metricCode === code ? " active" : ""}`}
              onClick={() => setMetricCode(code)}
            >
              {metricLabel(code, t)}
            </button>
          ))}
        </div>
        {metricCode === "" ? <p className="empty-state">{t("metrics.selectMetricForTrend")}</p> : null}
        {metricCode !== "" && trendQuery.isLoading ? <p>{t("common.loading")}</p> : null}
        {metricCode !== "" && !trendQuery.isLoading && trendSeries.length === 0 ? <p className="empty-state">{t("metrics.noTrendData")}</p> : null}
        {metricCode !== "" && !trendQuery.isLoading && activeTrend ? (
          <div className="trend-layout">
            <div className="trend-series-list">
              {trendSeries.map((series) => {
                const key = trendSeriesKey(series);
                return (
                  <button
                    type="button"
                    key={key}
                    className={`trend-series-button${key === selectedTrendSeries ? " active" : ""}`}
                    onClick={() => setSelectedTrendSeries(key)}
                  >
                    <strong>{series.hostName}</strong>
                    <span>{series.hostId}</span>
                    <small>{formatMetricValue(series.latestValue, series.unit)}</small>
                  </button>
                );
              })}
            </div>
            <div className="trend-view">
              <div className="trend-header">
                <div>
                  <h4>{activeTrend.hostName}</h4>
                  <p>{activeTrend.metricCode}</p>
                </div>
                <small>{activeTrend.agentName || activeTrend.agentId || "Agent"}</small>
              </div>
              <div className="trend-stats">
                <div className="trend-stat">
                  <span>{t("metrics.latestValue")}</span>
                  <strong>{formatMetricValue(activeTrend.latestValue, activeTrend.unit)}</strong>
                  <small>{metricLabel(activeTrend.metricCode, t)}</small>
                </div>
                <div className="trend-stat">
                  <span>{t("metrics.minValue")}</span>
                  <strong>{formatMetricValue(activeTrend.minValue, activeTrend.unit)}</strong>
                  <small>{metricLabel(activeTrend.metricCode, t)}</small>
                </div>
                <div className="trend-stat">
                  <span>{t("metrics.maxValue")}</span>
                  <strong>{formatMetricValue(activeTrend.maxValue, activeTrend.unit)}</strong>
                  <small>{metricLabel(activeTrend.metricCode, t)}</small>
                </div>
                <div className="trend-stat">
                  <span>{t("metrics.points")}</span>
                  <strong>{activeTrend.points.length}</strong>
                  <small>{t("metrics.rangeHours").replace("{hours}", String(trendHours))} / {activeTrend.granularity || trendGranularity}</small>
                </div>
              </div>
              <TrendChart series={activeTrend} />
            </div>
          </div>
        ) : null}
        {trendQuery.isError ? <p className="form-error">{trendQuery.error.message}</p> : null}
      </section>
      ) : null}

      {tab === "metrics" ? (
      <section className="panel table-panel">
        <div className="panel-title">
          <h3>{t("metrics.latestSnapshots")}</h3>
          <span>{latest.length} {t("common.items")}</span>
        </div>
        <div className="toolbar-row">
          <button type="button" onClick={() => metricsQuery.refetch()}>{t("common.refresh")}</button>
        </div>
        <div className="data-table">
          <table>
            <thead>
              <tr>
                <th>{t("metrics.host")}</th>
                <th>{t("metrics.agent")}</th>
                <th>{t("metrics.metric")}</th>
                <th>{t("metrics.value")}</th>
                <th>{t("metrics.collected")}</th>
                <th>{t("common.action")}</th>
              </tr>
            </thead>
            <tbody>
              {latest.map((metric) => (
                <tr key={`${metric.hostId}:${metric.metricCode}`}>
                  <td><strong>{metric.hostName}</strong><small>{metric.hostId}</small></td>
                  <td>{metric.agentName || "-"}</td>
                  <td><strong>{metricLabel(metric.metricCode, t)}</strong><small>{metric.metricCode}</small></td>
                  <td><strong>{formatMetricValue(metric.value, metric.unit)}</strong></td>
                  <td>{metric.collectedAt}</td>
                  <td className="action-cell">
                    <button type="button" onClick={() => openTrend(metric.hostId, metric.metricCode)}>{t("metrics.viewTrend")}</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!metricsQuery.isLoading && latest.length === 0 ? <p className="empty-state">{t("metrics.emptyMetrics")}</p> : null}
        {metricsQuery.isError ? <p className="form-error">{metricsQuery.error.message}</p> : null}
      </section>
      ) : null}
    </main>
  );
}

function AlertHistoryChart({ points, t }: { points: AlertHistoryPoint[]; t: (key: any) => string }) {
  const width = 920;
  const height = 280;
  const padding = { top: 20, right: 24, bottom: 36, left: 40 };
  const maxValue = Math.max(1, ...points.flatMap((point) => [point.firingCount, point.resolvedCount, point.acknowledgedCount, point.silencedCount]));
  const innerWidth = width - padding.left - padding.right;
  const innerHeight = height - padding.top - padding.bottom;
  const series = [
    { key: "firing", color: "#c94f3a", values: points.map((point) => point.firingCount), label: t("common.status.firing") },
    { key: "resolved", color: "#1f8a70", values: points.map((point) => point.resolvedCount), label: t("common.status.resolved") },
    { key: "acknowledged", color: "#175068", values: points.map((point) => point.acknowledgedCount), label: t("common.status.acknowledged") },
    { key: "silenced", color: "#d89216", values: points.map((point) => point.silencedCount), label: t("common.status.silenced") }
  ];

  return (
    <div className="alert-history-shell">
      <div className="legend-row">
        {series.map((item) => (
          <div key={item.key} className="legend-item">
            <span className="legend-swatch" style={{ background: item.color }} />
            <small>{item.label}</small>
          </div>
        ))}
      </div>
      <svg className="alert-history-chart" viewBox={`0 0 ${width} ${height}`} role="img" aria-label={t("metrics.alertActivity")}>
        <line x1={padding.left} y1={padding.top} x2={padding.left} y2={height - padding.bottom} className="trend-axis" />
        <line x1={padding.left} y1={height - padding.bottom} x2={width - padding.right} y2={height - padding.bottom} className="trend-axis" />
        {series.map((item) => {
          const polyline = item.values.map((value, index) => {
            const x = padding.left + (points.length === 1 ? innerWidth / 2 : (index / (points.length - 1)) * innerWidth);
            const y = padding.top + (1 - value / maxValue) * innerHeight;
            return `${x},${y}`;
          }).join(" ");
          return <polyline key={item.key} fill="none" stroke={item.color} strokeWidth="2.5" points={polyline} />;
        })}
        <text x={padding.left} y={padding.top - 6} className="trend-label">{maxValue}</text>
        <text x={padding.left} y={height - padding.bottom + 18} className="trend-label">0</text>
        <text x={padding.left} y={height - 6} className="trend-label">{points[0].bucketStart.slice(5, 16)}</text>
        <text x={width - padding.right} y={height - 6} textAnchor="end" className="trend-label">{points[points.length - 1].bucketStart.slice(5, 16)}</text>
      </svg>
    </div>
  );
}

function TrendChart({ series }: { series: MetricTrendSeries }) {
  if (series.points.length === 0) {
    return null;
  }
  const width = 760;
  const height = 260;
  const padding = { top: 20, right: 20, bottom: 28, left: 44 };
  const minValue = Math.min(...series.points.map((point) => point.value));
  const maxValue = Math.max(...series.points.map((point) => point.value));
  const spread = Math.max(maxValue - minValue, 1);
  const innerWidth = width - padding.left - padding.right;
  const innerHeight = height - padding.top - padding.bottom;
  const points = series.points.map((point, index) => {
    const x = padding.left + (series.points.length === 1 ? innerWidth / 2 : (index / (series.points.length - 1)) * innerWidth);
    const y = padding.top + (1 - (point.value - minValue) / spread) * innerHeight;
    return `${x},${y}`;
  }).join(" ");
  const lastPoint = series.points[series.points.length - 1];
  const firstPoint = series.points[0];

  return (
    <div className="trend-chart-shell">
      <svg className="trend-chart" viewBox={`0 0 ${width} ${height}`} role="img" aria-label={series.metricCode}>
        <line x1={padding.left} y1={padding.top} x2={padding.left} y2={height - padding.bottom} className="trend-axis" />
        <line x1={padding.left} y1={height - padding.bottom} x2={width - padding.right} y2={height - padding.bottom} className="trend-axis" />
        <polyline fill="none" stroke="currentColor" strokeWidth="3" points={points} />
        {series.points.map((point, index) => {
          const x = padding.left + (series.points.length === 1 ? innerWidth / 2 : (index / (series.points.length - 1)) * innerWidth);
          const y = padding.top + (1 - (point.value - minValue) / spread) * innerHeight;
          return <circle key={`${point.collectedAt}:${index}`} cx={x} cy={y} r="3.5" fill="currentColor" />;
        })}
        <text x={padding.left} y={padding.top - 6} className="trend-label">{formatShortValue(maxValue)}</text>
        <text x={padding.left} y={height - padding.bottom + 18} className="trend-label">{formatShortValue(minValue)}</text>
        <text x={padding.left} y={height - 6} className="trend-label">{firstPoint.collectedAt.slice(5)}</text>
        <text x={width - padding.right} y={height - 6} textAnchor="end" className="trend-label">{lastPoint.collectedAt.slice(5)}</text>
      </svg>
    </div>
  );
}

function metricLabel(code: string, t: (key: any) => string) {
  switch (code) {
    case "agent.os.cpu.percent":
      return t("metrics.cpuPercent");
    case "agent.os.memory.used_percent":
      return t("metrics.memoryPercent");
    case "agent.os.disk.used_percent":
      return t("metrics.diskPercent");
    case "agent.running_tasks":
      return t("metrics.runningTasks");
    case "agent.runtime.goroutines":
      return t("metrics.goroutines");
    case "agent.os.network.bytes_sent":
      return t("metrics.networkSent");
    case "agent.os.network.bytes_recv":
      return t("metrics.networkRecv");
    case "agent.os.memory.used_bytes":
      return t("metrics.memoryUsedBytes");
    case "agent.os.memory.total_bytes":
      return t("metrics.memoryTotalBytes");
    case "agent.os.disk.used_bytes":
      return t("metrics.diskUsedBytes");
    case "agent.os.disk.total_bytes":
      return t("metrics.diskTotalBytes");
    case "agent.runtime.alloc_bytes":
      return t("metrics.allocBytes");
    case "agent.runtime.sys_bytes":
      return t("metrics.sysBytes");
    case "agent.cpu.logical":
      return t("metrics.logicalCpu");
    default:
      return code;
  }
}

function buildHostOverview(metrics: Array<{ hostId: string; hostName: string; metricCode: string; value: number; unit?: string; collectedAt: string }>) {
  const map = new Map<string, {
    hostId: string;
    hostName: string;
    metrics: Record<string, { value: number; unit?: string; collectedAt: string }>;
  }>();
  for (const metric of metrics) {
    if (!map.has(metric.hostId)) {
      map.set(metric.hostId, {
        hostId: metric.hostId,
        hostName: metric.hostName,
        metrics: {}
      });
    }
    map.get(metric.hostId)!.metrics[metric.metricCode] = {
      value: metric.value,
      unit: metric.unit,
      collectedAt: metric.collectedAt
    };
  }
  return Array.from(map.values());
}

function summarizeAlerts(alerts: Array<{ status: string }>) {
  return alerts.reduce(
    (result, alert) => {
      switch (alert.status) {
        case "firing":
          result.firing += 1;
          break;
        case "acknowledged":
          result.acknowledged += 1;
          break;
        case "silenced":
          result.silenced += 1;
          break;
        case "resolved":
          result.resolved += 1;
          break;
      }
      return result;
    },
    { firing: 0, acknowledged: 0, silenced: 0, resolved: 0 }
  );
}

function formatAlertGroupHosts(group: AlertGroupSummary) {
  if (!group.hosts || group.hosts.length === 0) {
    return group.fingerprint.slice(0, 16);
  }
  if (group.hostCount <= group.hosts.length) {
    return group.hosts.join(", ");
  }
  return `${group.hosts.join(", ")} +${group.hostCount - group.hosts.length}`;
}

function formatAlertGroupCounts(group: AlertGroupSummary, t: (key: any) => string) {
  return [
    `${group.firingCount} ${t("common.status.firing")}`,
    `${group.acknowledgedCount} ${t("common.status.acknowledged")}`,
    `${group.silencedCount} ${t("common.status.silenced")}`,
    `${group.resolvedCount} ${t("common.status.resolved")}`
  ].join(" | ");
}

function trendSeriesKey(series: MetricTrendSeries) {
  return `${series.hostId}:${series.metricCode}`;
}

function formatMetricValue(value: number, unit?: string) {
  const formatted = Number.isInteger(value) ? String(value) : value.toFixed(2).replace(/\.?0+$/, "");
  return unit ? `${formatted} ${unit}` : formatted;
}

function formatShortValue(value: number) {
  if (Math.abs(value) >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(1).replace(/\.0$/, "")}M`;
  }
  if (Math.abs(value) >= 1_000) {
    return `${(value / 1_000).toFixed(1).replace(/\.0$/, "")}k`;
  }
  return Number.isInteger(value) ? String(value) : value.toFixed(1).replace(/\.0$/, "");
}

function formatRuleCondition(operator: string, threshold: number, durationSeconds: number, forLabel: string) {
  const thresholdText = Number.isInteger(threshold) ? String(threshold) : threshold.toFixed(2).replace(/\.?0+$/, "");
  return `${operator} ${thresholdText} ${forLabel} ${durationSeconds}s`;
}
