import axios, { type AxiosError } from "axios";
import { clearToken, getToken, isDevSession } from "../store/auth";

export const http = axios.create({
  baseURL: "/api/dashboard",
  // Server registration/diagnostics can take a while (the backend probes
  // upstreams synchronously); 60s avoids cutting those off mid-flight.
  timeout: 60000,
});

// Attach the access token (when not in a dev session) to every request.
http.interceptors.request.use((config) => {
  if (!isDevSession()) {
    const token = getToken();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
  }
  return config;
});

// On 401, clear the stored session and bounce to the login page.
http.interceptors.response.use(
  (resp) => resp,
  (error: AxiosError) => {
    if (error.response?.status === 401) {
      clearToken();
      if (!window.location.pathname.endsWith("/login")) {
        window.location.href = "/login";
      }
    }
    return Promise.reject(error);
  },
);

// Extract a human-readable message from an axios error, preferring the backend
// `{ error: "..." }` payload.
export function extractError(e: unknown): string {
  if (axios.isAxiosError(e)) {
    const data = e.response?.data as { error?: string } | undefined;
    return data?.error ?? e.message;
  }
  return e instanceof Error ? e.message : String(e);
}
