import axios, { type AxiosError } from "axios";

export const http = axios.create({
  baseURL: "/api/dashboard",
  withCredentials: true,
  // Server registration/diagnostics can take a while (the backend probes
  // upstreams synchronously); 60s avoids cutting those off mid-flight.
  timeout: 60000,
});

export const accessHttp = axios.create({
  baseURL: "/api/v1",
  withCredentials: true,
  timeout: 60000,
});

function redirectOnExpiredSession(error: AxiosError) {
  const requestURL = error.config?.url ?? "";
  const credentialCheck =
    requestURL.endsWith("/auth/login") || requestURL.endsWith("/auth/password");
  if (error.response?.status === 401 && !credentialCheck) {
    window.dispatchEvent(new Event("mcpjungle:session-expired"));
  }
  return Promise.reject(error);
}

http.interceptors.response.use((resp) => resp, redirectOnExpiredSession);
accessHttp.interceptors.response.use((resp) => resp, redirectOnExpiredSession);

// Extract a human-readable message from an axios error, preferring the backend
// `{ error: "..." }` payload.
export function extractError(e: unknown): string {
  if (axios.isAxiosError(e)) {
    const data = e.response?.data as { error?: string } | undefined;
    return data?.error ?? e.message;
  }
  return e instanceof Error ? e.message : String(e);
}
