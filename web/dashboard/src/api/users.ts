import { http } from "./client";
import type { UserListItem, CreateUserResponse } from "../types";

const v0 = { baseURL: "/api/v0" };

export const usersApi = {
  list: (signal?: AbortSignal) => http.get<UserListItem[]>("/users", { ...v0, signal }).then((r) => r.data),
  create: (body: { username: string; access_token?: string }) =>
    http.post<CreateUserResponse>("/users", body, v0).then((r) => r.data),
  update: (username: string, body: { allowed_servers?: string[] }) =>
    http.put(`/users/${encodeURIComponent(username)}`, body, v0).then((r) => r.data),
  remove: (username: string) => http.delete(`/users/${encodeURIComponent(username)}`, v0),
};
