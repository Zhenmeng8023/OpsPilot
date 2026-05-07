import { FormEvent, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { request } from "../../api/request";
import type { ManagedUser, Role } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";

export function UserManagementPage() {
  const queryClient = useQueryClient();
  const t = useLanguageStore((state) => state.t);
  const [form, setForm] = useState({ username: "", email: "", password: "User@123456", role: "member" });
  const usersQuery = useQuery({
    queryKey: ["users"],
    queryFn: () => request<ManagedUser[]>("/api/v1/users")
  });
  const rolesQuery = useQuery({
    queryKey: ["roles"],
    queryFn: () => request<Role[]>("/api/v1/roles")
  });

  const createUser = useMutation({
    mutationFn: () =>
      request<ManagedUser>("/api/v1/users", {
        method: "POST",
        body: JSON.stringify({
          username: form.username,
          email: form.email || undefined,
          password: form.password,
          roles: [form.role]
        })
      }),
    onSuccess: async () => {
      setForm({ username: "", email: "", password: "User@123456", role: "member" });
      await queryClient.invalidateQueries({ queryKey: ["users"] });
    }
  });

  const updateStatus = useMutation({
    mutationFn: ({ user, status }: { user: ManagedUser; status: string }) =>
      request<{ ok: boolean }>(`/api/v1/users/${user.id}/status`, {
        method: "PATCH",
        body: JSON.stringify({ status })
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["users"] })
  });

  const updateRole = useMutation({
    mutationFn: ({ user, role }: { user: ManagedUser; role: string }) =>
      request<ManagedUser>(`/api/v1/users/${user.id}/roles`, {
        method: "PUT",
        body: JSON.stringify({ roles: [role] })
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["users"] })
  });

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    createUser.mutate();
  }

  const roles = rolesQuery.data ?? [];
  const users = usersQuery.data ?? [];

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">Access Control</p>
          <h1>{t("users.title")}</h1>
        </div>
      </section>

      <section className="management-grid">
        <form className="panel management-form" onSubmit={submit}>
          <div className="panel-title">
            <h3>{t("users.create")}</h3>
            <span>{createUser.isPending ? t("common.saving") : t("common.ready")}</span>
          </div>
          <label>
            {t("auth.username")}
            <input value={form.username} onChange={(event) => setForm({ ...form, username: event.target.value })} />
          </label>
          <label>
            {t("auth.email")}
            <input value={form.email} onChange={(event) => setForm({ ...form, email: event.target.value })} />
          </label>
          <label>
            {t("users.initialPassword")}
            <input
              type="password"
              value={form.password}
              onChange={(event) => setForm({ ...form, password: event.target.value })}
            />
          </label>
          <label>
            {t("users.role")}
            <select value={form.role} onChange={(event) => setForm({ ...form, role: event.target.value })}>
              {roles.map((role) => (
                <option key={role.id} value={role.code}>
                  {role.name} ({role.code})
                </option>
              ))}
            </select>
          </label>
          {createUser.isError ? <p className="form-error">{String(createUser.error.message)}</p> : null}
          <button type="submit" disabled={createUser.isPending}>
            {t("users.create")}
          </button>
        </form>

        <section className="panel table-panel">
          <div className="panel-title">
            <h3>{t("users.list")}</h3>
            <span>
              {users.length} {t("nav.users")}
            </span>
          </div>
          <div className="data-table">
            <table>
              <thead>
                <tr>
                  <th>{t("users.user")}</th>
                  <th>{t("users.status")}</th>
                  <th>{t("users.role")}</th>
                  <th>{t("common.createdAt")}</th>
                </tr>
              </thead>
              <tbody>
                {users.map((user) => (
                  <tr key={user.id}>
                    <td>
                      <strong>{user.username}</strong>
                      <small>{user.email || user.id}</small>
                    </td>
                    <td>
                      <select
                        value={user.status}
                        onChange={(event) => updateStatus.mutate({ user, status: event.target.value })}
                      >
                        <option value="active">active</option>
                        <option value="disabled">disabled</option>
                        <option value="locked">locked</option>
                      </select>
                    </td>
                    <td>
                      <select
                        value={user.roles[0] ?? ""}
                        onChange={(event) => updateRole.mutate({ user, role: event.target.value })}
                      >
                        {roles.map((role) => (
                          <option key={role.id} value={role.code}>
                            {role.code}
                          </option>
                        ))}
                      </select>
                    </td>
                    <td>{user.createdAt || "-"}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          {usersQuery.isError ? <p className="form-error">{String(usersQuery.error.message)}</p> : null}
        </section>
      </section>
    </main>
  );
}
