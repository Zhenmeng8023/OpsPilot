# OpsPilot

> Language: [简体中文](./README.md) | English (current)

OpsPilot is an automation operations platform built with Go + Gin + React + TypeScript. The project now covers Agent onboarding, script templates, task execution, live logs, Cron scheduling, Webhook triggering, basic metrics, alerts, notifications, RBAC, and audit logs.

## Current Stage

- T1 Auth/RBAC: login, registration, JWT access/refresh token, users/roles/permissions, protected routes.
- T2 Agent/Host: Agent registration, token hashing, heartbeat, Agent/Host list, disable, token revoke, offline scan.
- T3 Task execution: script templates, task creation, target dispatch, Agent poll/claim, command execution, result reporting, status aggregation.
- T4 Realtime logs: Agent stdout/stderr/system log upload, redaction, chunk persistence, query APIs, and SSE streaming.
- T5 Cron scheduling: schedules API, preview, misfire policy, trigger history, background scanner, and Schedules page; due schedules reuse the current execution chain.
- T6 Webhook: source/rule management, tokenized trigger endpoint, split signing secret, HMAC-SHA256 validation, replay prevention, rate limiting, event persistence, matcher debugging, and task run generation.
- T7 Metrics/Alerts/Notifications: Agent OS metrics upload, host_metrics persistence, metric threshold alerts, site/Email/webhook-like notifications, delivery records, retry, and test send.

## Implemented

- Cron scheduling loop: schedules API, preview, misfire policy, trigger history, background scanner, and Schedules page; due schedules reuse the existing task_run / task_target / Agent poll / logs / result execution chain.
- Frontend permission matrix: routes declare `handle.meta.permission`, the route guard checks user permissions, and navigation/key actions are permission-aware.
- Webhook trigger loop: `POST /api/v1/webhooks/trigger/:token` requires `X-OpsPilot-Signature` or `X-Hub-Signature-256`; matched rules create task runs through the existing execution chain.
- Metrics / Alerts / Notifications loop: Agent OS metrics upload, host_metrics persistence, trend charts, metric threshold alert rules, alert events, site/Email notifications, webhook/dingtalk/wechat/slack dispatching, delivery records, retry, and test send.

- Unified response envelope: `code`, `message`, `data`, `traceId`.
- Backend layered architecture: handler/service/repository.
- RBAC for scripts, tasks, logs, agents, and hosts.
- Agent Token auth isolated from user JWT auth.
- Script revisioning via `script_versions.version_no`.
- Task state machine: `pending -> queued -> running -> success|failed|timeout`, and `pending|queued -> canceled`.
- Agent executor: concurrency control, timeout handling, log upload retry, result reporting.
- Frontend pages for scripts, script approvals, tasks, task logs, agents/hosts, and enrollment tokens.
- Policy-enforced approval gate: scripts with `approvalRequired=true` must have latest approval status `approved` before task execution.
- Task execution safety gate: high-risk command blocking, configurable command allowlist/denylist, maximum timeout enforcement, per-task Agent work directories, Agent-side log redaction before upload, and local process termination when canceling a running task.
- Audit events for auth, scripts, tasks, agents, and role permission updates.

## Not Yet Implemented

- Visual workflow orchestration beyond the current Cron scheduling loop.
- Advanced JSONPath webhook matcher expressions and replay-window tuning.
- Notification template editing and encrypted channel config storage.

## Requirements

- Go 1.26.2
- Node.js 24.x
- npm 11.x
- MySQL 8.0.39
- Redis 7.x

## Start Infrastructure

```powershell
cd D:\+\1108026_rust_go\OpsPilot
docker compose -f .\deploy\docker-compose.yml up -d
```

To start migrate/API/Web/Agent through Compose:

```powershell
cd D:\+\1108026_rust_go\OpsPilot
Copy-Item .\deploy\.env.example .\deploy\.env -ErrorAction SilentlyContinue
docker compose --env-file .\deploy\.env -f .\deploy\docker-compose.yml --profile full up --build
```

## Database Migration

