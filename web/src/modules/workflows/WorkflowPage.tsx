import { useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  cancelWorkflowRun,
  createWorkflow,
  getWorkflow,
  getWorkflowRun,
  listWorkflowRuns,
  listWorkflows,
  publishWorkflow,
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

const sampleDefinition = JSON.stringify({
  nodes: [
    { id: "collect", type: "task", name: "Collect diagnostics", config: { taskId: "replace-with-task-id" } },
    { id: "gate", type: "condition", name: "Prod gate", config: { path: "environment", operator: "equals", value: "prod", onFalse: "skip" } },
    { id: "cooldown", type: "wait", name: "Cooldown", config: { seconds: 30 } },
    { id: "notify", type: "notification", name: "Notify owner", config: { title: "Workflow completed", channelId: "optional-channel-id" } }
  ],
  edges: [
    { from: "collect", to: "gate" },
    { from: "gate", to: "cooldown" },
    { from: "cooldown", to: "notify" }
  ],
  maxParallel: 2,
  failurePolicy: "stop_on_failure"
}, null, 2);

type PendingAction =
  | { type: "publish"; workflow: WorkflowDefinitionSummary }
  | { type: "run"; workflow: WorkflowDefinitionSummary }
  | { type: "cancel"; run: WorkflowRunSummary }
  | { type: "retry"; run: WorkflowRunSummary };

export function WorkflowPage() {
  const t = useLanguageStore((state) => state.t);
  const user = useAuthStore((state) => state.user);
  const { notify } = useToast();
  const queryClient = useQueryClient();
  const [tab, setTab] = useState<"definitions" | "runs">("definitions");
  const [keyword, setKeyword] = useState("");
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(1);
  const [runPage, setRunPage] = useState(1);
  const [selectedWorkflowId, setSelectedWorkflowId] = useState("");
  const [selectedRunId, setSelectedRunId] = useState("");
  const [pendingAction, setPendingAction] = useState<PendingAction | null>(null);
  const [form, setForm] = useState({ name: "", description: "", definition: sampleDefinition });
  const pageSize = 20;
  const canWrite = hasPermission(user, "workflow:write");
  const canExecute = hasPermission(user, "workflow:execute");
  const canCancel = hasPermission(user, "workflow:cancel");
  const workflowQuery = useQuery({
    queryKey: ["workflows", keyword, status, page],
    queryFn: () => listWorkflows({ keyword, status, page, pageSize })
  });
  const selectedWorkflowQuery = useQuery({
    queryKey: ["workflow", selectedWorkflowId],
    enabled: Boolean(selectedWorkflowId),
    queryFn: () => getWorkflow(selectedWorkflowId)
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
  const runMutation = useMutation({
    mutationFn: (workflow: WorkflowDefinitionSummary) => runWorkflow(workflow.id, { triggerType: "manual" }),
    onSuccess: (run) => {
      notify(t("workflows.runCreatedToast"), "success");
      setTab("runs");
      setSelectedRunId(run.id);
      queryClient.invalidateQueries({ queryKey: ["workflowRuns"] });
      queryClient.invalidateQueries({ queryKey: ["workflowRun", run.id] });
    }
  });
  const cancelMutation = useMutation({
    mutationFn: (run: WorkflowRunSummary) => cancelWorkflowRun(run.id, "Canceled from UI"),
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
  const currentError = saveMutation.error ?? publishMutation.error ?? runMutation.error ?? cancelMutation.error ?? retryMutation.error;

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
                        {canExecute ? <button type="button" disabled={workflow.status !== "active" || runMutation.isPending} onClick={() => setPendingAction({ type: "run", workflow })}>{t("workflows.run")}</button> : null}
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
                        <th>TaskRun</th>
                      </tr>
                    </thead>
                    <tbody>
                      {selectedRun.nodes.map((node) => (
                        <tr key={node.id}>
                          <td>
                            <strong>{node.nodeName || node.nodeId}</strong>
                            <small>{node.nodeId}</small>
                            {node.errorMessage ? <small>{node.errorMessage}</small> : null}
                          </td>
                          <td>{node.nodeType}</td>
                          <td><span className={`status-chip status-${node.status}`}>{statusText(node.status)}</span></td>
                          <td>{node.taskRunId || "-"}</td>
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
        confirmLabel={t("common.confirm")}
        cancelLabel={t("common.cancel")}
        danger={pendingAction?.type === "cancel"}
        onCancel={() => setPendingAction(null)}
        onConfirm={() => {
          if (pendingAction?.type === "publish") publishMutation.mutate(pendingAction.workflow);
          if (pendingAction?.type === "run") runMutation.mutate(pendingAction.workflow);
          if (pendingAction?.type === "cancel") cancelMutation.mutate(pendingAction.run);
          if (pendingAction?.type === "retry") retryMutation.mutate(pendingAction.run);
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

function cancelable(status: string) {
  return ["pending", "queued", "running", "canceling"].includes(status);
}

function retryable(status: string) {
  return ["failed", "canceled"].includes(status);
}

function confirmTitle(action: PendingAction | null, t: (key: string) => string) {
  if (!action) return "";
  if (action.type === "publish") return t("workflows.confirmPublish");
  if (action.type === "run") return t("workflows.confirmRun");
  if (action.type === "retry") return t("workflows.confirmRun");
  return t("workflows.confirmCancel");
}
