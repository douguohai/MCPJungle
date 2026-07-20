import { accessHttp } from "./client";
import type { CreateUserResponse, UserAccount, UserRole } from "../types";

export const usersApi = {
  list: () =>
    accessHttp
      .get<{ users: UserAccount[] }>("/users")
      .then((response) => response.data.users ?? []),
  create: (body: { username: string; display_name?: string; role: UserRole }) =>
    accessHttp
      .post<CreateUserResponse>("/users", body)
      .then((response) => response.data),
  update: (
    id: number,
    body: { display_name?: string; role?: UserRole },
  ) => accessHttp.patch<{ user: UserAccount }>(`/users/${id}`, body),
  setEnabled: (id: number, enabled: boolean) =>
    accessHttp.post(`/users/${id}/${enabled ? "enable" : "disable"}`),
};
