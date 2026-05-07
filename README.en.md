# OpsPilot

> Language: [简体中文](./README.md) | English (current)

OpsPilot is an automation operations platform built with Go + Gin + React + TypeScript. The project now covers Agent onboarding, script templates, task execution, live logs, Cron scheduling, Webhook triggering, basic metrics, alerts, notifications, RBAC, and audit logs.

## Current Stage

- T1 Auth/RBAC: login, registration, JWT access/refresh token, users/roles/permissions, protected routes.
- T2 Agent/Host: Agent registration, token hashing, heartbeat, Agent/Host list, disable, token revoke, offline scan.
- T3 Task execution: script templates, task creation, target dispatch, Agent poll/claim, command execution, result reporting, status aggregation.
- T4 Realtime logs: Agent stdout/stderr/system log upload, redaction, chunk persistence, query APIs, and SSE streaming.
- T5 Cron scheduling: schedules API, background scanner, and Schedules page; due schedules reuse the current execution chain.
- T6 Webhook: source/rule management, tokenized trigger endpoint, HMAC-SHA256 signature validation, replay prevention, rate limiting, event persistence, and task run generation.
- T7 Metrics/Alerts/Notifications: Agent metrics upload, host_metrics persistence, metric threshold alerts, site notifications, webhook-like external delivery, and delivery records.

## Implemented

- Minimal Cron scheduling loop: schedules API, background scanner, and Schedules page; due schedules reuse the existing task_run / task_target / Agent poll / logs / result execution chain.
- Frontend permission matrix: routes declare `handle.meta.permission`, the route guard checks user permissions, and navigation/key actions are permission-aware.
- Webhook trigger loop: `POST /api/v1/webhooks/trigger/:token` requires `X-OpsPilot-Signature` or `X-Hub-Signature-256`; matched rules create task runs through the existing execution chain.
- Metrics / Alerts / Notifications loop: Agent metrics upload, host_metrics persistence, a Metrics page, metric threshold alert rules, alert events, site notifications, webhook/dingtalk/wechat/slack dispatching, and delivery records.

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
- Advanced webhook matcher expressions, replay-window tuning, and audit search.
- Email/SMTP notifications, notification templates, and retry management UI.

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
```

Frontend:

```powershell
cd D:\+\1108026_rust_go\OpsPilot\web
npm run build
```
