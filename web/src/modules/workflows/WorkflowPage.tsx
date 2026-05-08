import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";

import {
  approveWorkflowNode,
  cancelWorkflowRun,
  copyWorkflow,
  createWorkflow,
  disableWorkflow,
  getWorkflow,
  getWorkflowRun,
  listWorkflowVersions,
  listWorkflowRuns,
  listWorkflows,
  publishWorkflow,
  rejectWorkflowNode,
  retryWorkflowNode,
  retryWorkflowRun,
  runWorkflow,
  updateWorkflow
} from "../../api/workflows";
import type { WorkflowDefinitionSummary, WorkflowRunSummary } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";
import { ConfirmDialog } from "../../shared/components/ConfirmDialog";
import { DataTable } from "../../shared/components/DataTable";
import { FilterToolbar } from "../../shared/components/FilterToolbar";
import { JsonViewer } from "../../shared/components/JsonViewer";
import { PaginationBar } from "../../shared/components/PaginationBar";
import { Timeline, type TimelineItem } from "../../shared/components/Timeline";
import { useToast } from "../../shared/components/ToastProvider";
import { hasPermission } from "../auth/permissions";
import { useAuthStore } from "../auth/store";

type Translate = (key: string, params?: Record<string, string | number>) => string;

function buildSampleDefinition(t: Translate) {
  return JSON.stringify({
    nodes: [
      { id: "collect", type: "task", name: t("workflows.sampleCollectDiagnostics"), timeoutSeconds: 900, config: { taskId: "replace-with-task-id" } },
      { id: "gate", type: "condition", name: t("workflows.sampleProdGate"), config: { path: "environment", operator: "equals", value: "prod", onFalse: "skip" } },
      { id: "callback", type: "webhook-call", name: t("workflows.samplePostCallback"), config: { url: "https://example.com/hooks/${payload.service}", method: "POST", bodyPath: "payload", headers: { "X-Env": "${payload.environment}" } } },
      { id: "cooldown", type: "wait", name: t("workflows.sampleCooldown"), config: { seconds: 30 } },
      { id: "notify", type: "notification", name: t("workflows.sampleNotifyOwner"), config: { title: t("workflows.sampleCompletedTitle"), channelId: "optional-channel-id" } }
    ],
    edges: [
      { from: "collect", to: "gate" },
      { from: "gate", to: "callback" },
      { from: "callback", to: "cooldown" },
      { from: "cooldown", to: "notify" }
    ],
    maxParallel: 2,
    failurePolicy: "skip_downstream"
  }, null, 2);
}

type PendingAction =
  | { type: "publish"; workflow: WorkflowDefinitionSummary }
  | { type: "disable"; workflow: WorkflowDefinitionSummary }
  | { type: "copy"; workflow: WorkflowDefinitionSummary }
  | { type: "run"; workflow: WorkflowDefinitionSummary }
  | { type: "cancel"; run: WorkflowRunSummary }
  | { type: "retry"; run: WorkflowRunSummary }
  | { type: "retryNode"; runId: string; nodeId: string }
  | { type: "approve"; runId: string; nodeId: string }
  | { type: "reject"; runId: string; nodeId: string };

