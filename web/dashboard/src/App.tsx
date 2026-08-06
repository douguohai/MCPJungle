import { Suspense, lazy, useEffect, useState } from "react";
import { Routes, Route, Navigate } from "react-router-dom";
import { Spin } from "antd";
import DashboardLayout from "./layouts/DashboardLayout";
import AuthLayout from "./layouts/AuthLayout";
import RequireAuth from "./components/RouteGuard";
import { authApi } from "./api/auth";
import { setDevSession, setUser } from "./store/auth";
import LoginPage from "./pages/LoginPage";

// Code-split all dashboard pages (design doc perf optimization).
const OverviewPage = lazy(() => import("./pages/OverviewPage"));
const ServersPage = lazy(() => import("./pages/ServersPage"));
const ToolsPage = lazy(() => import("./pages/ToolsPage"));
const ToolCollectionsPage = lazy(() => import("./pages/ToolCollectionsPage"));
const PromptsPage = lazy(() => import("./pages/PromptsPage"));
const ResourcesPage = lazy(() => import("./pages/ResourcesPage"));
const DiagnosticsPage = lazy(() => import("./pages/DiagnosticsPage"));
const DeviceTokensPage = lazy(() => import("./pages/DeviceTokensPage"));
const UsersPage = lazy(() => import("./pages/UsersPage"));
const StatsPage = lazy(() => import("./pages/StatsPage"));
const MyServicesPage = lazy(() => import("./pages/MyServicesPage"));
const MyStatsPage = lazy(() => import("./pages/MyStatsPage"));
const PermissionGroupsPage = lazy(() => import("./pages/PermissionGroupsPage"));
const AuditPage = lazy(() => import("./pages/AuditPage"));

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
        <Route index element={<Suspense fallback={<Spin size="large" />}><OverviewPage /></Suspense>} />
        <Route path="servers" element={<Suspense fallback={<Spin size="large" />}><ServersPage /></Suspense>} />
        <Route path="tools" element={<Suspense fallback={<Spin size="large" />}><ToolsPage /></Suspense>} />
        <Route path="tool-collections" element={<Suspense fallback={<Spin size="large" />}><ToolCollectionsPage /></Suspense>} />
        <Route path="prompts" element={<Suspense fallback={<Spin size="large" />}><PromptsPage /></Suspense>} />
        <Route path="resources" element={<Suspense fallback={<Spin size="large" />}><ResourcesPage /></Suspense>} />
        <Route path="diagnostics" element={<Suspense fallback={<Spin size="large" />}><DiagnosticsPage /></Suspense>} />
        <Route path="device-tokens" element={<Suspense fallback={<Spin size="large" />}><DeviceTokensPage /></Suspense>} />
        <Route path="users" element={<Suspense fallback={<Spin size="large" />}><UsersPage /></Suspense>} />
        <Route path="stats" element={<Suspense fallback={<Spin size="large" />}><StatsPage /></Suspense>} />
        <Route path="my-services" element={<Suspense fallback={<Spin size="large" />}><MyServicesPage /></Suspense>} />
        <Route path="my-stats" element={<Suspense fallback={<Spin size="large" />}><MyStatsPage /></Suspense>} />
        <Route path="permission-groups" element={<Suspense fallback={<Spin size="large" />}><PermissionGroupsPage /></Suspense>} />
        <Route path="audit" element={<Suspense fallback={<Spin size="large" />}><AuditPage /></Suspense>} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
