import { useEffect, useMemo, useRef, useState } from "react";

import { getTaskLogs, streamTaskLogs } from "../../api/tasks";
import type { TaskLogEntry } from "../../api/types";

export function TaskLogViewer({ taskId, targetId }: { taskId: string; targetId?: string }) {
  const [logs, setLogs] = useState<TaskLogEntry[]>([]);
  const [paused, setPaused] = useState(false);
  const [stream, setStream] = useState("");
  const [error, setError] = useState("");
  const endRef = useRef<HTMLDivElement | null>(null);
  const cursor = useMemo(() => logs.reduce((max, log) => Math.max(max, log.id), 0), [logs]);

  useEffect(() => {
    const controller = new AbortController();
    setLogs([]);
    setError("");
    getTaskLogs(taskId, targetId, undefined, stream, 500)
      .then(setLogs)
      .catch((err: Error) => setError(err.message));
    streamTaskLogs(taskId, {
      targetId,
      afterSequence: cursor,
      stream,
      limit: 500,
      signal: controller.signal,
      onLog: (log) => setLogs((current) => (current.some((item) => item.id === log.id) ? current : [...current, log])),
      onError: (err) => setError(err.message)
    }).catch((err: Error) => {
      if (!controller.signal.aborted) setError(err.message);
    });
    return () => controller.abort();
  }, [taskId, targetId, stream]);

  useEffect(() => {
    if (!paused) endRef.current?.scrollIntoView({ block: "end" });
  }, [logs, paused]);

  const text = logs.map((log) => `[${log.createdAt}] ${log.stream} ${log.content}`).join("");

  return (
    <section className="panel log-panel">
      <div className="panel-title">
        <h3>Logs</h3>
        <div className="log-actions">
          <select value={stream} onChange={(event) => setStream(event.target.value)}>
            <option value="">All streams</option>
            <option value="stdout">stdout</option>
            <option value="stderr">stderr</option>
            <option value="system">system</option>
          </select>
          <button type="button" onClick={() => setPaused(!paused)}>{paused ? "Resume" : "Pause"}</button>
          <button type="button" onClick={() => navigator.clipboard.writeText(text)}>Copy</button>
          <button type="button" onClick={() => setLogs([])}>Clear</button>
        </div>
      </div>
      <div className="log-viewer">
        {logs.map((log) => (
          <pre key={log.id} className={`log-line log-${log.stream}`}>
            <span>{log.createdAt}</span> <strong>{log.stream}</strong> {log.content}
          </pre>
        ))}
        <div ref={endRef} />
      </div>
      {logs.length === 0 ? <p className="empty-state">Waiting for task logs.</p> : null}
      {error ? <p className="form-error">{error}</p> : null}
    </section>
  );
}
