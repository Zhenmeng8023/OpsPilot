param(
  [string]$ServerPath = "",
  [string[]]$Allowlist = @()
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
if ([string]::IsNullOrWhiteSpace($ServerPath)) {
  $ServerPath = Join-Path $repoRoot "server"
}
if (-not (Test-Path -LiteralPath $ServerPath)) {
  throw "Server path not found: $ServerPath"
}

$tool = Join-Path (go env GOPATH) "bin/govulncheck"
if (-not (Test-Path -LiteralPath $tool) -and $IsWindows) {
  $tool = "$tool.exe"
}
if (-not (Test-Path -LiteralPath $tool)) {
  go install golang.org/x/vuln/cmd/govulncheck@latest
}

$tmp = New-TemporaryFile
try {
  Push-Location $ServerPath
  & $tool ./... 2>&1 | Tee-Object -FilePath $tmp.FullName | Write-Host
  $exitCode = $LASTEXITCODE
  Pop-Location

  if ($exitCode -eq 0) {
    Write-Host "govulncheck passed."
    exit 0
  }

  $lines = Get-Content -LiteralPath $tmp.FullName
  $ids = @()
  foreach ($line in $lines) {
    if ($line -match "GO-\d{4}-\d+") {
      $ids += $Matches[0]
    }
  }
  $ids = $ids | Sort-Object -Unique
  if ($ids.Count -eq 0) {
    throw "govulncheck failed, but no vulnerability IDs were parsed from output."
  }

  $unknown = @()
  foreach ($id in $ids) {
    if ($Allowlist -notcontains $id) {
      $unknown += $id
    }
  }
  if ($unknown.Count -gt 0) {
    throw "govulncheck found unallowlisted vulnerabilities: $($unknown -join ', ')"
  }

  Write-Warning "Only allowlisted vulnerabilities were found: $($ids -join ', ')"
  Write-Warning "These should be removed from allowlist after upstream patched runtime/toolchain is available."
  exit 0
}
finally {
  if (Test-Path -LiteralPath $tmp.FullName) {
    Remove-Item -LiteralPath $tmp.FullName -Force
  }
}
