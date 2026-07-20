import { Routes, Route, Navigate } from "react-router-dom";
import { Spin } from "antd";
import DashboardLayout from "./layouts/DashboardLayout";
import AuthLayout from "./layouts/AuthLayout";
import RequireAuth from "./components/RouteGuard";
import { useAuth } from "./store/auth";
import LoginPage from "./pages/LoginPage";
import OverviewPage from "./pages/OverviewPage";
import ServersPage from "./pages/ServersPage";
import ToolsPage from "./pages/ToolsPage";
import ToolGroupsPage from "./pages/ToolGroupsPage";
import PromptsPage from "./pages/PromptsPage";
import ResourcesPage from "./pages/ResourcesPage";
import DiagnosticsPage from "./pages/DiagnosticsPage";
import UsersPage from "./pages/UsersPage";
import StatsPage from "./pages/StatsPage";
import ChangePasswordPage from "./pages/ChangePasswordPage";
import DeviceTokensPage from "./pages/DeviceTokensPage";
import PermissionGroupsPage from "./pages/PermissionGroupsPage";

export default function App() {
  const { ready } = useAuth();
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
        <Route
          path="/change-password"
          element={
            <RequireAuth allowPasswordChange>
              <ChangePasswordPage />
            </RequireAuth>
          }
        />
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
        <Route
          path="users"
          element={
            <RequireAuth requiredRole="system_admin">
              <UsersPage />
            </RequireAuth>
          }
        />
        <Route
          path="permission-groups"
          element={
            <RequireAuth requiredRole="system_admin">
              <PermissionGroupsPage />
            </RequireAuth>
          }
        />
        <Route
          path="stats"
          element={
            <RequireAuth requiredRole="system_admin">
              <StatsPage />
            </RequireAuth>
          }
        />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
