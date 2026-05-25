# OpsPilot V1.1 Release Evidence

Generated: 2026-05-25

## Scope

V1.1 release scope is now aligned to the three P0 epics:

- OP-11-TRACE: Trace Center with `/api/v1/traces/:id`, `/api/v1/traces/:id/events`, `/api/v1/traces/lookup`, and the existing `/api/v1/trace-center` compatibility endpoint.
- OP-11-WF-RELIABILITY: retry plan, retry, cancel, cancel propagation report, action history, definition diff, and node retry.
- OP-11-UI-SMOKE: Playwright desktop/narrow smoke with JSON summary and screenshot/trace artifact paths under `artifacts/ui-smoke/latest`.

P1 items included in this evidence:

- Agent Fleet aliases and diagnostics diff.
- Alert routing explanation and noise report.
- Incident timeline and alert-group association.
- Security Review page and APIs for permission diff, secret rotation history, and audit export hash headers.

## API Evidence

P0 API routes added or aligned:

- `GET /api/v1/traces/:id`
- `GET /api/v1/traces/:id/events`
- `GET /api/v1/traces/lookup`
- `GET /api/v1/workflow-runs/:id/action-history`
- `GET /api/v1/workflow-runs/:id/cancel-report`

P1 API routes added or aligned:

- `GET /api/v1/agents/version-distribution`
- `GET /api/v1/agents/:id/diagnostics/diff`
- `GET /api/v1/alerts/:id/routing-explanation`
- `GET /api/v1/alerts/noise-report`
- `GET /api/v1/incidents/:id/timeline`
- `POST /api/v1/incidents/:id/alert-groups`
- `GET /api/v1/security/permissions/diff`
- `GET /api/v1/security/secrets/:id/rotation-history`

## Validation Commands

Latest local validation:

```powershell
cd E:\Code\1108026_rust_go\OpsPilot\server
go test ./...

cd E:\Code\1108026_rust_go\OpsPilot\web
npm run build

cd E:\Code\1108026_rust_go\OpsPilot
.\scripts\migration-check.ps1 -SkipExecution
.\scripts\openapi-router-check.ps1
```

UI smoke command:

```powershell
cd E:\Code\1108026_rust_go\OpsPilot\web
npm run test:ui-smoke
```

Trace Center E2E command:

```powershell
cd E:\Code\1108026_rust_go\OpsPilot\web
$env:SMOKE_BASE_URL='http://127.0.0.1:5174'
$env:SMOKE_API_BASE_URL='http://127.0.0.1:18889'
npm run test:trace-e2e
```

Manual browser evidence:

```text
artifacts/ui-smoke/latest/manual-browser-verify/final/
artifacts/ui-smoke/latest/manual-browser-verify/p1-ui-final/
artifacts/ui-smoke/latest/trace-center-e2e/
```

Expected artifact locations:

```text
artifacts/ui-smoke/latest/
  summary.json
  html-report/
  test-results/
```

## Known Deviations

- `trace_events` remains optional. The implementation uses on-demand aggregation and should only introduce `trace_events` if p95 query latency exceeds the V1.1 target.
- `workflow_run_actions` is not introduced as a separate table. Action history is derived from `audit_logs` and `workflow_run_events`, which keeps V1.0 data compatibility while preserving traceability.
- `GET /api/v1/trace-center` remains as a compatibility endpoint for the existing UI; `/api/v1/traces/*` is now the plan-aligned API surface.

## Trace Center E2E Evidence

Latest local run:

- Created, published, and manually ran a generated workflow probe.
- Workflow run completed with `status=success`.
- Trace Center lookup by `workflowRunId` returned 6 timeline items.
- Timeline categories: audit, workflow, workflow-event, workflow-node.
- 20 repeated Trace Center lookups reported p95 `5.85ms` against a `1000ms` threshold.
- Browser opened `/trace-center`, searched the generated workflow run id, rendered results, and produced no actionable console errors or failed requests.

## P1 Status

- Agent Fleet: completed for version distribution alias, stale reasons, diagnostics diff, host group diagnostics, and maintenance window preview foundations.
- Alert governance: completed for noise report UI and per-alert routing explanation entry point.
- Incident governance: completed for timeline, merge/close lifecycle, and alert-group linking through existing alert associations.
- Security hardening: completed for Security Review UI, permission diff preview, secret rotation audit history, and audit export hash headers.
