import { useState } from "react";
import { Outlet, Navigate, useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "@/contexts/AuthContext";
import AdminSidebar from "./components/AdminSidebar";

// Map admin route prefixes to permission keys
const routePermissions: Record<string, string> = {
  "/admin/content": "content",
  "/admin/pages": "pages",
  "/admin/articles": "articles",
  "/admin/media": "media",
  "/admin/form-submissions": "form-submissions",
  "/admin/menus": "menus",
  "/admin/theme": "theme",
  "/admin/analytics": "analytics",
  "/admin/audit-logs": "audit-logs",
  "/admin/backups": "backups",
  "/admin/users": "users",
};

function getRequiredPermission(pathname: string): string | null {
  for (const [prefix, perm] of Object.entries(routePermissions)) {
    if (pathname === prefix || pathname.startsWith(prefix + "/")) {
      return perm;
    }
  }
  return null; // No permission required (e.g. /admin dashboard, /admin/login)
}

export default function AdminLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, isAuthenticated, isLoading, logout, hasPermission } = useAuth();

  const [collapsed, setCollapsed] = useState(() =>
    localStorage.getItem("admin_sidebar_collapsed") === "true"
  );
  const [mobileOpen, setMobileOpen] = useState(false);

  const handleToggle = () => {
    setCollapsed((prev) => {
      const next = !prev;
      localStorage.setItem("admin_sidebar_collapsed", String(next));
      return next;
    });
  };

  // Show loading state while checking auth
  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-100">
        <div className="text-gray-600">加载中...</div>
      </div>
    );
  }

  // Redirect to login if not authenticated
  if (!isAuthenticated && location.pathname !== "/admin/login") {
    return <Navigate to="/admin/login" replace />;
  }

  // Show login page without navigation frame
  if (location.pathname === "/admin/login") {
    return <Outlet />;
  }

  // Check route-level permission
  const requiredPerm = getRequiredPermission(location.pathname);
  if (requiredPerm && !hasPermission(requiredPerm)) {
    return <Navigate to="/admin" replace />;
  }

  const handleLogout = async () => {
    await logout();
    navigate("/admin/login");
  };

  const roleBadge = user?.isSuperAdmin
    ? "超级管理员"
    : user?.role === "admin"
      ? "管理员"
      : "编辑";

  return (
    <div className="min-h-screen bg-gray-100 flex">
      <AdminSidebar
        collapsed={collapsed}
        onToggle={handleToggle}
        mobileOpen={mobileOpen}
        onMobileClose={() => setMobileOpen(false)}
      />

      {/* Main Area */}
      <div
        className={`flex-1 flex flex-col transition-all duration-200 ${
          collapsed ? "md:ml-16" : "md:ml-64"
        }`}
      >
        {/* Top Header Bar */}
        <header className="sticky top-0 z-10 bg-white shadow-sm h-14 flex items-center justify-between px-6">
          {/* Left: mobile menu button */}
          <div className="flex items-center gap-3">
            <button
              onClick={() => setMobileOpen(true)}
              className="md:hidden p-1.5 rounded-md text-gray-500 hover:bg-gray-100 hover:text-gray-700"
              aria-label="打开菜单"
            >
              <svg className="w-6 h-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M4 6h16M4 12h16M4 18h16" />
              </svg>
            </button>
            <span className="text-sm font-medium text-gray-600 hidden sm:block">
              印迹官网 - 管理后台
            </span>
          </div>

          {/* Right: user info + logout */}
          <div className="flex items-center gap-3">
            <span className="text-sm text-gray-600">
              {user?.username || "管理员"}
            </span>
            <span className={`text-xs px-2 py-0.5 rounded ${
              user?.isSuperAdmin
                ? "text-amber-700 bg-amber-100"
                : "text-gray-500 bg-gray-100"
            }`}>
              {roleBadge}
            </span>
            <button
              onClick={handleLogout}
              className="px-3 py-1.5 text-sm font-medium text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-md transition-colors"
            >
              退出
            </button>
          </div>
        </header>

        {/* Main Content */}
        <main className="flex-1 p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
