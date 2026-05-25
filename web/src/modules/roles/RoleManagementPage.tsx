import { FormEvent, useEffect, useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { request } from "../../api/request";
import type { Permission, Role } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";

type PermissionState = Record<string, string[]>;

export function RoleManagementPage() {
  const queryClient = useQueryClient();
  const t = useLanguageStore((state) => state.t);
  const [newRole, setNewRole] = useState({ code: "", name: "", description: "" });
  const [selected, setSelected] = useState<PermissionState>({});
  const permissionsQuery = useQuery({
    queryKey: ["permissions"],
    queryFn: () => request<Permission[]>("/api/v1/permissions")
  });
  const rolesQuery = useQuery({
    queryKey: ["roles"],
    queryFn: () => request<Role[]>("/api/v1/roles")
  });

  useEffect(() => {
    if (!rolesQuery.data) {
      return;
    }
    setSelected((current) => {
      const next = { ...current };
      for (const role of rolesQuery.data) {
        if (!next[role.id]) {
          next[role.id] = role.permissions.map((permission) => permission.code);
        }
      }
      return next;
    });
  }, [rolesQuery.data]);

  const permissionsByModule = useMemo(() => {
    const groups = new Map<string, Permission[]>();
    for (const permission of permissionsQuery.data ?? []) {
      groups.set(permission.module, [...(groups.get(permission.module) ?? []), permission]);
    }
    return [...groups.entries()];
  }, [permissionsQuery.data]);

  const createRole = useMutation({
    mutationFn: () =>
      request<Role>("/api/v1/roles", {
        method: "POST",
        body: JSON.stringify({
          ...newRole,
          permissions: permissionsQuery.data?.filter((permission) => permission.code.endsWith(".read") || permission.code.endsWith(":read")).map((p) => p.code) ?? []
        })
      }),
    onSuccess: async () => {
      setNewRole({ code: "", name: "", description: "" });
      await queryClient.invalidateQueries({ queryKey: ["roles"] });
    }
  });

  const savePermissions = useMutation({
    mutationFn: (role: Role) =>
      request<Role>(`/api/v1/roles/${role.id}/permissions`, {
        method: "PUT",
        body: JSON.stringify({ permissions: selected[role.id] ?? [] })
      }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["roles"] })
  });

  function toggle(roleID: string, permission: string) {
    setSelected((current) => {
      const rolePermissions = new Set(current[roleID] ?? []);
      if (rolePermissions.has(permission)) {
        rolePermissions.delete(permission);
      } else {
        rolePermissions.add(permission);
      }
      return { ...current, [roleID]: [...rolePermissions].sort() };
    });
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    createRole.mutate();
  }

  const roles = rolesQuery.data ?? [];

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">RBAC</p>
          <h1>{t("roles.title")}</h1>
        </div>
      </section>

      <section className="management-grid">
        <form className="panel management-form" onSubmit={submit}>
          <div className="panel-title">
            <h3>{t("roles.create")}</h3>
            <span>{createRole.isPending ? t("common.saving") : t("common.custom")}</span>
          </div>
          <label>
            <span>{t("roles.code")}</span>
            <input
              value={newRole.code}
              placeholder="ops.viewer"
              onChange={(event) => setNewRole({ ...newRole, code: event.target.value })}
              required
            />
          </label>
          <label>
            <span>{t("roles.name")}</span>
            <input
              value={newRole.name}
              placeholder={t("roles.name")}
              onChange={(event) => setNewRole({ ...newRole, name: event.target.value })}
              required
            />
          </label>
          <label>
            <span>{t("roles.description")}</span>
            <input
              value={newRole.description}
              placeholder={t("roles.description")}
              onChange={(event) => setNewRole({ ...newRole, description: event.target.value })}
            />
          </label>
          {createRole.isError ? <p className="form-error">{String(createRole.error.message)}</p> : null}
          <button type="submit" disabled={createRole.isPending}>
            {t("roles.create")}
          </button>
        </form>

        <section className="panel permission-catalog">
          <div className="panel-title">
            <h3>{t("roles.catalog")}</h3>
            <span>
              {permissionsQuery.data?.length ?? 0} {t("roles.permissions")}
            </span>
          </div>
          <div className="permission-groups">
            {permissionsByModule.map(([module, permissions]) => (
              <div key={module}>
                <strong>{module}</strong>
                <div>
                  {permissions.map((permission) => (
                    <span key={permission.code}>{permission.code}</span>
                  ))}
                </div>
              </div>
            ))}
          </div>
        </section>
      </section>

      <section className="role-list">
        {roles.map((role) => (
          <article className="panel role-card" key={role.id}>
            <div className="panel-title">
              <div>
                <h3>
                  {role.name} <small>{role.code}</small>
                </h3>
                <p className="muted">{role.description || (role.builtIn ? t("roles.builtIn") : t("common.custom"))}</p>
              </div>
              <button type="button" disabled={role.code === "admin"} onClick={() => savePermissions.mutate(role)}>
                {t("roles.savePermissions")}
              </button>
            </div>
            <div className="permission-checks">
              {(permissionsQuery.data ?? []).map((permission) => (
                <label key={permission.code}>
                  <input
                    type="checkbox"
                    disabled={role.code === "admin"}
                    checked={(selected[role.id] ?? []).includes(permission.code)}
                    onChange={() => toggle(role.id, permission.code)}
                  />
                  <span>{permission.code}</span>
                </label>
              ))}
            </div>
          </article>
        ))}
      </section>
    </main>
  );
}
