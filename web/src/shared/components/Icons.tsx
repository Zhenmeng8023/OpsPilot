import React from "react";

// SVG icon paths — 24x24, stroke-based, minimal
function SvgIcon({ d, label }: { d: string; label?: string }) {
  return (
    <svg
      width="18"
      height="18"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.8"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-label={label}
      aria-hidden={!label}
    >
      <path d={d} />
    </svg>
  );
}

// Dashboard — grid/layout icon
export const IconDashboard = () => (
  <SvgIcon d="M3 3h7v7H3V3zm11 0h7v7h-7V3zM3 14h7v7H3v-7zm11 0h7v7h-7v-7z" label="Dashboard" />
);

// Users — people icon
export const IconUsers = () => (
  <SvgIcon d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2M9 11a4 4 0 1 0 0-8 4 4 0 0 0 0 8zm14 10v-2a4 4 0 0 0-3-3.87M16 3.13a4 4 0 0 1 0 7.75" label="Users" />
);

// Roles — shield/key icon
export const IconRoles = () => (
  <SvgIcon d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10zM12 8v4M12 16h.01" label="Roles" />
);

// Agents — terminal/bot icon
export const IconAgents = () => (
  <SvgIcon d="M12 2a4 4 0 0 1 4 4M8 2a4 4 0 0 0-4 4m16 6v1a2 2 0 0 1-2 2h-1M6 12v1a2 2 0 0 1-2 2H3m18-4a6 6 0 0 0-6-6h-1M3 8a6 6 0 0 1 6-6h1m2 14h4M12 18v4m-4 0h8" label="Agents" />
);

// Scripts — file/code icon
export const IconScripts = () => (
  <SvgIcon d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8zm-4 14l-4-4 4-4m4 8l4-4-4-4" label="Scripts" />
);

// Tasks — zap/lightning icon
export const IconTasks = () => (
  <SvgIcon d="M13 2L3 14h9l-1 8 10-12h-9l1-8z" label="Tasks" />
);

// Schedules — clock icon
export const IconSchedules = () => (
  <SvgIcon d="M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10zm0-6v-4l-3-3" label="Schedules" />
);

// Webhooks — link/hook icon
export const IconWebhooks = () => (
  <SvgIcon d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71" label="Webhooks" />
);

// Workflows — git-branch/flow icon
export const IconWorkflows = () => (
  <SvgIcon d="M6 3v12M18 9a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM6 21a3 3 0 1 0 0-6 3 3 0 0 0 0 6zM18 9a9 9 0 0 1-9 9" label="Workflows" />
);

// Metrics — bar chart icon
export const IconMetrics = () => (
  <SvgIcon d="M18 20V10M12 20V4M6 20v-6" label="Metrics" />
);

// Incidents — alert triangle icon
export const IconIncidents = () => (
  <SvgIcon d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0zM12 9v4M12 17h.01" label="Incidents" />
);

// Notifications — bell icon
export const IconNotifications = () => (
  <SvgIcon d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9M13.73 21a2 2 0 0 1-3.46 0" label="Notifications" />
);

// Trace Center — search icon
export const IconTrace = () => (
  <SvgIcon d="M11 19a8 8 0 1 0 0-16 8 8 0 0 0 0 16zm10 2l-4.35-4.35" label="Trace" />
);

// Audit Logs — clipboard icon
export const IconAudit = () => (
  <SvgIcon d="M16 4h2a2 2 0 0 1 2 2v14a2 2 0 0 1-2 2H6a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2h2M9 2h6a1 1 0 0 1 1 1v1a1 1 0 0 1-1 1H9a1 1 0 0 1-1-1V3a1 1 0 0 1 1-1z" label="Audit" />
);

// Quick Action icons
export const IconPlus = () => (
  <SvgIcon d="M12 5v14M5 12h14" label="Add" />
);

export const IconEdit = () => (
  <SvgIcon d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z" label="Edit" />
);

export const IconClock = () => (
  <SvgIcon d="M12 22c5.523 0 10-4.477 10-10S17.523 2 12 2 2 6.477 2 12s4.477 10 10 10zm0-6v-4l-3-3" label="Clock" />
);
