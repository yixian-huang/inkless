export type AdminCapabilityStatus = "production" | "experimental";
export type AdminRoutePermission = string | string[] | null;

export interface AdminRouteAccess {
  path: string;
  permission: AdminRoutePermission;
  status: AdminCapabilityStatus;
}

export const ADMIN_DEFAULT_PATH = "/admin";
export const ADMIN_PAGES_PATH = "/admin/pages";

export const adminRouteAccess: AdminRouteAccess[] = [
  { path: "/admin/login", permission: null, status: "production" },
  { path: "/admin/pages", permission: "pages:read", status: "production" },
  { path: "/admin/articles/categories", permission: "categories:read", status: "production" },
  { path: "/admin/articles/tags", permission: "tags:read", status: "production" },
  { path: "/admin/articles", permission: "articles:read", status: "production" },
  { path: "/admin/media", permission: "media:read", status: "production" },
  { path: "/admin/form-submissions", permission: "form_submissions:read", status: "production" },
  { path: "/admin/menus", permission: "menus:read", status: "production" },
  { path: "/admin/scheduled-publications", permission: ["pages:publish", "articles:publish"], status: "production" },
  { path: "/admin/comments", permission: "comments:read", status: "production" },
  { path: "/admin/theme", permission: "themes:read", status: "production" },
  { path: "/admin/site-config", permission: "settings:manage", status: "production" },
  { path: "/admin/features", permission: "settings:manage", status: "production" },
  { path: "/admin/analytics", permission: "analytics:read", status: "production" },
  { path: "/admin/audit-logs", permission: "audit_logs:read", status: "production" },
  { path: "/admin/backups", permission: "backups:read", status: "production" },
  { path: "/admin/users", permission: "users:read", status: "production" },
  { path: "/admin/roles", permission: "roles:read", status: "production" },
  { path: "/admin/email-settings", permission: "settings:manage", status: "production" },
  { path: "/admin/ai-settings", permission: "settings:manage", status: "production" },
  { path: "/admin/storage", permission: "settings:manage", status: "production" },
  { path: "/admin/translation", permission: "settings:manage", status: "production" },
  { path: "/admin/wizard", permission: "pages:create", status: "production" },
  { path: "/admin/migration", permission: "system:manage", status: "production" },
  { path: "/admin/system-status", permission: "system:manage", status: "production" },
  { path: "/admin/qa", permission: "settings:manage", status: "experimental" },
  { path: "/admin", permission: "dashboard:read", status: "production" },
];

export function getAdminRouteAccess(pathname: string): AdminRouteAccess | null {
  return (
    adminRouteAccess
      .filter(({ path }) => {
        if (path === "/admin") {
          return pathname === path;
        }
        return pathname === path || pathname.startsWith(`${path}/`);
      })
      .sort((left, right) => right.path.length - left.path.length)[0] ?? null
  );
}

export function getAdminRoutePermission(pathname: string): AdminRoutePermission {
  return getAdminRouteAccess(pathname)?.permission ?? null;
}

export function hasAdminRoutePermission(
  requiredPermission: AdminRoutePermission,
  hasPermission: (permission: string) => boolean,
): boolean {
  if (!requiredPermission) return true;
  if (Array.isArray(requiredPermission)) {
    return requiredPermission.some((permission) => hasPermission(permission));
  }
  return hasPermission(requiredPermission);
}

export function isAdminRouteVisibleInNavigation(pathname: string): boolean {
  return getAdminRouteAccess(pathname)?.status !== "experimental";
}
