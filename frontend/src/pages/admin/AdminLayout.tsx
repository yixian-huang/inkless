import { useEffect, useState } from "react";
import { Outlet, Navigate, useLocation, useNavigate } from "react-router-dom";
import { useAuth } from "@/contexts/AuthContext";
import { useBranding } from "@/hooks/useBranding";
import { useSetupStatus } from "@/hooks/useSetupStatus";
import { BROWSER_STORAGE_KEYS } from "@/lib/browserStorage";
import {
  prefetchAdminCorePack,
  prefetchAdminSecondaryRoutes,
} from "@/pages/admin/adminRoutePrefetch";
import { isAdminEditorPath } from "@/pages/admin/nav/adminNav";
import { getAdminRoutePermission, hasAdminRoutePermission } from "@/router/adminAccess";
import AdminKeepAliveOutlet from "./components/AdminKeepAliveOutlet";
import AdminSidebar from "./components/AdminSidebar";
import AdminTopbar from "./components/AdminTopbar";

// Pull the high-frequency admin pack into the AdminLayout chunk graph so the
// first authenticated shell load already warms dashboard/articles/pages/media/settings.
void import("./adminCorePages").then(() => {
  prefetchAdminCorePack();
});

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

  useEffect(() => {
    if (!isAuthenticated) return;
    prefetchAdminCorePack();
    prefetchAdminSecondaryRoutes();
  }, [isAuthenticated]);

  const handleToggle = () => {
    setCollapsed((prev) => {
      const next = !prev;
      localStorage.setItem(BROWSER_STORAGE_KEYS.adminSidebarCollapsed, String(next));
      return next;
    });
  };

  if (isLoading || setupLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-[#f5f1ea]">
        <div className="flex flex-col items-center gap-3 text-[#8a8378]">
          <div className="h-9 w-9 animate-spin rounded-full border-2 border-[#e4ddd2] border-t-[#1a1814]" />
          <p className="text-sm font-medium tracking-wide">加载中…</p>
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
    <div className="admin-scope flex min-h-screen bg-[#f5f1ea]">
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
              : "admin-main flex-1 p-4 sm:p-6 lg:p-7"
          }
        >
          <AdminKeepAliveOutlet />
        </main>
      </div>
    </div>
  );
}
