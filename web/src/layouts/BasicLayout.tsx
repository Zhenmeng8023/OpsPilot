import { NavLink, Outlet, useNavigate } from "react-router-dom";

import { useAuthStore } from "../modules/auth/store";

const navItems = [
  { to: "/dashboard", label: "仪表盘" },
  { to: "/users", label: "用户管理" },
  { to: "/roles", label: "权限管理" },
  { to: "/dashboard", label: "Agent 管理" },
  { to: "/dashboard", label: "任务执行" }
];

export function BasicLayout() {
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const logout = useAuthStore((state) => state.logout);

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
          {navItems.map((item) => (
            <NavLink key={item.label} to={item.to}>
              {item.label}
            </NavLink>
          ))}
        </nav>
      </aside>
      <div className="main-area">
        <header className="topbar">
          <div>
            <p className="eyebrow">Workspace</p>
            <h2>{user?.workspace.name ?? "OpsPilot"}</h2>
          </div>
          <div className="user-box">
            <span>{user?.username ?? "guest"}</span>
            <button
              type="button"
              onClick={() => {
                void logout();
                navigate("/login", { replace: true });
              }}
            >
              退出
            </button>
          </div>
        </header>
        <Outlet />
      </div>
    </div>
  );
}
