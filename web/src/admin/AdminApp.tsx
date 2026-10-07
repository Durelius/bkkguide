import { useQuery, useQueryClient } from "@tanstack/react-query";
import { Navigate, NavLink, Outlet, Route, Routes, useLocation, useNavigate } from "react-router-dom";
import { Icon } from "../components/Icon";
import { Logo } from "../components/Logo";
import { AccountPage } from "./AccountPage";
import { AdminsPage } from "./AdminsPage";
import { adminApi, ApiError } from "./api";
import { AuditPage } from "./AuditPage";
import { LoginPage } from "./LoginPage";
import "./admin.css";

export function AdminApp() {
  return (
    <Routes>
      <Route path="login" element={<LoginPage />} />
      <Route element={<AdminShell />}>
        <Route index element={<Navigate to="admins" replace />} />
        <Route path="admins" element={<AdminsPage />} />
        <Route path="log" element={<AuditPage />} />
        <Route path="account" element={<AccountPage />} />
        <Route path="*" element={<Navigate to="admins" replace />} />
      </Route>
    </Routes>
  );
}

export function useMe() {
  return useQuery({ queryKey: ["admin", "me"], queryFn: adminApi.me, retry: false, staleTime: 5 * 60_000 });
}

function AdminShell() {
  const me = useMe();
  const location = useLocation();
  const navigate = useNavigate();
  const qc = useQueryClient();

  if (me.isLoading) return <p className="admin-loading">Checking your session…</p>;
  if (me.error instanceof ApiError && me.error.status === 401) {
    return <Navigate to={`/admin/login?next=${encodeURIComponent(location.pathname)}`} replace />;
  }
  if (!me.data) return <p className="admin-loading">The admin area couldn't load. Reload the page to try again.</p>;

  const logout = async () => {
    await adminApi.logout().catch(() => {});
    qc.removeQueries({ queryKey: ["admin"] });
    navigate("/admin/login", { replace: true });
  };

  return (
    <div className="admin">
      <aside className="admin__side">
        <a href="/" className="admin__brand" aria-label="Back to the guide">
          <Logo compact />
        </a>
        <nav className="admin__nav" aria-label="Admin">
          <NavLink to="/admin/admins" className="admin__navlink">
            <Icon name="user" /> Admins
          </NavLink>
          <NavLink to="/admin/log" className="admin__navlink">
            <Icon name="list" /> Activity log
          </NavLink>
          <NavLink to="/admin/account" className="admin__navlink">
            <Icon name="lock" /> Account
          </NavLink>
        </nav>
        <div className="admin__me">
          <span className="admin__me-name">{me.data.fullName}</span>
          <span className="admin__me-id">{me.data.studentId}</span>
          <button type="button" className="admin__logout" onClick={logout}>
            <Icon name="sign-out" /> Log out
          </button>
        </div>
      </aside>
      <main className="admin__main">
        <Outlet />
      </main>
    </div>
  );
}
