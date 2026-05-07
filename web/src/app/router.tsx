import { createBrowserRouter, Link, Navigate, Outlet, useMatches } from "react-router-dom";

import { BasicLayout } from "../layouts/BasicLayout";
import { AuthLayout } from "../layouts/AuthLayout";
import { LoginPage } from "../modules/auth/LoginPage";
import { hasPermission } from "../modules/auth/permissions";
import { useAuthStore } from "../modules/auth/store";
import { AgentManagementPage } from "../modules/agents/AgentManagementPage";
import { AuditLogsPage } from "../modules/audits/AuditLogsPage";
import { DashboardPage } from "../modules/dashboard/DashboardPage";
import { MetricsPage } from "../modules/metrics/MetricsPage";
import { NotificationsPage } from "../modules/notifications/NotificationsPage";
import { RoleManagementPage } from "../modules/roles/RoleManagementPage";
import { UserManagementPage } from "../modules/users/UserManagementPage";
import { ScriptEditorPage } from "../modules/scripts/ScriptEditorPage";
import { ScriptListPage } from "../modules/scripts/ScriptListPage";
import { ScheduleListPage } from "../modules/schedules/ScheduleListPage";
import { TaskCreatePage } from "../modules/tasks/TaskCreatePage";
import { TaskDetailPage } from "../modules/tasks/TaskDetailPage";
import { TaskListPage } from "../modules/tasks/TaskListPage";
import { WebhookPage } from "../modules/webhooks/WebhookPage";

type RouteHandle = {
  meta?: {
    permission?: string;
  };
};

function ProtectedRoute() {
  const token = useAuthStore((state) => state.token);
  const user = useAuthStore((state) => state.user);
  const matches = useMatches();
  const permission = requiredPermission(matches.map((match) => match.handle));

  if (!token || !user) {
    return <Navigate to="/login" replace />;
  }
  if (permission && !hasPermission(user, permission)) {
    return <ForbiddenPage permission={permission} />;
  }
  return <Outlet />;
}

function ForbiddenPage({ permission }: { permission: string }) {
  return (
    <main className="page">
      <section className="panel empty-panel">
        <p className="eyebrow">Permission denied</p>
        <h1>403</h1>
        <p className="empty-state">Current account does not have {permission}.</p>
        <Link className="ghost-button" to="/dashboard">Back to dashboard</Link>
      </section>
    </main>
  );
}

function requiredPermission(handles: unknown[]) {
  for (let index = handles.length - 1; index >= 0; index -= 1) {
    const handle = handles[index] as RouteHandle | undefined;
    const permission = handle?.meta?.permission;
    if (permission) return permission;
  }
  return "";
}

export const router = createBrowserRouter([
  {
    path: "/login",
    element: (
      <AuthLayout>
        <LoginPage />
      </AuthLayout>
    )
  },
  {
    path: "/",
    element: <ProtectedRoute />,
    children: [
      {
        element: <BasicLayout />,
        children: [
          { index: true, element: <Navigate to="/dashboard" replace /> },
          { path: "dashboard", element: <DashboardPage />, handle: { meta: { permission: "workspace.read" } } },
          { path: "users", element: <UserManagementPage />, handle: { meta: { permission: "user.read" } } },
          { path: "roles", element: <RoleManagementPage />, handle: { meta: { permission: "role.read" } } },
          { path: "agents", element: <AgentManagementPage />, handle: { meta: { permission: "agent:read" } } },
          { path: "scripts", element: <ScriptListPage />, handle: { meta: { permission: "script:read" } } },
          { path: "scripts/new", element: <ScriptEditorPage />, handle: { meta: { permission: "script:write" } } },
          { path: "scripts/:id", element: <ScriptEditorPage />, handle: { meta: { permission: "script:write" } } },
          { path: "tasks", element: <TaskListPage />, handle: { meta: { permission: "task:read" } } },
          { path: "tasks/new", element: <TaskCreatePage />, handle: { meta: { permission: "task:execute" } } },
          { path: "tasks/:id", element: <TaskDetailPage />, handle: { meta: { permission: "task:read" } } },
          { path: "schedules", element: <ScheduleListPage />, handle: { meta: { permission: "schedule:read" } } },
          { path: "webhooks", element: <WebhookPage />, handle: { meta: { permission: "webhook:read" } } },
          { path: "metrics", element: <MetricsPage />, handle: { meta: { permission: "metric:read" } } },
          { path: "notifications", element: <NotificationsPage />, handle: { meta: { permission: "notification:read" } } },
          { path: "audit-logs", element: <AuditLogsPage />, handle: { meta: { permission: "audit.read" } } }
        ]
      }
    ]
  },
  {
    path: "*",
    element: <Navigate to="/dashboard" replace />
  }
]);
