import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { listTasks } from "../../api/tasks";
import { useLanguageStore } from "../../i18n/language";
import { FilterToolbar } from "../../shared/components/FilterToolbar";
import { PaginationBar } from "../../shared/components/PaginationBar";
import { hasPermission } from "../auth/permissions";
import { useAuthStore } from "../auth/store";

export function TaskListPage() {
  const user = useAuthStore((state) => state.user);
  const t = useLanguageStore((state) => state.t);
  const [keyword, setKeyword] = useState("");
  const [status, setStatus] = useState("");
  const [creator, setCreator] = useState("");
  const [createdFrom, setCreatedFrom] = useState("");
  const [createdTo, setCreatedTo] = useState("");
  const [page, setPage] = useState(1);
  const pageSize = 20;
  const tasksQuery = useQuery({
    queryKey: ["tasks", keyword, status, creator, createdFrom, createdTo, page],
    queryFn: () => listTasks({ keyword, status, creator, createdFrom, createdTo, page, pageSize })
  });
  const tasks = useMemo(() => tasksQuery.data?.items ?? [], [tasksQuery.data]);
  const total = tasksQuery.data?.total ?? 0;
  const statusText = (value: string) => t(`common.status.${value}`);

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">{t("tasks.eyebrow")}</p>
          <h1>{t("tasks.title")}</h1>
        </div>
        {hasPermission(user, "task:execute") ? <Link className="ghost-button" to="/tasks/new">{t("tasks.createAction")}</Link> : null}
      </section>
      <section className="panel table-panel">
        <FilterToolbar>
          <input placeholder={t("tasks.search")} value={keyword} onChange={(event) => { setKeyword(event.target.value); setPage(1); }} />
          <input placeholder={t("tasks.creator")} value={creator} onChange={(event) => { setCreator(event.target.value); setPage(1); }} />
          <input type="date" value={createdFrom} onChange={(event) => { setCreatedFrom(event.target.value); setPage(1); }} />
          <input type="date" value={createdTo} onChange={(event) => { setCreatedTo(event.target.value); setPage(1); }} />
          <select value={status} onChange={(event) => { setStatus(event.target.value); setPage(1); }}>
            <option value="">{t("common.allStatus")}</option>
            <option value="queued">{statusText("queued")}</option>
            <option value="running">{statusText("running")}</option>
            <option value="canceling">{statusText("canceling")}</option>
            <option value="success">{statusText("success")}</option>
            <option value="failed">{statusText("failed")}</option>
            <option value="timeout">{statusText("timeout")}</option>
            <option value="canceled">{statusText("canceled")}</option>
          </select>
          <button type="button" onClick={() => tasksQuery.refetch()}>{t("common.refresh")}</button>
        </FilterToolbar>
        <div className="data-table">
          <table>
            <thead>
              <tr>
                <th>{t("common.task")}</th>
                <th>{t("common.status")}</th>
                <th>{t("tasks.targets")}</th>
                <th>{t("tasks.progress")}</th>
                <th>{t("tasks.createdBy")}</th>
                <th>{t("common.createdAt")}</th>
              </tr>
            </thead>
            <tbody>
              {tasks.map((task) => (
                <tr key={task.id}>
                  <td>
                    <Link to={`/tasks/${task.id}`}><strong>{task.name}</strong></Link>
                    <small>{task.description || task.id}</small>
                  </td>
                  <td><span className={`status-chip status-${task.status}`}>{statusText(task.status)}</span></td>
                  <td>{task.targetCount}</td>
                  <td>
                    <strong>{task.successCount} {statusText("success")} / {task.failedCount} {statusText("failed")}</strong>
                    <small>{task.runningCount} {statusText("running")} / {task.queuedCount} {statusText("queued")}</small>
                  </td>
                  <td>{task.createdBy || "-"}</td>
                  <td>{task.createdAt}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!tasksQuery.isLoading && tasks.length === 0 ? <p className="empty-state">{t("tasks.noTasks")}</p> : null}
        <PaginationBar total={total} page={page} pageSize={pageSize} onPageChange={setPage} />
        {tasksQuery.isError ? <p className="form-error">{tasksQuery.error.message}</p> : null}
      </section>
    </main>
  );
}
