import { http } from "./client";

// Permission group list item (returned by GET /permission-groups)
export interface PermissionGroup {
  id: number;
  created_at: string;
  updated_at: string;
  name: string;
  display_name: string;
  description: string;
  status: "active" | "disabled";
  created_by_id: number;
}

// Full detail returned by GET /permission-groups/:id
export interface PermissionGroupDetail {
  group: PermissionGroup;
  members: PermissionGroupMember[];
  services: PermissionGroupService[];
}

export interface PermissionGroupMember {
  id: number;
  created_at: string;
  updated_at: string;
  permission_group_id: number;
  user_id: number;
  assigned_by_id: number;
  assigned_at: string;
}

export interface PermissionGroupService {
  id: number;
  created_at: string;
  updated_at: string;
  permission_group_id: number;
  mcp_server_id: number;
  assigned_by_id: number;
  assigned_at: string;
}

export const permissionGroupsApi = {
  list: () =>
    http.get<PermissionGroup[]>("/permission-groups").then((r) => r.data),
  get: (id: number) =>
    http.get<PermissionGroupDetail>(`/permission-groups/${id}`).then((r) => r.data),
  create: (body: { name: string; display_name?: string; description?: string }) =>
    http.post<PermissionGroup>("/permission-groups", body).then((r) => r.data),
  update: (id: number, body: { display_name: string; description: string }) =>
    http.patch(`/permission-groups/${id}`, body).then((r) => r.data),
  disable: (id: number) =>
    http.post(`/permission-groups/${id}/disable`).then((r) => r.data),
  addMember: (id: number, user_id: number) =>
    http.post(`/permission-groups/${id}/members`, { user_id }).then((r) => r.data),
  removeMember: (id: number, user_id: number) =>
    http.delete(`/permission-groups/${id}/members/${user_id}`).then((r) => r.data),
  addService: (id: number, mcp_server_id: number) =>
    http.post(`/permission-groups/${id}/services`, { mcp_server_id }).then((r) => r.data),
  removeService: (id: number, server_id: number) =>
    http.delete(`/permission-groups/${id}/services/${server_id}`).then((r) => r.data),
};
