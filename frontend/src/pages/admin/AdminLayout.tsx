import { Suspense, useState } from "react";
import { Outlet, Navigate, useLocation, useNavigate } from "react-router-dom";
import { AdminRouteFallback } from "@/components/admin/ui";
import { useAuth } from "@/contexts/AuthContext";
import { useBranding } from "@/hooks/useBranding";
import { useSetupStatus } from "@/hooks/useSetupStatus";
import { BROWSER_STORAGE_KEYS } from "@/lib/browserStorage";
import { isAdminEditorPath } from "@/pages/admin/nav/adminNav";
import { getAdminRoutePermission, hasAdminRoutePermission } from "@/router/adminAccess";
import AdminSidebar from "./components/AdminSidebar";
import AdminTopbar from "./components/AdminTopbar";

export default function AdminLayout() {
  const location = useLocation();
  const navigate = useNavigate();
  const { isAuthenticated, isLoading, logout, hasPermission } = useAuth();
  const { status: setupStatus, loading: setupLoading } = useSetupStatus();
  const branding = useBranding();

  const [collapsed, setCollapsed] = useState(
    () => localStorage.getItem(BROWSER_STORAGE_KEYS.adminSidebarCollapsed) === "true",
  );
  const [mobileOpen, setMobileOpen] = useState(false);

  const handleToggle = () => {
    setCollapsed((prev) => {
      const next = !prev;
      localStorage.setItem(BROWSER_STORAGE_KEYS.adminSidebarCollapsed, String(next));
      return next;
    });
  };

  if (isLoading || setupLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-slate-50">
        <div className="flex flex-col items-center gap-3 text-slate-500">
          <div className="h-8 w-8 animate-spin rounded-full border-2 border-slate-200 border-t-blue-600" />
          <p className="text-sm">加载中…</p>
        </div>
      </div>
    );
  }

  if (setupStatus && !setupStatus.installed) {
    return <Navigate to="/setup" replace />;
  }

  if (!isAuthenticated && location.pathname !== "/admin/login") {
    return <Navigate to="/admin/login" replace />;
  }

  if (location.pathname === "/admin/login") {
    return <Outlet />;
  }

  const requiredPerm = getAdminRoutePermission(location.pathname);
  if (!hasAdminRoutePermission(requiredPerm, hasPermission)) {
    return <Navigate to="/admin" replace />;
  }

  const handleLogout = async () => {
    await logout();
    navigate("/admin/login");
  };

  const flush = isAdminEditorPath(location.pathname);

  return (
    <div className="flex min-h-screen bg-slate-50">
      <AdminSidebar
        collapsed={collapsed}
        onToggle={handleToggle}
        mobileOpen={mobileOpen}
        onMobileClose={() => setMobileOpen(false)}
      />

      <div
        className={`flex min-w-0 flex-1 flex-col transition-all duration-200 ${
          collapsed ? "md:ml-16" : "md:ml-64"
        }`}
      >
        <AdminTopbar
          pathname={location.pathname}
          siteName={branding.siteName}
          onOpenMobileMenu={() => setMobileOpen(true)}
          onLogout={handleLogout}
        />

        <main
          className={
            flush
              ? "flex min-h-0 flex-1 flex-col overflow-hidden"
              : "flex-1 p-4 sm:p-6"
          }
        >
          <Suspense fallback={<AdminRouteFallback />}>
            <Outlet />
          </Suspense>
        </main>
      </div>
    </div>
  );
}
