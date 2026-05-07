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

export interface UserProfile {
  id: string;
  username: string;
  email?: string;
  roles: string[];
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
