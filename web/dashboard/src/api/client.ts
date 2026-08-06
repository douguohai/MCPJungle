import axios, { type AxiosError } from "axios";
import { clearUser } from "../store/auth";

export const http = axios.create({
  baseURL: "/api/dashboard",
  // Server registration/diagnostics can take a while (the backend probes
  // upstreams synchronously); 60s avoids cutting those off mid-flight.
  timeout: 60000,
});

// On 401, clear the stored user info and bounce to the login page.
// Browser requests carry the session cookie automatically (SameSite=Lax);
// no Authorization header is needed.
http.interceptors.response.use(
  (resp) => resp,
  (error: AxiosError) => {
    if (error.response?.status === 401) {
      clearUser();
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
