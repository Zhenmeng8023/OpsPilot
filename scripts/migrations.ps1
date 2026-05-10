param(
  [ValidateSet("apply", "verify", "status")]
  [string]$Mode = "apply",
  [string]$HostName = "127.0.0.1",
  [int]$Port = 3306,
  [string]$Database = "opspilot",
  [string]$User = "opspilot",
  [string]$Password = "opspilot",
  [string]$MysqlExe = "mysql",
  [string]$MigrationDir = ""
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
if ([string]::IsNullOrWhiteSpace($MigrationDir)) {
  $MigrationDir = Join-Path $repoRoot "server/migrations"
}

if (-not (Test-Path -LiteralPath $MigrationDir)) {
  throw "Migration directory not found: $MigrationDir"
}

$upMigrations = Get-ChildItem -LiteralPath $MigrationDir -Filter "*.up.sql" | Sort-Object Name
if ($upMigrations.Count -eq 0) {
  throw "No up migrations found under $MigrationDir"
}

foreach ($up in $upMigrations) {
  $downName = $up.Name -replace "\.up\.sql$", ".down.sql"
  $downPath = Join-Path $MigrationDir $downName
  if (-not (Test-Path -LiteralPath $downPath)) {
    throw "Missing down migration for $($up.Name): $downName"
  }
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

function Invoke-MySqlQuery {
  param([string]$Query)
  $Query | & $mysql.Source @mysqlArgs
  if ($LASTEXITCODE -ne 0) {
    throw "MySQL query failed."
  }
}

function Invoke-MySqlFile {
  param([string]$Path)
  Get-Content -Raw -LiteralPath $Path | & $mysql.Source @mysqlArgs
  if ($LASTEXITCODE -ne 0) {
    throw "MySQL file execution failed: $Path"
  }
}

function Escape-SqlLiteral {
  param([string]$Value)
  if ($null -eq $Value) {
    return ""
  }
  return $Value.Replace("'", "''")
}

function Ensure-SchemaMigrationTable {
  $ddl = @"
CREATE TABLE IF NOT EXISTS schema_migrations (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  version VARCHAR(128) NOT NULL,
  checksum CHAR(64) NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'applied',
  execution_ms INT UNSIGNED NOT NULL DEFAULT 0,
  notes VARCHAR(255) NULL,
  applied_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_schema_migrations_version (version),
  KEY idx_schema_migrations_applied_at (applied_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci ROW_FORMAT=DYNAMIC;
"@
  Invoke-MySqlQuery $ddl
}

function Get-AppliedMigrations {
  $result = @{}
  $rows = & $mysql.Source @mysqlArgs -N -B -e "SELECT version, COALESCE(checksum, '') AS checksum FROM schema_migrations"
  if ($LASTEXITCODE -ne 0) {
    throw "Failed to query schema_migrations."
  }
  foreach ($row in $rows) {
    if ([string]::IsNullOrWhiteSpace($row)) {
      continue
    }
    $parts = $row -split "`t", 2
    if ($parts.Count -eq 0) {
      continue
    }
    $version = $parts[0].Trim()
    if ([string]::IsNullOrWhiteSpace($version)) {
      continue
    }
    $checksum = ""
    if ($parts.Count -gt 1) {
      $checksum = $parts[1].Trim().ToLowerInvariant()
    }
    $result[$version] = $checksum
  }
  return $result
}

function Verify-Migrations {
  $applied = Get-AppliedMigrations
  $pending = @()
  $checksumMismatch = @()
  foreach ($migration in $upMigrations) {
    $name = $migration.Name
    if (-not $applied.ContainsKey($name)) {
      $pending += $name
      continue
    }
    $expected = (Get-FileHash -LiteralPath $migration.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    $actual = $applied[$name]
    if (-not [string]::IsNullOrWhiteSpace($actual) -and $actual -ne $expected) {
      $checksumMismatch += "$name (db=$actual file=$expected)"
    }
  }

  if ($pending.Count -gt 0) {
    throw "Pending migrations: $($pending -join ', ')"
  }
  if ($checksumMismatch.Count -gt 0) {
    throw "Migration checksum mismatch: $($checksumMismatch -join '; ')"
  }
}

Write-Host "Migration pair check passed: $($upMigrations.Count) up/down pairs."

Ensure-SchemaMigrationTable

if ($Mode -eq "apply") {
  $appliedMap = Get-AppliedMigrations
  foreach ($migration in $upMigrations) {
    $version = $migration.Name
    if ($appliedMap.ContainsKey($version)) {
      Write-Host "Skipping $version (already recorded)."
      continue
    }

    Write-Host "Applying $version ..."
    $checksum = (Get-FileHash -LiteralPath $migration.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
    $startedAt = Get-Date
    Invoke-MySqlFile $migration.FullName
    $elapsed = [int][Math]::Max(0, ((Get-Date) - $startedAt).TotalMilliseconds)
    $versionSql = Escape-SqlLiteral $version
    $checksumSql = Escape-SqlLiteral $checksum
    $notesSql = Escape-SqlLiteral "applied by scripts/migrations.ps1"
    Invoke-MySqlQuery "INSERT INTO schema_migrations(version, checksum, status, execution_ms, notes) VALUES ('$versionSql', '$checksumSql', 'applied', $elapsed, '$notesSql') ON DUPLICATE KEY UPDATE checksum = VALUES(checksum), status = VALUES(status), execution_ms = VALUES(execution_ms), notes = VALUES(notes), applied_at = NOW(3)"
  }
  Verify-Migrations
  Write-Host "Migration apply + verify passed."
  exit 0
}

if ($Mode -eq "verify") {
  Verify-Migrations
  Write-Host "Migration verify passed."
  exit 0
}

$appliedStatus = Get-AppliedMigrations
Write-Host "Applied migrations recorded: $($appliedStatus.Count)"
Write-Host "Expected root migrations: $($upMigrations.Count)"
