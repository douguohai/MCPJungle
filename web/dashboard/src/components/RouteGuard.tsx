import { useEffect, useState, type ReactNode } from "react";
import { Navigate, useLocation } from "react-router-dom";
import { authApi } from "../api/auth";
import { getUser, setUser, clearUser } from "../store/auth";

// Redirects to /login when no valid session is present. On first render, if a
// cached user exists the guard trusts it immediately; otherwise it calls
// /auth/verify asynchronously (the browser sends the session cookie
// automatically). A dev-mode marker set by App's bootstrap also counts as a
// valid session.
export default function RequireAuth({ children }: { children: ReactNode }) {
  const location = useLocation();
  const [ready, setReady] = useState(!!getUser());
  const [authenticated, setAuthenticated] = useState(!!getUser());

  useEffect(() => {
    if (getUser()) {
      setAuthenticated(true);
      setReady(true);
      return;
    }
    authApi
      .verify()
      .then((res) => {
        if (res.authenticated) {
          setUser({ username: res.username, role: res.role });
          setAuthenticated(true);
        }
      })
      .catch(() => {
        clearUser();
        setAuthenticated(false);
      })
      .finally(() => setReady(true));
  }, []);

  if (!ready) {
    return null; // loading — avoids flash of login page while verify is in-flight
  }
  if (!authenticated) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }
  return <>{children}</>;
}
