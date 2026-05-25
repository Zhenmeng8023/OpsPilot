import { useEffect } from "react";
import { NavLink, Outlet, useMatches, useNavigate } from "react-router-dom";

import type { MessageKey } from "../i18n/language";
import { useLanguageStore } from "../i18n/language";
import { hasPermissionCode } from "../modules/auth/permissions";
import { useAuthStore } from "../modules/auth/store";
import {
  IconDashboard, IconUsers, IconRoles, IconAgents, IconScripts,
  IconTasks, IconSchedules, IconWebhooks, IconWorkflows, IconMetrics,
  IconIncidents, IconNotifications, IconTrace, IconAudit,
} from "../shared/components/Icons";

type NavGroup = {
  labelKey: MessageKey;
  items: { to: string; labelKey: MessageKey; permission: string; icon: React.ReactNode }[];
};

const navGroups: NavGroup[] = [
  {
    labelKey: "nav.group.control",
    items: [
      { to: "/dashboard", labelKey: "nav.dashboard", permission: "workspace.read", icon: <IconDashboard /> },
      { to: "/users", labelKey: "nav.users", permission: "user.read", icon: <IconUsers /> },
      { to: "/roles", labelKey: "nav.roles", permission: "role.read", icon: <IconRoles /> },
    ]
  },
  {
    labelKey: "nav.group.execution",
    items: [
      { to: "/agents", labelKey: "nav.agents", permission: "agent:read", icon: <IconAgents /> },
      { to: "/scripts", labelKey: "nav.scripts", permission: "script:read", icon: <IconScripts /> },
      { to: "/tasks", labelKey: "nav.tasks", permission: "task:read", icon: <IconTasks /> },
      { to: "/schedules", labelKey: "nav.schedules", permission: "schedule:read", icon: <IconSchedules /> },
    ]
  },
  {
    labelKey: "nav.group.automation",
    items: [
      { to: "/webhooks", labelKey: "nav.webhooks", permission: "webhook:read", icon: <IconWebhooks /> },
      { to: "/workflows", labelKey: "nav.workflows", permission: "workflow:read", icon: <IconWorkflows /> },
    ]
  },
  {
    labelKey: "nav.group.monitoring",
    items: [
      { to: "/metrics", labelKey: "nav.metrics", permission: "metric:read", icon: <IconMetrics /> },
      { to: "/incidents", labelKey: "nav.incidents", permission: "alert:read", icon: <IconIncidents /> },
      { to: "/notifications", labelKey: "nav.notifications", permission: "notification:read", icon: <IconNotifications /> },
    ]
  },
  {
    labelKey: "nav.group.audit",
    items: [
      { to: "/trace-center", labelKey: "nav.traceCenter", permission: "traces:read", icon: <IconTrace /> },
      { to: "/audit-logs", labelKey: "nav.auditLogs", permission: "audit.read", icon: <IconAudit /> },
      { to: "/security-review", labelKey: "nav.securityReview", permission: "security:review", icon: <IconRoles /> },
    ]
  }
];

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
  const roleSummary = (user?.roles ?? []).join(" / ") || t("layout.controlPlane");

  useEffect(() => {
    document.title = `${pageTitle} - OpsPilot`;
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
          <nav>
            {navGroups.map((group) => {
              const visibleItems = group.items.filter((item) =>
                hasPermissionCode(user?.permissions, item.permission)
              );
              if (visibleItems.length === 0) return null;
              return (
                <div className="nav-group" key={group.labelKey}>
                  <div className="nav-group-label">{t(group.labelKey)}</div>
                  <div className="nav-list">
                    {visibleItems.map((item) => (
                      <NavLink key={item.to} to={item.to}>
                        <span className="nav-icon" aria-hidden="true">{item.icon}</span>
                        {t(item.labelKey)}
                      </NavLink>
                    ))}
                  </div>
                </div>
              );
            })}
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
          <div className="topbar-shell">
            <div className="topbar-context">
              <span className="topbar-kicker">{user?.workspace.name ?? "OpsPilot"}</span>
              <strong>{pageTitle}</strong>
              <p>{sectionTitle}</p>
            </div>
            <div className="user-box">
              <div className="user-meta">
                <strong>{user?.username ?? "guest"}</strong>
                <span>{roleSummary}</span>
              </div>
              <button
                type="button"
                className="language-button"
                aria-label={t("language.label")}
                onClick={toggleLanguage}
              >
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
          </div>
        </header>
        <Outlet />
      </div>
    </div>
  );
}
