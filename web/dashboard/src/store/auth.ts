const TOKEN_KEY = "mcpjungle.dashboard.token";
const USER_KEY = "mcpjungle.dashboard.user";

// DEV_MARKER is written when the backend is in development mode so the SPA can
// skip the login page without storing a real access token.
const DEV_MARKER = "mcpjungle.dashboard.dev";

// Tokens are kept in sessionStorage: they survive in-tab refresh but are
// dropped when the tab/browser closes, limiting exposure of the session JWT.
export function getToken(): string | null {
  return sessionStorage.getItem(TOKEN_KEY);
}

export function setToken(token: string): void {
  sessionStorage.setItem(TOKEN_KEY, token);
}

export function setDevSession(user: { username?: string; role?: string }): void {
  sessionStorage.setItem(TOKEN_KEY, DEV_MARKER);
  sessionStorage.setItem(USER_KEY, JSON.stringify(user));
}

export function isDevSession(): boolean {
  return sessionStorage.getItem(TOKEN_KEY) === DEV_MARKER;
}

export function clearToken(): void {
  sessionStorage.removeItem(TOKEN_KEY);
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