```powershell
cd D:\+\1108026_rust_go\OpsPilot
mysql -h 127.0.0.1 -P 3306 -uopspilot -popspilot opspilot < .\server\migrations\000001_init_mysql_schema.up.sql
mysql -h 127.0.0.1 -P 3306 -uopspilot -popspilot opspilot < .\server\migrations\000002_seed_initial_auth_data.up.sql
mysql -h 127.0.0.1 -P 3306 -uopspilot -popspilot opspilot < .\server\migrations\000003_task_execution_security.up.sql
```

## Backend

```powershell
cd D:\+\1108026_rust_go\OpsPilot\server
Copy-Item .\.env.example .\.env
go mod tidy
go run .\cmd\api
```

Key environment variables:

```text
DATABASE_DSN=opspilot:opspilot@tcp(127.0.0.1:3306)/opspilot?charset=utf8mb4&parseTime=True&loc=Local
HTTP_ADDR=:8080
HTTP_ALLOW_ORIGIN=http://localhost:5173
AGENT_BOOTSTRAP_SECRET=dev-agent-bootstrap-secret
AGENT_REGISTRATION_ENABLED=true
AGENT_OFFLINE_SCAN_INTERVAL_SECONDS=45
TASK_COMMAND_ALLOW_PATTERNS=
TASK_COMMAND_DENY_PATTERNS=
SCHEDULE_SCAN_INTERVAL_SECONDS=30
ALERT_SCAN_INTERVAL_SECONDS=30
NOTIFICATION_DISPATCH_INTERVAL_SECONDS=15
NOTIFICATION_HTTP_TIMEOUT_SECONDS=10
```

Production safeguards:

- When `APP_ENV=prod`, `/auth/register` is disabled by default unless `AUTH_PUBLIC_REGISTRATION_ENABLED=true` is set explicitly.
- When `APP_ENV=prod`, `JWT_ACCESS_SECRET` and `JWT_REFRESH_SECRET` must be non-empty, must not use the default development values, and must be at least 32 characters.
- `HTTP_ALLOW_ORIGIN` supports a comma-separated whitelist; unmatched `Origin` values do not receive an allow header.

## Frontend

```powershell
cd D:\+\1108026_rust_go\OpsPilot\web
Copy-Item .\.env.example .\.env
npm install
npm run dev
```

## Agent

```powershell
cd D:\+\1108026_rust_go\OpsPilot\server
$env:AGENT_API_BASE_URL='http://localhost:8080'
$env:AGENT_BOOTSTRAP_SECRET='dev-agent-bootstrap-secret'
$env:AGENT_WORK_DIR='.tmp/agent-work'
$env:AGENT_POLL_INTERVAL_SECONDS='5'
$env:AGENT_HEARTBEAT_INTERVAL_SECONDS='30'
$env:AGENT_MAX_CONCURRENT_TASKS='2'
go run .\cmd\agent
```

Enrollment token registration:

```powershell
$env:AGENT_ENROLLMENT_TOKEN='<token-from-UI-or-API>'
go run .\cmd\agent
```

## Tests

Backend:

```powershell
cd D:\+\1108026_rust_go\OpsPilot\server
go test ./...
go vet ./...
```

Frontend:

```powershell
cd D:\+\1108026_rust_go\OpsPilot\web
npm run build
```

Migration check:

```powershell
cd D:\+\1108026_rust_go\OpsPilot
.\scripts\migration-check.ps1 -SkipExecution
```

Version endpoint:

```text
GET /api/v1/version
```

For the full manual acceptance flow, see [E2E Demo](./docs/e2e-demo.md).

Release/security documents:
- [Security Permission Matrix](./docs/security-permission-matrix.en.md) / [安全权限矩阵](./docs/security-permission-matrix.md)
- [V0.7 Security Release Checklist](./docs/v0.7-security-release-checklist.en.md) / [V0.7 安全发布检查清单](./docs/v0.7-security-release-checklist.md)
- [V0.7 Release Notes](./docs/v0.7-release-notes.en.md) / [V0.7 发布说明](./docs/v0.7-release-notes.md)
- [V1.0 Product Development Design](./docs/version/v1.0/v1.0-product-development-design.en.md) / [V1.0 项目开发设计书](./docs/version/v1.0/v1.0-product-development-design.md)
- [V1.0 Release Evidence](./docs/v1.0-release-notes.en.md) / [V1.0 验收记录](./docs/v1.0-release-notes.md)
