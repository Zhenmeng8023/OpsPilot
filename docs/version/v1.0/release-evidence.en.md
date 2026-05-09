# OpsPilot V1.0 Release Evidence

> Language: [简体中文](./release-evidence.md) | English (current)

This document records V1.0 productization validation commands, results, environment, and risks. The current state is the final `v1.0.0` release validation record.

## V1.0 Update Log

| Date | Version / Commit | Update | Sync Notes |
| --- | --- | --- | --- |
| 2026-05-09 | `v1.0.0` release closure | The root version marker is now `1.0.0`, and remote CI has passed. | GitHub Actions is green on both `master` and the `v1.0.0-beta.1` tag; remaining V1.0 risks are now tracked as V1.1 hardening work. |
| 2026-05-09 | V1.0 in progress / pending | Cleared the remaining GitHub Actions runtime deprecation risk. | CI now uses `actions/checkout@v5`, `actions/setup-node@v6`, and `actions/setup-go@v6`, moving the workflow onto the native Node 24 action runtime. |
| 2026-05-09 | V1.0 in progress / pending | Closed the remaining strict V1.0 matcher and documentation gaps. | Added `payload_exists`, `payload_not_equals`, `payload_regex`, and JSONPath array wildcard matcher support across backend/frontend/OpenAPI/E2E smoke, then added the V1.0 deployment guide, acceptance guide, and UI guidelines. |
| 2026-05-09 | V1.0 in progress / pending | Fixed the CI schedule smoke timezone mismatch. | The E2E smoke script now uses a timezone-neutral every-minute cron for the schedule workflow check and disables the smoke schedule after the first fired trigger, avoiding false failures when the GitHub runner timezone differs from `Asia/Shanghai`. |
| 2026-05-09 | V1.0 in progress / pending | Docker/Compose release validation and full-stack smoke are now closed locally. | Removed Dockerfile frontend fetch dependency, parameterized published ports and dropped fixed container names in Compose, aligned API/Agent timezone handling for wait/schedule correctness, expanded `scripts/e2e-smoke.ps1`, and added a CI full-stack E2E job that also verifies encrypted notification config at rest. |
| 2026-05-09 | V1.0 in progress / pending | Squashed the active V1.0 SQL migration chain. | Replaced root `000005`-`000015` with `000005_v10_productization_bundle`, archived the original incremental files, validated MySQL apply/rollback and schema equivalence against the legacy chain, and aligned `VERSION` to `1.0.0`. |
| 2026-05-09 | V1.0 in progress / pending | Another frontend i18n closure pass. | Removed remaining hard-coded copy from Workflow/Audit/Agent/Webhook pages, localized default samples, export-failure copy, actor type/result rendering, and UI action reasons, then synced README, CHANGELOG, global summary, and the V1.0 design docs; validated frontend build and `git diff --check`. |
| 2026-05-08 | V1.0 in progress / pending | Alert grouping operations view. | Synced README, CHANGELOG, OpenAPI, UML, global summary, and V1.0 design docs; validated Go test/vet, frontend build, migration pair check, OpenAPI/router check, and git diff check. |
| 2026-05-08 | V1.0 in progress / pending | Agent Host Group batch operations and group-scoped maintenance windows. | Synced README, CHANGELOG, OpenAPI, UML, global summary, and V1.0 design docs; validated Go test/vet, frontend build, migration pair check, and OpenAPI/router check. |
| 2026-05-08 | V1.0 in progress / pending | Alert suppression/routing now supports Host Group matching. | Synced OpenAPI, UML, global summary, and V1.0 design docs; validated Go test/vet, frontend build, migration pair check, and OpenAPI/router check. |
| 2026-05-08 | V1.0 in progress / pending | Metrics rollup/retention background workers with configurable intervals. | Synced README, `.env.example`, Compose, UML, global summary, and V1.0 design doc; validated Go test/vet, migration pair check, and OpenAPI/router check. |
| 2026-05-08 | `3870522` | Maintenance windows integrated with alert notification suppression and schedule firing skips. | Synced UML and this Release Evidence; validated Go test/vet, migration pair check, and OpenAPI/router check. |
| 2026-05-08 | `412fd6d` | Agent/Host tags and Host Group API/UI. | Synced OpenAPI, UML, and this Release Evidence; validated Go test/vet, frontend build, and OpenAPI/router check. |
| 2026-05-08 | `b443390` | Agent diagnostics, version inventory, and maintenance windows API/UI. | Synced migrations, OpenAPI, UML, and this Release Evidence; validated migration pair check, Go test/vet, and frontend build. |
| 2026-05-08 | `ebe1dec` | Alert suppression rules and routing policies. | Synced OpenAPI, UML, and this Release Evidence; backend/frontend checks passed. |
| 2026-05-08 | `ff792ad` | Metrics rollup, retention, saved dashboards, and trend granularity. | Synced OpenAPI, UML, and this Release Evidence; backend/frontend checks passed. |
| 2026-05-08 | `82de7fb`, `e1cd3d9` | Notification templates, channel-specific templates, and bulk retry. | Synced OpenAPI and this Release Evidence; backend/frontend checks passed. |
| 2026-05-08 | `422b5e5` | Workflow node retry, timeout, and failure policy. | Synced UML and this Release Evidence; backend/frontend checks passed. |
| 2026-05-08 | `3a05248` | V1.0 Workflow, Webhook, Audit, Secret governance, and release validation baseline. | Established OpenAPI/router diff, E2E smoke, Release Evidence, and UML updates. |

