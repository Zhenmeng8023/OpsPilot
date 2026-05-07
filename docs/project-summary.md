# OpsPilot Project Summary

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
