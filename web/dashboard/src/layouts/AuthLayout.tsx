import { Outlet } from "react-router-dom";

// Centered full-viewport wrapper used by the login page.
export default function AuthLayout() {
  return (
    <div className="login-page">
      <Outlet />
    </div>
  );
}
