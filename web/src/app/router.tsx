import { Suspense, lazy } from "react";
import type { ComponentType, ReactNode } from "react";
import { createBrowserRouter, Link, Navigate, Outlet, useMatches } from "react-router-dom";

import type { MessageKey } from "../i18n/language";
import { useLanguageStore } from "../i18n/language";
import { BasicLayout } from "../layouts/BasicLayout";
import { AuthLayout } from "../layouts/AuthLayout";
import { LoginPage } from "../modules/auth/LoginPage";
import { hasPermission } from "../modules/auth/permissions";
import { useAuthStore } from "../modules/auth/store";

const AgentManagementPage = lazyNamed(() => import("../modules/agents/AgentManagementPage"), "AgentManagementPage");
const AuditLogsPage = lazyNamed(() => import("../modules/audits/AuditLogsPage"), "AuditLogsPage");
const DashboardPage = lazyNamed(() => import("../modules/dashboard/DashboardPage"), "DashboardPage");
const IncidentPage = lazyNamed(() => import("../modules/incidents/IncidentPage"), "IncidentPage");
const MetricsPage = lazyNamed(() => import("../modules/metrics/MetricsPage"), "MetricsPage");
const NotificationsPage = lazyNamed(() => import("../modules/notifications/NotificationsPage"), "NotificationsPage");
const RoleManagementPage = lazyNamed(() => import("../modules/roles/RoleManagementPage"), "RoleManagementPage");
const SecurityReviewPage = lazyNamed(() => import("../modules/security/SecurityReviewPage"), "SecurityReviewPage");
const UserManagementPage = lazyNamed(() => import("../modules/users/UserManagementPage"), "UserManagementPage");
const ScriptEditorPage = lazyNamed(() => import("../modules/scripts/ScriptEditorPage"), "ScriptEditorPage");
const ScriptListPage = lazyNamed(() => import("../modules/scripts/ScriptListPage"), "ScriptListPage");
const ScheduleListPage = lazyNamed(() => import("../modules/schedules/ScheduleListPage"), "ScheduleListPage");
const TaskCreatePage = lazyNamed(() => import("../modules/tasks/TaskCreatePage"), "TaskCreatePage");
const TaskDetailPage = lazyNamed(() => import("../modules/tasks/TaskDetailPage"), "TaskDetailPage");
const TaskListPage = lazyNamed(() => import("../modules/tasks/TaskListPage"), "TaskListPage");
const WebhookPage = lazyNamed(() => import("../modules/webhooks/WebhookPage"), "WebhookPage");
const WorkflowPage = lazyNamed(() => import("../modules/workflows/WorkflowPage"), "WorkflowPage");
const TraceCenterPage = lazyNamed(() => import("../modules/tracecenter/TraceCenterPage"), "TraceCenterPage");

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

function LazyRoute({ children }: { children: ReactNode }) {
  return (
    <Suspense fallback={<PageFallback />}>
      {children}
    </Suspense>
  );
}

function PageFallback() {
  const t = useLanguageStore((state) => state.t);
  return (
    <main className="page">
      <section className="panel empty-panel">
        <p className="empty-state">{t("common.loading")}</p>
      </section>
    </main>
  );
}

