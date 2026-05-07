export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
  traceId: string;
}

export interface HealthData {
  status: string;
  service: string;
  version: string;
  env: string;
  time: string;
  checks?: {
    database?: string;
    redis?: string;
  };
}

export interface VersionInfo {
  service: string;
  version: string;
  commit: string;
  buildTime: string;
  goVersion: string;
  env: string;
}

export interface UserProfile {
  id: string;
  username: string;
  email?: string;
  roles: string[];
  permissions: string[];
  workspace: {
    id: string;
    name: string;
    slug: string;
  };
}

export interface AuthResponse {
  accessToken: string;
  refreshToken: string;
  tokenType: string;
  expiresIn: number;
  user: UserProfile;
}

export interface ManagedUser {
  id: string;
  username: string;
  email?: string;
  status: string;
  roles: string[];
  lastLoginAt?: string;
  createdAt: string;
}

export interface Permission {
  code: string;
  module: string;
  name: string;
  description?: string;
}

export interface Role {
  id: string;
  code: string;
  name: string;
  description?: string;
  builtIn: boolean;
  status: string;
  permissions: Permission[];
}

export interface Host {
  id: string;
  name: string;
  hostname?: string;
  ip?: string;
  os?: string;
  arch?: string;
  status: string;
  agentCount?: number;
  onlineAgentCount?: number;
  lastHeartbeatAt?: string;
  createdAt: string;
}

export interface Agent {
  id: string;
  name: string;
  status: string;
  tokenPrefix?: string;
  version?: string;
  ip?: string;
  os?: string;
  arch?: string;
  lastHeartbeatAt?: string;
  disabledAt?: string;
  createdAt: string;
  host?: Host;
}

export interface OfflineScanResult {
  offlineAgents: number;
  offlineHosts: number;
  thresholdSeconds: number;
}

export interface EnrollmentTokenSummary {
  id: string;
  tokenPrefix: string;
  status: string;
  maxUses: number;
  usedCount: number;
  bindWorkspaceSlug?: string;
  expiresAt: string;
  createdBy?: string;
  createdAt: string;
}

export interface EnrollmentTokenDetail extends EnrollmentTokenSummary {
  token?: string;
}

export interface PageResult<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}

export interface Script {
  id: string;
  name: string;
  description?: string;
  scriptType: string;
  status: string;
  version: number;
  approvalRequired: boolean;
  approvalStatus?: string;
  content?: string;
  changeSummary?: string;
  createdBy?: string;
  createdAt: string;
  updatedAt: string;
}

export interface ScriptApprovalSummary {
  id: number;
  versionId: string;
  scriptId: string;
  scriptName: string;
  version: number;
  status: string;
  comment?: string;
  approver?: string;
  approvedAt?: string;
  createdAt: string;
}

export interface TaskSummary {
  id: string;
  taskId: string;
  name: string;
  description?: string;
  status: string;
  timeoutSeconds: number;
  targetCount: number;
  successCount: number;
  failedCount: number;
  canceledCount: number;
  runningCount: number;
  queuedCount: number;
  createdBy?: string;
  createdAt: string;
  queuedAt?: string;
  startedAt?: string;
  finishedAt?: string;
  errorMessage?: string;
}

export interface TaskTarget {
  id: string;
  status: string;
  agentId?: string;
  agentName?: string;
  agentStatus?: string;
  hostId?: string;
  hostName?: string;
  exitCode?: number;
  errorMessage?: string;
  startedAt?: string;
  finishedAt?: string;
  createdAt: string;
}

export interface TaskDetail extends TaskSummary {
  targets?: TaskTarget[];
}

export interface TaskLogEntry {
  id: number;
  runId: string;
  targetId: string;
  sequence: number;
  stream: "stdout" | "stderr" | "system";
  content: string;
  createdAt: string;
  agentName?: string;
  hostName?: string;
}

export interface ScheduleSummary {
  id: string;
  name: string;
  taskId: string;
  taskName: string;
  scheduleType: string;
  cronExpr: string;
  timezone: string;
  misfirePolicy: string;
  status: string;
  nextFireAt?: string;
  lastFireAt?: string;
  createdBy?: string;
  createdAt: string;
}

export interface ScheduleTrigger {
  id: number;
  taskRunId?: string;
  plannedFireAt: string;
  actualFireAt?: string;
  status: string;
  errorMessage?: string;
  createdAt: string;
}

export interface WebhookSource {
  id: string;
  name: string;
  sourceType: string;
  status: string;
  lastReceivedAt?: string;
  createdBy?: string;
  createdAt: string;
  token?: string;
  signingSecret?: string;
}

