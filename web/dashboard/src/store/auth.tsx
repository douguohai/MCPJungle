import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { authApi } from "../api/auth";
import type { AuthUser } from "../types";

interface AuthState {
  ready: boolean;
  mode: string;
  user: AuthUser | null;
  setUser: (user: AuthUser | null) => void;
  refresh: () => Promise<void>;
}

const AuthContext = createContext<AuthState | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
  const [ready, setReady] = useState(false);
  const [mode, setMode] = useState("");
  const [user, setUser] = useState<AuthUser | null>(null);

  const refresh = useCallback(async () => {
    try {
      const account = await authApi.me();
      setUser(account);
      setMode("enterprise");
    } catch {
      try {
        const dashboard = await authApi.verifyDashboardMode();
        if (dashboard.authenticated && dashboard.mode === "development") {
          setMode(dashboard.mode);
          setUser({
            ID: 0,
            username: dashboard.username ?? "developer",
            display_name: "开发模式",
            role: "system_admin",
            status: "active",
            must_change_password: false,
          });
          return;
        }
      } catch {
        // No active browser session. The route guard will show the login page.
      }
      setUser(null);
    } finally {
      setReady(true);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    const clearExpiredSession = () => setUser(null);
    window.addEventListener("mcpjungle:session-expired", clearExpiredSession);
    return () =>
      window.removeEventListener("mcpjungle:session-expired", clearExpiredSession);
  }, []);

  const value = useMemo(
    () => ({ ready, mode, user, setUser, refresh }),
    [mode, ready, refresh, user],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const value = useContext(AuthContext);
  if (!value) throw new Error("useAuth must be used inside AuthProvider");
  return value;
}
