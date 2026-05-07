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
