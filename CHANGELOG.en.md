# Changelog

> Language: [简体中文](./CHANGELOG.md) | English (current)

## Unreleased

### V1.0 Productization Validation Engineering

- Agent Fleet now supports Host Group batch Agent disable, group-scoped diagnostics, and group-scoped maintenance windows.
- Maintenance-window alert suppression and schedule skips now honor Host Group membership.
- Alert suppression/routing now supports Host Group matching, and Metrics rollup/retention workers are wired into background maintenance.
- README, CHANGELOG, V1.0 Release Evidence, project summary, OpenAPI, UML, and V1.0 design docs now reflect the current V1.0 acceptance status.
- Added `scripts/openapi-router-check.ps1`, which extracts routes from Go router/handler source and compares them with `server/docs/openapi/openapi.yaml`.
- Added `scripts/e2e-smoke.ps1` for health, version, login, and authenticated core-route smoke checks against a running environment.
- CI now includes OpenAPI/router diff and `docker compose --profile full config` validation.
- Added `docs/v1.0-release-notes.md` and the English version to record V1.0 validation commands, results, environment, and remaining blockers.
- Fixed README V1.0 design document links and added the V1.0 release evidence entry.
- Aligned Workflow permissions on `workflow:manage` while keeping `workflow:write` / `workflow:cancel` as compatibility aliases.
- Removed several hard-coded English labels from Audit, Workflow, Agent, and Webhook pages by adding zh/en i18n keys.

### V0.7 Release Closure And Documentation

- Added `docs/v0.7-release-notes.md` and `docs/v0.7-release-notes.en.md` for V0.7 positioning, completed capabilities, validation commands, known gaps, and follow-up order.
- Standardized newly added explanatory docs into Chinese default files plus `.en.md` English files: `security-permission-matrix`, `v0.7-security-release-checklist`, and `v0.7-release-notes`.
- Added Release/security document links to README and README.en.
- Preserved the existing changelog section structure; new changes are appended as new sections without compressing historical entries.

### V0.7 Webhook Security And Debugging

- Split Webhook trigger token and signing secret: URL tokens only locate the source, while HMAC validation uses an independent signing secret.
- Trigger requests now support timestamp, nonce, and delivery-id replay protection; failed attempts are persisted with failure reasons.
- Matchers support AND conditions with `header_equals`, `payload_equals`, `payload_contains`, `event_type_equals`, `ref_equals`, and `branch_equals`.
- Webhook event queries now support `receivedFrom` / `receivedTo` time range filters.
- Sources and rules support pause, resume, and disable; rules can be edited.
- The Webhooks UI now includes event details, headers, payload, matcher results, failure reasons, and task run debugging.

### V0.7 Metrics / Alerts Production Improvements

- Agent now collects real OS metrics for CPU, memory, disk, and network sent/received bytes.
- Agent uploads metrics immediately after the first heartbeat to reduce empty first-screen states for new Hosts.
- Added metrics trend API, Host overview, trend charts, and quick metric filters.
- Alert state machine now supports ack, silence, unsilence, and resolve.
- Alert rules support update, pause, resume, disable, rule templates, cooldown, and recovery notifications.
- Alert history, alert events, and status/severity/rule/host filters are wired through backend and frontend.
- `durationSeconds` now evaluates recent metric windows for sustained threshold violations instead of a single latest point.

### V0.7 Notification Delivery Improvements

- Added SMTP Email sender with configuration injected through `.env` / environment variables; example files document variable names only.
- Added notification delivery listing, failed delivery retry, and channel test send.
- Notification channel lists return only `targetSummary` and do not expose plaintext secret/password values.
- Outbound webhook deliveries include `X-OpsPilot-*` signing headers.
- Fixed delivery attempts being incremented twice in some paths.

### V0.7 Audit And Redaction Improvements

- Added `GET /api/v1/audit-logs` with filters for action, actor type, result, resource type, trace id, keyword, time range, and pagination.
- Added an Audit Logs frontend page wired to the `audit.read` route permission and bilingual UI copy.
- Enhanced `security.Redact` to cover Bearer tokens, `token/password/secret/authorization` fields, and camelCase secret fields.
- `audit.Write` now redacts serialized `before`, `after`, and `metadata` before persistence.
- Added redaction and audit-write tests to prevent SMTP passwords, signing secrets, access tokens, and similar values from leaking into audit payloads.

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
