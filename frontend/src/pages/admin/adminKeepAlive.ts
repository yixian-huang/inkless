/** Max number of list/settings screens kept mounted. */
export const ADMIN_KEEP_ALIVE_MAX = 8;

/** Local copy to avoid importing adminNav (circular: AuthContext → keepAlive → nav). */
function isEditorPath(pathname: string): boolean {
  return (
    pathname === "/admin/pages/new" ||
    pathname.startsWith("/admin/pages/edit/") ||
    pathname === "/admin/articles/new" ||
    pathname.startsWith("/admin/articles/edit/")
  );
}

/**
 * Routes eligible for keep-alive (exact path match).
 * Editors and one-shot tools are excluded — they are heavy or ephemeral.
 */
export const ADMIN_KEEP_ALIVE_PATHS = new Set<string>([
  "/admin",
  "/admin/articles",
  "/admin/articles/categories",
  "/admin/articles/tags",
  "/admin/pages",
  "/admin/media",
  "/admin/settings",
  "/admin/analytics",
  "/admin/form-submissions",
  "/admin/menus",
  "/admin/scheduled-publications",
  "/admin/users",
  "/admin/roles",
  "/admin/comments",
  "/admin/audit-logs",
  "/admin/theme",
  "/admin/site-config",
  "/admin/features",
  "/admin/backups",
  "/admin/email-settings",
  "/admin/ai-settings",
  "/admin/storage",
  "/admin/api-keys",
  "/admin/translation",
  "/admin/system-status",
]);

/** Normalize pathname and return keep-alive cache key, or null if not cached. */
export function resolveKeepAliveKey(pathname: string): string | null {
  if (!pathname.startsWith("/admin")) return null;
  if (pathname === "/admin/login") return null;
  if (isEditorPath(pathname)) return null;

  const normalized =
    pathname.length > 1 && pathname.endsWith("/") ? pathname.slice(0, -1) : pathname;

  if (ADMIN_KEEP_ALIVE_PATHS.has(normalized)) {
    return normalized;
  }
  return null;
}

export function isKeepAlivePath(pathname: string): boolean {
  return resolveKeepAliveKey(pathname) !== null;
}

/** Apply LRU: put key first, drop overflow keys. Returns keys that were evicted. */
export function touchKeepAliveLru(
  order: string[],
  key: string,
  max: number = ADMIN_KEEP_ALIVE_MAX,
): { order: string[]; evicted: string[] } {
  const without = order.filter((k) => k !== key);
  const next = [key, ...without];
  if (next.length <= max) {
    return { order: next, evicted: [] };
  }
  const kept = next.slice(0, max);
  const evicted = next.slice(max);
  return { order: kept, evicted };
}

type ClearListener = () => void;
const clearListeners = new Set<ClearListener>();

/** Drop all keep-alive panes (e.g. on logout). */
export function clearAdminKeepAlive(): void {
  for (const listener of clearListeners) {
    listener();
  }
}

export function onAdminKeepAliveClear(listener: ClearListener): () => void {
  clearListeners.add(listener);
  return () => {
    clearListeners.delete(listener);
  };
}
