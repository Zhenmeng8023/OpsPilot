import { useMemo, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";

import { getAlertNoiseReport, getAlertRoutingExplanation, listAlertRules, listAlerts, runAlertRoutingDryRun, runAlertSuppressionDryRun } from "../../api/alerts";
import type { AlertRoutingDryRunResult, AlertSuppressionDryRunResult } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";
import { Button } from "../../shared/components/Button";
import { FormGrid } from "../../shared/components/FormGrid";
import { SelectField } from "../../shared/components/SelectField";
import { TextField } from "../../shared/components/TextField";

export function AlertNoiseGovernancePanel() {
  const t = useLanguageStore((state) => state.t);
  const [hours, setHours] = useState(24);
  const [limit, setLimit] = useState(8);
  const [dryRunForm, setDryRunForm] = useState({
    ruleId: "",
    hostId: "",
    hostGroupId: "",
    severity: "",
    channelId: ""
  });
  const [routingAlertId, setRoutingAlertId] = useState("");
  const [submittedRoutingAlertId, setSubmittedRoutingAlertId] = useState("");
  const [suppressionResult, setSuppressionResult] = useState<AlertSuppressionDryRunResult | null>(null);
  const [routingResult, setRoutingResult] = useState<AlertRoutingDryRunResult | null>(null);

  const noiseReportQuery = useQuery({
    queryKey: ["alertNoiseReport", hours, limit],
    queryFn: () => getAlertNoiseReport({ hours, limit })
  });
  const rulesQuery = useQuery({
    queryKey: ["alertRules", "noiseGovernance"],
    queryFn: listAlertRules
  });
  const alertsQuery = useQuery({
    queryKey: ["alerts", "routingExplanation"],
    queryFn: () => listAlerts({ status: "firing" })
  });
  const routingExplanationQuery = useQuery({
    queryKey: ["alertRoutingExplanation", submittedRoutingAlertId],
    queryFn: () => getAlertRoutingExplanation(submittedRoutingAlertId),
    enabled: submittedRoutingAlertId !== ""
  });

  const suppressionDryRun = useMutation({
    mutationFn: () => runAlertSuppressionDryRun({
      ruleId: dryRunForm.ruleId.trim(),
      hostId: dryRunForm.hostId.trim(),
      hostGroupId: dryRunForm.hostGroupId.trim(),
      severity: dryRunForm.severity
    }),
    onSuccess: (result) => setSuppressionResult(result)
  });

  const routingDryRun = useMutation({
    mutationFn: () => runAlertRoutingDryRun({
      ruleId: dryRunForm.ruleId.trim(),
      hostId: dryRunForm.hostId.trim(),
      hostGroupId: dryRunForm.hostGroupId.trim(),
      severity: dryRunForm.severity,
      channelId: dryRunForm.channelId.trim()
    }),
    onSuccess: (result) => setRoutingResult(result)
  });

  const trendSummary = useMemo(() => {
    const rows = noiseReportQuery.data?.trends ?? [];
    const map = new Map<string, { hostGroupName: string; severity: string; count: number }>();
    for (const row of rows) {
      const key = `${row.hostGroupName}:${row.severity}`;
      const current = map.get(key);
      if (!current) {
        map.set(key, { hostGroupName: row.hostGroupName, severity: row.severity, count: row.alertCount });
      } else {
        current.count += row.alertCount;
      }
    }
    return Array.from(map.values()).sort((left, right) => right.count - left.count).slice(0, 12);
  }, [noiseReportQuery.data]);

  const ruleOptions = useMemo(() => {
    const base = [{ value: "", label: t("metrics.allRules") }];
    const rules = rulesQuery.data ?? [];
    return base.concat(rules.map((rule) => ({ value: rule.id, label: `${rule.name} (${rule.id})` })));
  }, [rulesQuery.data, t]);
  const alertOptions = useMemo(() => {
    const base = [{ value: "", label: t("metrics.selectAlert") }];
    return base.concat((alertsQuery.data ?? []).map((alert) => ({ value: alert.id, label: `${alert.title} (${alert.id})` })));
  }, [alertsQuery.data, t]);

  return (
    <section className="panel table-panel">
      <div className="panel-title">
        <h3>{t("metrics.noiseGovernance")}</h3>
        <span>{t("metrics.noiseGovernanceHint")}</span>
      </div>
      <div className="toolbar-row">
        <select value={String(hours)} onChange={(event) => setHours(Number(event.target.value))}>
          {[6, 12, 24, 48, 72, 168].map((item) => (
            <option key={item} value={item}>{t("metrics.rangeHours").replace("{hours}", String(item))}</option>
          ))}
        </select>
        <select value={String(limit)} onChange={(event) => setLimit(Number(event.target.value))}>
          {[5, 8, 10, 15, 20].map((item) => (
            <option key={item} value={item}>{t("metrics.topN").replace("{count}", String(item))}</option>
          ))}
        </select>
        <Button type="button" onClick={() => {
          noiseReportQuery.refetch();
        }}>{t("common.refresh")}</Button>
      </div>

      <div className="panel-grid">
        <div className="data-table">
          <table>
            <thead>
              <tr>
                <th>{t("metrics.rule")}</th>
                <th>{t("common.severity")}</th>
                <th>{t("metrics.alertCount")}</th>
                <th>{t("metrics.activeAlerts")}</th>
                <th>{t("metrics.hostCount")}</th>
                <th>{t("metrics.lastSeen")}</th>
              </tr>
            </thead>
            <tbody>
              {(noiseReportQuery.data?.noisyRules ?? []).map((row) => (
                <tr key={`${row.ruleId || row.ruleName}:${row.lastSeenAt}`}>
                  <td><strong>{row.ruleName}</strong><small>{row.ruleId || "-"}</small></td>
                  <td>{row.severity}</td>
                  <td>{row.alertCount}</td>
                  <td>{row.activeCount}</td>
                  <td>{row.hostCount}</td>
                  <td>{row.lastSeenAt}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {!noiseReportQuery.isLoading && (noiseReportQuery.data?.noisyRules ?? []).length === 0 ? <p className="empty-state">{t("common.empty")}</p> : null}
          {noiseReportQuery.isError ? <p className="form-error">{noiseReportQuery.error.message}</p> : null}
        </div>

        <div className="data-table">
          <table>
            <thead>
              <tr>
                <th>{t("metrics.hostGroup")}</th>
                <th>{t("common.severity")}</th>
                <th>{t("metrics.alertCount")}</th>
              </tr>
            </thead>
            <tbody>
              {trendSummary.map((row) => (
                <tr key={`${row.hostGroupName}:${row.severity}`}>
                  <td>{row.hostGroupName}</td>
                  <td>{row.severity}</td>
                  <td>{row.count}</td>
                </tr>
              ))}
            </tbody>
          </table>
          {!noiseReportQuery.isLoading && trendSummary.length === 0 ? <p className="empty-state">{t("common.empty")}</p> : null}
        </div>
      </div>

      <form
        className="editor-form"
        onSubmit={(event) => {
          event.preventDefault();
          suppressionDryRun.mutate();
        }}
      >
        <h4>{t("metrics.dryRun")}</h4>
        <FormGrid columns={3}>
          <SelectField
            label={t("metrics.rule")}
            value={dryRunForm.ruleId}
            options={ruleOptions}
            onChange={(event) => setDryRunForm((current) => ({ ...current, ruleId: event.target.value }))}
          />
          <TextField
            label={t("metrics.host")}
            value={dryRunForm.hostId}
            onChange={(event) => setDryRunForm((current) => ({ ...current, hostId: event.target.value }))}
            placeholder="host_xxx"
          />
          <TextField
            label={t("agents.hostGroups")}
            value={dryRunForm.hostGroupId}
            onChange={(event) => setDryRunForm((current) => ({ ...current, hostGroupId: event.target.value }))}
            placeholder="group_xxx"
          />
          <SelectField
            label={t("common.severity")}
            value={dryRunForm.severity}
            options={[
              { value: "", label: t("metrics.allSeverities") },
              { value: "info", label: "info" },
              { value: "warning", label: "warning" },
              { value: "critical", label: "critical" }
            ]}
            onChange={(event) => setDryRunForm((current) => ({ ...current, severity: event.target.value }))}
          />
          <TextField
            label={t("notifications.channel")}
            value={dryRunForm.channelId}
            onChange={(event) => setDryRunForm((current) => ({ ...current, channelId: event.target.value }))}
            placeholder="channel_xxx"
          />
        </FormGrid>
        <div className="toolbar-row">
          <Button type="submit" disabled={suppressionDryRun.isPending}>{t("metrics.dryRunSuppression")}</Button>
          <Button
            type="button"
            onClick={() => routingDryRun.mutate()}
            disabled={routingDryRun.isPending}
          >
            {t("metrics.dryRunRouting")}
          </Button>
        </div>
      </form>

      <div className="summary-grid">
        <div className="summary-card">
          <strong>{t("metrics.dryRunSuppression")}</strong>
          <small>{t("metrics.matchedAlerts")}: {suppressionResult?.matchedAlerts ?? 0}</small>
          <small>{t("metrics.sampleAlerts")}: {renderIDs(suppressionResult?.sampleAlertIds)}</small>
        </div>
        <div className="summary-card">
          <strong>{t("metrics.dryRunRouting")}</strong>
          <small>{t("metrics.matchedAlerts")}: {routingResult?.matchedAlerts ?? 0}</small>
          <small>{t("metrics.matchedPolicies")}: {routingResult?.matchedPolicies ?? 0}</small>
          <small>{t("metrics.sampleAlerts")}: {renderIDs(routingResult?.sampleAlertIds)}</small>
        </div>
      </div>
      {suppressionDryRun.isError ? <p className="form-error">{suppressionDryRun.error.message}</p> : null}
      {routingDryRun.isError ? <p className="form-error">{routingDryRun.error.message}</p> : null}

      <form
        className="editor-form"
        onSubmit={(event) => {
          event.preventDefault();
          setSubmittedRoutingAlertId(routingAlertId.trim());
        }}
      >
        <h4>{t("metrics.routingExplanation")}</h4>
        <FormGrid columns={2}>
          <SelectField
            label={t("metrics.alert")}
            value={routingAlertId}
            options={alertOptions}
            onChange={(event) => setRoutingAlertId(event.target.value)}
          />
          <TextField
            label={t("metrics.alertId")}
            value={routingAlertId}
            onChange={(event) => setRoutingAlertId(event.target.value)}
            placeholder="alert_xxx"
          />
        </FormGrid>
        <div className="toolbar-row">
          <Button type="submit" disabled={!routingAlertId.trim() || routingExplanationQuery.isFetching}>{t("metrics.explainRouting")}</Button>
        </div>
      </form>

      {routingExplanationQuery.data ? (
        <div className="summary-grid">
          <div className="summary-card">
            <strong>{t("metrics.routing")}</strong>
            <small>{routingExplanationQuery.data.routing || "-"}</small>
          </div>
          <div className="summary-card">
            <strong>{t("metrics.suppression")}</strong>
            <small>{routingExplanationQuery.data.suppression || "-"}</small>
          </div>
          <div className="summary-card">
            <strong>{t("metrics.matchedPolicies")}</strong>
            <small>{renderIDs(routingExplanationQuery.data.policyIds)}</small>
          </div>
          <div className="summary-card">
            <strong>{t("metrics.sampleAlerts")}</strong>
            <small>{routingExplanationQuery.data.alert.id}</small>
          </div>
        </div>
      ) : null}
      {routingExplanationQuery.data?.explanations.length ? (
        <div className="data-table">
          <table>
            <thead>
              <tr>
                <th>{t("common.category")}</th>
                <th>{t("common.status")}</th>
                <th>{t("common.reason")}</th>
              </tr>
            </thead>
            <tbody>
              {routingExplanationQuery.data.explanations.map((item) => (
                <tr key={`${item.dimension}:${item.value ?? ""}:${item.reason ?? ""}`}>
                  <td><strong>{item.dimension}</strong><small>{item.value || "-"}</small></td>
                  <td><span className={`status-chip status-${item.matched ? "matched" : "disabled"}`}>{item.matched ? "matched" : "skipped"}</span></td>
                  <td>{item.reason || "-"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
      {routingExplanationQuery.isError ? <p className="form-error">{routingExplanationQuery.error.message}</p> : null}
    </section>
  );
}

function renderIDs(values: string[] | undefined) {
  if (!values || values.length === 0) return "-";
  return values.slice(0, 3).join(", ");
}
