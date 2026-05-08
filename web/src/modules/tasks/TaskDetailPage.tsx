import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useParams } from "react-router-dom";

import { cancelTask, getTask } from "../../api/tasks";
import { useLanguageStore } from "../../i18n/language";
import { TaskLogViewer } from "./TaskLogViewer";

export function TaskDetailPage() {
  const { id } = useParams();
  const queryClient = useQueryClient();
  const t = useLanguageStore((state) => state.t);
  const [selectedTarget, setSelectedTarget] = useState<string | undefined>();
  const taskQuery = useQuery({ queryKey: ["task", id], enabled: Boolean(id), queryFn: () => getTask(id ?? "") });
  const cancelMutation = useMutation({
    mutationFn: () => cancelTask(id ?? ""),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["task", id] })
  });
  const task = taskQuery.data;
  const stats = useMemo(() => ({
    success: task?.successCount ?? 0,
    failed: task?.failedCount ?? 0,
    running: task?.runningCount ?? 0,
    queued: task?.queuedCount ?? 0,
    total: task?.targetCount ?? 0
  }), [task]);
  const statusText = (value: string) => t(`common.status.${value}`);

  if (!id) return null;

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">{t("tasks.detailEyebrow")}</p>
          <h1>{task?.name ?? t("common.task")}</h1>
        </div>
        <button className="danger-button" type="button" disabled={!task || !["pending", "queued"].includes(task.status) || cancelMutation.isPending} onClick={() => cancelMutation.mutate()}>
          {t("tasks.cancel")}
        </button>
      </section>
      {task ? (
        <>
          <section className="agent-summary">
            <div className="panel agent-stat"><span>{t("common.status")}</span><strong>{statusText(task.status)}</strong></div>
            <div className="panel agent-stat"><span>{t("tasks.targets")}</span><strong>{stats.total}</strong></div>
            <div className="panel agent-stat"><span>{t("tasks.running")}</span><strong>{stats.running}</strong></div>
            <div className="panel agent-stat"><span>{statusText("failed")}</span><strong>{stats.failed}</strong></div>
          </section>
          <section className="panel table-panel">
            <div className="panel-title"><h3>{t("tasks.targets")}</h3><span>{selectedTarget ? t("tasks.filtered") : t("tasks.allLogs")}</span></div>
            <div className="data-table">
              <table>
                <thead>
                  <tr><th>{t("tasks.targets")}</th><th>{t("common.status")}</th><th>{t("tasks.exit")}</th><th>{t("tasks.started")}</th><th>{t("tasks.finished")}</th><th>{t("tasks.logs")}</th></tr>
                </thead>
                <tbody>
                  {(task.targets ?? []).map((target) => (
                    <tr key={target.id}>
                      <td><strong>{target.agentName || target.hostName || target.id}</strong><small>{target.agentId || target.hostId}</small></td>
                      <td><span className={`status-chip status-${target.status}`}>{statusText(target.status)}</span></td>
                      <td>{target.exitCode ?? "-"}</td>
                      <td>{target.startedAt || "-"}</td>
                      <td>{target.finishedAt || "-"}</td>
                      <td><button type="button" onClick={() => setSelectedTarget(selectedTarget === target.id ? undefined : target.id)}>{selectedTarget === target.id ? t("common.hide") : t("common.view")}</button></td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
          <TaskLogViewer taskId={id} targetId={selectedTarget} />
        </>
      ) : taskQuery.isLoading ? (
        <section className="panel"><p>{t("tasks.loading")}</p></section>
      ) : null}
      {taskQuery.isError ? <p className="form-error">{taskQuery.error.message}</p> : null}
      {cancelMutation.isError ? <p className="form-error">{cancelMutation.error.message}</p> : null}
    </main>
  );
}
