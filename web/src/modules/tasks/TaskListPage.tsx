import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";

import { listTasks } from "../../api/tasks";

export function TaskListPage() {
  const [keyword, setKeyword] = useState("");
  const [status, setStatus] = useState("");
  const tasksQuery = useQuery({
    queryKey: ["tasks", keyword, status],
    queryFn: () => listTasks({ keyword, status })
  });
  const tasks = useMemo(() => tasksQuery.data ?? [], [tasksQuery.data]);

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">Execution</p>
          <h1>Task Runs</h1>
        </div>
        <Link className="ghost-button" to="/tasks/new">Create task</Link>
      </section>
      <section className="panel table-panel">
        <div className="toolbar-row">
          <input placeholder="Search tasks" value={keyword} onChange={(event) => setKeyword(event.target.value)} />
          <select value={status} onChange={(event) => setStatus(event.target.value)}>
            <option value="">All status</option>
            <option value="queued">queued</option>
            <option value="running">running</option>
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
        {tasksQuery.isError ? <p className="form-error">{tasksQuery.error.message}</p> : null}
      </section>
    </main>
  );
}