export function WorkflowPage() {
  const t = useLanguageStore((state) => state.t);
  const user = useAuthStore((state) => state.user);
  const { notify } = useToast();
  const queryClient = useQueryClient();
  const sampleDefinition = useMemo(() => buildSampleDefinition(t), [t]);
  const [tab, setTab] = useState<"definitions" | "runs">("definitions");
  const [keyword, setKeyword] = useState("");
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(1);
  const [runPage, setRunPage] = useState(1);
  const [selectedWorkflowId, setSelectedWorkflowId] = useState("");
  const [selectedRunId, setSelectedRunId] = useState("");
  const [pendingAction, setPendingAction] = useState<PendingAction | null>(null);
  const [form, setForm] = useState(() => ({ name: "", description: "", definition: buildSampleDefinition(t) }));
  const [runInput, setRunInput] = useState("{}");
  const pageSize = 20;
  const canWrite = hasPermission(user, "workflow:manage");
  const canExecute = hasPermission(user, "workflow:execute");
  const canCancel = hasPermission(user, "workflow:manage");
  const workflowQuery = useQuery({
    queryKey: ["workflows", keyword, status, page],
    queryFn: () => listWorkflows({ keyword, status, page, pageSize })
  });
  const selectedWorkflowQuery = useQuery({
    queryKey: ["workflow", selectedWorkflowId],
    enabled: Boolean(selectedWorkflowId),
    queryFn: () => getWorkflow(selectedWorkflowId)
  });
  const versionsQuery = useQuery({
    queryKey: ["workflowVersions", selectedWorkflowId],
    enabled: Boolean(selectedWorkflowId),
    queryFn: () => listWorkflowVersions(selectedWorkflowId)
  });
  const runsQuery = useQuery({
    queryKey: ["workflowRuns", keyword, status, runPage],
    queryFn: () => listWorkflowRuns({ keyword, status, page: runPage, pageSize })
  });
  const selectedRunQuery = useQuery({
    queryKey: ["workflowRun", selectedRunId],
    enabled: Boolean(selectedRunId),
    queryFn: () => getWorkflowRun(selectedRunId)
  });
  const workflows = useMemo(() => workflowQuery.data?.items ?? [], [workflowQuery.data]);
  const runs = useMemo(() => runsQuery.data?.items ?? [], [runsQuery.data]);
  const selectedWorkflow = selectedWorkflowQuery.data;
  const selectedRun = selectedRunQuery.data;
  const statusText = (value: string) => t(`common.status.${value}`);

  useEffect(() => {
    if (selectedWorkflow) {
      setForm({
        name: selectedWorkflow.name,
        description: selectedWorkflow.description ?? "",
        definition: formatJSON(selectedWorkflow.definition)
      });
    }
  }, [selectedWorkflow]);

  const saveMutation = useMutation({
    mutationFn: () => selectedWorkflowId ? updateWorkflow(selectedWorkflowId, form) : createWorkflow(form),
    onSuccess: (item) => {
      notify(t("workflows.savedToast"), "success");
      setSelectedWorkflowId(item.id);
      queryClient.invalidateQueries({ queryKey: ["workflows"] });
      queryClient.invalidateQueries({ queryKey: ["workflow", item.id] });
    }
  });
  const publishMutation = useMutation({
    mutationFn: (workflow: WorkflowDefinitionSummary) => publishWorkflow(workflow.id),
    onSuccess: (item) => {
      notify(t("workflows.publishedToast"), "success");
      queryClient.invalidateQueries({ queryKey: ["workflows"] });
      queryClient.invalidateQueries({ queryKey: ["workflow", item.id] });
    }
  });
  const disableMutation = useMutation({
    mutationFn: (workflow: WorkflowDefinitionSummary) => disableWorkflow(workflow.id),
    onSuccess: (item) => {
      notify(t("workflows.disabledToast"), "success");
      queryClient.invalidateQueries({ queryKey: ["workflows"] });
      queryClient.invalidateQueries({ queryKey: ["workflow", item.id] });
    }
  });
  const copyMutation = useMutation({
    mutationFn: (workflow: WorkflowDefinitionSummary) => copyWorkflow(workflow.id),
    onSuccess: (item) => {
      notify(t("workflows.copiedToast"), "success");
      setSelectedWorkflowId(item.id);
      queryClient.invalidateQueries({ queryKey: ["workflows"] });
    }
  });
  const runMutation = useMutation({
    mutationFn: ({ workflow, input }: { workflow: WorkflowDefinitionSummary; input: string }) => runWorkflow(workflow.id, { triggerType: "manual", input: input.trim() }),
    onSuccess: (run) => {
      notify(t("workflows.runCreatedToast"), "success");
      setTab("runs");
      setSelectedRunId(run.id);
      queryClient.invalidateQueries({ queryKey: ["workflowRuns"] });
      queryClient.invalidateQueries({ queryKey: ["workflowRun", run.id] });
    }
  });
  const cancelMutation = useMutation({
    mutationFn: (run: WorkflowRunSummary) => cancelWorkflowRun(run.id, t("workflows.uiCancelReason")),
    onSuccess: (run) => {
      notify(t("workflows.canceledToast"), "success");
      queryClient.invalidateQueries({ queryKey: ["workflowRuns"] });
      queryClient.invalidateQueries({ queryKey: ["workflowRun", run.id] });
    }
  });
  const retryMutation = useMutation({
    mutationFn: (run: WorkflowRunSummary) => retryWorkflowRun(run.id),
    onSuccess: (run) => {
      notify(t("workflows.runCreatedToast"), "success");
      setTab("runs");
      setSelectedRunId(run.id);
      queryClient.invalidateQueries({ queryKey: ["workflowRuns"] });
      queryClient.invalidateQueries({ queryKey: ["workflowRun", run.id] });
    }
  });
  const approveMutation = useMutation({
    mutationFn: ({ runId, nodeId }: { runId: string; nodeId: string }) => approveWorkflowNode(runId, nodeId, t("workflows.uiApproveReason")),
    onSuccess: (run) => {
      notify(t("workflows.savedToast"), "success");
      setSelectedRunId(run.id);
      queryClient.invalidateQueries({ queryKey: ["workflowRuns"] });
      queryClient.invalidateQueries({ queryKey: ["workflowRun", run.id] });
    }
  });
  const rejectMutation = useMutation({
    mutationFn: ({ runId, nodeId }: { runId: string; nodeId: string }) => rejectWorkflowNode(runId, nodeId, t("workflows.uiRejectReason")),
    onSuccess: (run) => {
      notify(t("workflows.canceledToast"), "success");
      setSelectedRunId(run.id);
      queryClient.invalidateQueries({ queryKey: ["workflowRuns"] });
      queryClient.invalidateQueries({ queryKey: ["workflowRun", run.id] });
    }
  });
  const retryNodeMutation = useMutation({
    mutationFn: ({ runId, nodeId }: { runId: string; nodeId: string }) => retryWorkflowNode(runId, nodeId),
    onSuccess: (run) => {
      notify(t("workflows.nodeRetryToast"), "success");
      setSelectedRunId(run.id);
      queryClient.invalidateQueries({ queryKey: ["workflowRuns"] });
      queryClient.invalidateQueries({ queryKey: ["workflowRun", run.id] });
    }
  });
  const currentError = saveMutation.error ?? publishMutation.error ?? disableMutation.error ?? copyMutation.error ?? runMutation.error ?? cancelMutation.error ?? retryMutation.error ?? retryNodeMutation.error ?? approveMutation.error ?? rejectMutation.error;

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">{t("workflows.eyebrow")}</p>
          <h1>{t("workflows.title")}</h1>
        </div>
        <div className="segmented">
          <button type="button" className={tab === "definitions" ? "active" : ""} onClick={() => setTab("definitions")}>{t("workflows.definitions")}</button>
          <button type="button" className={tab === "runs" ? "active" : ""} onClick={() => setTab("runs")}>{t("workflows.runs")}</button>
        </div>
      </section>

      {tab === "definitions" ? (
        <section className="workflow-grid">
          <section className="panel table-panel">
            <FilterToolbar>
              <input placeholder={t("workflows.search")} value={keyword} onChange={(event) => { setKeyword(event.target.value); setPage(1); }} />
              <select value={status} onChange={(event) => { setStatus(event.target.value); setPage(1); }}>
                <option value="">{t("common.allStatus")}</option>
                <option value="draft">{statusText("draft")}</option>
                <option value="active">{statusText("active")}</option>
                <option value="disabled">{statusText("disabled")}</option>
              </select>
              <button type="button" onClick={() => workflowQuery.refetch()}>{t("common.refresh")}</button>
            </FilterToolbar>
            <DataTable loading={workflowQuery.isLoading} empty={workflows.length === 0} emptyMessage={t("workflows.empty")} error={workflowQuery.isError ? workflowQuery.error.message : null}>
              <table>
                <thead>
                  <tr>
                    <th>{t("common.name")}</th>
                    <th>{t("common.status")}</th>
                    <th>{t("workflows.graph")}</th>
                    <th>{t("common.version")}</th>
                    <th>{t("common.action")}</th>
                  </tr>
                </thead>
                <tbody>
                  {workflows.map((workflow) => (
                    <tr key={workflow.id}>
                      <td><strong>{workflow.name}</strong><small>{workflow.description || workflow.id}</small></td>
                      <td><span className={`status-chip status-${workflow.status}`}>{statusText(workflow.status)}</span></td>
                      <td><strong>{workflow.nodeCount} / {workflow.edgeCount}</strong><small>{t("workflows.nodesEdges")}</small></td>
                      <td><strong>v{workflow.version}</strong><small>{workflow.updatedAt}</small></td>
                      <td className="action-cell">
                        <button type="button" onClick={() => setSelectedWorkflowId(workflow.id)}>{t("common.edit")}</button>
                        {canWrite ? <button type="button" disabled={publishMutation.isPending} onClick={() => setPendingAction({ type: "publish", workflow })}>{t("workflows.publish")}</button> : null}
                        {canWrite ? <button type="button" disabled={copyMutation.isPending} onClick={() => setPendingAction({ type: "copy", workflow })}>{t("workflows.copy")}</button> : null}
                        {canWrite && workflow.status !== "disabled" ? <button type="button" disabled={disableMutation.isPending} onClick={() => setPendingAction({ type: "disable", workflow })}>{t("workflows.disable")}</button> : null}
                        {canExecute ? <button type="button" disabled={workflow.status !== "active" || runMutation.isPending} onClick={() => { setRunInput("{}"); setPendingAction({ type: "run", workflow }); }}>{t("workflows.run")}</button> : null}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </DataTable>
            <PaginationBar total={workflowQuery.data?.total ?? 0} page={page} pageSize={pageSize} onPageChange={setPage} />
          </section>

          {canWrite ? (
            <section className="panel form-panel">
              <div className="panel-title">
                <h3>{selectedWorkflowId ? t("workflows.edit") : t("workflows.create")}</h3>
                <button type="button" className="ghost-button" onClick={() => { setSelectedWorkflowId(""); setForm({ name: "", description: "", definition: sampleDefinition }); }}>
                  {t("workflows.newDraft")}
                </button>
              </div>
              <form className="editor-form" onSubmit={(event) => { event.preventDefault(); saveMutation.mutate(); }}>
                <label>
                  {t("common.name")}
                  <input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required />
                </label>
                <label>
                  {t("workflows.description")}
                  <input value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} />
                </label>
                <label>
                  {t("workflows.definitionJson")}
                  <textarea className="code-input workflow-json-input" value={form.definition} onChange={(event) => setForm({ ...form, definition: event.target.value })} required />
                </label>
                <button type="submit" disabled={saveMutation.isPending}>{t("common.save")}</button>
              </form>
              {selectedWorkflow ? <JsonViewer value={selectedWorkflow.definition} /> : null}
              {selectedWorkflowId ? (
                <div className="event-payload">
                  <strong>{t("workflows.versions")}</strong>
                  <DataTable loading={versionsQuery.isLoading} empty={(versionsQuery.data ?? []).length === 0} emptyMessage={t("common.empty")} error={versionsQuery.isError ? versionsQuery.error.message : null}>
                    <table>
                      <thead>
                        <tr>
                          <th>{t("common.version")}</th>
                          <th>{t("common.status")}</th>
                          <th>{t("workflows.versionHash")}</th>
                          <th>{t("common.created")}</th>
                        </tr>
                      </thead>
                      <tbody>
                        {(versionsQuery.data ?? []).map((version) => (
                          <tr key={version.id}>
                            <td>v{version.version}</td>
                            <td><span className={`status-chip status-${version.status}`}>{statusText(version.status)}</span></td>
                            <td><small>{version.definitionHash.slice(0, 12)}</small></td>
                            <td><strong>{version.createdAt}</strong><small>{version.publishedAt || version.createdBy || "-"}</small></td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </DataTable>
                </div>
              ) : null}
            </section>
          ) : null}
        </section>
      ) : (
        <section className="workflow-grid">
          <section className="panel table-panel">
            <FilterToolbar>
              <input placeholder={t("workflows.searchRuns")} value={keyword} onChange={(event) => { setKeyword(event.target.value); setRunPage(1); }} />
              <select value={status} onChange={(event) => { setStatus(event.target.value); setRunPage(1); }}>
                <option value="">{t("common.allStatus")}</option>
                <option value="pending">{statusText("pending")}</option>
                <option value="running">{statusText("running")}</option>
                <option value="success">{statusText("success")}</option>
                <option value="failed">{statusText("failed")}</option>
                <option value="canceled">{statusText("canceled")}</option>
              </select>
              <button type="button" onClick={() => runsQuery.refetch()}>{t("common.refresh")}</button>
            </FilterToolbar>
            <DataTable loading={runsQuery.isLoading} empty={runs.length === 0} emptyMessage={t("workflows.emptyRuns")} error={runsQuery.isError ? runsQuery.error.message : null}>
              <table>
                <thead>
                  <tr>
                    <th>{t("workflows.run")}</th>
                    <th>{t("workflows.workflow")}</th>
                    <th>{t("common.status")}</th>
                    <th>{t("workflows.progress")}</th>
                    <th>{t("common.createdAt")}</th>
                    <th>{t("common.action")}</th>
                  </tr>
                </thead>
                <tbody>
                  {runs.map((run) => (
                    <tr key={run.id}>
                      <td><strong>{run.id}</strong><small>{run.triggerType}</small></td>
                      <td><strong>{run.workflowName}</strong><small>v{run.workflowVersion}</small></td>
                      <td><span className={`status-chip status-${run.status}`}>{statusText(run.status)}</span></td>
                      <td><strong>{run.successNodes}/{run.totalNodes}</strong><small>{run.failedNodes} {statusText("failed")}</small></td>
                      <td><strong>{run.createdAt}</strong><small>{run.finishedAt || run.startedAt || run.queuedAt || "-"}</small></td>
                      <td className="action-cell">
                        <button type="button" onClick={() => setSelectedRunId(run.id)}>{t("common.view")}</button>
                        {canCancel && cancelable(run.status) ? <button type="button" disabled={cancelMutation.isPending} onClick={() => setPendingAction({ type: "cancel", run })}>{t("tasks.cancel")}</button> : null}
                        {canExecute && retryable(run.status) ? <button type="button" disabled={retryMutation.isPending} onClick={() => setPendingAction({ type: "retry", run })}>{t("notifications.retryDelivery")}</button> : null}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </DataTable>
            <PaginationBar total={runsQuery.data?.total ?? 0} page={runPage} pageSize={pageSize} onPageChange={setRunPage} />
          </section>

          <section className="panel">
            <div className="panel-title">
              <h3>{selectedRun?.workflowName ?? t("workflows.runDetail")}</h3>
              <span>{selectedRun?.status ? statusText(selectedRun.status) : t("workflows.selectRun")}</span>
            </div>
            {selectedRunQuery.isLoading ? <p className="empty-state">{t("common.loading")}</p> : null}
            {selectedRun ? (
              <>
                <div className="event-detail-grid">
                  <div><strong>{t("workflows.totalNodes")}</strong><span>{selectedRun.totalNodes}</span></div>
                  <div><strong>{statusText("success")}</strong><span>{selectedRun.successNodes}</span></div>
                  <div><strong>{statusText("failed")}</strong><span>{selectedRun.failedNodes}</span></div>
                  <div><strong>{statusText("skipped")}</strong><span>{selectedRun.skippedNodes}</span></div>
                </div>
                <DataTable empty={selectedRun.nodes.length === 0} emptyMessage={t("workflows.emptyNodes")}>
                  <table>
                    <thead>
                      <tr>
                        <th>{t("workflows.node")}</th>
                        <th>{t("common.type")}</th>
                        <th>{t("common.status")}</th>
                        <th>{t("workflows.links")}</th>
                      </tr>
                    </thead>
                    <tbody>
                      {selectedRun.nodes.map((node) => (
                        <tr key={node.id}>
                          <td>
                            <strong>{node.nodeName || node.nodeId}</strong>
                            <small>{node.nodeId}</small>
                            {node.errorMessage ? <small>{node.errorMessage}</small> : null}
                            {node.input ? <small>{truncateJSON(node.input)}</small> : null}
                            {node.output ? <small>{truncateJSON(node.output)}</small> : null}
                          </td>
                          <td>{node.nodeType}</td>
                          <td><span className={`status-chip status-${node.status}`}>{statusText(node.status)}</span></td>
                          <td className="action-cell">
                            {node.taskRunId ? <Link to={`/tasks/${node.taskRunId}`}>{node.taskRunId}</Link> : "-"}
                            {canExecute && retryableNode(node.status) ? <button type="button" disabled={retryNodeMutation.isPending} onClick={() => setPendingAction({ type: "retryNode", runId: selectedRun.id, nodeId: node.nodeId })}>{t("workflows.retryNode")}</button> : null}
                            {canExecute && node.nodeType === "approval" && node.status === "running" ? <button type="button" onClick={() => setPendingAction({ type: "approve", runId: selectedRun.id, nodeId: node.nodeId })}>{t("incidents.acknowledge")}</button> : null}
                            {canExecute && node.nodeType === "approval" && node.status === "running" ? <button type="button" onClick={() => setPendingAction({ type: "reject", runId: selectedRun.id, nodeId: node.nodeId })}>{t("common.cancel")}</button> : null}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </DataTable>
                <Timeline
                  items={selectedRun.events.map((event): TimelineItem => ({
                    id: event.id,
                    title: event.eventType,
                    time: event.createdAt,
                    description: event.message || event.nodeId || event.eventType,
                    tone: event.eventType === "canceled" ? "danger" : "default"
                  }))}
                />
                <div className="event-payload-stack">
                  <div className="event-payload">
                    <strong>{t("workflows.runInput")}</strong>
                    <JsonViewer value={selectedRun.input} emptyLabel="-" />
                  </div>
                  <div className="event-payload">
                    <strong>{t("workflows.runOutput")}</strong>
                    <JsonViewer value={selectedRun.output} emptyLabel="-" />
                  </div>
                  <div className="event-payload">
                    <strong>Definition</strong>
                    <JsonViewer value={selectedRun.definition} emptyLabel="-" />
                  </div>
                  {selectedRun.events.filter((event) => event.payload).map((event) => (
                    <div className="event-payload" key={`workflow-event-${event.id}`}>
                      <strong>{event.eventType}</strong>
                      <JsonViewer value={event.payload} emptyLabel="-" />
                    </div>
                  ))}
                </div>
              </>
            ) : <p className="empty-state">{t("workflows.selectRunHint")}</p>}
            {selectedRunQuery.isError ? <p className="form-error">{selectedRunQuery.error.message}</p> : null}
          </section>
        </section>
      )}

      {currentError ? <p className="form-error">{currentError.message}</p> : null}
      <ConfirmDialog
        open={Boolean(pendingAction)}
        title={confirmTitle(pendingAction, t)}
        message={pendingAction?.type === "run" ? (
          <label>
            {t("workflows.runInputJson")}
            <textarea className="code-input" value={runInput} onChange={(event) => setRunInput(event.target.value)} />
          </label>
        ) : null}
        confirmLabel={t("common.confirm")}
        cancelLabel={t("common.cancel")}
        danger={pendingAction?.type === "cancel"}
        onCancel={() => setPendingAction(null)}
        onConfirm={() => {
          if (pendingAction?.type === "publish") publishMutation.mutate(pendingAction.workflow);
          if (pendingAction?.type === "disable") disableMutation.mutate(pendingAction.workflow);
          if (pendingAction?.type === "copy") copyMutation.mutate(pendingAction.workflow);
          if (pendingAction?.type === "run") runMutation.mutate({ workflow: pendingAction.workflow, input: runInput });
          if (pendingAction?.type === "cancel") cancelMutation.mutate(pendingAction.run);
          if (pendingAction?.type === "retry") retryMutation.mutate(pendingAction.run);
          if (pendingAction?.type === "retryNode") retryNodeMutation.mutate({ runId: pendingAction.runId, nodeId: pendingAction.nodeId });
          if (pendingAction?.type === "approve") approveMutation.mutate({ runId: pendingAction.runId, nodeId: pendingAction.nodeId });
          if (pendingAction?.type === "reject") rejectMutation.mutate({ runId: pendingAction.runId, nodeId: pendingAction.nodeId });
          setPendingAction(null);
        }}
      />
    </main>
  );
}

function formatJSON(value: string) {
  try {
    return JSON.stringify(JSON.parse(value), null, 2);
  } catch {
    return value;
  }
}

function truncateJSON(value?: string) {
  if (!value) return "";
  const text = value.replace(/\s+/g, " ").trim();
  if (text.length <= 140) return text;
  return `${text.slice(0, 137)}...`;
}

function cancelable(status: string) {
  return ["pending", "queued", "running", "canceling"].includes(status);
}

function retryable(status: string) {
  return ["failed", "canceled"].includes(status);
}

function retryableNode(status: string) {
  return ["failed", "canceled"].includes(status);
}

function confirmTitle(action: PendingAction | null, t: (key: string) => string) {
  if (!action) return "";
  if (action.type === "publish") return t("workflows.confirmPublish");
  if (action.type === "disable") return t("workflows.confirmDisable");
  if (action.type === "copy") return t("workflows.confirmCopy");
  if (action.type === "run") return t("workflows.confirmRun");
  if (action.type === "retry") return t("workflows.confirmRun");
  if (action.type === "retryNode") return t("workflows.confirmRetryNode");
  if (action.type === "approve") return t("incidents.confirmAcknowledge");
  if (action.type === "reject") return t("workflows.confirmCancel");
  return t("workflows.confirmCancel");
}
