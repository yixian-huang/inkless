/**
 * High-frequency admin screens packed into one shared chunk.
 *
 * Loaded with AdminLayout (via side-effect import) so the first visit to
 * /admin already has dashboard / articles / pages / media / settings ready.
 * Route config still uses lazy(() => import("./adminCorePages")) so the
 * public SPA entry does not pull these modules.
 */
export { default as AdminDashboardPage } from "./dashboard/page";
export { default as AdminArticlesPage } from "./articles/page";
export { default as AdminPagesPage } from "./pages/page";
export { default as AdminMediaPage } from "./media/page";
export { default as AdminSettingsPage } from "./settings/page";
