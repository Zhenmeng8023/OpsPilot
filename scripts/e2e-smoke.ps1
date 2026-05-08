param(
  [string]$ApiBaseUrl = "http://localhost:8080",
  [string]$Username = "admin",
  [string]$Password = "Admin@123456",
  [switch]$SkipAuthenticatedChecks
)

$ErrorActionPreference = "Stop"

function Invoke-OpsPilotJson {
  param(
    [string]$Method,
    [string]$Url,
    [object]$Body = $null,
    [hashtable]$Headers = @{}
  )

  $args = @{
    Method = $Method
    Uri = $Url
    Headers = $Headers
  }
  if ($null -ne $Body) {
    $args.ContentType = "application/json"
    $args.Body = ($Body | ConvertTo-Json -Depth 20)
  }
  return Invoke-RestMethod @args
}

function Assert-Envelope {
  param(
    [object]$Response,
    [string]$Name
  )

  if ($null -eq $Response -or $Response.code -ne 0) {
    throw "$Name failed: unexpected response envelope"
  }
}

$ApiBaseUrl = $ApiBaseUrl.TrimEnd("/")

Write-Host "Checking health..."
$health = Invoke-OpsPilotJson -Method GET -Url "$ApiBaseUrl/health"
Assert-Envelope $health "health"
if ($health.data.status -ne "ok") {
  throw "health failed: status=$($health.data.status)"
}

Write-Host "Checking version..."
$version = Invoke-OpsPilotJson -Method GET -Url "$ApiBaseUrl/api/v1/version"
Assert-Envelope $version "version"

if ($SkipAuthenticatedChecks) {
  Write-Host "E2E smoke passed: health/version only."
  exit 0
}

Write-Host "Logging in..."
$login = Invoke-OpsPilotJson -Method POST -Url "$ApiBaseUrl/api/v1/auth/login" -Body @{
  username = $Username
  password = $Password
}
Assert-Envelope $login "login"
$token = $login.data.accessToken
if ([string]::IsNullOrWhiteSpace($token)) {
  throw "login failed: accessToken missing"
}

$headers = @{ Authorization = "Bearer $token" }

Write-Host "Checking authenticated core routes..."
$checks = @(
  @{ Name = "me"; Url = "$ApiBaseUrl/api/v1/me" },
  @{ Name = "agents"; Url = "$ApiBaseUrl/api/v1/agents?page=1&pageSize=1" },
  @{ Name = "scripts"; Url = "$ApiBaseUrl/api/v1/scripts?page=1&pageSize=1" },
  @{ Name = "tasks"; Url = "$ApiBaseUrl/api/v1/tasks?page=1&pageSize=1" },
  @{ Name = "schedules"; Url = "$ApiBaseUrl/api/v1/schedules?page=1&pageSize=1" },
  @{ Name = "webhook sources"; Url = "$ApiBaseUrl/api/v1/webhooks/sources?page=1&pageSize=1" },
  @{ Name = "workflow definitions"; Url = "$ApiBaseUrl/api/v1/workflows?page=1&pageSize=1" },
  @{ Name = "workflow runs"; Url = "$ApiBaseUrl/api/v1/workflow-runs?page=1&pageSize=1" },
  @{ Name = "metrics"; Url = "$ApiBaseUrl/api/v1/metrics/hosts?limit=1" },
  @{ Name = "alerts"; Url = "$ApiBaseUrl/api/v1/alerts?page=1&pageSize=1" },
  @{ Name = "incidents"; Url = "$ApiBaseUrl/api/v1/incidents?page=1&pageSize=1" },
  @{ Name = "notifications"; Url = "$ApiBaseUrl/api/v1/notifications?page=1&pageSize=1" },
  @{ Name = "audit logs"; Url = "$ApiBaseUrl/api/v1/audit-logs?page=1&pageSize=1" }
)

foreach ($check in $checks) {
  $response = Invoke-OpsPilotJson -Method GET -Url $check.Url -Headers $headers
  Assert-Envelope $response $check.Name
}

Write-Host "E2E smoke passed: health, version, login, and authenticated core route checks."
