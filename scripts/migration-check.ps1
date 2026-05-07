param(
  [string]$HostName = "127.0.0.1",
  [int]$Port = 3306,
  [string]$Database = "opspilot",
  [string]$User = "opspilot",
  [string]$Password = "opspilot",
  [string]$MysqlExe = "mysql",
  [switch]$SkipExecution
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$migrationDir = Join-Path $repoRoot "server/migrations"

$upMigrations = Get-ChildItem -LiteralPath $migrationDir -Filter "*.up.sql" | Sort-Object Name
if ($upMigrations.Count -eq 0) {
  throw "No up migrations found under $migrationDir"
}

foreach ($up in $upMigrations) {
  $downName = $up.Name -replace "\.up\.sql$", ".down.sql"
  $downPath = Join-Path $migrationDir $downName
  if (-not (Test-Path -LiteralPath $downPath)) {
    throw "Missing down migration for $($up.Name): $downName"
  }
}

Write-Host "Migration pair check passed: $($upMigrations.Count) up/down pairs."

if ($SkipExecution) {
  Write-Host "Skipping SQL execution because -SkipExecution was provided."
  exit 0
}

$mysql = Get-Command $MysqlExe -ErrorAction Stop
$mysqlArgs = @(
  "--host=$HostName",
  "--port=$Port",
  "--user=$User",
  "--password=$Password",
  "--database=$Database",
  "--default-character-set=utf8mb4"
)

foreach ($migration in $upMigrations) {
  Write-Host "Applying $($migration.Name)..."
  Get-Content -Raw -LiteralPath $migration.FullName | & $mysql.Source @mysqlArgs
  if ($LASTEXITCODE -ne 0) {
    throw "Migration failed: $($migration.Name)"
  }
}

Write-Host "All migrations applied successfully."