## 2026-05-09 SQL Migration Squash Validation

Commands executed:

```powershell
cd D:\+\1108026_rust_go\OpsPilot
powershell -File .\scripts\migration-check.ps1 `
  -HostName 127.0.0.1 -Port 3306 `
  -Database opspilot_migcheck_squash_1778261001 `
  -User root -Password 123456 `
  -MysqlExe "C:\Program Files\MySQL\MySQL Server 8.0\bin\mysql.exe"
```

Results:

| Area | Result |
| --- | --- |
| Active migration pair check | Passed, 5 root up/down migration pairs. |
| Squashed V1.0 apply | Passed, `000005_v10_productization_bundle.up.sql` applied after `000001`-`000004`. |
| Squashed V1.0 rollback | Passed, `000005_v10_productization_bundle.down.sql` restored the schema to the pre-V1.0 state. |
| Legacy-chain equivalence | Passed, normalized schema dumps and table row-count signatures match the archived `000005`-`000015` chain. |

Notes:

- The active root migration path is now optimized for new environment bootstrap.
- Historical incremental V1.0 migrations remain available in `server/migrations/archive/v1.0-incremental/`.

## 2026-05-09 Release Validation Snapshot

Environment:

- OS: Windows / PowerShell
- Node.js: local Node 24 toolchain
- Docker: available locally
- Workspace: `D:\+\1108026_rust_go\OpsPilot`

Commands executed:

```powershell
cd D:\+\1108026_rust_go\OpsPilot\server
go test ./...
go vet ./...

cd D:\+\1108026_rust_go\OpsPilot\web
npm install --no-audit --no-fund
npm run build

cd D:\+\1108026_rust_go\OpsPilot
powershell -File .\scripts\migration-check.ps1 -SkipExecution
powershell -File .\scripts\openapi-router-check.ps1
docker compose -f deploy/docker-compose.yml --profile full config
docker build -f server/Dockerfile --build-arg APP_VERSION="$(Get-Content VERSION)" .
docker build -f web/Dockerfile .

$env:MYSQL_PUBLISHED_PORT="33306"
$env:REDIS_PUBLISHED_PORT="36379"
$env:API_PUBLISHED_PORT="18080"
$env:WEB_PUBLISHED_PORT="15173"
$env:SCHEDULE_SCAN_INTERVAL_SECONDS="2"
$env:ALERT_SCAN_INTERVAL_SECONDS="2"
$env:NOTIFICATION_DISPATCH_INTERVAL_SECONDS="2"
$env:METRIC_ROLLUP_INTERVAL_SECONDS="3600"
$env:METRIC_RETENTION_INTERVAL_SECONDS="3600"
docker compose -p opspilot-local-e2e -f deploy/docker-compose.yml --profile full up --build -d
powershell -File .\scripts\e2e-smoke.ps1 -ApiBaseUrl http://127.0.0.1:18080 -ScheduleTimeoutSeconds 180
docker compose -p opspilot-local-e2e -f deploy/docker-compose.yml exec -T mysql mysql -uopspilot -popspilot opspilot -Nse "SELECT config FROM notification_channels WHERE config IS NOT NULL ORDER BY id DESC LIMIT 1;"
```

