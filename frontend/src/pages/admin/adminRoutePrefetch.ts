/**
 * Shared dynamic import map for admin routes.
 * Used by React.lazy route definitions and sidebar hover/focus prefetch.
 */

import type { ComponentType } from "react";

type ModuleDefault = { default: ComponentType<unknown> };
export type AdminRouteLoader = () => Promise<ModuleDefault>;

/** Paths resolved longest-prefix-first when prefetching nested URLs. */
export const adminRouteLoaders: Record<string, AdminRouteLoader> = {
  // Core pack (single shared chunk)
  "/admin": () =>
    import("./adminCorePages").then((m) => ({ default: m.AdminDashboardPage })),
  "/admin/articles": () =>
    import("./adminCorePages").then((m) => ({ default: m.AdminArticlesPage })),
  "/admin/pages": () =>
    import("./adminCorePages").then((m) => ({ default: m.AdminPagesPage })),
  "/admin/media": () =>
    import("./adminCorePages").then((m) => ({ default: m.AdminMediaPage })),
  "/admin/settings": () =>
    import("./adminCorePages").then((m) => ({ default: m.AdminSettingsPage })),

  // Secondary screens (individual chunks)
  "/admin/login": () => import("./login/page"),
  "/admin/analytics": () => import("./analytics/page"),
  "/admin/articles/new": () => import("./articles/editor/page"),
  "/admin/articles/edit": () => import("./articles/editor/page"),
  "/admin/articles/categories": () => import("./articles/categories/page"),
  "/admin/articles/tags": () => import("./articles/tags/page"),
  "/admin/audit-logs": () => import("./audit-logs/page"),
  "/admin/backups": () => import("./backups/page"),
  "/admin/pages/new": () => import("./pages/editor/page"),
  "/admin/pages/edit": () => import("./pages/editor/page"),
  "/admin/theme": () => import("./theme/page"),
  "/admin/form-submissions": () => import("./form-submissions/page"),
  "/admin/menus": () => import("./menus/page"),
  "/admin/scheduled-publications": () => import("./scheduled-publications/page"),
  "/admin/users": () => import("./users/page"),
  "/admin/ai-settings": () => import("./ai-settings/page"),
  "/admin/translation": () => import("./translation/page"),
  "/admin/qa": () => import("../../modules/qa/admin/page"),
  "/admin/wizard": () => import("./wizard/page"),
  "/admin/comments": () => import("../../modules/comment/admin/page"),
  "/admin/roles": () => import("./roles/page"),
  "/admin/storage": () => import("./storage/page"),
  "/admin/email-settings": () => import("./email-settings/page"),
  "/admin/site-config": () => import("./site-config/page"),
  "/admin/features": () => import("./features/page"),
  "/admin/migration": () => import("./migration/page"),
  "/admin/system-status": () => import("./system-status/page"),
};

const sortedLoaderPaths = Object.keys(adminRouteLoaders).sort(
  (a, b) => b.length - a.length,
);

const inflight = new Map<string, Promise<unknown>>();

export function resolveAdminLoaderPath(pathname: string): string | null {
  if (!pathname.startsWith("/admin")) return null;
  // Exact dashboard only — do not treat every /admin/* as dashboard
  if (pathname === "/admin" || pathname === "/admin/") {
    return "/admin";
  }
  for (const path of sortedLoaderPaths) {
    if (path === "/admin") continue;
    if (pathname === path || pathname.startsWith(`${path}/`)) {
      return path;
    }
  }
  return null;
}

/** Prefetch the JS module for an admin path (deduped; safe to call often). */
export function prefetchAdminRoute(pathname: string): void {
  const key = resolveAdminLoaderPath(pathname);
  if (!key) return;
  if (inflight.has(key)) return;
  const loader = adminRouteLoaders[key];
  if (!loader) return;
  const promise = loader().catch(() => {
    // Allow retry after a failed network prefetch
    inflight.delete(key);
  });
  inflight.set(key, promise);
}

/** Warm the shared core admin pack (dashboard / articles / pages / media / settings). */
export function prefetchAdminCorePack(): void {
  if (inflight.has("__core__")) return;
  const promise = import("./adminCorePages").catch(() => {
    inflight.delete("__core__");
  });
  inflight.set("__core__", promise);
  // Mark individual core routes as satisfied once the pack resolves
  promise.then(() => {
    for (const path of [
      "/admin",
      "/admin/articles",
      "/admin/pages",
      "/admin/media",
      "/admin/settings",
    ]) {
      inflight.set(path, promise);
    }
  });
}

/** Idle-time warm of common secondary menus (non-blocking). */
export function prefetchAdminSecondaryRoutes(): void {
  const secondary = [
    "/admin/site-config",
    "/admin/theme",
    "/admin/analytics",
    "/admin/menus",
    "/admin/form-submissions",
    "/admin/users",
    "/admin/roles",
  ];
  const run = () => {
    for (const path of secondary) {
      prefetchAdminRoute(path);
    }
  };
  if (typeof window !== "undefined" && "requestIdleCallback" in window) {
    window.requestIdleCallback(() => run(), { timeout: 2500 });
  } else {
    setTimeout(run, 800);
  }
}

/** Test helper: clear prefetch cache. */
export function __resetAdminPrefetchCacheForTests(): void {
  inflight.clear();
}
