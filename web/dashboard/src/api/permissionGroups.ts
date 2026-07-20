import { accessHttp } from "./client";
import type { PermissionGroup, PermissionGroupDetail } from "../types";

export interface CreatePermissionGroupInput {
  name: string;
  display_name: string;
  description?: string;
}

export const permissionGroupsApi = {
  list: () =>
    accessHttp
      .get<{ permission_groups: PermissionGroup[] }>("/permission-groups")
      .then((response) => response.data.permission_groups ?? []),
  get: (id: number) =>
    accessHttp
      .get<PermissionGroupDetail>(`/permission-groups/${id}`)
      .then((response) => response.data),
  create: (body: CreatePermissionGroupInput) =>
    accessHttp
      .post<{ permission_group: PermissionGroup }>("/permission-groups", body)
      .then((response) => response.data.permission_group),
  update: (
    id: number,
    body: { display_name?: string; description?: string },
  ) => accessHttp.patch(`/permission-groups/${id}`, body),
  replaceUsers: (id: number, ids: number[]) =>
    accessHttp.put(`/permission-groups/${id}/users`, { ids }),
  replaceServices: (id: number, ids: number[]) =>
    accessHttp.put(`/permission-groups/${id}/services`, { ids }),
  setEnabled: (id: number, enabled: boolean) =>
    accessHttp.post(`/permission-groups/${id}/${enabled ? "enable" : "disable"}`),
};
