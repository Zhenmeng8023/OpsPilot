param(
  [string]$OpenApiPath = "",
  [string]$ServerPath = ""
)

$ErrorActionPreference = "Stop"

function Join-RoutePath {
  param(
    [string]$Prefix,
    [string]$Path
  )

  if ([string]::IsNullOrWhiteSpace($Prefix)) {
    $Prefix = ""
  }
  if ([string]::IsNullOrWhiteSpace($Path)) {
    $Path = ""
  }

  $joined = ($Prefix.TrimEnd("/") + "/" + $Path.TrimStart("/")).TrimEnd("/")
  if ([string]::IsNullOrWhiteSpace($joined)) {
    return "/"
  }
  if (-not $joined.StartsWith("/")) {
    $joined = "/" + $joined
  }
  return $joined
}

function Normalize-RoutePath {
  param([string]$Path)
  return ($Path -replace ":([A-Za-z0-9_]+)", '{$1}')
}

function Add-Route {
  param(
    [hashtable]$Routes,
    [string]$Method,
    [string]$Path,
    [string]$Source
  )

  $normalized = Normalize-RoutePath $Path
  $key = "$($Method.ToUpperInvariant()) $normalized"
  if (-not $Routes.ContainsKey($key)) {
    $Routes[$key] = @()
  }
  $Routes[$key] += $Source
}

$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
if ([string]::IsNullOrWhiteSpace($OpenApiPath)) {
  $OpenApiPath = Join-Path $RepoRoot "server\docs\openapi\openapi.yaml"
}
if ([string]::IsNullOrWhiteSpace($ServerPath)) {
  $ServerPath = Join-Path $RepoRoot "server"
}

if (-not (Test-Path $OpenApiPath)) {
  throw "OpenAPI file not found: $OpenApiPath"
}
if (-not (Test-Path $ServerPath)) {
  throw "Server path not found: $ServerPath"
}

$openApiRoutes = @{}
$currentPath = $null
foreach ($line in Get-Content -Encoding UTF8 $OpenApiPath) {
  if ($line -match "^  (/[^:]+):\s*$") {
    $currentPath = $Matches[1]
    continue
  }
  if ($currentPath -and $line -match "^\s{4}(get|post|put|patch|delete):\s*$") {
    $key = "$($Matches[1].ToUpperInvariant()) $currentPath"
    $openApiRoutes[$key] = $true
  }
}

$routerRoutes = @{}
$files = @()
$files += Get-Item (Join-Path $ServerPath "internal\app\router.go")
$files += Get-ChildItem -Path (Join-Path $ServerPath "internal\modules") -Recurse -Filter "handler.go"

foreach ($file in $files) {
  $groups = @{
    "router" = ""
    "api" = "/api/v1"
  }

  foreach ($line in Get-Content -Encoding UTF8 $file.FullName) {
    if ($line -match "^\s*(\w+)\s*:=\s*(\w+)\.Group\(`"([^`"]*)`"\)") {
      $name = $Matches[1]
      $parent = $Matches[2]
      $path = $Matches[3]
      $parentPrefix = ""
      if ($groups.ContainsKey($parent)) {
        $parentPrefix = $groups[$parent]
      }
      $groups[$name] = Join-RoutePath $parentPrefix $path
      continue
    }

    if ($line -match "^\s*(\w+)\.(GET|POST|PUT|PATCH|DELETE)\(`"([^`"]*)`"") {
      $group = $Matches[1]
      $method = $Matches[2]
      $path = $Matches[3]
      if (-not $groups.ContainsKey($group)) {
        continue
      }
      $fullPath = Join-RoutePath $groups[$group] $path
      Add-Route $routerRoutes $method $fullPath $file.FullName
    }
  }
}

$missing = @()
foreach ($key in ($routerRoutes.Keys | Sort-Object)) {
  if (-not $openApiRoutes.ContainsKey($key)) {
    $sources = ($routerRoutes[$key] | Sort-Object -Unique) -join ", "
    $missing += "$key    # $sources"
  }
}

if ($missing.Count -gt 0) {
  Write-Error "OpenAPI is missing $($missing.Count) route(s):`n$($missing -join "`n")"
}

Write-Host "OpenAPI/router check passed: $($routerRoutes.Count) router routes documented."
