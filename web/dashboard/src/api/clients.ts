import { http } from "./client";
import type { McpClient, CreateClientInput } from "../types";

// MCP clients live under /api/v0 (admin API). The dashboard reuses its session
// JWT by overriding the baseURL per request.
const v0 = { baseURL: "/api/v0" };

export const clientsApi = {
  list: (signal?: AbortSignal) => http.get<McpClient[]>("/clients", { ...v0, signal }).then((r) => r.data),
  create: (body: CreateClientInput) =>
    http.post<McpClient>("/clients", body, v0).then((r) => r.data),
  update: (name: string, body: Partial<CreateClientInput>) =>
    http.put<McpClient>(`/clients/${encodeURIComponent(name)}`, body, v0).then((r) => r.data),
  remove: (name: string) => http.delete(`/clients/${encodeURIComponent(name)}`, v0),
};
