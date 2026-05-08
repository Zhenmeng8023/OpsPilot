import { useMemo, useState } from "react";
import { Link } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import {
  approveScriptApproval,
  disableScript,
  listScriptApprovals,
  listScripts,
  rejectScriptApproval
} from "../../api/scripts";
import type { Script } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";
import { ConfirmDialog } from "../../shared/components/ConfirmDialog";
import { FilterToolbar } from "../../shared/components/FilterToolbar";
import { PaginationBar } from "../../shared/components/PaginationBar";
import { useToast } from "../../shared/components/ToastProvider";
import { hasPermission } from "../auth/permissions";
import { useAuthStore } from "../auth/store";

type PendingAction =
  | { type: "disable"; script: Script }
  | { type: "approve"; id: number }
  | { type: "reject"; id: number };

export function ScriptListPage() {
  const user = useAuthStore((state) => state.user);
  const t = useLanguageStore((state) => state.t);
  const { notify } = useToast();
  const queryClient = useQueryClient();
  const [keyword, setKeyword] = useState("");
  const [status, setStatus] = useState("");
  const [scriptType, setScriptType] = useState("");
  const [approvalStatus, setApprovalStatus] = useState("");
  const [pendingAction, setPendingAction] = useState<PendingAction | null>(null);
  const [page, setPage] = useState(1);
  const pageSize = 20;
  const scriptsQuery = useQuery({
    queryKey: ["scripts", keyword, status, scriptType, approvalStatus, page],
    queryFn: () => listScripts({ keyword, status, type: scriptType, approvalStatus, page, pageSize })
  });
  const approvalsQuery = useQuery({
    queryKey: ["scriptApprovals", "pending"],
    queryFn: () => listScriptApprovals({ status: "pending" })
  });
  const disableMutation = useMutation({
    mutationFn: (script: Script) => disableScript(script.id),
    onSuccess: () => {
      notify(t("scripts.disabledToast"), "success");
      queryClient.invalidateQueries({ queryKey: ["scripts"] });
    }
  });
  const approveMutation = useMutation({
    mutationFn: ({ id, comment }: { id: number; comment: string }) => approveScriptApproval(id, comment),
    onSuccess: () => {
      notify(t("scripts.approvedToast"), "success");
      queryClient.invalidateQueries({ queryKey: ["scriptApprovals"] });
    }
  });
  const rejectMutation = useMutation({
    mutationFn: ({ id, comment }: { id: number; comment: string }) => rejectScriptApproval(id, comment),
    onSuccess: () => {
      notify(t("scripts.rejectedToast"), "success");
      queryClient.invalidateQueries({ queryKey: ["scriptApprovals"] });
    }
  });
  const scripts = useMemo(() => scriptsQuery.data?.items ?? [], [scriptsQuery.data]);
  const total = scriptsQuery.data?.total ?? 0;
  const approvals = useMemo(() => approvalsQuery.data ?? [], [approvalsQuery.data]);
  const canApprove = hasPermission(user, "script:approve");
  const canWrite = hasPermission(user, "script:write");
  const statusText = (value: string) => t(`common.status.${value}`);
  const confirmTitle = pendingAction?.type === "disable"
    ? t("scripts.confirmDisable")
    : pendingAction?.type === "approve"
      ? t("scripts.confirmApprove")
      : t("scripts.confirmReject");
  const confirmDanger = pendingAction?.type === "disable" || pendingAction?.type === "reject";

  function confirmPendingAction() {
    if (!pendingAction) return;
    if (pendingAction.type === "disable") {
      disableMutation.mutate(pendingAction.script);
    } else if (pendingAction.type === "approve") {
      approveMutation.mutate({ id: pendingAction.id, comment: "" });
    } else {
      rejectMutation.mutate({ id: pendingAction.id, comment: "" });
    }
    setPendingAction(null);
  }

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">{t("scripts.eyebrow")}</p>
          <h1>{t("scripts.title")}</h1>
        </div>
        {canWrite ? <Link className="ghost-button" to="/scripts/new">{t("scripts.createAction")}</Link> : null}
      </section>

      <section className="panel table-panel">
        <FilterToolbar>
          <input placeholder={t("scripts.search")} value={keyword} onChange={(event) => { setKeyword(event.target.value); setPage(1); }} />
          <select value={scriptType} onChange={(event) => { setScriptType(event.target.value); setPage(1); }}>
            <option value="">{t("scripts.allType")}</option>
            <option value="shell">shell</option>
            <option value="powershell">powershell</option>
            <option value="bash">bash</option>
            <option value="custom">custom</option>
          </select>
          <select value={status} onChange={(event) => { setStatus(event.target.value); setPage(1); }}>
            <option value="">{t("common.allStatus")}</option>
            <option value="active">{statusText("active")}</option>
            <option value="disabled">{statusText("disabled")}</option>
            <option value="draft">{statusText("draft")}</option>
          </select>
          <select value={approvalStatus} onChange={(event) => { setApprovalStatus(event.target.value); setPage(1); }}>
            <option value="">{t("scripts.allApproval")}</option>
            <option value="approved">{statusText("approved")}</option>
            <option value="pending">{statusText("pending")}</option>
            <option value="rejected">{statusText("rejected")}</option>
            <option value="canceled">{statusText("canceled")}</option>
            <option value="not_required">{statusText("not_required")}</option>
          </select>
          <button type="button" onClick={() => scriptsQuery.refetch()}>{t("common.refresh")}</button>
        </FilterToolbar>
        <div className="data-table">
          <table>
            <thead>
              <tr>
                <th>{t("common.name")}</th>
                <th>{t("common.type")}</th>
                <th>{t("common.status")}</th>
                <th>{t("scripts.approval")}</th>
                <th>{t("scripts.revision")}</th>
                <th>{t("common.created")}</th>
                <th>{t("common.action")}</th>
              </tr>
            </thead>
            <tbody>
              {scripts.map((script) => (
                <tr key={script.id}>
                  <td>
                    <strong>{script.name}</strong>
                    <small>{script.description || script.id}</small>
                  </td>
                  <td>{script.scriptType}</td>
                  <td><span className={`status-chip status-${script.status}`}>{statusText(script.status)}</span></td>
                  <td>
                    <span className={`status-chip status-${script.approvalStatus || "none"}`}>
                      {script.approvalRequired ? statusText(script.approvalStatus || "required") : statusText("not_required")}
                    </span>
                  </td>
                  <td>v{script.version}</td>
                  <td>{script.createdAt}</td>
                  <td className="action-cell">
                    {canWrite ? <Link to={`/scripts/${script.id}`}>{t("common.edit")}</Link> : null}
                    {canWrite ? (
                      <button type="button" disabled={script.status === "disabled"} onClick={() => setPendingAction({ type: "disable", script })}>
                        {t("common.disable")}
                      </button>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!scriptsQuery.isLoading && scripts.length === 0 ? <p className="empty-state">{t("scripts.noScripts")}</p> : null}
        <PaginationBar total={total} page={page} pageSize={pageSize} onPageChange={setPage} />
        {scriptsQuery.isError ? <p className="form-error">{scriptsQuery.error.message}</p> : null}
        {disableMutation.isError ? <p className="form-error">{disableMutation.error.message}</p> : null}
      </section>

      <section className="panel table-panel">
        <div className="panel-title">
          <h3>{t("scripts.pendingApprovals")}</h3>
          <span>{t("scripts.pendingCount", { count: approvals.length })}</span>
        </div>
        <div className="data-table">
          <table>
            <thead>
              <tr>
                <th>{t("common.name")}</th>
                <th>{t("common.version")}</th>
                <th>{t("common.status")}</th>
                <th>{t("common.created")}</th>
                <th>{t("common.action")}</th>
              </tr>
            </thead>
            <tbody>
              {approvals.map((approval) => (
                <tr key={approval.id}>
                  <td>
                    <strong>{approval.scriptName}</strong>
                    <small>{approval.scriptId}</small>
                  </td>
                  <td>v{approval.version}</td>
                  <td><span className={`status-chip status-${approval.status}`}>{statusText(approval.status)}</span></td>
                  <td>{approval.createdAt}</td>
                  <td className="action-cell">
                    {canApprove ? (
                      <>
                        <button
                          type="button"
                          disabled={approveMutation.isPending || rejectMutation.isPending}
                          onClick={() => setPendingAction({ type: "approve", id: approval.id })}
                        >
                          {t("scripts.approve")}
                        </button>
                        <button
                          type="button"
                          disabled={approveMutation.isPending || rejectMutation.isPending}
                          onClick={() => setPendingAction({ type: "reject", id: approval.id })}
                        >
                          {t("scripts.reject")}
                        </button>
                      </>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!approvalsQuery.isLoading && approvals.length === 0 ? <p className="empty-state">{t("scripts.noPendingApprovals")}</p> : null}
        {approvalsQuery.isError ? <p className="form-error">{approvalsQuery.error.message}</p> : null}
        {approveMutation.isError ? <p className="form-error">{approveMutation.error.message}</p> : null}
        {rejectMutation.isError ? <p className="form-error">{rejectMutation.error.message}</p> : null}
      </section>
      <ConfirmDialog
        open={Boolean(pendingAction)}
        title={confirmTitle}
        confirmLabel={t("common.confirm")}
        cancelLabel={t("common.cancel")}
        danger={confirmDanger}
        onCancel={() => setPendingAction(null)}
        onConfirm={confirmPendingAction}
      />
    </main>
  );
}
