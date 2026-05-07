import { NavLink, Outlet, useNavigate } from "react-router-dom";

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
  { to: "/metrics", labelKey: "nav.metrics", permission: "metric:read" },
  { to: "/notifications", labelKey: "nav.notifications", permission: "notification:read" },
  { to: "/audit-logs", labelKey: "nav.auditLogs", permission: "audit.read" }
] as const;

export function BasicLayout() {
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const logout = useAuthStore((state) => state.logout);
  const t = useLanguageStore((state) => state.t);
  const toggleLanguage = useLanguageStore((state) => state.toggleLanguage);

  return (
    <div className="app-shell">
      <aside className="sidebar">
        <div className="brand">
          <span className="brand-mark">OP</span>
          <div>
            <strong>OpsPilot</strong>
            <small>Automation Ops</small>
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
      </aside>
      <div className="main-area">
        <header className="topbar">
          <div>
            <p className="eyebrow">{t("layout.workspace")}</p>
            <h2>{user?.workspace.name ?? "OpsPilot"}</h2>
          </div>
          <div className="user-box">
            <span>{user?.username ?? "guest"}</span>
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
