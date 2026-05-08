# OpsPilot Project Summary

## Version Update Log

| Date | Version / Commit | Development Change | Implementation Scope | Documentation / Validation |
| --- | --- | --- | --- | --- |
| 2026-05-08 | V1.0 in progress / pending | Added Alert host-group matching for suppression and routing. | Added `host_group_uid` migration, `hostGroupId` API payload/summary fields, runtime membership matching, Metrics page host-group selectors, and OpenAPI schema updates. | Synced V1.0 release evidence, version design docs, and UML update notes; validated Go test/vet, web build, migration pair check, OpenAPI/router check. |
| 2026-05-08 | V1.0 in progress / pending | Added background Metrics rollup and retention maintenance workers. | `server/internal/modules/metrics/scanner.go`, API startup wiring, metric interval config, `.env.example`, Compose, README. | Synced V1.0 release evidence, UML update notes, README config examples; validated `go test ./...`, `go vet ./...`, migration pair check, OpenAPI/router check. |
| 2026-05-08 | `3870522` | Integrated maintenance windows into runtime behavior. | Alerts suppress notification delivery while recording firing/suppression events; schedules write skipped triggers and advance next fire during maintenance. | Synced V1.0 release evidence and UML flow notes; validated backend tests, vet, migration check, OpenAPI/router check. |
| 2026-05-08 | `412fd6d` | Completed Agent/Host tags and Host Group baseline. | Added fleet tag APIs, resource tag assignment, host group/member APIs, Agent page UI, OpenAPI routes, shared tag styles. | Synced V1.0 release evidence and UML; OpenAPI/router documented 145 routes; validated backend/frontend builds. |
| 2026-05-08 | `b443390` | Completed Agent Fleet operations baseline. | Added agent diagnostics snapshots, version inventory display, maintenance window storage/API/UI, heartbeat diagnostic writes. | Added migration pair `000013`, OpenAPI, release evidence, UML ER updates; validated migration check, Go tests/vet, web build. |
| 2026-05-08 | `ebe1dec` | Added Alert suppression and routing operations. | Added suppression rules and routing policies with API/UI and alert firing integration. | Synced release evidence and OpenAPI; validated backend/frontend checks. |
| 2026-05-08 | `ff792ad` | Added Metrics lifecycle baseline. | Added 5m/1h rollups, raw/rollup retention dry-run/execution, saved dashboards, trend granularity selection. | Synced V1.0 release evidence and UML metric lifecycle notes; validated Go tests and frontend build. |
| 2026-05-08 | `82de7fb` / `e1cd3d9` | Added Notification template and bulk retry operations. | Added template rendering/channel-specific templates and failed delivery batch retry. | Synced release evidence, OpenAPI, and UI/API docs; validated backend/frontend checks. |
| 2026-05-08 | `422b5e5` | Added Workflow node retry, timeout, and failure policy baseline. | Workflow nodes support retry/timeout/failure handling and retry-from-failed behavior. | Synced V1.0 release evidence and UML workflow notes; validated Go tests and web build. |
| 2026-05-08 | `3a05248` | Landed the first broad V1.0 productization slice. | Workflow, Webhook, Audit, Secret governance, OpenAPI/router validation, E2E smoke script, release notes, and UML updates. | Established V1.0 release evidence and architecture update documents; validated OpenAPI/router and CI-oriented checks. |

## Project Introduction

OpsPilot is an automation operations platform for managing Agents, Hosts, script templates, task execution, and realtime execution logs. It provides a control plane for administrators to create scripts or inline commands, dispatch tasks to registered Agents, observe execution status, and review audit trails.

## Tech Stack

- Backend: Go, Gin, GORM, MySQL, Redis, JWT.
- Agent: Go command-line executor with HTTP polling and cross-platform shell execution.
- Frontend: React, TypeScript, Vite, TanStack Query, Zustand, CSS modules through shared global styles.
- Infrastructure: Docker Compose, MySQL 8.0.39, Redis 7.

## Core Features

- Authentication and RBAC with users, roles, permissions, access tokens, and refresh tokens.
- Agent registration, Agent Token validation, heartbeat, disable, revoke, and offline scan.
- Host inventory with status aggregation from all Agents under a Host.
- Script templates with revisions and disable support.
- Task creation from scripts or inline commands with required timeout and selected targets.
- Agent polling, claim, command execution, log upload, and result reporting.
- Task detail page with target status table and realtime log terminal.
- Audit records for sensitive user and Agent operations.

## Technical Highlights

- Separates user JWT auth from Agent Token auth to avoid privilege mixing.
- Uses DTO/request/response structs instead of exposing database models directly.
- Maintains a strict task state machine and rejects illegal transitions.
- Persists logs as ordered chunks rather than storing an unbounded full log blob.
- Applies basic redaction before task logs enter storage.
- Uses SSE for realtime logs while preserving Authorization headers through a fetch-based stream wrapper.

## Difficulties

- Keeping task run status, per-target status, and Agent concurrency consistent under repeated polling.
- Avoiding duplicate execution with Agent-side in-memory guards and backend claim transitions.
- Supporting both Windows and Linux shell execution while enforcing task timeouts.
- Preserving existing Agent/Host management semantics while adding background offline scanning and Host status aggregation.

## Personal Work Description

- Designed and implemented script/template and task execution modules on top of the existing auth and Agent foundation.
- Built Agent private APIs and upgraded the Agent binary into a practical executor.
- Added realtime task log upload, storage, query, and streaming paths.
- Implemented frontend task/script management pages and live log viewer.
- Added security boundaries, log redaction, audit records, migrations, OpenAPI updates, README, changelog, and tests.

## Follow-Up Plan

- Build UI and APIs for one-time Agent enrollment token management.
- Add script approval workflow for high-risk operations.
- Add scheduled tasks and webhook triggers.
- Add metrics, alerts, and richer operational dashboards.
- Add integration tests against a real MySQL instance in CI.
