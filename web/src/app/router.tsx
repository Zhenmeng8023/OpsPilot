import { createBrowserRouter, Link, Navigate, Outlet, useMatches } from "react-router-dom";

import type { MessageKey } from "../i18n/language";
import { useLanguageStore } from "../i18n/language";
import { BasicLayout } from "../layouts/BasicLayout";
import { AuthLayout } from "../layouts/AuthLayout";
import { LoginPage } from "../modules/auth/LoginPage";
import { hasPermission } from "../modules/auth/permissions";
import { useAuthStore } from "../modules/auth/store";
import { AgentManagementPage } from "../modules/agents/AgentManagementPage";
import { AuditLogsPage } from "../modules/audits/AuditLogsPage";
import { DashboardPage } from "../modules/dashboard/DashboardPage";
import { IncidentPage } from "../modules/incidents/IncidentPage";
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
import { WorkflowPage } from "../modules/workflows/WorkflowPage";

type RouteHandle = {
  meta?: {
    permission?: string;
    titleKey?: MessageKey;
    sectionKey?: MessageKey;
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
  const t = useLanguageStore((state) => state.t);

  return (
    <main className="page">
      <section className="panel empty-panel">
        <p className="eyebrow">{t("layout.permissionDenied")}</p>
        <h1>403</h1>
        <p className="empty-state">{t("layout.noPermission", { permission })}</p>
        <Link className="ghost-button" to="/dashboard">{t("layout.backDashboard")}</Link>
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
          {
            path: "dashboard",
            element: <DashboardPage />,
            handle: { meta: { permission: "workspace.read", titleKey: "dashboard.title", sectionKey: "layout.controlPlane" } }
          },
          {
            path: "users",
            element: <UserManagementPage />,
            handle: { meta: { permission: "user.read", titleKey: "users.title", sectionKey: "layout.controlPlane" } }
          },
          {
            path: "roles",
            element: <RoleManagementPage />,
            handle: { meta: { permission: "role.read", titleKey: "roles.title", sectionKey: "layout.controlPlane" } }
          },
          {
            path: "agents",
            element: <AgentManagementPage />,
            handle: { meta: { permission: "agent:read", titleKey: "agents.title", sectionKey: "layout.controlPlane" } }
          },
          { path: "scripts", element: <ScriptListPage />, handle: { meta: { permission: "script:read", titleKey: "scripts.title", sectionKey: "layout.controlPlane" } } },
          { path: "scripts/new", element: <ScriptEditorPage />, handle: { meta: { permission: "script:write", titleKey: "scripts.createTitle", sectionKey: "layout.controlPlane" } } },
          { path: "scripts/:id", element: <ScriptEditorPage />, handle: { meta: { permission: "script:write", titleKey: "scripts.editTitle", sectionKey: "layout.controlPlane" } } },
          { path: "tasks", element: <TaskListPage />, handle: { meta: { permission: "task:read", titleKey: "tasks.title", sectionKey: "layout.controlPlane" } } },
          { path: "tasks/new", element: <TaskCreatePage />, handle: { meta: { permission: "task:execute", titleKey: "tasks.createTitle", sectionKey: "layout.controlPlane" } } },
          { path: "tasks/:id", element: <TaskDetailPage />, handle: { meta: { permission: "task:read", titleKey: "tasks.detailEyebrow", sectionKey: "layout.controlPlane" } } },
          {
            path: "schedules",
            element: <ScheduleListPage />,
            handle: { meta: { permission: "schedule:read", titleKey: "schedules.title", sectionKey: "automation.eyebrow" } }
          },
          {
            path: "webhooks",
            element: <WebhookPage />,
            handle: { meta: { permission: "webhook:read", titleKey: "webhooks.title", sectionKey: "automation.eyebrow" } }
          },
          {
            path: "workflows",
            element: <WorkflowPage />,
            handle: { meta: { permission: "workflow:read", titleKey: "workflows.title", sectionKey: "automation.eyebrow" } }
          },
          {
            path: "metrics",
            element: <MetricsPage />,
            handle: { meta: { permission: "metric:read", titleKey: "metrics.title", sectionKey: "monitoring.eyebrow" } }
          },
          {
            path: "incidents",
            element: <IncidentPage />,
            handle: { meta: { permission: "alert:read", titleKey: "incidents.title", sectionKey: "monitoring.eyebrow" } }
          },
          {
            path: "notifications",
            element: <NotificationsPage />,
            handle: { meta: { permission: "notification:read", titleKey: "notifications.title", sectionKey: "layout.controlPlane" } }
          },
          {
            path: "audit-logs",
            element: <AuditLogsPage />,
            handle: { meta: { permission: "audit.read", titleKey: "audit.title", sectionKey: "audit.eyebrow" } }
          }
        ]
      }
    ]
  },
  {
    path: "*",
    element: <Navigate to="/dashboard" replace />
  }
]);
