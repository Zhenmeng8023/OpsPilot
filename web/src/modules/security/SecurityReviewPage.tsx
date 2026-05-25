import { useEffect, useMemo, useState } from "react";
import { useQuery } from "@tanstack/react-query";

import { request } from "../../api/request";
import { getPermissionDiff, listSecretRotationHistory } from "../../api/security";
import type { Permission, Role } from "../../api/types";
import { useLanguageStore } from "../../i18n/language";
import { Button } from "../../shared/components/Button";
import { DataTable } from "../../shared/components/DataTable";
import { PagePanel } from "../../shared/components/PagePanel";
import { SectionToolbar } from "../../shared/components/SectionToolbar";
import { SelectField } from "../../shared/components/SelectField";
import { TextField } from "../../shared/components/TextField";

type PermissionState = Record<string, string[]>;

export function SecurityReviewPage() {
  const t = useLanguageStore((state) => state.t);
  const [roleId, setRoleId] = useState("");
  const [selected, setSelected] = useState<PermissionState>({});
  const [secretId, setSecretId] = useState("webhook_source");
  const [submittedSecretId, setSubmittedSecretId] = useState("webhook_source");

  const rolesQuery = useQuery({
    queryKey: ["securityRoles"],
    queryFn: () => request<Role[]>("/api/v1/roles")
  });
  const permissionsQuery = useQuery({
    queryKey: ["securityPermissions"],
    queryFn: () => request<Permission[]>("/api/v1/permissions")
  });

  const roles = rolesQuery.data ?? [];
  const permissions = permissionsQuery.data ?? [];
  const selectedRole = roles.find((role) => role.id === roleId);
  const requestedPermissions = selected[roleId] ?? [];

  useEffect(() => {
    if (!roleId && roles.length > 0) {
      setRoleId(roles[0].id);
    }
  }, [roleId, roles]);

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
    for (const permission of permissions) {
      groups.set(permission.module, [...(groups.get(permission.module) ?? []), permission]);
    }
    return [...groups.entries()];
  }, [permissions]);

  const diffQuery = useQuery({
    queryKey: ["securityPermissionDiff", roleId, requestedPermissions],
    queryFn: () => getPermissionDiff({ roleId, permissionCodes: requestedPermissions }),
    enabled: Boolean(roleId)
  });
  const rotationQuery = useQuery({
    queryKey: ["securitySecretRotation", submittedSecretId],
    queryFn: () => listSecretRotationHistory(submittedSecretId),
    enabled: submittedSecretId.trim() !== ""
  });

  function toggle(permissionCode: string) {
    if (!roleId || selectedRole?.code === "admin") {
      return;
    }
    setSelected((current) => {
      const rolePermissions = new Set(current[roleId] ?? []);
      if (rolePermissions.has(permissionCode)) {
        rolePermissions.delete(permissionCode);
      } else {
        rolePermissions.add(permissionCode);
      }
      return { ...current, [roleId]: [...rolePermissions].sort() };
    });
  }

  const roleOptions = roles.map((role) => ({ value: role.id, label: `${role.name} (${role.code})` }));
  const diff = diffQuery.data;
  const rotationEvents = rotationQuery.data ?? [];

  return (
    <main className="page">
      <section className="page-heading">
        <div>
          <p className="eyebrow">{t("security.eyebrow")}</p>
          <h1>{t("security.title")}</h1>
        </div>
      </section>

      <PagePanel title={t("security.permissionDiff")} subtitle={selectedRole?.name ?? t("common.loading")}>
        <SectionToolbar>
          <SelectField
            value={roleId}
            options={roleOptions}
            onChange={(event) => setRoleId(event.target.value)}
          />
          <Button type="button" onClick={() => diffQuery.refetch()} disabled={!roleId || diffQuery.isFetching}>
            {t("common.refresh")}
          </Button>
        </SectionToolbar>

        <div className="summary-grid">
          <div className="summary-card">
            <strong>{t("security.affectedUsers")}</strong>
            <small>{diff?.affectedUsers ?? 0}</small>
          </div>
          <div className="summary-card">
            <strong>{t("security.added")}</strong>
            <small>{diff?.added.length ?? 0}</small>
          </div>
          <div className="summary-card">
            <strong>{t("security.removed")}</strong>
            <small>{diff?.removed.length ?? 0}</small>
          </div>
          <div className="summary-card">
            <strong>{t("security.highRisk")}</strong>
            <small>{diff?.highRisk ? t("common.status.warning") : t("common.status.ok")}</small>
          </div>
        </div>

        <div className="management-grid">
          <div className="permission-catalog">
            <div className="permission-groups">
              {permissionsByModule.map(([module, modulePermissions]) => (
                <div key={module}>
                  <strong>{module}</strong>
                  <div>
                    {modulePermissions.map((permission) => (
                      <label key={permission.code}>
                        <input
                          type="checkbox"
                          disabled={selectedRole?.code === "admin"}
                          checked={requestedPermissions.includes(permission.code)}
                          onChange={() => toggle(permission.code)}
                        />
                        <span>{permission.code}</span>
                      </label>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </div>

          <div className="data-table">
            <table>
              <thead>
                <tr>
                  <th>{t("security.changeType")}</th>
                  <th>{t("roles.permissions")}</th>
                </tr>
              </thead>
              <tbody>
                {(diff?.added ?? []).map((permission) => (
                  <tr key={`added:${permission}`}>
                    <td><span className="status-chip status-warning">{t("security.added")}</span></td>
                    <td>{permission}</td>
                  </tr>
                ))}
                {(diff?.removed ?? []).map((permission) => (
                  <tr key={`removed:${permission}`}>
                    <td><span className="status-chip status-disabled">{t("security.removed")}</span></td>
                    <td>{permission}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {!diffQuery.isLoading && (diff?.added.length ?? 0) + (diff?.removed.length ?? 0) === 0 ? (
              <p className="empty-state">{t("security.noPermissionChanges")}</p>
            ) : null}
            {diffQuery.isError ? <p className="form-error">{diffQuery.error.message}</p> : null}
          </div>
        </div>
      </PagePanel>

      <PagePanel title={t("security.secretRotation")} subtitle={submittedSecretId}>
        <form
          className="toolbar-row"
          onSubmit={(event) => {
            event.preventDefault();
            setSubmittedSecretId(secretId.trim());
          }}
        >
          <TextField
            label={t("security.secretId")}
            value={secretId}
            onChange={(event) => setSecretId(event.target.value)}
            placeholder="webhook_source"
          />
          <Button type="submit" disabled={rotationQuery.isFetching}>{t("security.queryRotation")}</Button>
        </form>

        <DataTable loading={rotationQuery.isLoading} empty={rotationEvents.length === 0} emptyMessage={t("common.empty")} error={rotationQuery.isError ? rotationQuery.error.message : null}>
          <table>
            <thead>
              <tr>
                <th>{t("common.action")}</th>
                <th>{t("security.operator")}</th>
                <th>{t("common.source")}</th>
                <th>{t("traceCenter.traceId")}</th>
                <th>{t("common.createdAt")}</th>
              </tr>
            </thead>
            <tbody>
              {rotationEvents.map((event) => (
                <tr key={event.id}>
                  <td><strong>{event.action}</strong><small>{event.secretId || submittedSecretId}</small></td>
                  <td>{event.operator || "-"}</td>
                  <td>{[event.resourceType, event.resourceId].filter(Boolean).join(" / ") || "-"}</td>
                  <td>{event.traceId || "-"}</td>
                  <td>{event.createdAt}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </DataTable>
      </PagePanel>
    </main>
  );
}