export interface WebhookRule {
  id: string;
  sourceId: string;
  sourceName: string;
  taskId: string;
  taskName: string;
  name: string;
  eventType?: string;
  matcher?: WebhookMatcher;
  status: string;
  createdBy?: string;
  createdAt: string;
}

export interface WebhookMatcher {
  conditions: WebhookMatcherCondition[];
}

export interface WebhookMatcherCondition {
  type: "header_equals" | "payload_equals" | "payload_contains" | "event_type_equals" | "ref_equals" | "branch_equals";
  key?: string;
  path?: string;
  value: string;
}

export interface WebhookEvent {
  id: string;
  sourceId?: string;
  sourceName?: string;
  eventType?: string;
  deliveryId?: string;
  sourceTimestamp?: string;
  nonce?: string;
  signatureHeader?: string;
  signatureValid: boolean;
  replayed: boolean;
  status: string;
  errorMessage?: string;
  remoteIp?: string;
  payloadHash?: string;
  receivedAt: string;
}

export interface WebhookEventMatch {
  id: number;
  ruleId?: string;
  ruleName?: string;
  matched: boolean;
  reason?: string;
  taskRunId?: string;
  createdAt: string;
}

export interface WebhookEventDetail extends WebhookEvent {
  headers: Array<{ key: string; value: string }>;
  payload?: string;
  matches: WebhookEventMatch[];
}

export interface HostMetric {
  id: number;
  hostId: string;
  hostName: string;
  agentId?: string;
  agentName?: string;
  metricCode: string;
  value: number;
  unit?: string;
  collectedAt: string;
  createdAt: string;
}

export interface MetricTrendPoint {
  collectedAt: string;
  value: number;
}

export interface MetricTrendSeries {
  hostId: string;
  hostName: string;
  agentId?: string;
  agentName?: string;
  metricCode: string;
  unit?: string;
  latestValue: number;
  minValue: number;
  maxValue: number;
  points: MetricTrendPoint[];
}

export interface AlertRule {
  id: string;
  name: string;
  ruleType: string;
  metricCode?: string;
  operator?: string;
  threshold?: number;
  durationSeconds: number;
  cooldownSeconds: number;
  severity: string;
  status: string;
  createdBy?: string;
  createdAt: string;
}

export interface AlertSummary {
  id: string;
  ruleId?: string;
  ruleName?: string;
  resourceType: string;
  resourceId?: number;
  hostId?: string;
  hostName?: string;
  title: string;
  message?: string;
  severity: string;
  status: string;
  acknowledgedAt?: string;
  acknowledgedBy?: string;
  silencedUntil?: string;
  silenceReason?: string;
  cooldownUntil?: string;
  firstSeenAt: string;
  lastSeenAt: string;
  resolvedAt?: string;
  createdAt: string;
}

export interface AlertEventSummary {
  id: number;
  eventType: string;
  message?: string;
  actor?: string;
  payload?: string;
  createdAt: string;
}

export interface AlertHistoryPoint {
  bucketStart: string;
  firingCount: number;
  resolvedCount: number;
  acknowledgedCount: number;
  silencedCount: number;
}

export interface NotificationChannel {
  id: string;
  name: string;
  channelType: string;
  targetSummary?: string;
  status: string;
  createdBy?: string;
  createdAt: string;
}

export interface NotificationChannelTestResult {
  channelId: string;
  notificationId: string;
  deliveryId: number;
  status: string;
  errorMessage?: string;
}

export interface NotificationSummary {
  id: string;
  title: string;
  content?: string;
  category: string;
  severity: string;
  resourceType?: string;
  resourceId?: number;
  readAt?: string;
  createdAt: string;
}

export interface NotificationDelivery {
  id: number;
  notificationId?: string;
  title?: string;
  channelId?: string;
  channelName?: string;
  channelType?: string;
  status: string;
  attempts: number;
  nextRetryAt?: string;
  deliveredAt?: string;
  errorMessage?: string;
  createdAt: string;
  updatedAt: string;
}

export interface AuditLog {
  id: number;
  actorType: string;
  actorUser?: string;
  actorAgent?: string;
  action: string;
  resourceType?: string;
  resourceId?: number;
  result: string;
  ip?: string;
  userAgent?: string;
  traceId?: string;
  requestMethod?: string;
  requestPath?: string;
  before?: string;
  after?: string;
  metadata?: string;
  createdAt: string;
}