function lazyNamed<T extends Record<string, ComponentType<any>>>(loader: () => Promise<T>, exportName: keyof T) {
  return lazy(async () => ({ default: (await loader())[exportName] as ComponentType<any> }));
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
            element: <LazyRoute><DashboardPage /></LazyRoute>,
            handle: { meta: { permission: "workspace.read", titleKey: "dashboard.title", sectionKey: "layout.controlPlane" } }
          },
          {
            path: "users",
            element: <LazyRoute><UserManagementPage /></LazyRoute>,
            handle: { meta: { permission: "user.read", titleKey: "users.title", sectionKey: "layout.controlPlane" } }
          },
          {
            path: "roles",
            element: <LazyRoute><RoleManagementPage /></LazyRoute>,
            handle: { meta: { permission: "role.read", titleKey: "roles.title", sectionKey: "layout.controlPlane" } }
          },
          {
            path: "agents",
            element: <LazyRoute><AgentManagementPage /></LazyRoute>,
            handle: { meta: { permission: "agent:read", titleKey: "agents.title", sectionKey: "layout.controlPlane" } }
          },
          { path: "scripts", element: <LazyRoute><ScriptListPage /></LazyRoute>, handle: { meta: { permission: "script:read", titleKey: "scripts.title", sectionKey: "layout.controlPlane" } } },
          { path: "scripts/new", element: <LazyRoute><ScriptEditorPage /></LazyRoute>, handle: { meta: { permission: "script:write", titleKey: "scripts.createTitle", sectionKey: "layout.controlPlane" } } },
          { path: "scripts/:id", element: <LazyRoute><ScriptEditorPage /></LazyRoute>, handle: { meta: { permission: "script:write", titleKey: "scripts.editTitle", sectionKey: "layout.controlPlane" } } },
          { path: "tasks", element: <LazyRoute><TaskListPage /></LazyRoute>, handle: { meta: { permission: "task:read", titleKey: "tasks.title", sectionKey: "layout.controlPlane" } } },
          { path: "tasks/new", element: <LazyRoute><TaskCreatePage /></LazyRoute>, handle: { meta: { permission: "task:execute", titleKey: "tasks.createTitle", sectionKey: "layout.controlPlane" } } },
          { path: "tasks/:id", element: <LazyRoute><TaskDetailPage /></LazyRoute>, handle: { meta: { permission: "task:read", titleKey: "tasks.detailEyebrow", sectionKey: "layout.controlPlane" } } },
          {
            path: "schedules",
            element: <LazyRoute><ScheduleListPage /></LazyRoute>,
            handle: { meta: { permission: "schedule:read", titleKey: "schedules.title", sectionKey: "automation.eyebrow" } }
          },
          {
            path: "webhooks",
            element: <LazyRoute><WebhookPage /></LazyRoute>,
            handle: { meta: { permission: "webhook:read", titleKey: "webhooks.title", sectionKey: "automation.eyebrow" } }
          },
          {
            path: "workflows",
            element: <LazyRoute><WorkflowPage /></LazyRoute>,
            handle: { meta: { permission: "workflow:read", titleKey: "workflows.title", sectionKey: "automation.eyebrow" } }
          },
          {
            path: "metrics",
            element: <LazyRoute><MetricsPage /></LazyRoute>,
            handle: { meta: { permission: "metric:read", titleKey: "metrics.title", sectionKey: "monitoring.eyebrow" } }
          },
          {
            path: "incidents",
            element: <LazyRoute><IncidentPage /></LazyRoute>,
            handle: { meta: { permission: "alert:read", titleKey: "incidents.title", sectionKey: "monitoring.eyebrow" } }
          },
          {
            path: "notifications",
            element: <LazyRoute><NotificationsPage /></LazyRoute>,
            handle: { meta: { permission: "notification:read", titleKey: "notifications.title", sectionKey: "layout.controlPlane" } }
          },
          {
            path: "audit-logs",
            element: <LazyRoute><AuditLogsPage /></LazyRoute>,
            handle: { meta: { permission: "audit.read", titleKey: "audit.title", sectionKey: "audit.eyebrow" } }
          },
          {
            path: "security-review",
            element: <LazyRoute><SecurityReviewPage /></LazyRoute>,
            handle: { meta: { permission: "security:review", titleKey: "security.title", sectionKey: "audit.eyebrow" } }
          },
          {
            path: "traces",
            element: <LazyRoute><TraceCenterPage /></LazyRoute>,
            handle: { meta: { permission: "traces:read", titleKey: "traceCenter.title", sectionKey: "audit.eyebrow" } }
          },
          {
            path: "trace-center",
            element: <Navigate to="/traces" replace />
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
