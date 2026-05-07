import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { listTasks } from "../../api/tasks";
import { hasPermission } from "../auth/permissions";
import { useAuthStore } from "../auth/store";

export function TaskListPage() {
  const user = useAuthStore((state) => state.user);
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

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">Execution</p>
          <h1>Task Runs</h1>
        </div>
        {hasPermission(user, "task:execute") ? <Link className="ghost-button" to="/tasks/new">Create task</Link> : null}
      </section>
      <section className="panel table-panel">
        <div className="toolbar-row">
          <input placeholder="Search tasks" value={keyword} onChange={(event) => { setKeyword(event.target.value); setPage(1); }} />
          <input placeholder="Creator" value={creator} onChange={(event) => { setCreator(event.target.value); setPage(1); }} />
          <input type="date" value={createdFrom} onChange={(event) => { setCreatedFrom(event.target.value); setPage(1); }} />
          <input type="date" value={createdTo} onChange={(event) => { setCreatedTo(event.target.value); setPage(1); }} />
          <select value={status} onChange={(event) => { setStatus(event.target.value); setPage(1); }}>
            <option value="">All status</option>
            <option value="queued">queued</option>
            <option value="running">running</option>
            <option value="canceling">canceling</option>
            <option value="success">success</option>
            <option value="failed">failed</option>
            <option value="timeout">timeout</option>
            <option value="canceled">canceled</option>
          </select>
          <button type="button" onClick={() => tasksQuery.refetch()}>Refresh</button>
        </div>
        <div className="data-table">
          <table>
            <thead>
              <tr>
                <th>Task</th>
                <th>Status</th>
                <th>Targets</th>
                <th>Progress</th>
                <th>Created by</th>
                <th>Created at</th>
              </tr>
            </thead>
            <tbody>
              {tasks.map((task) => (
                <tr key={task.id}>
                  <td>
                    <Link to={`/tasks/${task.id}`}><strong>{task.name}</strong></Link>
                    <small>{task.description || task.id}</small>
                  </td>
                  <td><span className={`status-chip status-${task.status}`}>{task.status}</span></td>
                  <td>{task.targetCount}</td>
                  <td>
                    <strong>{task.successCount} ok / {task.failedCount} failed</strong>
                    <small>{task.runningCount} running / {task.queuedCount} queued</small>
                  </td>
                  <td>{task.createdBy || "-"}</td>
                  <td>{task.createdAt}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!tasksQuery.isLoading && tasks.length === 0 ? <p className="empty-state">No tasks found.</p> : null}
        <div className="toolbar-row">
          <span>{total} total</span>
          <button type="button" disabled={page <= 1} onClick={() => setPage((value) => Math.max(1, value - 1))}>Previous</button>
          <span>Page {page}</span>
          <button type="button" disabled={page * pageSize >= total} onClick={() => setPage((value) => value + 1)}>Next</button>
        </div>
        {tasksQuery.isError ? <p className="form-error">{tasksQuery.error.message}</p> : null}
      </section>
    </main>
  );
}
