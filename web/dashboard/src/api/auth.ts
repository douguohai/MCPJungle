import { http } from "./client";
import type { LoginResponse, VerifyTokenResponse } from "../types";

export const authApi = {
  // login exchanges username+password for a short-lived session JWT.
  login: (username: string, password: string) =>
    http
      .post<LoginResponse>("/auth/login", { username, password })
      .then((r) => r.data),

  // verify checks whether the current session is still valid. Dev mode always
  // succeeds; enterprise mode reads the session cookie automatically.
  verify: () =>
    http
      .post<VerifyTokenResponse>("/auth/verify")
      .then((r) => r.data),
};
