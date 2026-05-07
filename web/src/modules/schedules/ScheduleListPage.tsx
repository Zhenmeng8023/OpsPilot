import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { listTasks } from "../../api/tasks";
import { createSchedule, disableSchedule, listSchedules, pauseSchedule, resumeSchedule } from "../../api/schedules";
import type { ScheduleSummary } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";
import { hasPermission } from "../auth/permissions";
import { useAuthStore } from "../auth/store";

export function ScheduleListPage() {
  const user = useAuthStore((state) => state.user);
  const t = useLanguageStore((state) => state.t);
  const queryClient = useQueryClient();
  const [keyword, setKeyword] = useState("");
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(1);
  const [form, setForm] = useState({
    name: "",
    taskId: "",
    cronExpr: "*/5 * * * *",
    timezone: "Asia/Shanghai"
  });
  const pageSize = 20;
  const canWrite = hasPermission(user, "schedule:write");
  const schedulesQuery = useQuery({
    queryKey: ["schedules", keyword, status, page],
    queryFn: () => listSchedules({ keyword, status, page, pageSize })
  });
  const tasksQuery = useQuery({
    queryKey: ["tasks", "schedule-options"],
    queryFn: () => listTasks({ pageSize: 100 })
  });
  const createMutation = useMutation({
    mutationFn: createSchedule,
    onSuccess: () => {
      setForm((current) => ({ ...current, name: "" }));
      queryClient.invalidateQueries({ queryKey: ["schedules"] });
    }
  });
  const pauseMutation = useMutation({
    mutationFn: (schedule: ScheduleSummary) => pauseSchedule(schedule.id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["schedules"] })
  });
  const resumeMutation = useMutation({
    mutationFn: (schedule: ScheduleSummary) => resumeSchedule(schedule.id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["schedules"] })
  });
  const disableMutation = useMutation({
    mutationFn: (schedule: ScheduleSummary) => disableSchedule(schedule.id),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["schedules"] })
  });
  const schedules = useMemo(() => schedulesQuery.data?.items ?? [], [schedulesQuery.data]);
  const total = schedulesQuery.data?.total ?? 0;
  const taskOptions = useMemo(() => {
    const map = new Map<string, NonNullable<typeof tasksQuery.data>["items"][number]>();
    for (const task of tasksQuery.data?.items ?? []) {
      if (!map.has(task.taskId)) {
        map.set(task.taskId, task);
      }
    }
    return Array.from(map.values());
  }, [tasksQuery.data]);

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">{t("automation.eyebrow")}</p>
          <h1>{t("schedules.title")}</h1>
        </div>
      </section>

      {canWrite ? (
        <section className="panel form-panel">
          <div className="panel-title">
            <h3>{t("schedules.create")}</h3>
            <span>{t("schedules.createHint")}</span>
          </div>
          <form
            className="form-grid"
            onSubmit={(event) => {
              event.preventDefault();
              createMutation.mutate(form);
            }}
          >
            <label>
              {t("common.name")}
              <input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required />
            </label>
            <label>
              {t("common.task")}
              <select value={form.taskId} onChange={(event) => setForm({ ...form, taskId: event.target.value })} required>
                <option value="">{t("common.selectTask")}</option>
                {taskOptions.map((task) => (
                  <option key={task.taskId} value={task.taskId}>{task.name}</option>
                ))}
              </select>
            </label>
            <label>
              {t("schedules.cron")}
              <input value={form.cronExpr} onChange={(event) => setForm({ ...form, cronExpr: event.target.value })} required />
            </label>
            <label>
              {t("schedules.timezone")}
              <input value={form.timezone} onChange={(event) => setForm({ ...form, timezone: event.target.value })} required />
            </label>
            <button type="submit" disabled={createMutation.isPending}>{t("schedules.createAction")}</button>
          </form>
          {createMutation.isError ? <p className="form-error">{createMutation.error.message}</p> : null}
        </section>
      ) : null}

      <section className="panel table-panel">
        <div className="toolbar-row">
          <input placeholder={t("schedules.search")} value={keyword} onChange={(event) => { setKeyword(event.target.value); setPage(1); }} />
          <select value={status} onChange={(event) => { setStatus(event.target.value); setPage(1); }}>
            <option value="">{t("common.allStatus")}</option>
            <option value="active">active</option>
            <option value="paused">paused</option>
            <option value="disabled">disabled</option>
          </select>
          <button type="button" onClick={() => schedulesQuery.refetch()}>{t("common.refresh")}</button>
        </div>
        <div className="data-table">
          <table>
            <thead>
              <tr>
                <th>{t("common.name")}</th>
                <th>{t("common.task")}</th>
                <th>{t("schedules.cron")}</th>
                <th>{t("common.status")}</th>
                <th>{t("schedules.nextFire")}</th>
                <th>{t("schedules.lastFire")}</th>
                <th>{t("common.action")}</th>
              </tr>
            </thead>
            <tbody>
              {schedules.map((schedule) => (
                <tr key={schedule.id}>
                  <td>
                    <strong>{schedule.name}</strong>
                    <small>{schedule.id}</small>
                  </td>
                  <td>
                    <strong>{schedule.taskName}</strong>
                    <small>{schedule.taskId}</small>
                  </td>
                  <td>
                    <strong>{schedule.cronExpr}</strong>
                    <small>{schedule.timezone}</small>
                  </td>
                  <td><span className={`status-chip status-${schedule.status}`}>{schedule.status}</span></td>
                  <td>{schedule.nextFireAt || "-"}</td>
                  <td>{schedule.lastFireAt || "-"}</td>
                  <td className="action-cell">
                    {canWrite && schedule.status === "active" ? <button type="button" onClick={() => pauseMutation.mutate(schedule)}>{t("schedules.pause")}</button> : null}
                    {canWrite && schedule.status === "paused" ? <button type="button" onClick={() => resumeMutation.mutate(schedule)}>{t("schedules.resume")}</button> : null}
                    {canWrite && schedule.status !== "disabled" ? <button type="button" onClick={() => disableMutation.mutate(schedule)}>{t("schedules.disable")}</button> : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!schedulesQuery.isLoading && schedules.length === 0 ? <p className="empty-state">{t("schedules.empty")}</p> : null}
        <div className="toolbar-row">
          <span>{total} {t("common.total")}</span>
          <button type="button" disabled={page <= 1} onClick={() => setPage((value) => Math.max(1, value - 1))}>{t("common.previous")}</button>
          <span>{t("common.page").replace("{page}", String(page))}</span>
          <button type="button" disabled={page * pageSize >= total} onClick={() => setPage((value) => value + 1)}>{t("common.next")}</button>
        </div>
        {schedulesQuery.isError ? <p className="form-error">{schedulesQuery.error.message}</p> : null}
        {pauseMutation.isError ? <p className="form-error">{pauseMutation.error.message}</p> : null}
        {resumeMutation.isError ? <p className="form-error">{resumeMutation.error.message}</p> : null}
        {disableMutation.isError ? <p className="form-error">{disableMutation.error.message}</p> : null}
      </section>
    </main>
  );
}
