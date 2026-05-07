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
import { hasPermission } from "../auth/permissions";
import { useAuthStore } from "../auth/store";

export function ScriptListPage() {
  const user = useAuthStore((state) => state.user);
  const queryClient = useQueryClient();
  const [keyword, setKeyword] = useState("");
  const [status, setStatus] = useState("");
  const [scriptType, setScriptType] = useState("");
  const [approvalStatus, setApprovalStatus] = useState("");
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
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["scripts"] })
  });
  const approveMutation = useMutation({
    mutationFn: ({ id, comment }: { id: number; comment: string }) => approveScriptApproval(id, comment),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["scriptApprovals"] })
  });
  const rejectMutation = useMutation({
    mutationFn: ({ id, comment }: { id: number; comment: string }) => rejectScriptApproval(id, comment),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["scriptApprovals"] })
  });
  const scripts = useMemo(() => scriptsQuery.data?.items ?? [], [scriptsQuery.data]);
  const total = scriptsQuery.data?.total ?? 0;
  const approvals = useMemo(() => approvalsQuery.data ?? [], [approvalsQuery.data]);
  const canApprove = hasPermission(user, "script:approve");
  const canWrite = hasPermission(user, "script:write");

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">Scripts</p>
          <h1>Script Templates</h1>
        </div>
        {canWrite ? <Link className="ghost-button" to="/scripts/new">Create script</Link> : null}
      </section>

      <section className="panel table-panel">
        <div className="toolbar-row">
          <input placeholder="Search scripts" value={keyword} onChange={(event) => { setKeyword(event.target.value); setPage(1); }} />
          <select value={scriptType} onChange={(event) => { setScriptType(event.target.value); setPage(1); }}>
            <option value="">All type</option>
            <option value="shell">shell</option>
            <option value="powershell">powershell</option>
            <option value="bash">bash</option>
            <option value="custom">custom</option>
          </select>
          <select value={status} onChange={(event) => { setStatus(event.target.value); setPage(1); }}>
            <option value="">All status</option>
            <option value="active">active</option>
            <option value="disabled">disabled</option>
            <option value="draft">draft</option>
          </select>
          <select value={approvalStatus} onChange={(event) => { setApprovalStatus(event.target.value); setPage(1); }}>
            <option value="">All approval</option>
            <option value="approved">approved</option>
            <option value="pending">pending</option>
            <option value="rejected">rejected</option>
            <option value="canceled">canceled</option>
            <option value="not_required">not_required</option>
          </select>
          <button type="button" onClick={() => scriptsQuery.refetch()}>Refresh</button>
        </div>
        <div className="data-table">
          <table>
            <thead>
              <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Status</th>
                <th>Approval</th>
                <th>Revision</th>
                <th>Created</th>
                <th>Action</th>
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
                  <td><span className={`status-chip status-${script.status}`}>{script.status}</span></td>
                  <td>
                    <span className={`status-chip status-${script.approvalStatus || "none"}`}>
                      {script.approvalRequired ? script.approvalStatus || "required" : "not_required"}
                    </span>
                  </td>
                  <td>v{script.version}</td>
                  <td>{script.createdAt}</td>
                  <td className="action-cell">
                    {canWrite ? <Link to={`/scripts/${script.id}`}>Edit</Link> : null}
                    {canWrite ? (
                      <button type="button" disabled={script.status === "disabled"} onClick={() => disableMutation.mutate(script)}>
                        Disable
                      </button>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!scriptsQuery.isLoading && scripts.length === 0 ? <p className="empty-state">No scripts found.</p> : null}
        <div className="toolbar-row">
          <span>{total} total</span>
          <button type="button" disabled={page <= 1} onClick={() => setPage((value) => Math.max(1, value - 1))}>Previous</button>
          <span>Page {page}</span>
          <button type="button" disabled={page * pageSize >= total} onClick={() => setPage((value) => value + 1)}>Next</button>
        </div>
        {scriptsQuery.isError ? <p className="form-error">{scriptsQuery.error.message}</p> : null}
        {disableMutation.isError ? <p className="form-error">{disableMutation.error.message}</p> : null}
      </section>

      <section className="panel table-panel">
        <div className="panel-title">
          <h3>Pending Approvals</h3>
          <span>{approvals.length} pending</span>
        </div>
        <div className="data-table">
          <table>
            <thead>
              <tr>
                <th>Script</th>
                <th>Version</th>
                <th>Status</th>
                <th>Created</th>
                <th>Action</th>
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
                  <td><span className={`status-chip status-${approval.status}`}>{approval.status}</span></td>
                  <td>{approval.createdAt}</td>
                  <td className="action-cell">
                    {canApprove ? (
                      <>
                        <button
                          type="button"
                          disabled={approveMutation.isPending || rejectMutation.isPending}
                          onClick={() => {
                            const comment = window.prompt("Approval comment (optional):", "") ?? "";
                            approveMutation.mutate({ id: approval.id, comment });
                          }}
                        >
                          Approve
                        </button>
                        <button
                          type="button"
                          disabled={approveMutation.isPending || rejectMutation.isPending}
                          onClick={() => {
                            const comment = window.prompt("Rejection reason (optional):", "") ?? "";
                            rejectMutation.mutate({ id: approval.id, comment });
                          }}
                        >
                          Reject
                        </button>
                      </>
                    ) : null}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
        {!approvalsQuery.isLoading && approvals.length === 0 ? <p className="empty-state">No pending approvals.</p> : null}
        {approvalsQuery.isError ? <p className="form-error">{approvalsQuery.error.message}</p> : null}
        {approveMutation.isError ? <p className="form-error">{approveMutation.error.message}</p> : null}
        {rejectMutation.isError ? <p className="form-error">{rejectMutation.error.message}</p> : null}
      </section>
    </main>
  );
}
