# Changelog

> Language: [简体中文](./CHANGELOG.md) | English (current)

## Unreleased

### V0.7 Release Foundation

- Added GitHub Actions CI for backend tests, `go vet`, migration pair checks, frontend build, and Docker build.
- Added root `VERSION`, build metadata injection, and public `GET /api/v1/version`.
- Added API/Web Dockerfiles and extended the Compose `full` profile for migrate/API/Web/Agent startup.
- Added migration checker `scripts/migration-check.ps1`.
- Synced OpenAPI with current T5-T8 core routes and added `docs/e2e-demo.md` for manual acceptance.

### V0.7 Security Baseline

- Added `AUTH_PUBLIC_REGISTRATION_ENABLED`; public registration is disabled by default in production.
- Added production JWT secret validation to reject empty values, default development values, and short secrets.
- Updated CORS to use comma-separated origin whitelists and to omit allow headers for unmatched origins.

### T5 Cron Scheduling

- Added a schedules backend module with Cron schedule creation, paginated listing, pause, resume, and disable actions.
- Added a background schedule scanner controlled by `SCHEDULE_SCAN_INTERVAL_SECONDS`; due schedules create task runs through the existing `task_runs` / `task_run_targets` / Agent poll / logs / result execution chain.
- Added a Schedules page in the web app for creating Cron schedules from existing task definitions and viewing next/last fire times.
- Added a frontend route permission matrix: routes declare `handle.meta.permission`, the guard checks the current user's permissions, and navigation/actions are permission-aware.

### T6 Webhook Triggering

- Added Webhook source/rule management APIs and a Webhooks page.
- Added public trigger endpoint `POST /api/v1/webhooks/trigger/:token`; the source token identifies the sender.
- Trigger requests write `webhook_events` and `webhook_event_matches`; matched rules create `task_runs` through the existing execution chain.
- Added basic protections: `X-OpsPilot-Signature` / `X-Hub-Signature-256` HMAC-SHA256 validation, `X-Delivery-Id` replay prevention, per-source per-minute rate limiting, payload hashing, and request header persistence.

### T7 Metrics Monitoring

- Added Agent metrics upload endpoint `POST /api/v1/agent/metrics`, persisted into `host_metrics`.
- Agent now uploads running task count, logical CPU count, goroutine count, and runtime memory metrics on the heartbeat loop.
- Added management API `GET /api/v1/metrics/hosts` and a Metrics page for recent host/Agent metrics.
- Added alert rules / alerts APIs and a background scanner; metric threshold rules create firing alerts and automatically resolve when values recover.
- The Metrics page now includes alert rule creation, current firing alerts, and rule listing.

### T8 Notifications

- Added notification channel / notification management APIs and a Notifications page.
- First-time firing alerts now create `notifications` and `notification_deliveries`.
- Site notification channels mark deliveries as success immediately; webhook/dingtalk/wechat/slack channels are dispatched by a background sender from pending delivery rows.
- The web app supports notification listing, unread filtering, mark-as-read, channel listing, and webhook-like channel URL configuration.

### T1 Authentication and RBAC

- Added login, registration, refresh/logout, current user, users, roles, and permissions APIs.
- Added JWT middleware and permission checks for management APIs.
- Seeded admin/member roles and base permissions.

### T2 Agent and Host

- Added Agent registration, token hashing, heartbeat, Agent list, Host list, Agent disable, token revoke, and offline scan.
- Added Host status aggregation from Agent states.
- Added background offline scan in API process.

### T3 Scripts and Task Execution

- Added script template CRUD, disable, and revision tracking through `script_versions`.
- Added task creation from script template or inline command.
- Added task run/target creation, status machine, cancel, Agent poll/claim, result report, and aggregate status update.
- Upgraded `cmd/agent` into an executor with concurrency control, timeout, stdout/stderr capture, and result reporting.

### T4 Realtime Logs

- Added Agent log upload for stdout/stderr/system streams.
- Added chunked task log persistence with sequence ordering and source timestamps.
- Added management log query APIs and SSE streaming.
- Added frontend task log terminal with pause/copy/clear/target filtering.

### Security and Audit

- Added redaction for token/password/secret/authorization/bearer patterns.
- Added audit records for auth, scripts, tasks, agents, and role permission changes.
- Added production CORS guard and Agent registration toggle.

### Workflow and UX

- Added enrollment token management UI: create, one-time reveal, copy, list, and revoke.
- Added script approval UI: pending queue, approve/reject actions, and request-approval action in script editor.
- Synced OpenAPI for enrollment token and script approval endpoints.
- Synced RBAC seed data with `script:approve`.
- Added script approval execution gate: task creation now rejects scripts with `approvalRequired=true` when latest approval status is not `approved`.
- Added approval visibility in UI: script list now includes an approval status column, and task creation shows script approval status with blocking guidance for non-executable scripts.
- Added task execution safety gate: blocks high-risk commands such as `rm/del/format/shutdown/reboot/mkfs/Remove-Item` and caps task timeout at 3600 seconds.
- Added environment-configurable command policy: `TASK_COMMAND_ALLOW_PATTERNS` and `TASK_COMMAND_DENY_PATTERNS` can extend allowlist / denylist behavior with regular expressions.
- Updated Agent executor to use one work directory per task and redact stdout/stderr/system logs before upload.
- Completed running-task cancellation: backend now uses `canceling` state and exposes Agent target status, while Agent stops the local command and reports `canceled` when cancellation is requested.
