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
  token: string;
  expires_at: string;
  user: { username: string; role: string };
}

export interface DashboardUser {
  username?: string;
  role?: string;
}

export interface McpClient {
  name: string;
  description: string;
  user_id?: number;
  access_token?: string;
  is_custom_access_token?: boolean;
  allow_list: string[];
}

export interface CreateClientInput {
  name: string;
  description?: string;
  allow_list: string[];
}

export interface CreateUserResponse {
  username: string;
  role: string;
  access_token: string;
}

export interface UserListItem {
  username: string;
  role: string;
  allowed_servers?: string[] | null;
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
