# OpsPilot V0.7 Security Permission Matrix

> Language: [简体中文](./security-permission-matrix.md) | English (current)

This document tracks the current RBAC surface for V0.7 release validation.

## Permission Codes

| Module | Canonical permission | Legacy aliases | Main capability |
| --- | --- | --- | --- |
| workspace | `workspace.read` | - | View workspace and dashboard metadata. |
| user | `user.read` | - | View users and memberships. |
| user | `user.write` | - | Create and update users. |
| role | `role.read` | - | View roles and permission catalog. |
| role | `role.write` | - | Manage role permission assignments. |
| agent | `agent:read` | `agent.read` | View agents and hosts. |
| agent | `agent:write` | `agent.disable` | Disable agents and revoke tokens. |
| host | `host:read` | - | View host inventory. |
| host | `host:write` | - | Manage host metadata. |
| script | `script:read` | `script.read` | View scripts and versions. |
| script | `script:write` | `script.write` | Create and update scripts. |
| script | `script:approve` | `script.approve` | Approve high-risk script versions. |
| task | `task:read` | `task.read` | View task runs and targets. |
| task | `task:write` | - | Manage task definitions. |
| task | `task:execute` | `task.run` | Create and run tasks. |
| task | `task:cancel` | `task.cancel` | Cancel queued or running tasks. |
| task | `task:log:read` | `log.read` | View and stream task logs. |
| schedule | `schedule:read` | - | View schedules. |
| schedule | `schedule:write` | `schedule.write` | Create and manage schedules. |
| metric | `metric:read` | `metric.read` | View host metrics, trends, alerts. |
| alert | `alert:read` | - | View alert rules, alerts, events, and history. |
| alert | `alert:write` | `alert.write` | Manage alert rules and alert state. |
| webhook | `webhook:read` | - | View webhook sources, rules, and events. |
| webhook | `webhook:manage` | `webhook.manage` | Manage webhook sources and trigger rules. |
| notification | `notification:read` | - | View notification channels, messages, and deliveries. |
| notification | `notification:write` | `notification.write` | Manage notification channels and retry/test deliveries. |
| audit | `audit.read` | - | View audit logs. |

## Built-In Roles

| Role | Permissions |
| --- | --- |
| `admin` | All seeded permissions. |
| `member` | `workspace.read`, `agent:read`, `host:read`, `script:read`, `task:read`, `task:log:read`, `metric:read`. |

## Route Gate Summary

| Surface | Read permission | Write/execute permission |
| --- | --- | --- |
| Dashboard | `workspace.read` | - |
| Users | `user.read` | `user.write` |
| Roles | `role.read` | `role.write` |
| Agents and hosts | `agent:read`, `host:read` | `agent:write`, `host:write` |
| Scripts | `script:read` | `script:write`, `script:approve` |
| Tasks | `task:read`, `task:log:read` | `task:execute`, `task:cancel` |
| Schedules | `schedule:read` | `schedule:write` |
| Webhooks | `webhook:read` | `webhook:manage` |
| Metrics and alerts | `metric:read`, `alert:read` | `alert:write` |
| Notifications | `notification:read` | `notification:write` |
| Audit Logs | `audit.read` | - |

## Compatibility Notes

- Backend permission checks accept canonical and legacy aliases through `permissionMatchCodes`.
- Role assignment normalizes aliases to canonical codes when possible.
- Frontend route guards preserve compatibility with legacy dot-style permissions.
- New V0.7 code should prefer colon-style canonical permissions, except `workspace.read`, `user.read`, `role.read`, and similar existing dot-style workspace/admin permissions.
