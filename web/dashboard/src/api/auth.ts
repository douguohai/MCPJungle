import { http } from "./client";
import type { LoginResponse, VerifyTokenResponse } from "../types";

export const authApi = {
  // login exchanges username+password for a short-lived session JWT.
  login: (username: string, password: string) =>
    http
      .post<LoginResponse>("/auth/login", { username, password })
      .then((r) => r.data),

  // verify checks whether a token is still valid. Used by App bootstrap to
  // detect dev mode (no token required) and kept for legacy-token compatibility.
  verify: (accessToken: string) =>
    http
      .post<VerifyTokenResponse>("/auth/verify", { access_token: accessToken })
      .then((r) => r.data),
};
