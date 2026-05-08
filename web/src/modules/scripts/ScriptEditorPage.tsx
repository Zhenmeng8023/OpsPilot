import { FormEvent, useEffect, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";

import { createScript, getScript, requestScriptApproval, updateScript } from "../../api/scripts";
import { useLanguageStore } from "../../i18n/language";
import { useToast } from "../../shared/components/ToastProvider";

const emptyForm = {
  name: "",
  description: "",
  scriptType: "shell",
  content: "",
  changeSummary: ""
};

export function ScriptEditorPage() {
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const t = useLanguageStore((state) => state.t);
  const { notify } = useToast();
  const { id } = useParams();
  const isNew = !id || id === "new";
  const [form, setForm] = useState(emptyForm);
  const scriptQuery = useQuery({
    queryKey: ["script", id],
    enabled: !isNew && Boolean(id),
    queryFn: () => getScript(id ?? "")
  });
  useEffect(() => {
    if (!scriptQuery.data) return;
    setForm({
      name: scriptQuery.data.name,
      description: scriptQuery.data.description ?? "",
      scriptType: scriptQuery.data.scriptType,
      content: scriptQuery.data.content ?? "",
      changeSummary: ""
    });
  }, [scriptQuery.data]);

  const saveMutation = useMutation({
    mutationFn: () => (isNew ? createScript(form) : updateScript(id ?? "", form)),
    onSuccess: (script) => {
      notify(t("scripts.savedToast"), "success");
      navigate(`/scripts/${script.id}`);
    }
  });
  const requestApprovalMutation = useMutation({
    mutationFn: () => requestScriptApproval(id ?? ""),
    onSuccess: () => {
      notify(t("scripts.approvalSubmitted"), "success");
      queryClient.invalidateQueries({ queryKey: ["scriptApprovals"] });
    }
  });

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!form.content.trim()) return;
    saveMutation.mutate();
  }

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">{t("scripts.eyebrow")}</p>
          <h1>{isNew ? t("scripts.createTitle") : t("scripts.editTitle")}</h1>
        </div>
      </section>
      <form className="panel editor-form" onSubmit={submit}>
        <div className="form-grid">
          <label>
            {t("common.name")}
            <input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} required />
          </label>
          <label>
            {t("common.type")}
            <select value={form.scriptType} onChange={(event) => setForm({ ...form, scriptType: event.target.value })}>
              <option value="shell">shell</option>
              <option value="bash">bash</option>
              <option value="powershell">powershell</option>
            </select>
          </label>
        </div>
        <label>
          {t("scripts.description")}
          <input value={form.description} onChange={(event) => setForm({ ...form, description: event.target.value })} />
        </label>
        <label>
          {t("scripts.changeSummary")}
          <input value={form.changeSummary} onChange={(event) => setForm({ ...form, changeSummary: event.target.value })} />
        </label>
        <label>
          {t("scripts.content")}
          <textarea
            className="code-input"
            value={form.content}
            onChange={(event) => setForm({ ...form, content: event.target.value })}
            required
          />
        </label>
        {!form.content.trim() ? <p className="form-error">{t("scripts.contentEmpty")}</p> : null}
        {saveMutation.isError ? <p className="form-error">{saveMutation.error.message}</p> : null}
        {scriptQuery.isError ? <p className="form-error">{scriptQuery.error.message}</p> : null}
        <div className="enrollment-actions">
          <button className="ghost-button" type="submit" disabled={saveMutation.isPending || !form.content.trim()}>
            {saveMutation.isPending ? t("common.saving") : t("common.save")}
          </button>
          {!isNew ? (
            <button
              className="ghost-button"
              type="button"
              disabled={requestApprovalMutation.isPending}
              onClick={() => requestApprovalMutation.mutate()}
            >
              {requestApprovalMutation.isPending ? t("scripts.requestingApproval") : t("scripts.requestApproval")}
            </button>
          ) : null}
        </div>
        {requestApprovalMutation.isError ? <p className="form-error">{requestApprovalMutation.error.message}</p> : null}
        {requestApprovalMutation.isSuccess ? <p className="empty-state">{t("scripts.approvalSubmitted")}</p> : null}
      </form>
    </main>
  );
}
