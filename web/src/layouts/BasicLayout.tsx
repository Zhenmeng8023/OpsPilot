import { useEffect } from "react";
import { NavLink, Outlet, useMatches, useNavigate } from "react-router-dom";

import type { MessageKey } from "../i18n/language";
import { useLanguageStore } from "../i18n/language";
import { hasPermissionCode } from "../modules/auth/permissions";
import { useAuthStore } from "../modules/auth/store";

const navItems = [
  { to: "/dashboard", labelKey: "nav.dashboard", permission: "workspace.read" },
  { to: "/users", labelKey: "nav.users", permission: "user.read" },
  { to: "/roles", labelKey: "nav.roles", permission: "role.read" },
  { to: "/agents", labelKey: "nav.agents", permission: "agent:read" },
  { to: "/scripts", labelKey: "nav.scripts", permission: "script:read" },
  { to: "/tasks", labelKey: "nav.tasks", permission: "task:read" },
  { to: "/schedules", labelKey: "nav.schedules", permission: "schedule:read" },
  { to: "/webhooks", labelKey: "nav.webhooks", permission: "webhook:read" },
  { to: "/workflows", labelKey: "nav.workflows", permission: "workflow:read" },
  { to: "/metrics", labelKey: "nav.metrics", permission: "metric:read" },
  { to: "/incidents", labelKey: "nav.incidents", permission: "alert:read" },
  { to: "/notifications", labelKey: "nav.notifications", permission: "notification:read" },
  { to: "/trace-center", labelKey: "nav.traceCenter", permission: "audit.read" },
  { to: "/audit-logs", labelKey: "nav.auditLogs", permission: "audit.read" }
] as const;

type LayoutHandle = {
  meta?: {
    titleKey?: MessageKey;
    sectionKey?: MessageKey;
  };
};

export function BasicLayout() {
  const navigate = useNavigate();
  const matches = useMatches();
  const user = useAuthStore((state) => state.user);
  const logout = useAuthStore((state) => state.logout);
  const t = useLanguageStore((state) => state.t);
  const toggleLanguage = useLanguageStore((state) => state.toggleLanguage);
  const currentMeta = [...matches]
    .reverse()
    .map((match) => (match.handle as LayoutHandle | undefined)?.meta)
    .find((meta) => meta?.titleKey);
  const pageTitle = currentMeta?.titleKey ? t(currentMeta.titleKey) : user?.workspace.name ?? "OpsPilot";
  const sectionTitle = currentMeta?.sectionKey ? t(currentMeta.sectionKey) : t("layout.controlPlane");
  const roleSummary = (user?.roles ?? []).join(" · ") || t("layout.controlPlane");

  useEffect(() => {
    document.title = `${pageTitle} · OpsPilot`;
  }, [pageTitle]);

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="shell-stack">
          <div className="brand">
            <span className="brand-mark">OP</span>
            <div>
              <strong>OpsPilot</strong>
              <small>{t("layout.controlPlane")}</small>
            </div>
          </div>
          <nav className="nav-list">
            {navItems
              .filter((item) => hasPermissionCode(user?.permissions, item.permission))
              .map((item) => (
                <NavLink key={item.to} to={item.to}>
                  {t(item.labelKey)}
                </NavLink>
              ))}
          </nav>
        </div>
        <div className="sidebar-footer">
          <small>{t("layout.workspace")}</small>
          <strong>{user?.workspace.name ?? "OpsPilot"}</strong>
          <span>{roleSummary}</span>
        </div>
      </aside>
      <div className="main-area">
        <header className="topbar">
          <div className="topbar-context">
            <p className="eyebrow">{sectionTitle}</p>
            <h2>{pageTitle}</h2>
            <p className="topbar-subtitle">{user?.workspace.slug ?? user?.workspace.name ?? "OpsPilot"}</p>
          </div>
          <div className="user-box">
            <div className="user-meta">
              <strong>{user?.username ?? "guest"}</strong>
              <span>{roleSummary}</span>
            </div>
            <button type="button" className="language-button" aria-label={t("language.label")} onClick={toggleLanguage}>
              {t("language.toggle")}
            </button>
            <button
              type="button"
              onClick={() => {
                void logout();
                navigate("/login", { replace: true });
              }}
            >
              {t("common.logout")}
            </button>
          </div>
        </header>
        <Outlet />
      </div>
    </div>
  );
}
