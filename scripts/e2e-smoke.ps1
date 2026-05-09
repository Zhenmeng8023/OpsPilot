param(
  [string]$ApiBaseUrl = "http://localhost:8080",
  [string]$Username = "admin",
  [string]$Password = "Admin@123456",
  [string]$ScheduleTimezone = "Asia/Shanghai",
  [int]$PollIntervalSeconds = 2,
  [int]$RunTimeoutSeconds = 90,
  [int]$ScheduleTimeoutSeconds = 150,
  [switch]$SkipAuthenticatedChecks
)

$ErrorActionPreference = "Stop"

$terminalWorkflowStatuses = @("success", "failed", "canceled", "timeout")

function Invoke-OpsPilotJson {
  param(
    [string]$Method,
    [string]$Url,
    [object]$Body = $null,
    [hashtable]$Headers = @{}
  )

  $args = @{
    Method  = $Method
    Uri     = $Url
    Headers = $Headers
  }
  if ($null -ne $Body) {
    $args.ContentType = "application/json"
    if ($Body -is [string]) {
      $args.Body = $Body
    } else {
      $args.Body = ($Body | ConvertTo-Json -Depth 20 -Compress)
    }
  }
  return Invoke-RestMethod @args
}

function Invoke-OpsPilotExport {
  param(
    [string]$Url,
    [hashtable]$Headers
  )

  return Invoke-WebRequest -Method GET -Uri $Url -Headers $Headers -UseBasicParsing
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

function Assert-True {
  param(
    [bool]$Condition,
    [string]$Message
  )

  if (-not $Condition) {
    throw $Message
  }
}

function Assert-NotContains {
  param(
    [string]$Text,
    [string]$Needle,
    [string]$Message
  )

  if ($Text -like "*$Needle*") {
    throw $Message
  }
}

function New-UniqueName {
  param([string]$Prefix)

  return "{0}-{1}" -f $Prefix, [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
}

function Wait-Until {
  param(
    [scriptblock]$Condition,
    [int]$TimeoutSeconds = 60,
    [int]$IntervalSeconds = 2,
    [string]$Description = "condition"
  )

  $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
  while ((Get-Date) -lt $deadline) {
    $result = & $Condition
    if ($null -ne $result -and $result -ne $false) {
      return $result
    }
    Start-Sleep -Seconds $IntervalSeconds
  }
  throw "Timed out waiting for $Description after ${TimeoutSeconds}s"
}

function New-WorkflowDefinition {
  param(
    [array]$Nodes,
    [array]$Edges = @(),
    [int]$MaxParallel = 1,
    [string]$FailurePolicy = "stop_workflow"
  )

  return (@{
      nodes         = $Nodes
      edges         = $Edges
      maxParallel   = $MaxParallel
      failurePolicy = $FailurePolicy
    } | ConvertTo-Json -Depth 20 -Compress)
}

function New-Workflow {
  param(
    [hashtable]$Headers,
    [string]$Name,
    [string]$Definition,
    [string]$Description = ""
  )

  $created = Invoke-OpsPilotJson -Method POST -Url "$ApiBaseUrl/api/v1/workflows" -Headers $Headers -Body @{
    name        = $Name
    description = $Description
    definition  = $Definition
  }
  Assert-Envelope $created "create workflow $Name"

  $published = Invoke-OpsPilotJson -Method POST -Url "$ApiBaseUrl/api/v1/workflows/$($created.data.id)/publish" -Headers $Headers
  Assert-Envelope $published "publish workflow $Name"

  return $published.data
}

function Start-WorkflowRun {
  param(
    [hashtable]$Headers,
    [string]$WorkflowId,
    [string]$TriggerType = "manual",
    [string]$Input = ""
  )

  $payload = @{ triggerType = $TriggerType }
  if (-not [string]::IsNullOrWhiteSpace($Input)) {
    $payload.input = $Input
  }
  $run = Invoke-OpsPilotJson -Method POST -Url "$ApiBaseUrl/api/v1/workflows/$WorkflowId/run" -Headers $Headers -Body $payload
  Assert-Envelope $run "run workflow $WorkflowId"
  return $run.data
}

function Get-WorkflowRun {
  param(
    [hashtable]$Headers,
    [string]$RunId
  )

  $run = Invoke-OpsPilotJson -Method GET -Url "$ApiBaseUrl/api/v1/workflow-runs/$RunId" -Headers $Headers
  Assert-Envelope $run "get workflow run $RunId"
  return $run.data
}

function Wait-WorkflowTerminal {
  param(
    [hashtable]$Headers,
    [string]$RunId,
    [int]$TimeoutSeconds = 90
  )

  return Wait-Until -TimeoutSeconds $TimeoutSeconds -IntervalSeconds $PollIntervalSeconds -Description "workflow run $RunId terminal status" -Condition {
    $detail = Get-WorkflowRun -Headers $Headers -RunId $RunId
    if ($terminalWorkflowStatuses -contains $detail.status) {
      return $detail
    }
    return $null
  }
}

function Build-WebhookSignature {
  param(
    [string]$Secret,
    [string]$Body
  )

  $hmac = [System.Security.Cryptography.HMACSHA256]::new([Text.Encoding]::UTF8.GetBytes($Secret))
  try {
    $hash = $hmac.ComputeHash([Text.Encoding]::UTF8.GetBytes($Body))
    $hex = ([BitConverter]::ToString($hash)).Replace("-", "").ToLowerInvariant()
    return "sha256=$hex"
  } finally {
    $hmac.Dispose()
  }
}

function Assert-WorkflowTraceability {
  param(
    [object]$Run,
    [string]$ExpectedTrigger
  )

  Assert-True ($Run.triggerType -eq $ExpectedTrigger) "Expected triggerType $ExpectedTrigger, got $($Run.triggerType)"
  Assert-True ($Run.nodes.Count -gt 0) "Expected workflow run nodes to be present"
  Assert-True ($Run.events.Count -gt 0) "Expected workflow run events to be present"
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
  @{ Name = "webhook sources"; Url = "$ApiBaseUrl/api/v1/webhooks/sources" },
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

Write-Host "Validating notification secret masking and audit export/retention..."
$channelSecret = "smoke-channel-secret"
$channelName = New-UniqueName "smoke-webhook-channel"
$channel = Invoke-OpsPilotJson -Method POST -Url "$ApiBaseUrl/api/v1/notification-channels" -Headers $headers -Body @{
  name        = $channelName
  channelType = "webhook"
  config      = @{
    url           = "https://hooks.example.com/smoke"
    signingSecret = $channelSecret
  }
}
Assert-Envelope $channel "create notification channel"
$channelText = ($channel | ConvertTo-Json -Depth 20 -Compress)
Assert-NotContains -Text $channelText -Needle $channelSecret -Message "Notification channel create API leaked the secret"

$channels = Invoke-OpsPilotJson -Method GET -Url "$ApiBaseUrl/api/v1/notification-channels" -Headers $headers
Assert-Envelope $channels "list notification channels"
$channelsText = ($channels | ConvertTo-Json -Depth 20 -Compress)
Assert-NotContains -Text $channelsText -Needle $channelSecret -Message "Notification channel list API leaked the secret"
Assert-True ((@($channels.data | Where-Object { $_.id -eq $channel.data.id -and $_.targetSummary -eq "https://hooks.example.com/..." }).Count) -gt 0) "Expected notification channel target to stay masked"

$auditLogs = Invoke-OpsPilotJson -Method GET -Url "$ApiBaseUrl/api/v1/audit-logs?action=notification.channel.create&page=1&pageSize=20" -Headers $headers
Assert-Envelope $auditLogs "list audit logs"
$auditLogsText = ($auditLogs | ConvertTo-Json -Depth 20 -Compress)
Assert-NotContains -Text $auditLogsText -Needle $channelSecret -Message "Audit log API leaked the notification secret"

$export = Invoke-OpsPilotExport -Url "$ApiBaseUrl/api/v1/audit-logs/export?format=json&action=notification.channel.create" -Headers $headers
$exportText = if ($export.Content -is [byte[]]) {
  [Text.Encoding]::UTF8.GetString($export.Content)
} else {
  [string]$export.Content
}
Assert-NotContains -Text $exportText -Needle $channelSecret -Message "Audit export leaked the notification secret"

$retention = Invoke-OpsPilotJson -Method POST -Url "$ApiBaseUrl/api/v1/audit-logs/retention/run" -Headers $headers -Body @{
  days   = 1
  dryRun = $true
}
Assert-Envelope $retention "audit retention"
Assert-True ($retention.data.dryRun -eq $true) "Expected audit retention dryRun result"

$auditExportEntry = Wait-Until -TimeoutSeconds 30 -IntervalSeconds $PollIntervalSeconds -Description "audit export audit record" -Condition {
  $response = Invoke-OpsPilotJson -Method GET -Url "$ApiBaseUrl/api/v1/audit-logs?action=audit.export&page=1&pageSize=20" -Headers $headers
  Assert-Envelope $response "list audit export records"
  $entry = $response.data.items | Select-Object -First 1
  if ($null -ne $entry) {
    return $entry
  }
  return $null
}
Assert-True ($auditExportEntry.action -eq "audit.export") "Expected audit export action to be recorded"

$auditRetentionEntry = Wait-Until -TimeoutSeconds 30 -IntervalSeconds $PollIntervalSeconds -Description "audit retention audit record" -Condition {
  $response = Invoke-OpsPilotJson -Method GET -Url "$ApiBaseUrl/api/v1/audit-logs?action=audit.retention.run&page=1&pageSize=20" -Headers $headers
  Assert-Envelope $response "list audit retention records"
  $entry = $response.data.items | Select-Object -First 1
  if ($null -ne $entry) {
    return $entry
  }
  return $null
}
Assert-True ($auditRetentionEntry.action -eq "audit.retention.run") "Expected audit retention action to be recorded"

Write-Host "Creating workflows for manual/schedule/webhook/cancel/retry validation..."
$waitDefinition = New-WorkflowDefinition -Nodes @(
  @{ id = "pause"; type = "wait"; name = "Pause"; config = @{ seconds = 1 } }
)
$waitWorkflow = New-Workflow -Headers $headers -Name (New-UniqueName "smoke-workflow-wait") -Definition $waitDefinition -Description "Wait workflow for manual/schedule/webhook smoke"

$cancelDefinition = New-WorkflowDefinition -Nodes @(
  @{ id = "pause"; type = "wait"; name = "Long Pause"; config = @{ seconds = 20 } }
)
$cancelWorkflow = New-Workflow -Headers $headers -Name (New-UniqueName "smoke-workflow-cancel") -Definition $cancelDefinition -Description "Cancelable wait workflow"

$failDefinition = New-WorkflowDefinition -Nodes @(
  @{ id = "gate"; type = "condition"; name = "Gate"; config = @{ path = "allow"; operator = "equals"; value = $true; onFalse = "fail" } }
)
$failWorkflow = New-Workflow -Headers $headers -Name (New-UniqueName "smoke-workflow-fail") -Definition $failDefinition -Description "Failing workflow for retry validation"

Write-Host "Running manual workflow..."
$manualRun = Start-WorkflowRun -Headers $headers -WorkflowId $waitWorkflow.id -TriggerType "manual" -Input (@{ source = "manual" } | ConvertTo-Json -Compress)
$manualDetail = Wait-WorkflowTerminal -Headers $headers -RunId $manualRun.id -TimeoutSeconds $RunTimeoutSeconds
Assert-True ($manualDetail.status -eq "success") "Expected manual workflow run to succeed, got $($manualDetail.status)"
Assert-WorkflowTraceability -Run $manualDetail -ExpectedTrigger "manual"

Write-Host "Running cancelable workflow..."
$cancelRun = Start-WorkflowRun -Headers $headers -WorkflowId $cancelWorkflow.id -TriggerType "manual" -Input (@{ source = "cancel" } | ConvertTo-Json -Compress)
$null = Wait-Until -TimeoutSeconds 20 -IntervalSeconds $PollIntervalSeconds -Description "cancel workflow to become active" -Condition {
  $detail = Get-WorkflowRun -Headers $headers -RunId $cancelRun.id
  if ($detail.status -in @("queued", "running")) {
    return $detail
  }
  return $null
}
$canceled = Invoke-OpsPilotJson -Method POST -Url "$ApiBaseUrl/api/v1/workflow-runs/$($cancelRun.id)/cancel" -Headers $headers -Body @{ reason = "smoke cancel" }
Assert-Envelope $canceled "cancel workflow run"
$cancelDetail = Wait-WorkflowTerminal -Headers $headers -RunId $cancelRun.id -TimeoutSeconds $RunTimeoutSeconds
Assert-True ($cancelDetail.status -eq "canceled") "Expected canceled workflow run, got $($cancelDetail.status)"
Assert-WorkflowTraceability -Run $cancelDetail -ExpectedTrigger "manual"

Write-Host "Running retryable workflow..."
$failedRun = Start-WorkflowRun -Headers $headers -WorkflowId $failWorkflow.id -TriggerType "manual" -Input (@{ allow = $false } | ConvertTo-Json -Compress)
$failedDetail = Wait-WorkflowTerminal -Headers $headers -RunId $failedRun.id -TimeoutSeconds $RunTimeoutSeconds
Assert-True ($failedDetail.status -eq "failed") "Expected failing workflow run to fail, got $($failedDetail.status)"
Assert-WorkflowTraceability -Run $failedDetail -ExpectedTrigger "manual"
Assert-True ((@($failedDetail.nodes | Where-Object { $_.nodeId -eq "gate" -and $_.status -eq "failed" }).Count) -eq 1) "Expected failed gate node"

$retriedRunResponse = Invoke-OpsPilotJson -Method POST -Url "$ApiBaseUrl/api/v1/workflow-runs/$($failedRun.id)/retry" -Headers $headers
Assert-Envelope $retriedRunResponse "retry workflow run"
$retriedDetail = Wait-WorkflowTerminal -Headers $headers -RunId $retriedRunResponse.data.id -TimeoutSeconds $RunTimeoutSeconds
Assert-True ($retriedDetail.status -eq "failed") "Expected retried workflow run to fail with the same input, got $($retriedDetail.status)"
Assert-WorkflowTraceability -Run $retriedDetail -ExpectedTrigger "retry"

$nodeRetryResponse = Invoke-OpsPilotJson -Method POST -Url "$ApiBaseUrl/api/v1/workflow-runs/$($failedRun.id)/nodes/gate/retry" -Headers $headers
Assert-Envelope $nodeRetryResponse "retry workflow node"
$nodeRetriedDetail = Wait-WorkflowTerminal -Headers $headers -RunId $failedRun.id -TimeoutSeconds $RunTimeoutSeconds
Assert-True ($nodeRetriedDetail.status -eq "failed") "Expected node retried workflow run to remain failed with the same input, got $($nodeRetriedDetail.status)"
Assert-True ((@($nodeRetriedDetail.events | Where-Object { $_.eventType -eq "node_retry" }).Count) -gt 0) "Expected node retry event to be recorded"

Write-Host "Running schedule-triggered workflow..."
# Use an every-minute cron so the smoke is independent from the runner's local timezone.
# The schedule itself still exercises the configured schedule timezone inside the service.
$cronExpr = "* * * * *"
$schedule = Invoke-OpsPilotJson -Method POST -Url "$ApiBaseUrl/api/v1/schedules" -Headers $headers -Body @{
  name          = New-UniqueName "smoke-workflow-schedule"
  targetType    = "workflow"
  workflowId    = $waitWorkflow.id
  cronExpr      = $cronExpr
  timezone      = $ScheduleTimezone
  misfirePolicy = "fire_once"
}
Assert-Envelope $schedule "create schedule"

$scheduleTrigger = Wait-Until -TimeoutSeconds $ScheduleTimeoutSeconds -IntervalSeconds $PollIntervalSeconds -Description "schedule trigger" -Condition {
  $response = Invoke-OpsPilotJson -Method GET -Url "$ApiBaseUrl/api/v1/schedules/$($schedule.data.id)/triggers?limit=10" -Headers $headers
  Assert-Envelope $response "list schedule triggers"
  $trigger = $response.data | Where-Object { $_.status -eq "fired" -and -not [string]::IsNullOrWhiteSpace($_.workflowRunId) } | Select-Object -First 1
  if ($null -ne $trigger) {
    return $trigger
  }
  return $null
}
$scheduleRunDetail = Wait-WorkflowTerminal -Headers $headers -RunId $scheduleTrigger.workflowRunId -TimeoutSeconds $RunTimeoutSeconds
Assert-True ($scheduleRunDetail.status -eq "success") "Expected schedule workflow run to succeed, got $($scheduleRunDetail.status)"
Assert-WorkflowTraceability -Run $scheduleRunDetail -ExpectedTrigger "schedule"
$scheduleDisabled = Invoke-OpsPilotJson -Method POST -Url "$ApiBaseUrl/api/v1/schedules/$($schedule.data.id)/disable" -Headers $headers
Assert-Envelope $scheduleDisabled "disable smoke schedule"

Write-Host "Running webhook-triggered workflow plus matcher simulator and replay..."
$webhookSource = Invoke-OpsPilotJson -Method POST -Url "$ApiBaseUrl/api/v1/webhooks/sources" -Headers $headers -Body @{
  name       = New-UniqueName "smoke-webhook-source"
  sourceType = "custom"
}
Assert-Envelope $webhookSource "create webhook source"
Assert-True (-not [string]::IsNullOrWhiteSpace($webhookSource.data.token)) "Expected webhook source token"
Assert-True (-not [string]::IsNullOrWhiteSpace($webhookSource.data.signingSecret)) "Expected webhook source signing secret"

$ruleMatcher = @{
  conditions = @(
    @{
      type  = "payload_equals"
      path  = "payload.service"
      value = "api"
    }
  )
}
$webhookRule = Invoke-OpsPilotJson -Method POST -Url "$ApiBaseUrl/api/v1/webhooks/rules" -Headers $headers -Body @{
  sourceId   = $webhookSource.data.id
  targetType = "workflow"
  workflowId = $waitWorkflow.id
  name       = New-UniqueName "smoke-webhook-rule"
  eventType  = "push"
  matcher    = $ruleMatcher
}
Assert-Envelope $webhookRule "create webhook rule"

$payloadObject = @{
  payload = @{
    service     = "api"
    environment = "staging"
  }
}
$payloadText = $payloadObject | ConvertTo-Json -Depth 20 -Compress

$simulation = Invoke-OpsPilotJson -Method POST -Url "$ApiBaseUrl/api/v1/webhooks/matcher/simulate" -Headers $headers -Body @{
  ruleId    = $webhookRule.data.id
  eventType = "push"
  payload   = $payloadText
}
Assert-Envelope $simulation "simulate webhook matcher"
Assert-True ($simulation.data.matched -eq $true) "Expected webhook matcher simulation to match"
Assert-True ($simulation.data.conditions.Count -eq 1 -and $simulation.data.conditions[0].matched -eq $true) "Expected webhook matcher condition to match"

$timestamp = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds().ToString()
$nonce = New-UniqueName "nonce"
$deliveryId = New-UniqueName "delivery"
$signature = Build-WebhookSignature -Secret $webhookSource.data.signingSecret -Body $payloadText
$webhookHeaders = @{
  "Content-Type"          = "application/json"
  "X-Event-Type"          = "push"
  "X-Delivery-Id"         = $deliveryId
  "X-OpsPilot-Timestamp"  = $timestamp
  "X-OpsPilot-Nonce"      = $nonce
  "X-OpsPilot-Signature"  = $signature
}
$triggered = Invoke-RestMethod -Method POST -Uri "$ApiBaseUrl/api/v1/webhooks/trigger/$($webhookSource.data.token)" -Headers $webhookHeaders -Body $payloadText
Assert-Envelope $triggered "trigger webhook"
Assert-True ($triggered.data.triggeredRuns.Count -eq 1) "Expected webhook trigger to start one workflow run"

$webhookEvent = Invoke-OpsPilotJson -Method GET -Url "$ApiBaseUrl/api/v1/webhooks/events/$($triggered.data.eventId)" -Headers $headers
Assert-Envelope $webhookEvent "get webhook event"
Assert-True ($webhookEvent.data.matches.Count -eq 1) "Expected webhook event match detail"

$webhookRunDetail = Wait-WorkflowTerminal -Headers $headers -RunId $triggered.data.triggeredRuns[0] -TimeoutSeconds $RunTimeoutSeconds
Assert-True ($webhookRunDetail.status -eq "success") "Expected webhook workflow run to succeed, got $($webhookRunDetail.status)"
Assert-WorkflowTraceability -Run $webhookRunDetail -ExpectedTrigger "webhook"

$replayed = Invoke-OpsPilotJson -Method POST -Url "$ApiBaseUrl/api/v1/webhooks/events/$($triggered.data.eventId)/replay" -Headers $headers -Body @{
  simulateOnly  = $false
  idempotencyKey = New-UniqueName "replay"
}
Assert-Envelope $replayed "replay webhook event"
Assert-True ($replayed.data.triggeredRuns.Count -eq 1) "Expected webhook replay to start one workflow run"
$replayRunDetail = Wait-WorkflowTerminal -Headers $headers -RunId $replayed.data.triggeredRuns[0] -TimeoutSeconds $RunTimeoutSeconds
Assert-True ($replayRunDetail.status -eq "success") "Expected replayed webhook workflow run to succeed, got $($replayRunDetail.status)"
Assert-WorkflowTraceability -Run $replayRunDetail -ExpectedTrigger "webhook"

Write-Host "E2E smoke passed: health/version/login, secret masking, audit export/retention, manual/schedule/webhook workflow triggers, run cancel/retry/node-retry, matcher simulator, and webhook replay."
