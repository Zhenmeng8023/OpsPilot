# OpsPilot V0.7 Security Release Checklist

> Language: [简体中文](./security-release-checklist.md) | English (current)

This checklist reflects the current implementation status for release review.

## Security Configuration

| Item | Status | Evidence |
| --- | --- | --- |
| Public registration disabled by default in production | Done | `APP_ENV=prod` requires explicit `AUTH_PUBLIC_REGISTRATION_ENABLED=true`. |
| Production JWT secret validation | Done | Empty, default, and short secrets are rejected in production. |
| CORS allowlist | Done | Origins are matched against configured allowlist entries. |
| Agent token boundary | Done | Agent opaque tokens are only accepted by Agent endpoints. |
| User JWT boundary | Done | User JWTs cannot access Agent private execution endpoints. |
| Webhook token boundary | Done | Trigger token only locates source; HMAC uses independent signing secret. |

## Sensitive Data Handling

| Surface | Expected behavior |
| --- | --- |
| Logs | `security.Redact` masks bearer tokens, token-like keys, password-like keys, secret-like keys, and authorization values. |
| Audit logs | `audit.Write` redacts serialized `before`, `after`, and `metadata` payloads before insert. |
| Notification channel list | Channel config is not returned; UI receives `targetSummary`. |
| Webhook source creation | Trigger token and signing secret are one-time creation outputs. Lists do not return secrets. |
| Agent enrollment token | Full token is only returned on creation. Lists return token prefix only. |
| SMTP config | Local `.env` is ignored by git; examples document env names without real credentials. |

## Audit Coverage

| Domain | Audited actions |
| --- | --- |
| Auth | Login and logout. |
| Users/Roles | Role permission updates. |
| Agents | Registration, disable, token revoke, enrollment token create/revoke. |
| Scripts | Create, update, disable, approval request, approve/reject. |
| Tasks | Create/cancel task runs. |
| Schedules | Create and status changes. |
| Webhooks | Source create/status change, rule create/update/status change. |
| Alerts | Rule create/update/status change, ack, silence, unsilence, resolve. |
| Notifications | Channel create, test send, delivery retry. |
| Audit Logs | Audit log search and detail views are protected by `audit.read`. |

## Release Validation Commands

Run these before tagging V0.7:

```powershell
cd server
go test ./...
go vet ./...
cd ..\web
npm run build
cd ..
powershell -File .\scripts\migration-check.ps1 -SkipExecution
```

When Docker is available, also run:

```powershell
docker compose -f deploy/docker-compose.yml --profile full config
docker build -f server/Dockerfile .
docker build -f web/Dockerfile .
```

## Remaining Risks

| Risk | Follow-up |
| --- | --- |
| Docker validation was not available on the local Windows environment earlier. | Rely on CI and rerun locally where Docker CLI is installed. |
| Notification channel config remains stored as JSON. | Keep API masked, audit redacted, and consider encrypted-at-rest config in a later version. |
