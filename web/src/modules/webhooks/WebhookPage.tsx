import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import { createWebhookRule, createWebhookSource, disableWebhookRule, disableWebhookSource, getWebhookEvent, listWebhookEvents, listWebhookRules, listWebhookSources, pauseWebhookRule, pauseWebhookSource, resumeWebhookRule, resumeWebhookSource, updateWebhookRule } from "../../api/webhooks";
import { listTasks } from "../../api/tasks";
import { API_BASE_URL } from "../../api/request";
import type { WebhookEvent, WebhookMatcherCondition, WebhookRule, WebhookSource } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";
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

export function WebhookPage() {
  const user = useAuthStore((state) => state.user);
  const t = useLanguageStore((state) => state.t);
  const queryClient = useQueryClient();
  const canManage = hasPermission(user, "webhook:manage");
  const [sourceForm, setSourceForm] = useState({ name: "", sourceType: "custom" });
  const [ruleForm, setRuleForm] = useState({
    sourceId: "",
    taskId: "",
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
  const createSourceMutation = useMutation({
    mutationFn: createWebhookSource,
    onSuccess: (source) => {
      setIssuedToken(source.token ?? "");
      setIssuedSigningSecret(source.signingSecret ?? "");
      setSourceForm({ name: "", sourceType: "custom" });
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
        taskId: "",
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
  const sources = useMemo(() => sourcesQuery.data ?? [], [sourcesQuery.data]);
  const rules = useMemo(() => rulesQuery.data ?? [], [rulesQuery.data]);
  const events = useMemo(() => eventsQuery.data?.items ?? [], [eventsQuery.data]);
  const totalEvents = eventsQuery.data?.total ?? 0;
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
            ...(patch.type === "payload_equals" || patch.type === "payload_contains" ? { key: "" } : {}),
            ...(patch.type === "event_type_equals" || patch.type === "ref_equals" || patch.type === "branch_equals" ? { key: "", path: "" } : {}),
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
      taskId: rule.taskId,
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
      taskId: "",
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
  const formatPayload = (payload?: string) => {
    const value = payload?.trim() ?? "";
    if (!value) {
      return t("webhooks.noPayload");
    }
    try {
      const parsed = JSON.parse(value) as unknown;
      if (typeof parsed === "string") {
        return parsed;
      }
      return JSON.stringify(parsed, null, 2);
    } catch {
      return value;
    }
  };

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
              <strong>{triggerURL}</strong>
              <span>{issuedSigningSecret || t("webhooks.secretMissing")}</span>
            </div>
          ) : null}
          {createSourceMutation.isError ? <p className="form-error">{createSourceMutation.error.message}</p> : null}
        </section>
      ) : null}

      <section className="panel table-panel">
        <div className="panel-title"><h3>{t("webhooks.sources")}</h3><span>{sources.length} {t("common.total")}</span></div>
        <div className="data-table">
          <table>
            <thead><tr><th>{t("common.name")}</th><th>{t("common.type")}</th><th>{t("common.status")}</th><th>{t("common.created")}</th><th>{t("common.action")}</th></tr></thead>
            <tbody>
              {sources.map((source) => (
                <tr key={source.id}>
                  <td><strong>{source.name}</strong><small>{source.id}</small></td>
                  <td>{source.sourceType}</td>
                  <td><span className={`status-chip status-${source.status}`}>{source.status}</span></td>
                  <td><strong>{source.createdAt}</strong><small>{source.lastReceivedAt || source.createdBy || "-"}</small></td>
                  <td className="action-cell">
                    {canManage && source.status === "active" ? <button type="button" onClick={() => pauseSourceMutation.mutate(source)}>{t("webhooks.pauseSource")}</button> : null}
                    {canManage && source.status === "paused" ? <button type="button" onClick={() => resumeSourceMutation.mutate(source)}>{t("webhooks.resumeSource")}</button> : null}
                    {canManage && source.status !== "disabled" ? <button type="button" onClick={() => disableSourceMutation.mutate(source)}>{t("webhooks.disableSource")}</button> : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!sourcesQuery.isLoading && sources.length === 0 ? <p className="empty-state">{t("webhooks.emptySources")}</p> : null}
        {sourcesQuery.isError ? <p className="form-error">{sourcesQuery.error.message}</p> : null}
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
              taskId: ruleForm.taskId,
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
              {t("common.task")}
              <select value={ruleForm.taskId} onChange={(event) => setRuleForm({ ...ruleForm, taskId: event.target.value })} required disabled={Boolean(editingRuleID)}>
                <option value="">{t("webhooks.selectTask")}</option>
                {taskOptions.map((task) => <option key={task.taskId} value={task.taskId}>{task.name}</option>)}
              </select>
            </label>
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
                  {draft.type === "payload_equals" || draft.type === "payload_contains" ? (
                    <label>{t("webhooks.matcherPath")}<input value={draft.path} onChange={(event) => {
                      setMatcherError("");
                      updateMatcherDraft(index, { path: event.target.value });
                    }} placeholder="repository.name" /></label>
                  ) : null}
                  {draft.type ? (
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

      <section className="panel table-panel">
        <div className="panel-title"><h3>{t("webhooks.rules")}</h3><span>{rules.length} {t("common.total")}</span></div>
        <div className="data-table">
          <table>
            <thead><tr><th>{t("common.name")}</th><th>{t("common.source")}</th><th>{t("common.task")}</th><th>{t("webhooks.event")}</th><th>{t("common.status")}</th><th>{t("common.created")}</th><th>{t("common.action")}</th></tr></thead>
            <tbody>
              {rules.map((rule) => (
                <tr key={rule.id}>
                  <td><strong>{rule.name}</strong><small>{rule.id}</small></td>
                  <td>{rule.sourceName}</td>
                  <td><strong>{rule.taskName}</strong><small>{rule.taskId}</small></td>
                  <td><strong>{rule.eventType || "*"}</strong><small>{formatMatcher(rule.matcher)}</small></td>
                  <td><span className={`status-chip status-${rule.status}`}>{rule.status}</span></td>
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
        </div>
        {!rulesQuery.isLoading && rules.length === 0 ? <p className="empty-state">{t("webhooks.emptyRules")}</p> : null}
        {pauseRuleMutation.isError ? <p className="form-error">{pauseRuleMutation.error.message}</p> : null}
        {resumeRuleMutation.isError ? <p className="form-error">{resumeRuleMutation.error.message}</p> : null}
        {disableRuleMutation.isError ? <p className="form-error">{disableRuleMutation.error.message}</p> : null}
      </section>

      <section className="panel table-panel">
        <div className="panel-title"><h3>{t("webhooks.events")}</h3><span>{totalEvents} {t("common.total")}</span></div>
        <div className="toolbar-row">
          <select value={eventFilters.sourceId} onChange={(event) => setEventFilters((current) => ({ ...current, sourceId: event.target.value, page: 1 }))}>
            <option value="">{t("webhooks.allSources")}</option>
            {sources.map((source) => <option key={source.id} value={source.id}>{source.name}</option>)}
          </select>
          <select value={eventFilters.status} onChange={(event) => setEventFilters((current) => ({ ...current, status: event.target.value, page: 1 }))}>
            <option value="">{t("webhooks.allStatuses")}</option>
            <option value="received">received</option>
            <option value="ignored">ignored</option>
            <option value="triggered">triggered</option>
            <option value="failed">failed</option>
            <option value="rejected">rejected</option>
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
        </div>
        <div className="data-table">
          <table>
            <thead><tr><th>{t("common.source")}</th><th>{t("webhooks.event")}</th><th>{t("webhooks.deliveryId")}</th><th>{t("common.status")}</th><th>{t("webhooks.security")}</th><th>{t("common.created")}</th><th>{t("common.action")}</th></tr></thead>
            <tbody>
              {events.map((item) => (
                <tr key={item.id}>
                  <td><strong>{item.sourceName || "-"}</strong><small>{item.sourceId || item.id}</small></td>
                  <td><strong>{item.eventType || "*"}</strong><small>{item.payloadHash || "-"}</small></td>
                  <td><strong>{item.deliveryId || "-"}</strong><small>{item.nonce || "-"}</small></td>
                  <td><span className={`status-chip status-${item.status}`}>{item.status}</span></td>
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
        </div>
        {!eventsQuery.isLoading && events.length === 0 ? <p className="empty-state">{t("webhooks.emptyEvents")}</p> : null}
        {selectedEventID && eventDetailQuery.data ? (
          <section className="event-detail">
            <div className="panel-title">
              <h3>{t("webhooks.eventDetail")}</h3>
              <span>{eventDetailQuery.data.id}</span>
            </div>
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
              <pre className="json-block">{formatPayload(eventDetailQuery.data.payload)}</pre>
            </div>
            <div className="data-table">
              <table>
                <thead><tr><th>{t("webhooks.rule")}</th><th>{t("common.status")}</th><th>{t("common.reason")}</th><th>{t("common.task")}</th><th>{t("common.created")}</th></tr></thead>
                <tbody>
                  {eventDetailQuery.data.matches.map((match) => (
                    <tr key={match.id}>
                      <td><strong>{match.ruleName || "-"}</strong><small>{match.ruleId || "-"}</small></td>
                      <td><span className={`status-chip status-${match.matched ? "success" : "rejected"}`}>{match.matched ? "matched" : "skipped"}</span></td>
                      <td>{formatMatchReason(match.reason)}</td>
                      <td>{match.taskRunId ? <Link to={`/tasks/${match.taskRunId}`}>{match.taskRunId}</Link> : "-"}</td>
                      <td>{match.createdAt}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        ) : null}
        {eventDetailQuery.isError ? <p className="form-error">{eventDetailQuery.error.message}</p> : null}
        <div className="toolbar-row">
          <span>{totalEvents} {t("common.total")}</span>
          <button type="button" disabled={eventFilters.page <= 1} onClick={() => setEventFilters((current) => ({ ...current, page: Math.max(1, current.page - 1) }))}>{t("common.previous")}</button>
          <span>{t("common.page").replace("{page}", String(eventFilters.page))}</span>
          <button
            type="button"
            disabled={eventFilters.page * 20 >= totalEvents}
            onClick={() => setEventFilters((current) => ({ ...current, page: current.page + 1 }))}
          >
            {t("common.next")}
          </button>
        </div>
        {eventsQuery.isError ? <p className="form-error">{eventsQuery.error.message}</p> : null}
      </section>
    </main>
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
    if (!type || !value) {
      return { ok: false };
    }
    if (type === "header_equals") {
      if (!key) {
        return { ok: false };
      }
      conditions.push({ type, key, value });
      continue;
    }
    if (type === "event_type_equals" || type === "ref_equals" || type === "branch_equals") {
      conditions.push({ type, value });
      continue;
    }
    if (!path) {
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
  return "main";
}
