import { createBrowserRouter, Navigate, Outlet } from "react-router-dom";

import { BasicLayout } from "../layouts/BasicLayout";
import { AuthLayout } from "../layouts/AuthLayout";
import { LoginPage } from "../modules/auth/LoginPage";
import { useAuthStore } from "../modules/auth/store";
import { AgentManagementPage } from "../modules/agents/AgentManagementPage";
import { DashboardPage } from "../modules/dashboard/DashboardPage";
import { RoleManagementPage } from "../modules/roles/RoleManagementPage";
import { UserManagementPage } from "../modules/users/UserManagementPage";
import { ScriptEditorPage } from "../modules/scripts/ScriptEditorPage";
import { ScriptListPage } from "../modules/scripts/ScriptListPage";
import { TaskCreatePage } from "../modules/tasks/TaskCreatePage";
import { TaskDetailPage } from "../modules/tasks/TaskDetailPage";
import { TaskListPage } from "../modules/tasks/TaskListPage";

function ProtectedRoute() {
  const token = useAuthStore((state) => state.token);
  if (!token) {
    return <Navigate to="/login" replace />;
  }
  return <Outlet />;
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
          { path: "dashboard", element: <DashboardPage /> },
          { path: "users", element: <UserManagementPage /> },
          { path: "roles", element: <RoleManagementPage /> },
          { path: "agents", element: <AgentManagementPage /> },
          { path: "scripts", element: <ScriptListPage /> },
          { path: "scripts/:id", element: <ScriptEditorPage /> },
          { path: "tasks", element: <TaskListPage /> },
          { path: "tasks/new", element: <TaskCreatePage /> },
          { path: "tasks/:id", element: <TaskDetailPage /> }
        ]
      }
    ]
  },
  {
    path: "*",
    element: <Navigate to="/dashboard" replace />
  }
]);
