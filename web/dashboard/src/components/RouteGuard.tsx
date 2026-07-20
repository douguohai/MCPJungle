import type { ReactNode } from "react";
import { Navigate, useLocation } from "react-router-dom";
import { useAuth } from "../store/auth";
import type { UserRole } from "../types";

export default function RequireAuth({
  children,
  allowPasswordChange = false,
  requiredRole,
}: {
  children: ReactNode;
  allowPasswordChange?: boolean;
  requiredRole?: UserRole;
}) {
  const location = useLocation();
  const { user } = useAuth();
  if (!user) {
    return <Navigate to="/login" state={{ from: location }} replace />;
  }
  if (user.must_change_password && !allowPasswordChange) {
    return <Navigate to="/change-password" replace />;
  }
  if (requiredRole && user.role !== requiredRole) {
    return <Navigate to="/" replace />;
  }
  return <>{children}</>;
}
