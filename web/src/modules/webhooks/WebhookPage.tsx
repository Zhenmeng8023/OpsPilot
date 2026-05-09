import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { createWebhookRule, createWebhookSource, disableWebhookRule, disableWebhookSource, getWebhookEvent, listWebhookEvents, listWebhookRules, listWebhookSources, pauseWebhookRule, pauseWebhookSource, replayWebhookEvent, resumeWebhookRule, resumeWebhookSource, rotateWebhookSourceSecret, simulateWebhookMatcher, updateWebhookRule } from "../../api/webhooks";
import { listTasks } from "../../api/tasks";
import { listWorkflows } from "../../api/workflows";
import { API_BASE_URL } from "../../api/request";
import type { WebhookEvent, WebhookMatcherCondition, WebhookMatcherSimulationResult, WebhookRule, WebhookSource } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";
import { DataTable } from "../../shared/components/DataTable";
import { FilterToolbar } from "../../shared/components/FilterToolbar";
import { JsonViewer } from "../../shared/components/JsonViewer";
import { PaginationBar } from "../../shared/components/PaginationBar";
import { hasPermission } from "../auth/permissions";
import { useAuthStore } from "../auth/store";

type MatcherDraft = {
  type: "" | WebhookMatcherCondition["type"];
  key: string;
  path: string;
  value: string;
};

const EMPTY_MATCHER_DRAFT: MatcherDraft = {
  type: "",
  key: "",
  path: "",
  value: ""
};

const DEFAULT_SIMULATION_PAYLOAD = JSON.stringify({
  ref: "refs/heads/main",
  repository: {
    name: "OpsPilot"
  },
  commits: [
    { id: "commit-1", author: { name: "ops" } },
    { id: "commit-2", author: { name: "pilot" } }
  ],
  deployment: {
    environment: "staging"
  }
}, null, 2);

