import { accessHttp, http } from "./client";
import type { AuthUser, LoginResponse, VerifyTokenResponse } from "../types";

export const authApi = {
  login: (username: string, password: string) =>
    accessHttp
      .post<LoginResponse>("/auth/login", { username, password })
      .then((r) => r.data),
  logout: () => accessHttp.post("/auth/logout"),
  me: () =>
    accessHttp
      .get<{ user: AuthUser }>("/auth/me")
      .then((r) => r.data.user),
  changePassword: (currentPassword: string, newPassword: string) =>
    accessHttp
      .put<{ user: AuthUser }>("/auth/password", {
        current_password: currentPassword,
        new_password: newPassword,
      })
      .then((r) => r.data.user),
  verifyDashboardMode: () =>
    http
      .post<VerifyTokenResponse>("/auth/verify", {})
      .then((r) => r.data),
};
