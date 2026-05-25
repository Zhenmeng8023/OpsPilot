# OpsPilot V1.1 Release Evidence

Generated: 2026-05-25

## Scope

V1.1 release scope is now aligned to the three P0 epics:

- OP-11-TRACE: Trace Center with `/api/v1/traces/:id`, `/api/v1/traces/:id/events`, `/api/v1/traces/lookup`, rebuildable `trace_events` cache, and `/traces` as the primary UI route.
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
- `POST /api/v1/traces/retention/run`
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
go vet ./...

cd E:\Code\1108026_rust_go\OpsPilot\web
npm run build
```

UI smoke command:

```powershell
cd E:\Code\1108026_rust_go\OpsPilot\web
$env:SMOKE_BASE_URL='http://127.0.0.1:5176'
npm run test:ui-smoke
```

Trace Center E2E command:

```powershell
cd E:\Code\1108026_rust_go\OpsPilot\web
$env:SMOKE_BASE_URL='http://127.0.0.1:5176'
$env:SMOKE_API_BASE_URL='http://127.0.0.1:18890'
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

Latest generated runtime logs used for this round:

```text
artifacts/api-18890.out.log
artifacts/web-5176.out.log
```

## Implementation Alignment

- `trace_events` is now enabled as a rebuildable cache populated from workflow/task/webhook/notification/audit facts during trace lookups.
- `trace_events` retention is now documented with `TRACE_RETENTION_DAYS=30`, and `POST /api/v1/traces/retention/run` provides dry-run plus targeted cleanup by `traceId`.
- `workflow_run_actions` is now persisted as a dedicated action history table for retry/cancel/approval flows, while legacy runs still fall back to audit/event reconstruction.
- `/traces` is now the primary UI route. `/trace-center` remains as a compatibility redirect.

## UI Smoke Evidence

Latest Playwright smoke (`artifacts/ui-smoke/latest/summary.json`) on 2026-05-25:

- 2/2 projects passed (`desktop`, `narrow`), 0 unexpected, total duration `9199.56ms`.
- Desktop screenshot: `artifacts/ui-smoke/latest/test-results/smoke-login-and-open-production-critical-pages-desktop/test-finished-1.png`
- Narrow screenshot: `artifacts/ui-smoke/latest/test-results/smoke-login-and-open-production-critical-pages-narrow/test-finished-1.png`
- HTML report: `artifacts/ui-smoke/latest/html-report/index.html`

## Trace Center E2E Evidence

Latest local run (`artifacts/ui-smoke/latest/trace-center-e2e/summary.json`):

- Created, published, and manually ran workflow `6TQ7JZ2SCB364VHY43HC2ZS6Q9`; success run `BPCRGSR8MED89CHN70EZ4KSF2N` completed with `status=success`.
- Trace Center lookup resolved `traceId=trc_dd3a13a33b94d23383b39b77` and returned 6 timeline items.
- Timeline categories: audit, workflow, workflow-event, workflow-node.
- `GET /api/v1/traces/trc_dd3a13a33b94d23383b39b77/events?page=1&pageSize=2` returned 2 items from a 6-item cached timeline, verifying `/traces/:id/events`.
- `POST /api/v1/traces/retention/run` with `{"dryRun":true,"traceId":"trc_dd3a13a33b94d23383b39b77"}` matched 6 cached rows and deleted 0, proving `trace_events` was materially populated and preview-cleanable.
- Failure probe run `D3YPXK574T317EH0T8KG79FTQA` reached `status=failed`; retry created run `P3AD98CBFFNV89HA1DNA2MY321`.
- `GET /api/v1/workflow-runs/D3YPXK574T317EH0T8KG79FTQA/action-history` returned `retry_requested`; `GET /api/v1/workflow-runs/P3AD98CBFFNV89HA1DNA2MY321/action-history` returned `retry_started`, providing live verification that `workflow_run_actions` persists action history for both source and retry runs.
- 20 repeated Trace Center lookups reported p95 `13.53ms` against a `1000ms` threshold.
- Browser opened `/traces`, searched the generated workflow run id, rendered results, and produced no actionable console errors or failed requests.
- Trace Center screenshot: `artifacts/ui-smoke/latest/trace-center-e2e/trace-center-workflow-run.png`

## P1 Status

- Agent Fleet: completed for version distribution alias, stale reasons, diagnostics diff, host group diagnostics, and maintenance window preview foundations.
- Alert governance: completed for noise report UI and per-alert routing explanation entry point.
- Incident governance: completed for timeline, merge/close lifecycle, and alert-group linking through existing alert associations.
- Security hardening: completed for Security Review UI, permission diff preview, secret rotation audit history, and audit export hash headers.
