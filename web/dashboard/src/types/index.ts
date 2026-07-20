export interface DashboardEmptyState {
  title: string;
  description: string;
  commands?: string[];
}

export interface DashboardEndpoint {
  label: string;
  url: string;
}

export interface DashboardOverviewResponse {
  status: "running" | "degraded" | "unknown";
  mode: string;
  version: string;
  endpoints: DashboardEndpoint[];
  server_count: number;
  tool_count: number;
  prompt_count: number;
  resource_count: number;
  empty_state?: DashboardEmptyState;
  troubleshooting?: string[];
}

export interface DashboardServerConfigSummary {
  kind: string;
  target?: string;
  command?: string;
  argument_count?: number;
  env_keys?: string[];
  header_keys?: string[];
  session_mode?: string;
  description?: string;
  sanitized_summary: string;
}

export interface DashboardServer {
  id: number;
  name: string;
  transport: string;
  enabled: boolean;
  status: "connected" | "reachable" | "failed" | "unknown";
  tool_count: number;
  prompt_count: number;
  resource_count: number;
  last_discovered_at?: string;
  updated_at?: string;
  connection_summary: string;
  config_summary: DashboardServerConfigSummary;
}

export interface DashboardServersResponse {
  servers: DashboardServer[];
  empty_state?: DashboardEmptyState;
}

export interface DashboardTool {
  name: string;
  canonical_name: string;
  server: string;
  description: string;
  enabled: boolean;
  server_enabled: boolean;
  input_schema?: Record<string, unknown>;
  input_preview?: string;
  transport?: string;
  server_status?: string;
  annotation_keys?: string[];
}

export interface DashboardToolsResponse {
  tools: DashboardTool[];
  empty_state?: DashboardEmptyState;
}

export interface DashboardToolGroupTool {
  name: string;
  canonical_name: string;
  server: string;
  description?: string;
}

export interface DashboardToolGroup {
  name: string;
  description?: string;
  tool_count: number;
  tools: DashboardToolGroupTool[];
  streamable_http_endpoint: string;
  sse_endpoint: string;
  sse_message_endpoint: string;
}

export interface DashboardToolGroupsResponse {
  tool_groups: DashboardToolGroup[];
  empty_state?: DashboardEmptyState;
}

export interface DashboardPrompt {
  name: string;
  canonical_name: string;
  server: string;
  description: string;
  enabled: boolean;
  server_enabled: boolean;
  arguments?: Array<Record<string, unknown>>;
  arguments_preview?: string;
  transport?: string;
  server_status?: string;
}

export interface DashboardPromptsResponse {
  prompts: DashboardPrompt[];
  empty_state?: DashboardEmptyState;
}

export interface DashboardResource {
  uri: string;
  name: string;
  server: string;
  description: string;
  mime_type?: string;
  enabled: boolean;
  transport?: string;
  server_status?: string;
}

export interface DashboardResourcesResponse {
  resources: DashboardResource[];
  empty_state?: DashboardEmptyState;
}

export interface DashboardDiagnosticsResponse {
  version: string;
  mode: string;
  config_source?: string;
  config_path?: string;
  database: string;
  enabled_transports: string[];
  metrics_endpoint?: string;
  primary_endpoint: string;
  troubleshooting_hints: string[];
  server_count: number;
  tool_count: number;
  prompt_count: number;
  resource_count: number;
  empty_state?: DashboardEmptyState;
}

export type McpTransport = "stdio" | "streamable_http" | "sse";

export interface DashboardRegisterServerInput {
  name: string;
  transport: McpTransport;
  description?: string;
  url?: string;
  bearer_token?: string;
  headers?: Record<string, string>;
  command?: string;
  args?: string[];
  env?: Record<string, string>;
  session_mode?: "stateless" | "stateful";
}

export interface DashboardCreateToolGroupInput {
  name: string;
  description?: string;
  tools: string[];
}

export interface DashboardOAuthAuthorizationRequired {
  session_id: string;
  authorization_url: string;
  expires_at: string;
}

export interface DashboardRegisterServerResponse {
  name?: string;
  transport?: string;
  enabled?: boolean;
  description?: string;
  authorization_required?: DashboardOAuthAuthorizationRequired;
}

export interface VerifyTokenResponse {
  authenticated: boolean;
  mode: string;
  username?: string;
  role?: string;
}

export interface LoginResponse {
  user: AuthUser;
  must_change_password: boolean;
}

export type UserRole = "system_admin" | "service_admin" | "member" | "auditor";
export type UserStatus = "pending" | "active" | "disabled";
export type DeviceTokenScope = "inherit_all" | "restricted";

export interface AuthUser {
  ID: number;
  username: string;
  display_name: string;
  role: UserRole;
  status: UserStatus;
  must_change_password: boolean;
  last_login_at?: string;
}

export interface UserAccount extends AuthUser {
  CreatedAt?: string;
  UpdatedAt?: string;
}

export interface CreateUserResponse {
  user: UserAccount;
  initial_password: string;
}

export interface PermissionGroup {
  ID: number;
  name: string;
  display_name: string;
  description?: string;
  enabled: boolean;
  CreatedAt?: string;
  UpdatedAt?: string;
}

export interface PermissionGroupDetail {
  group: PermissionGroup;
  user_ids: number[];
  service_ids: number[];
}

export interface DeviceToken {
  ID: number;
  user_id: number;
  name: string;
  token_prefix: string;
  scope: DeviceTokenScope;
  expires_at: string;
  last_used_at?: string;
  last_used_ip?: string;
  client_info?: string;
  revoked_at?: string;
  CreatedAt?: string;
}

export interface CreateDeviceTokenResponse {
  device_token: DeviceToken;
  token: string;
}

export interface CallStat {
  username: string;
  server_name: string;
  date: string;
  count: number;
}

export interface CallStatsResponse {
  stats: CallStat[];
}