Results:

| Area | Result |
| --- | --- |
| Backend tests | Passed, `go test ./...` is green. |
| Backend vet | Passed, `go vet ./...` produced no output. |
| Frontend build | Passed, `tsc --noEmit && vite build` succeeded. |
| Migration pair check | Passed, 5 active root up/down migration pairs. |
| OpenAPI/router diff | Passed, 148 router routes are documented in OpenAPI. |
| Docker compose config | Passed. |
| Docker image build | Passed for both `server/Dockerfile` and `web/Dockerfile`. |
| Compose full-profile smoke | Passed with MySQL, Redis, migrate, API, Web, and Agent running together. |
| E2E smoke | Passed for manual/schedule/webhook workflow triggers, run cancel, whole-run retry, node retry, matcher simulator, webhook replay, notification masking, and audit export/retention. |
| Secret at rest | Passed; the latest `notification_channels.config` row contains encrypted JSON (`"_encrypted"`) and does not contain the plaintext smoke secret. |

Notes:

- Removed the Dockerfile syntax directive from API/Web images so Docker builds no longer depend on fetching `docker/dockerfile:1` at build time.
- Compose now parameterizes published ports, removes fixed `container_name` values, and sets `TZ` for API/Agent so wait/schedule execution stays correct when the database and application timezone would otherwise diverge.
- `.github/workflows/ci.yml` now boots the full Compose profile, runs `scripts/e2e-smoke.ps1`, verifies encrypted notification config at rest, and always tears the stack down.

## Currently Available V1.0 Capabilities

- Workflow create, update, publish, disable, copy, version listing, manual runs with input, cancel, whole-run retry, node-level retry, approval node operations, node `timeoutSeconds`, and `stop_workflow` / `skip_downstream` / `continue` failure policies.
- Workflow permissions are aligned to `workflow:read`, `workflow:manage`, and `workflow:execute`; legacy `workflow:write` / `workflow:cancel` remain compatibility aliases.
- Schedule/Webhook triggered Workflow runs.
- Basic Workflow task, condition, approval, notification, wait, and webhook-call node execution.
- Notification/Webhook secret encryption and rotation basics; Notification template and channel-specific rendering, plus single and bulk failed-delivery retry.
- Audit export, retention run, and part of advanced search filtering.
- Webhook matcher simulator and event replay.
- Incident projection model, list, detail, and timeline.
- Alert grouping operations view with aggregation by rule, host group, severity, and fingerprint.
- Alert suppression rules and routing policies now have API/UI coverage and are applied during alert firing by rule, host, host group, and severity.
- Metrics 5m/1h rollups, raw/rollup retention dry-run and execution endpoints, background rollup/retention workers, saved dashboard API/UI, and trend `auto/raw/5m/1h` granularity selection.
- Agent diagnostic snapshots, version inventory, tags/groups, and maintenance windows now have API/UI coverage. Host Groups support batch Agent disable and group diagnostics. Heartbeats write the latest diagnostic snapshot; maintenance windows support all/agent/host/group scope and suppress alert notifications and skip schedule firing during maintenance.
- Shared frontend DataTable, FilterToolbar, Confirm, Toast, Timeline, and JsonViewer components.
- CI now covers migration pairs, OpenAPI/router diff, Go test/vet, npm build, Docker build, compose config, and full-stack Compose smoke with an encrypted-secret-at-rest assertion.

## V1.0.0 Release Closure Status

- Remote GitHub Actions has passed on both `master` and the `v1.0.0-beta.1` tag, so the V1.0 CI release gate is satisfied.
- The root version marker has moved from `1.0.0-dev` to `1.0.0`.
- The original V1.0 incremental migration chain remains archived under `server/migrations/archive/v1.0-incremental/`; the active root migration path uses `000005_v10_productization_bundle`.
- The final release report can still retain desktop and narrow-screen screenshot evidence for Dashboard, Workflow, Notifications, and Audit. This is release evidence hardening, not a V1.0 blocker.
- Workflow retry/timeout, metrics rollup/retention, and alert grouping paths now have working baseline coverage; deeper database integration coverage moves into the V1.1 hardening plan.