export function WebhookPage() {
  const user = useAuthStore((state) => state.user);
  const t = useLanguageStore((state) => state.t);
  const queryClient = useQueryClient();
  const canManage = hasPermission(user, "webhook:manage");
  const [sourceForm, setSourceForm] = useState({ name: "", sourceType: "custom" });
  const [ruleForm, setRuleForm] = useState({
    sourceId: "",
    targetType: "task",
    taskId: "",
    workflowId: "",
    name: "",
    eventType: ""
  });
  const [matcherDrafts, setMatcherDrafts] = useState<MatcherDraft[]>([]);
  const [matcherError, setMatcherError] = useState("");
  const [eventFilters, setEventFilters] = useState({ sourceId: "", status: "", deliveryId: "", receivedFrom: "", receivedTo: "", page: 1 });
  const [selectedEventID, setSelectedEventID] = useState<string | undefined>();
  const [editingRuleID, setEditingRuleID] = useState<string | undefined>();
  const [issuedToken, setIssuedToken] = useState("");
  const [issuedSigningSecret, setIssuedSigningSecret] = useState("");
  const [issuedSourceName, setIssuedSourceName] = useState("");
  const [simulator, setSimulator] = useState({
    ruleId: "",
    eventType: "push",
    headers: JSON.stringify({ "X-GitHub-Event": "push" }, null, 2),
    payload: DEFAULT_SIMULATION_PAYLOAD
  });
  const sourcesQuery = useQuery({ queryKey: ["webhookSources"], queryFn: listWebhookSources });
  const rulesQuery = useQuery({ queryKey: ["webhookRules"], queryFn: listWebhookRules });
  const eventsQuery = useQuery({
    queryKey: ["webhookEvents", eventFilters],
    queryFn: () => listWebhookEvents({ ...eventFilters, pageSize: 20 })
  });
  const eventDetailQuery = useQuery({
    queryKey: ["webhookEvent", selectedEventID],
    enabled: Boolean(selectedEventID),
    queryFn: () => getWebhookEvent(selectedEventID ?? "")
  });
  const tasksQuery = useQuery({ queryKey: ["tasks", "webhook-options"], queryFn: () => listTasks({ pageSize: 100 }) });
  const workflowsQuery = useQuery({ queryKey: ["workflows", "webhook-options"], queryFn: () => listWorkflows({ pageSize: 100, status: "active" }) });
  const createSourceMutation = useMutation({
    mutationFn: createWebhookSource,
    onSuccess: (source) => {
      setIssuedSourceName(source.name);
      setIssuedToken(source.token ?? "");
      setIssuedSigningSecret(source.signingSecret ?? "");
      setSourceForm({ name: "", sourceType: "custom" });
      queryClient.invalidateQueries({ queryKey: ["webhookSources"] });
    }
  });
  const rotateSourceMutation = useMutation({
    mutationFn: ({ source, rotateToken }: { source: WebhookSource; rotateToken: boolean }) =>
      rotateWebhookSourceSecret(source.id, { rotateToken }),
    onSuccess: (source) => {
      setIssuedSourceName(source.name);
      setIssuedToken(source.token ?? "");
      setIssuedSigningSecret(source.signingSecret ?? "");
      queryClient.invalidateQueries({ queryKey: ["webhookSources"] });
    }
  });
  const pauseSourceMutation = useMutation({
    mutationFn: (source: WebhookSource) => pauseWebhookSource(source.id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["webhookSources"] })
  });
  const resumeSourceMutation = useMutation({
    mutationFn: (source: WebhookSource) => resumeWebhookSource(source.id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["webhookSources"] })
  });
  const disableSourceMutation = useMutation({
    mutationFn: (source: WebhookSource) => disableWebhookSource(source.id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["webhookSources"] })
  });
  const createRuleMutation = useMutation({
    mutationFn: createWebhookRule,
    onSuccess: () => {
      setRuleForm({
        sourceId: "",
        targetType: "task",
        taskId: "",
        workflowId: "",
        name: "",
        eventType: ""
      });
      setMatcherDrafts([]);
      setMatcherError("");
      queryClient.invalidateQueries({ queryKey: ["webhookRules"] });
    }
  });
  const updateRuleMutation = useMutation({
    mutationFn: (payload: { id: string; name: string; eventType?: string; matcher?: { conditions: WebhookMatcherCondition[] } }) =>
      updateWebhookRule(payload.id, { name: payload.name, eventType: payload.eventType, matcher: payload.matcher }),
    onSuccess: () => {
      resetRuleEditor();
      queryClient.invalidateQueries({ queryKey: ["webhookRules"] });
    }
  });
  const pauseRuleMutation = useMutation({
    mutationFn: (rule: WebhookRule) => pauseWebhookRule(rule.id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["webhookRules"] })
  });
  const resumeRuleMutation = useMutation({
    mutationFn: (rule: WebhookRule) => resumeWebhookRule(rule.id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["webhookRules"] })
  });
  const disableRuleMutation = useMutation({
    mutationFn: (rule: WebhookRule) => disableWebhookRule(rule.id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["webhookRules"] })
  });
  const simulateMatcherMutation = useMutation({
    mutationFn: (payload: { ruleId?: string; eventType?: string; headers?: Record<string, string>; payload?: string }) =>
      simulateWebhookMatcher(payload)
  });
  const replayEventMutation = useMutation({
    mutationFn: ({ id, simulateOnly }: { id: string; simulateOnly: boolean }) =>
      replayWebhookEvent(id, { simulateOnly }),
    onSuccess: (_, variables) => {
      queryClient.invalidateQueries({ queryKey: ["webhookEvents"] });
      queryClient.invalidateQueries({ queryKey: ["webhookEvent", variables.id] });
    }
  });
  const sources = useMemo(() => sourcesQuery.data ?? [], [sourcesQuery.data]);
  const rules = useMemo(() => rulesQuery.data ?? [], [rulesQuery.data]);
  const events = useMemo(() => eventsQuery.data?.items ?? [], [eventsQuery.data]);
  const totalEvents = eventsQuery.data?.total ?? 0;
  const statusText = (value: string) => t(`common.status.${value}`);
  const taskOptions = useMemo(() => {
    const map = new Map<string, NonNullable<typeof tasksQuery.data>["items"][number]>();
    for (const task of tasksQuery.data?.items ?? []) {
      if (!map.has(task.taskId)) {
        map.set(task.taskId, task);
      }
    }
    return Array.from(map.values());
  }, [tasksQuery.data]);
  const workflowOptions = useMemo(() => workflowsQuery.data?.items ?? [], [workflowsQuery.data]);
  const triggerURL = issuedToken ? `${API_BASE_URL || window.location.origin}/api/v1/webhooks/trigger/${issuedToken}` : "";
  const addMatcherDraft = () => {
    setMatcherDrafts((current) => [...current, { ...EMPTY_MATCHER_DRAFT }]);
  };
  const updateMatcherDraft = (index: number, patch: Partial<MatcherDraft>) => {
    setMatcherDrafts((current) => current.map((item, itemIndex) => (
      itemIndex === index
        ? {
            ...item,
            ...patch,
            ...(patch.type === "header_equals" ? { path: "" } : {}),
            ...(isPayloadMatcherType(patch.type) ? { key: "" } : {}),
            ...(isEventMatcherType(patch.type) ? { key: "", path: "" } : {}),
            ...(patch.type === "payload_exists" ? { value: "" } : {}),
            ...(patch.type === "" ? { key: "", path: "", value: "" } : {})
          }
        : item
    )));
  };
  const removeMatcherDraft = (index: number) => {
    setMatcherDrafts((current) => current.filter((_, itemIndex) => itemIndex !== index));
  };
  const ruleMatcherResult = buildRuleMatcher(matcherDrafts);
  const startRuleEdit = (rule: WebhookRule) => {
    setEditingRuleID(rule.id);
    setMatcherError("");
    setRuleForm({
      sourceId: rule.sourceId,
      targetType: rule.targetType || "task",
      taskId: rule.taskId ?? "",
      workflowId: rule.workflowId ?? "",
      name: rule.name,
      eventType: rule.eventType ?? ""
    });
    setMatcherDrafts(matcherDraftsFromMatcher(rule.matcher));
  };
  function resetRuleEditor() {
    setEditingRuleID(undefined);
    setMatcherError("");
    setMatcherDrafts([]);
    setRuleForm({
      sourceId: "",
      targetType: "task",
      taskId: "",
      workflowId: "",
      name: "",
      eventType: ""
    });
  }
  const formatSecurityStatus = (item: Pick<WebhookEvent, "signatureValid" | "replayed">) => {
    if (item.replayed) {
      return t("webhooks.securityReplayed");
    }
    return item.signatureValid ? t("webhooks.securityOk") : t("webhooks.securityFailed");
  };
  const formatSecurityDetail = (item: Pick<WebhookEvent, "replayed" | "errorMessage" | "signatureHeader">) => {
    if (item.replayed) {
      return item.errorMessage || t("webhooks.securityReplayed");
    }
    return item.errorMessage || item.signatureHeader || "-";
  };
  const formatMatchReason = (reason?: string) => {
    const value = reason?.trim() ?? "";
    if (!value) {
      return "-";
    }
    if (value === "event_type_mismatch") {
      return t("webhooks.reasonEventTypeMismatch");
    }
    if (value === "matcher_invalid") {
      return t("webhooks.reasonMatcherInvalid");
    }
    if (value === "triggered") {
      return t("webhooks.reasonTriggered");
    }
    if (value.startsWith("header_mismatch:")) {
      return t("webhooks.reasonHeaderMismatch").replace("{value}", value.slice("header_mismatch:".length));
    }
    if (value.startsWith("payload_mismatch:")) {
      return t("webhooks.reasonPayloadMismatch").replace("{value}", value.slice("payload_mismatch:".length));
    }
    if (value.startsWith("payload_contains_mismatch:")) {
      return t("webhooks.reasonPayloadContainsMismatch").replace("{value}", value.slice("payload_contains_mismatch:".length));
    }
    if (value.startsWith("payload_not_equals_mismatch:")) {
      return t("webhooks.reasonPayloadNotEqualsMismatch").replace("{value}", value.slice("payload_not_equals_mismatch:".length));
    }
    if (value.startsWith("payload_regex_mismatch:")) {
      return t("webhooks.reasonPayloadRegexMismatch").replace("{value}", value.slice("payload_regex_mismatch:".length));
    }
    if (value.startsWith("payload_missing:")) {
      return t("webhooks.reasonPayloadMissing").replace("{value}", value.slice("payload_missing:".length));
    }
    if (value === "event_type_condition_mismatch") {
      return t("webhooks.reasonEventTypeConditionMismatch");
    }
    if (value === "ref_mismatch") {
      return t("webhooks.reasonRefMismatch");
    }
    if (value === "branch_mismatch") {
      return t("webhooks.reasonBranchMismatch");
    }
    return value;
  };

  const simulatorResult = simulateMatcherMutation.data;
  const currentRuleForSimulation = rules.find((item) => item.id === simulator.ruleId);
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
          {triggerURL ? (
            <div className="token-secret">
              <strong>{issuedSourceName ? `${issuedSourceName}: ${triggerURL}` : triggerURL}</strong>
              <span>{issuedSigningSecret || t("webhooks.secretMissing")}</span>
            </div>
          ) : null}
          {createSourceMutation.isError ? <p className="form-error">{createSourceMutation.error.message}</p> : null}
          {rotateSourceMutation.isError ? <p className="form-error">{rotateSourceMutation.error.message}</p> : null}
        </section>
      ) : null}

      <section className="panel table-panel">
        <div className="panel-title"><h3>{t("webhooks.sources")}</h3><span>{sources.length} {t("common.total")}</span></div>
        <DataTable loading={sourcesQuery.isLoading} empty={sources.length === 0} emptyMessage={t("webhooks.emptySources")} error={sourcesQuery.isError ? sourcesQuery.error.message : null}>
          <table>
            <thead><tr><th>{t("common.name")}</th><th>{t("common.type")}</th><th>{t("common.status")}</th><th>{t("common.created")}</th><th>{t("common.action")}</th></tr></thead>
            <tbody>
              {sources.map((source) => (
                <tr key={source.id}>
                  <td><strong>{source.name}</strong><small>{source.id}</small></td>
                  <td>{source.sourceType}</td>
                  <td><span className={`status-chip status-${source.status}`}>{statusText(source.status)}</span></td>
                  <td><strong>{source.createdAt}</strong><small>{source.lastReceivedAt || source.createdBy || "-"}</small></td>
                  <td className="action-cell">
                    {canManage && source.status === "active" ? <button type="button" onClick={() => rotateSourceMutation.mutate({ source, rotateToken: false })}>{t("webhooks.rotateSecret")}</button> : null}
                    {canManage && source.status === "active" ? <button type="button" onClick={() => rotateSourceMutation.mutate({ source, rotateToken: true })}>{t("webhooks.rotateToken")}</button> : null}
                    {canManage && source.status === "active" ? <button type="button" onClick={() => pauseSourceMutation.mutate(source)}>{t("webhooks.pauseSource")}</button> : null}
                    {canManage && source.status === "paused" ? <button type="button" onClick={() => resumeSourceMutation.mutate(source)}>{t("webhooks.resumeSource")}</button> : null}
                    {canManage && source.status !== "disabled" ? <button type="button" onClick={() => disableSourceMutation.mutate(source)}>{t("webhooks.disableSource")}</button> : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </DataTable>
        {pauseSourceMutation.isError ? <p className="form-error">{pauseSourceMutation.error.message}</p> : null}
        {resumeSourceMutation.isError ? <p className="form-error">{resumeSourceMutation.error.message}</p> : null}
        {disableSourceMutation.isError ? <p className="form-error">{disableSourceMutation.error.message}</p> : null}
      </section>

      {canManage ? (
        <section className="panel form-panel">
          <div className="panel-title"><h3>{editingRuleID ? t("webhooks.editRule") : t("webhooks.createRule")}</h3><span>{t("webhooks.ruleHint")}</span></div>
          <form className="form-grid" onSubmit={(event) => {
            event.preventDefault();
            if (!ruleMatcherResult.ok) {
              setMatcherError(t("webhooks.matcherInvalidInput"));
              return;
            }
            setMatcherError("");
            if (editingRuleID) {
              updateRuleMutation.mutate({
                id: editingRuleID,
                name: ruleForm.name,
                eventType: ruleForm.eventType,
                matcher: ruleMatcherResult.matcher
              });
              return;
            }
            createRuleMutation.mutate({
              sourceId: ruleForm.sourceId,
              targetType: ruleForm.targetType,
              taskId: ruleForm.targetType === "task" ? ruleForm.taskId : undefined,
              workflowId: ruleForm.targetType === "workflow" ? ruleForm.workflowId : undefined,
              name: ruleForm.name,
              eventType: ruleForm.eventType,
              matcher: ruleMatcherResult.matcher
            });
          }}>
            <label>
              {t("common.source")}
              <select value={ruleForm.sourceId} onChange={(event) => setRuleForm({ ...ruleForm, sourceId: event.target.value })} required disabled={Boolean(editingRuleID)}>
                <option value="">{t("webhooks.selectSource")}</option>
                {sources.map((source) => <option key={source.id} value={source.id}>{source.name}</option>)}
              </select>
            </label>
            <label>
              {t("common.type")}
              <select value={ruleForm.targetType} onChange={(event) => setRuleForm({ ...ruleForm, targetType: event.target.value, taskId: "", workflowId: "" })} disabled={Boolean(editingRuleID)}>
                <option value="task">{t("common.task")}</option>
                <option value="workflow">{t("workflows.workflow")}</option>
              </select>
            </label>
            {ruleForm.targetType === "workflow" ? (
              <label>
                {t("workflows.workflow")}
                <select value={ruleForm.workflowId} onChange={(event) => setRuleForm({ ...ruleForm, workflowId: event.target.value })} required disabled={Boolean(editingRuleID)}>
                  <option value="">{t("common.selectWorkflow")}</option>
                  {workflowOptions.map((workflow) => <option key={workflow.id} value={workflow.id}>{workflow.name}</option>)}
                </select>
              </label>
            ) : (
              <label>
                {t("common.task")}
                <select value={ruleForm.taskId} onChange={(event) => setRuleForm({ ...ruleForm, taskId: event.target.value })} required disabled={Boolean(editingRuleID)}>
                  <option value="">{t("webhooks.selectTask")}</option>
                  {taskOptions.map((task) => <option key={task.taskId} value={task.taskId}>{task.name}</option>)}
                </select>
              </label>
            )}
            <label>{t("common.name")}<input value={ruleForm.name} onChange={(event) => setRuleForm({ ...ruleForm, name: event.target.value })} required /></label>
            <label>{t("webhooks.eventType")}<input value={ruleForm.eventType} onChange={(event) => setRuleForm({ ...ruleForm, eventType: event.target.value })} /></label>
            <div className="matcher-builder">
              <div className="matcher-builder-header">
                <strong>{t("webhooks.matcherConditions")}</strong>
                <button type="button" className="ghost-button" onClick={addMatcherDraft}>{t("webhooks.addMatcherCondition")}</button>
              </div>
              {matcherDrafts.length === 0 ? <p className="muted">{t("webhooks.noMatcher")}</p> : null}
              {matcherDrafts.map((draft, index) => (
                <div key={`matcher-${index}`} className="matcher-row">
                  <label>
                    {t("webhooks.matcherType")}
                    <select
                      value={draft.type}
                      onChange={(event) => {
                        setMatcherError("");
                        updateMatcherDraft(index, { type: event.target.value as MatcherDraft["type"] });
                      }}
                    >
                      <option value="">{t("webhooks.noMatcher")}</option>
                      <option value="header_equals">header_equals</option>
                      <option value="payload_equals">payload_equals</option>
                      <option value="payload_contains">payload_contains</option>
                      <option value="payload_not_equals">payload_not_equals</option>
                      <option value="payload_exists">payload_exists</option>
                      <option value="payload_regex">payload_regex</option>
                      <option value="event_type_equals">event_type_equals</option>
                      <option value="ref_equals">ref_equals</option>
                      <option value="branch_equals">branch_equals</option>
                    </select>
                  </label>
                  {draft.type === "header_equals" ? (
                    <label>{t("webhooks.matcherKey")}<input value={draft.key} onChange={(event) => {
                      setMatcherError("");
                      updateMatcherDraft(index, { key: event.target.value });
                    }} placeholder="X-GitHub-Event" /></label>
                  ) : null}
                  {isPayloadMatcherType(draft.type) ? (
                    <label>{t("webhooks.matcherPath")}<input value={draft.path} onChange={(event) => {
                      setMatcherError("");
                      updateMatcherDraft(index, { path: event.target.value });
                    }} placeholder="commits[*].author.name" /></label>
                  ) : null}
                  {draft.type && matcherTypeRequiresValue(draft.type) ? (
                    <label>{t("webhooks.matcherValue")}<input value={draft.value} onChange={(event) => {
                      setMatcherError("");
                      updateMatcherDraft(index, { value: event.target.value });
                    }} placeholder={matcherValuePlaceholder(draft.type)} /></label>
                  ) : null}
                  <button type="button" className="danger-button" onClick={() => removeMatcherDraft(index)}>{t("webhooks.removeMatcherCondition")}</button>
                </div>
              ))}
            </div>
            <button type="submit" disabled={createRuleMutation.isPending || updateRuleMutation.isPending}>{editingRuleID ? t("webhooks.updateRuleAction") : t("webhooks.createRuleAction")}</button>
            {editingRuleID ? <button type="button" className="ghost-button" onClick={resetRuleEditor}>{t("webhooks.cancelRuleEdit")}</button> : null}
          </form>
          {matcherError ? <p className="form-error">{matcherError}</p> : null}
          {createRuleMutation.isError ? <p className="form-error">{createRuleMutation.error.message}</p> : null}
          {updateRuleMutation.isError ? <p className="form-error">{updateRuleMutation.error.message}</p> : null}
        </section>
      ) : null}

      {canManage ? (
        <section className="panel form-panel">
          <div className="panel-title"><h3>{t("webhooks.matcherSimulator")}</h3><span>{t("webhooks.matcherSimulatorHint")}</span></div>
          <form className="form-grid" onSubmit={(event) => {
            event.preventDefault();
            const headers = parseJSONMap(simulator.headers);
            if (!headers) {
              return;
            }
            simulateMatcherMutation.mutate({
              ruleId: simulator.ruleId || undefined,
              eventType: simulator.eventType,
              headers,
              payload: simulator.payload
            });
          }}>
            <label>
              {t("webhooks.rule")}
              <select value={simulator.ruleId} onChange={(event) => setSimulator((current) => ({ ...current, ruleId: event.target.value, eventType: current.eventType || "push" }))}>
                <option value="">{t("webhooks.selectRuleOptional")}</option>
                {rules.map((rule) => <option key={rule.id} value={rule.id}>{rule.name}</option>)}
              </select>
            </label>
            <label>{t("webhooks.eventType")}<input value={simulator.eventType} onChange={(event) => setSimulator((current) => ({ ...current, eventType: event.target.value }))} /></label>
            <label>
              {t("webhooks.headers")}
              <textarea className="code-input" value={simulator.headers} onChange={(event) => setSimulator((current) => ({ ...current, headers: event.target.value }))} />
            </label>
            <label>
              {t("webhooks.payload")}
              <textarea className="code-input" value={simulator.payload} onChange={(event) => setSimulator((current) => ({ ...current, payload: event.target.value }))} />
            </label>
            <button type="submit" disabled={simulateMatcherMutation.isPending}>{t("webhooks.simulateMatcherAction")}</button>
          </form>
          {simulateMatcherMutation.isError ? <p className="form-error">{simulateMatcherMutation.error.message}</p> : null}
          {simulatorResult ? <MatcherSimulationPanel result={simulatorResult} ruleName={currentRuleForSimulation?.name} t={t} formatReason={formatMatchReason} /> : null}
        </section>
      ) : null}

      <section className="panel table-panel">
        <div className="panel-title"><h3>{t("webhooks.rules")}</h3><span>{rules.length} {t("common.total")}</span></div>
        <DataTable loading={rulesQuery.isLoading} empty={rules.length === 0} emptyMessage={t("webhooks.emptyRules")} error={rulesQuery.isError ? rulesQuery.error.message : null}>
          <table>
            <thead><tr><th>{t("common.name")}</th><th>{t("common.source")}</th><th>{t("common.task")}</th><th>{t("webhooks.event")}</th><th>{t("common.status")}</th><th>{t("common.created")}</th><th>{t("common.action")}</th></tr></thead>
            <tbody>
              {rules.map((rule) => (
                <tr key={rule.id}>
                  <td><strong>{rule.name}</strong><small>{rule.id}</small></td>
                  <td>{rule.sourceName}</td>
                  <td><strong>{rule.targetType === "workflow" ? (rule.workflowName || t("workflows.workflow")) : (rule.taskName || t("common.task"))}</strong><small>{rule.targetType === "workflow" ? rule.workflowId : rule.taskId}</small></td>
                  <td><strong>{rule.eventType || "*"}</strong><small>{formatMatcher(rule.matcher)}</small></td>
                  <td><span className={`status-chip status-${rule.status}`}>{statusText(rule.status)}</span></td>
                  <td>{rule.createdAt}</td>
                  <td className="action-cell">
                    {canManage && rule.status !== "disabled" ? <button type="button" onClick={() => startRuleEdit(rule)}>{t("webhooks.editRule")}</button> : null}
                    {canManage && rule.status === "active" ? <button type="button" onClick={() => pauseRuleMutation.mutate(rule)}>{t("webhooks.pauseRule")}</button> : null}
                    {canManage && rule.status === "paused" ? <button type="button" onClick={() => resumeRuleMutation.mutate(rule)}>{t("webhooks.resumeRule")}</button> : null}
                    {canManage && rule.status !== "disabled" ? <button type="button" onClick={() => disableRuleMutation.mutate(rule)}>{t("webhooks.disableRule")}</button> : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </DataTable>
        {pauseRuleMutation.isError ? <p className="form-error">{pauseRuleMutation.error.message}</p> : null}
        {resumeRuleMutation.isError ? <p className="form-error">{resumeRuleMutation.error.message}</p> : null}
        {disableRuleMutation.isError ? <p className="form-error">{disableRuleMutation.error.message}</p> : null}
      </section>

      <section className="panel table-panel">
        <div className="panel-title"><h3>{t("webhooks.events")}</h3><span>{totalEvents} {t("common.total")}</span></div>
        <FilterToolbar>
          <select value={eventFilters.sourceId} onChange={(event) => setEventFilters((current) => ({ ...current, sourceId: event.target.value, page: 1 }))}>
            <option value="">{t("webhooks.allSources")}</option>
            {sources.map((source) => <option key={source.id} value={source.id}>{source.name}</option>)}
          </select>
          <select value={eventFilters.status} onChange={(event) => setEventFilters((current) => ({ ...current, status: event.target.value, page: 1 }))}>
            <option value="">{t("webhooks.allStatuses")}</option>
            <option value="received">{t("common.status.received")}</option>
            <option value="ignored">{t("common.status.ignored")}</option>
            <option value="triggered">{t("common.status.triggered")}</option>
            <option value="failed">{t("common.status.failed")}</option>
            <option value="rejected">{t("common.status.rejected")}</option>
          </select>
          <input
            placeholder={t("webhooks.deliveryId")}
            value={eventFilters.deliveryId}
            onChange={(event) => setEventFilters((current) => ({ ...current, deliveryId: event.target.value, page: 1 }))}
          />
          <input
            aria-label={t("webhooks.receivedFrom")}
            title={t("webhooks.receivedFrom")}
            type="datetime-local"
            value={eventFilters.receivedFrom}
            onChange={(event) => setEventFilters((current) => ({ ...current, receivedFrom: event.target.value, page: 1 }))}
          />
          <input
            aria-label={t("webhooks.receivedTo")}
            title={t("webhooks.receivedTo")}
            type="datetime-local"
            value={eventFilters.receivedTo}
            onChange={(event) => setEventFilters((current) => ({ ...current, receivedTo: event.target.value, page: 1 }))}
          />
          <button type="button" onClick={() => eventsQuery.refetch()}>{t("common.refresh")}</button>
        </FilterToolbar>
        <DataTable loading={eventsQuery.isLoading} empty={events.length === 0} emptyMessage={t("webhooks.emptyEvents")} error={eventsQuery.isError ? eventsQuery.error.message : null}>
          <table>
            <thead><tr><th>{t("common.source")}</th><th>{t("webhooks.event")}</th><th>{t("webhooks.deliveryId")}</th><th>{t("common.status")}</th><th>{t("webhooks.security")}</th><th>{t("common.created")}</th><th>{t("common.action")}</th></tr></thead>
            <tbody>
              {events.map((item) => (
                <tr key={item.id}>
                  <td><strong>{item.sourceName || "-"}</strong><small>{item.sourceId || item.id}</small></td>
                  <td><strong>{item.eventType || "*"}</strong><small>{item.payloadHash || "-"}</small></td>
                  <td><strong>{item.deliveryId || "-"}</strong><small>{item.nonce || "-"}</small></td>
                  <td><span className={`status-chip status-${item.status}`}>{statusText(item.status)}</span></td>
                  <td>
                    <strong>{formatSecurityStatus(item)}</strong>
                    <small>{formatSecurityDetail(item)}</small>
                  </td>
                  <td><strong>{item.receivedAt}</strong><small>{item.sourceTimestamp || item.remoteIp || "-"}</small></td>
                  <td><button type="button" onClick={() => setSelectedEventID((current) => current === item.id ? undefined : item.id)}>{selectedEventID === item.id ? t("common.hide") : t("common.view")}</button></td>
                </tr>
              ))}
            </tbody>
          </table>
        </DataTable>
        {selectedEventID && eventDetailQuery.data ? (
          <section className="event-detail">
            <div className="panel-title">
              <h3>{t("webhooks.eventDetail")}</h3>
              <span>{eventDetailQuery.data.id}</span>
            </div>
            {canManage ? (
              <div className="toolbar-row">
                <button type="button" disabled={replayEventMutation.isPending} onClick={() => replayEventMutation.mutate({ id: eventDetailQuery.data.id, simulateOnly: true })}>
                  {t("webhooks.replaySimulate")}
                </button>
                <button type="button" disabled={replayEventMutation.isPending} onClick={() => replayEventMutation.mutate({ id: eventDetailQuery.data.id, simulateOnly: false })}>
                  {t("webhooks.replayEvent")}
                </button>
              </div>
            ) : null}
            <div className="event-detail-grid">
              <div>
                <strong>{t("webhooks.security")}</strong>
                <small>{eventDetailQuery.data.signatureHeader || "-"}</small>
                <small>{formatSecurityStatus(eventDetailQuery.data)}</small>
                <small>{formatSecurityDetail(eventDetailQuery.data)}</small>
              </div>
              <div>
                <strong>{t("webhooks.headers")}</strong>
                {(eventDetailQuery.data.headers.length > 0 ? eventDetailQuery.data.headers : [{ key: "-", value: "-" }]).map((header) => (
                  <small key={`${header.key}-${header.value}`}>{header.key}: {header.value}</small>
                ))}
              </div>
            </div>
            <div className="event-payload">
              <strong>{t("webhooks.payload")}</strong>
              <JsonViewer value={eventDetailQuery.data.payload} emptyLabel={t("webhooks.noPayload")} />
            </div>
            <DataTable empty={eventDetailQuery.data.matches.length === 0}>
              <table>
                <thead><tr><th>{t("webhooks.rule")}</th><th>{t("common.status")}</th><th>{t("common.reason")}</th><th>{t("common.task")}</th><th>{t("common.created")}</th></tr></thead>
                <tbody>
                  {eventDetailQuery.data.matches.map((match) => (
                    <tr key={match.id}>
                      <td><strong>{match.ruleName || "-"}</strong><small>{match.ruleId || "-"}</small></td>
                      <td><span className={`status-chip status-${match.matched ? "success" : "rejected"}`}>{statusText(match.matched ? "matched" : "skipped")}</span></td>
                      <td>{formatMatchReason(match.reason)}</td>
                      <td>{match.workflowRunId ? <Link to="/workflows">{match.workflowRunId}</Link> : match.taskRunId ? <Link to={`/tasks/${match.taskRunId}`}>{match.taskRunId}</Link> : "-"}</td>
                      <td>{match.createdAt}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </DataTable>
          </section>
        ) : null}
        {eventDetailQuery.isError ? <p className="form-error">{eventDetailQuery.error.message}</p> : null}
        {replayEventMutation.isError ? <p className="form-error">{replayEventMutation.error.message}</p> : null}
        {replayEventMutation.data ? <p className="empty-state">{t("webhooks.replayResult").replace("{count}", String(replayEventMutation.data.matchedRules)).replace("{status}", replayEventMutation.data.status)}</p> : null}
        <PaginationBar total={totalEvents} page={eventFilters.page} pageSize={20} onPageChange={(page) => setEventFilters((current) => ({ ...current, page }))} />
      </section>
    </main>
  );
}

function MatcherSimulationPanel({ result, ruleName, t, formatReason }: { result: WebhookMatcherSimulationResult; ruleName?: string; t: (key: string) => string; formatReason: (reason?: string) => string }) {
  return (
    <div className="event-payload-stack">
      <div className="event-payload">
        <strong>{ruleName || result.ruleName || t("webhooks.matcherResult")}</strong>
        <small>{result.matched ? t("common.status.matched") : formatReason(result.reason) || t("common.status.failed")}</small>
      </div>
      <DataTable empty={result.conditions.length === 0} emptyMessage={t("webhooks.noConditionsEvaluated")}>
        <table>
          <thead>
            <tr>
              <th>{t("common.condition")}</th>
              <th>{t("common.status")}</th>
              <th>{t("common.actual")}</th>
              <th>{t("common.reason")}</th>
            </tr>
          </thead>
          <tbody>
            {result.conditions.map((condition, index) => (
              <tr key={`${condition.type}-${index}`}>
                <td><strong>{condition.type}</strong><small>{condition.key || condition.path || condition.value}</small></td>
                <td><span className={`status-chip status-${condition.matched ? "success" : "failed"}`}>{condition.matched ? t("common.status.matched") : t("common.status.failed")}</span></td>
                <td>{condition.actual || "-"}</td>
                <td>{formatReason(condition.reason)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </DataTable>
      <div className="event-payload">
        <strong>{t("webhooks.payloadUsed")}</strong>
        <JsonViewer value={result.payloadUsed} emptyLabel="-" />
      </div>
    </div>
  );
}

function buildRuleMatcher(drafts: MatcherDraft[]): { ok: true; matcher?: { conditions: WebhookMatcherCondition[] } } | { ok: false } {
  const conditions: WebhookMatcherCondition[] = [];
  for (const draft of drafts) {
    const type = draft.type;
    const key = draft.key.trim();
    const path = draft.path.trim();
    const value = draft.value.trim();
    if (!type && !key && !path && !value) {
      continue;
    }
    if (!type) {
      return { ok: false };
    }
    if (type === "header_equals") {
      if (!key || !value) {
        return { ok: false };
      }
      conditions.push({ type, key, value });
      continue;
    }
    if (type === "event_type_equals" || type === "ref_equals" || type === "branch_equals") {
      if (!value) {
        return { ok: false };
      }
      conditions.push({ type, value });
      continue;
    }
    if (!isPayloadMatcherType(type) || !path) {
      return { ok: false };
    }
    if (matcherTypeRequiresValue(type) && !value) {
      return { ok: false };
    }
    conditions.push({ type, path, value });
  }
  if (conditions.length === 0) {
    return { ok: true };
  }
  return { ok: true, matcher: { conditions } };
}

function matcherDraftsFromMatcher(matcher?: { conditions: Array<{ type: string; key?: string; path?: string; value: string }> }): MatcherDraft[] {
  return (matcher?.conditions ?? []).map((condition) => ({
    type: condition.type as MatcherDraft["type"],
    key: condition.key ?? "",
    path: condition.path ?? "",
    value: condition.value
  }));
}

function formatMatcher(matcher?: { conditions: Array<{ type: string; key?: string; path?: string; value: string }> }) {
  const conditions = matcher?.conditions ?? [];
  if (conditions.length === 0) {
    return "-";
  }
  return conditions.map((condition) => {
    if (condition.type === "header_equals") {
      return `${condition.type}:${condition.key}=${condition.value}`;
    }
    if (condition.type === "event_type_equals" || condition.type === "ref_equals" || condition.type === "branch_equals") {
      return `${condition.type}=${condition.value}`;
    }
    if (condition.type === "payload_exists") {
      return `${condition.type}:${condition.path}`;
    }
    return `${condition.type}:${condition.path}=${condition.value}`;
  }).join(" AND ");
}

function matcherValuePlaceholder(type: MatcherDraft["type"]) {
  if (type === "event_type_equals") {
    return "push";
  }
  if (type === "ref_equals") {
    return "refs/heads/main";
  }
  if (type === "payload_regex") {
    return "^commit-[0-9]+$";
  }
  if (type === "payload_not_equals") {
    return "prod";
  }
  return "main";
}

function matcherTypeRequiresValue(type: MatcherDraft["type"] | undefined) {
  return Boolean(type) && type !== "payload_exists";
}

function isPayloadMatcherType(type: MatcherDraft["type"] | undefined) {
  return type === "payload_equals" || type === "payload_contains" || type === "payload_not_equals" || type === "payload_exists" || type === "payload_regex";
}

function isEventMatcherType(type: MatcherDraft["type"] | undefined) {
  return type === "event_type_equals" || type === "ref_equals" || type === "branch_equals";
}

function parseJSONMap(value: string) {
  try {
    const parsed = JSON.parse(value) as Record<string, unknown>;
    if (!parsed || Array.isArray(parsed)) {
      return null;
    }
    const headers: Record<string, string> = {};
    for (const [key, item] of Object.entries(parsed)) {
      headers[key] = typeof item === "string" ? item : JSON.stringify(item);
    }
    return headers;
  } catch {
    return null;
  }
}
