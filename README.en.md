# OpsPilot

> Language: [简体中文](./README.md) | English (current)

OpsPilot is an automation operations platform built with Go + Gin + React + TypeScript. The project currently covers the T3/T4 execution chain: script templates, task execution, Agent polling/execution, task logs, SSE live logs, RBAC, and audit logs.

## Current Stage

- T1 Auth/RBAC: login, registration, JWT access/refresh token, users/roles/permissions, protected routes.
- T2 Agent/Host: Agent registration, token hashing, heartbeat, Agent/Host list, disable, token revoke, offline scan.
- T3 Task execution: script templates, task creation, target dispatch, Agent poll/claim, command execution, result reporting, status aggregation.
- T4 Realtime logs: Agent stdout/stderr/system log upload, redaction, chunk persistence, query APIs, and SSE streaming.

## Implemented

- Unified response envelope: `code`, `message`, `data`, `traceId`.
- Backend layered architecture: handler/service/repository.
- RBAC for scripts, tasks, logs, agents, and hosts.
- Agent Token auth isolated from user JWT auth.
- Script revisioning via `script_versions.version_no`.
- Task state machine: `pending -> queued -> running -> success|failed|timeout`, and `pending|queued -> canceled`.
- Agent executor: concurrency control, timeout handling, log upload retry, result reporting.
- Frontend pages for scripts, script approvals, tasks, task logs, agents/hosts, and enrollment tokens.
- Audit events for auth, scripts, tasks, agents, and role permission updates.

## Not Yet Implemented

- Visual workflow orchestration and cron scheduling UI.
- Policy-enforced approval gate (for example, high-risk scripts must be approved before execution).
- Webhook trigger execution path.
- Metrics and alerts pages beyond schema placeholders.

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
