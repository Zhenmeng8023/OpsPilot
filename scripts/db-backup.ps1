param(
  [ValidateSet("backup", "restore", "dry-run")]
  [string]$Mode = "dry-run",
  [string]$HostName = "127.0.0.1",
  [int]$Port = 3306,
  [string]$Database = "opspilot",
  [string]$User = "opspilot",
  [string]$Password = "opspilot",
  [string]$OutputDir = "",
  [string]$BackupFile = "",
  [string]$MysqlExe = "mysql",
  [string]$MysqldumpExe = "mysqldump"
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
if ([string]::IsNullOrWhiteSpace($OutputDir)) {
  $OutputDir = Join-Path $repoRoot "artifacts/backups"
}

if (-not (Test-Path -LiteralPath $OutputDir)) {
  New-Item -ItemType Directory -Path $OutputDir | Out-Null
}

if ([string]::IsNullOrWhiteSpace($BackupFile)) {
  $stamp = Get-Date -Format "yyyyMMdd-HHmmss"
  $BackupFile = Join-Path $OutputDir "$Database-$stamp.sql"
}

$mysql = Get-Command $MysqlExe -ErrorAction Stop
$dump = Get-Command $MysqldumpExe -ErrorAction Stop

$baseArgs = @(
  "--host=$HostName",
  "--port=$Port",
  "--user=$User",
  "--password=$Password",
  "--default-character-set=utf8mb4"
)

if ($Mode -eq "dry-run") {
  Write-Host "Dry-run only."
  Write-Host "Backup target: $BackupFile"
  Write-Host "Backup command: $($dump.Source) $($baseArgs -join ' ') --single-transaction --routines --triggers --events --set-gtid-purged=OFF $Database > $BackupFile"
  Write-Host "Restore command: $($mysql.Source) $($baseArgs -join ' ') $Database < <backup-file>"
  exit 0
}

if ($Mode -eq "backup") {
  Write-Host "Backing up $Database to $BackupFile ..."
  & $dump.Source @baseArgs "--single-transaction" "--routines" "--triggers" "--events" "--set-gtid-purged=OFF" $Database | Out-File -Encoding utf8 -FilePath $BackupFile
  if ($LASTEXITCODE -ne 0) {
    throw "mysqldump failed"
  }
  Write-Host "Backup completed: $BackupFile"
  exit 0
}

if (-not (Test-Path -LiteralPath $BackupFile)) {
  throw "Backup file not found: $BackupFile"
}

Write-Host "Restoring $Database from $BackupFile ..."
Get-Content -Raw -LiteralPath $BackupFile | & $mysql.Source @baseArgs $Database
if ($LASTEXITCODE -ne 0) {
  throw "mysql restore failed"
}
Write-Host "Restore completed from: $BackupFile"
