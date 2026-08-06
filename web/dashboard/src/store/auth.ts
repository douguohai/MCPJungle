// DEV_MARKER is written when the backend is in development mode so the SPA can
// skip the login page without a real session.
const DEV_MARKER = "mcpjungle.dashboard.dev";
const USER_KEY = "mcpjungle.dashboard.user";

export function setDevSession(user: { username?: string; role?: string }): void {
  sessionStorage.setItem(DEV_MARKER, "1");
  sessionStorage.setItem(USER_KEY, JSON.stringify(user));
}

export function isDevSession(): boolean {
  return sessionStorage.getItem(DEV_MARKER) !== null;
}

export function clearDevSession(): void {
  sessionStorage.removeItem(DEV_MARKER);
  sessionStorage.removeItem(USER_KEY);
}

export function setUser(user: { username?: string; role?: string }): void {
  sessionStorage.setItem(USER_KEY, JSON.stringify(user));
}

export function getUser(): { username?: string; role?: string } | null {
  try {
    return JSON.parse(sessionStorage.getItem(USER_KEY) ?? "null");
  } catch {
    return null;
  }
}

export function clearUser(): void {
  sessionStorage.removeItem(USER_KEY);
  sessionStorage.removeItem(DEV_MARKER);
}
