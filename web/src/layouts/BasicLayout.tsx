import { useEffect } from "react";
import { NavLink, Outlet, useMatches, useNavigate } from "react-router-dom";

import type { MessageKey } from "../i18n/language";
import { useLanguageStore } from "../i18n/language";
import { hasPermissionCode } from "../modules/auth/permissions";
import { useAuthStore } from "../modules/auth/store";

type NavGroup = {
  labelKey: MessageKey;
  items: { to: string; labelKey: MessageKey; permission: string; icon: string }[];
};

const navGroups: NavGroup[] = [
  {
    labelKey: "nav.group.control",
    items: [
      { to: "/dashboard", labelKey: "nav.dashboard", permission: "workspace.read", icon: "?" },
      { to: "/users", labelKey: "nav.users", permission: "user.read", icon: "??" },
      { to: "/roles", labelKey: "nav.roles", permission: "role.read", icon: "??" },
    ]
  },
  {
    labelKey: "nav.group.execution",
    items: [
      { to: "/agents", labelKey: "nav.agents", permission: "agent:read", icon: "??" },
      { to: "/scripts", labelKey: "nav.scripts", permission: "script:read", icon: "??" },
      { to: "/tasks", labelKey: "nav.tasks", permission: "task:read", icon: "?" },
      { to: "/schedules", labelKey: "nav.schedules", permission: "schedule:read", icon: "?" },
    ]
  },
  {
    labelKey: "nav.group.automation",
    items: [
      { to: "/webhooks", labelKey: "nav.webhooks", permission: "webhook:read", icon: "??" },
      { to: "/workflows", labelKey: "nav.workflows", permission: "workflow:read", icon: "??" },
    ]
  },
  {
    labelKey: "nav.group.monitoring",
    items: [
      { to: "/metrics", labelKey: "nav.metrics", permission: "metric:read", icon: "??" },
      { to: "/incidents", labelKey: "nav.incidents", permission: "alert:read", icon: "??" },
      { to: "/notifications", labelKey: "nav.notifications", permission: "notification:read", icon: "??" },
    ]
  },
  {
    labelKey: "nav.group.audit",
    items: [
      { to: "/trace-center", labelKey: "nav.traceCenter", permission: "audit.read", icon: "??" },
      { to: "/audit-logs", labelKey: "nav.auditLogs", permission: "audit.read", icon: "??" },
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
  const roleSummary = (user?.roles ?? []).join(" ? ") || t("layout.controlPlane");

  useEffect(() => {
    document.title = `${pageTitle} ? OpsPilot`;
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
          <div className="topbar-context">
            <p className="eyebrow">{sectionTitle}</p>
            <h2>{pageTitle}</h2>
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
        </header>
        <Outlet />
      </div>
    </div>
  );
}
