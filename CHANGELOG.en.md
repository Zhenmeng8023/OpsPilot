# Changelog

> Language: [简体中文](./CHANGELOG.md) | English (current)

## Unreleased

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
