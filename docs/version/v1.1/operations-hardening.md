# V1.1 Operations Hardening Runbook

## 1) Backup / Restore Drill

### Dry-run (verify command and output path)

```powershell
cd D:\+\1108026_rust_go\OpsPilot
.\scripts\db-backup.ps1 -Mode dry-run
```

### Backup

```powershell
cd D:\+\1108026_rust_go\OpsPilot
.\scripts\db-backup.ps1 -Mode backup -HostName 127.0.0.1 -Port 3306 -Database opspilot -User opspilot -Password opspilot
```

### Restore

```powershell
cd D:\+\1108026_rust_go\OpsPilot
.\scripts\db-backup.ps1 -Mode restore -BackupFile .\artifacts\backups\opspilot-YYYYMMDD-HHMMSS.sql -HostName 127.0.0.1 -Port 3306 -Database opspilot -User opspilot -Password opspilot
```

## 2) Migration Governance Check

```powershell
cd D:\+\1108026_rust_go\OpsPilot
.\scripts\migration-check.ps1 -SkipExecution
```

## 3) OpenAPI vs Router Check

```powershell
cd D:\+\1108026_rust_go\OpsPilot
.\scripts\openapi-router-check.ps1
```

## 4) Vulnerability Gate

### Go

```powershell
cd D:\+\1108026_rust_go\OpsPilot
.\scripts\govulncheck.ps1
```

Notes:
- Current allowlist only covers known Go stdlib advisories that require an upstream toolchain patch (`go1.26.3`) not yet available in this environment.
- Any unallowlisted advisory fails CI.

### npm

```powershell
cd D:\+\1108026_rust_go\OpsPilot\web
npm audit --audit-level=high --omit=dev --registry=https://registry.npmjs.org
```

## 5) Retention + Audit Evidence

- Metrics retention endpoint already supports `dryRun=true` and execution mode.
- Every retention execution should keep:
  - request payload,
  - response summary,
  - related `traceId`,
  - matching audit log record in `/api/v1/audit-logs`.
