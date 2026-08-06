import { useEffect, useState } from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import { Spin } from "antd";
import DashboardLayout from "./layouts/DashboardLayout";
import AuthLayout from "./layouts/AuthLayout";
import RequireAuth from "./components/RouteGuard";
import { authApi } from "./api/auth";
import { setDevSession, setUser } from "./store/auth";
import LoginPage from "./pages/LoginPage";
import OverviewPage from "./pages/OverviewPage";
import ServersPage from "./pages/ServersPage";
import ToolsPage from "./pages/ToolsPage";
import ToolGroupsPage from "./pages/ToolGroupsPage";
import PromptsPage from "./pages/PromptsPage";
import ResourcesPage from "./pages/ResourcesPage";
import DiagnosticsPage from "./pages/DiagnosticsPage";
import DeviceTokensPage from "./pages/DeviceTokensPage";
import UsersPage from "./pages/UsersPage";
import StatsPage from "./pages/StatsPage";

// On first load, probe the backend. In development mode it lets the dashboard
// in without a token; in enterprise mode the probe 401s and we show the login
// page. This lets dev users skip the login screen entirely.
function useBootstrap() {
  const [ready, setReady] = useState(false);
  useEffect(() => {
    authApi
      .verify()
      .then((res) => {
        if (res.authenticated) {
          if (res.mode === "development") {
            setDevSession({ username: res.username ?? "developer", role: res.role });
          } else {
            setUser({ username: res.username, role: res.role });
          }
        }
      })
      .catch(() => {
        // no valid session — will show login page via RouteGuard
      })
      .finally(() => setReady(true));
  }, []);
  return ready;
}

export default function App() {
  const ready = useBootstrap();
  if (!ready) {
    return (
      <div
        style={{
          display: "flex",
          justifyContent: "center",
          alignItems: "center",
          height: "100vh",
        }}
      >
        <Spin size="large" />
      </div>
    );
  }
  return (
    <Routes>
      <Route element={<AuthLayout />}>
        <Route path="/login" element={<LoginPage />} />
      </Route>
      <Route
        element={
          <RequireAuth>
            <DashboardLayout />
          </RequireAuth>
        }
      >
        <Route index element={<OverviewPage />} />
        <Route path="servers" element={<ServersPage />} />
        <Route path="tools" element={<ToolsPage />} />
        <Route path="tool-groups" element={<ToolGroupsPage />} />
        <Route path="prompts" element={<PromptsPage />} />
        <Route path="resources" element={<ResourcesPage />} />
        <Route path="diagnostics" element={<DiagnosticsPage />} />
        <Route path="device-tokens" element={<DeviceTokensPage />} />
        <Route path="users" element={<UsersPage />} />
        <Route path="stats" element={<StatsPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
