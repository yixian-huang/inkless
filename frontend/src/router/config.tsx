import { lazy } from "react";
import { RouteObject } from "react-router-dom";
import { FeatureGate } from "@/components/feature/FeatureGate";
import { withSiteLayout } from "@/plugins/withSiteLayout";
import { commentModuleConfig } from "@/modules/comment";
import { adminRouteLoaders } from "@/pages/admin/adminRoutePrefetch";

// Admin shell stays lazy so public pages do not download the admin bundle.
const AdminLayout = lazy(() => import("../pages/admin/AdminLayout"));

// Core + secondary admin pages share the same loaders as sidebar prefetch
// so hover-prefetched modules resolve instantly on navigation.
const AdminLoginPage = lazy(adminRouteLoaders["/admin/login"]);
const AdminMediaPage = lazy(adminRouteLoaders["/admin/media"]);
const AdminAnalyticsPage = lazy(adminRouteLoaders["/admin/analytics"]);
const AdminArticlesPage = lazy(adminRouteLoaders["/admin/articles"]);
const AdminArticleEditorPage = lazy(adminRouteLoaders["/admin/articles/edit"]);
const AdminCategoriesPage = lazy(adminRouteLoaders["/admin/articles/categories"]);
const AdminTagsPage = lazy(adminRouteLoaders["/admin/articles/tags"]);
const AdminAuditLogsPage = lazy(adminRouteLoaders["/admin/audit-logs"]);
const AdminBackupsPage = lazy(adminRouteLoaders["/admin/backups"]);
const AdminPagesPage = lazy(adminRouteLoaders["/admin/pages"]);
const AdminPageEditorPage = lazy(adminRouteLoaders["/admin/pages/edit"]);
const AdminThemePage = lazy(adminRouteLoaders["/admin/theme"]);
const AdminDashboardPage = lazy(adminRouteLoaders["/admin"]);
const AdminFormSubmissionsPage = lazy(adminRouteLoaders["/admin/form-submissions"]);
const AdminMenusPage = lazy(adminRouteLoaders["/admin/menus"]);
const AdminScheduledPublicationsPage = lazy(
  adminRouteLoaders["/admin/scheduled-publications"],
);
const AdminUsersPage = lazy(adminRouteLoaders["/admin/users"]);
const AdminAISettingsPage = lazy(adminRouteLoaders["/admin/ai-settings"]);
const AdminTranslationPage = lazy(adminRouteLoaders["/admin/translation"]);
const AdminQAPage = lazy(adminRouteLoaders["/admin/qa"]);
const AdminWizardPage = lazy(adminRouteLoaders["/admin/wizard"]);
const AdminRolesPage = lazy(adminRouteLoaders["/admin/roles"]);
const AdminStoragePage = lazy(adminRouteLoaders["/admin/storage"]);
const AdminAPIKeysPage = lazy(adminRouteLoaders["/admin/api-keys"]);
const AdminEmailSettingsPage = lazy(adminRouteLoaders["/admin/email-settings"]);
const AdminSiteConfigPage = lazy(adminRouteLoaders["/admin/site-config"]);
const AdminFeaturesPage = lazy(adminRouteLoaders["/admin/features"]);
const AdminMigrationPage = lazy(adminRouteLoaders["/admin/migration"]);
const AdminSystemStatusPage = lazy(adminRouteLoaders["/admin/system-status"]);
const AdminSettingsPage = lazy(adminRouteLoaders["/admin/settings"]);
const AdminCommentsPage = lazy(adminRouteLoaders["/admin/comments"]);

const SetupPage = lazy(() => import("../pages/setup/page"));
const NotFound = lazy(() => import("../pages/NotFound"));

// Public blog routes
const BlogPage = lazy(() => import("../pages/blog/page"));
const BlogDetailPage = lazy(() => import("../pages/blog/[slug]/page"));

// Public category & tag routes
const CategoriesPage = lazy(() => import("../pages/categories/page"));
const CategoryDetailPage = lazy(() => import("../pages/categories/[slug]/page"));
const TagsPage = lazy(() => import("../pages/tags/page"));
const TagDetailPage = lazy(() => import("../pages/tags/[slug]/page"));

// Dynamic page (section-based rendering)
const DynamicPage = lazy(() => import("../theme/DynamicPage"));

/**
 * Static routes — blog, admin, dynamic CMS pages, 404.
 * Theme-driven public page routes (/, /about, etc.) are generated
 * dynamically from activeTheme.pages in AppRoutes.
 */
export const staticRoutes: RouteObject[] = [
  {
    path: "/setup",
    element: <SetupPage />,
  },
  {
    path: "/blog",
    element: withSiteLayout(
      <FeatureGate feature="blog">
        <BlogPage />
      </FeatureGate>,
    ),
  },
  {
    path: "/blog/:slug",
    element: withSiteLayout(
      <FeatureGate feature="blog">
        <BlogDetailPage />
      </FeatureGate>,
    ),
  },
  {
    path: "/categories",
    element: withSiteLayout(<CategoriesPage />),
  },
  {
    path: "/categories/:slug",
    element: withSiteLayout(<CategoryDetailPage />),
  },
  {
    path: "/tags",
    element: withSiteLayout(<TagsPage />),
  },
  {
    path: "/tags/:slug",
    element: withSiteLayout(<TagDetailPage />),
  },
  {
    path: "/admin",
    element: <AdminLayout />,
    children: [
      { index: true, element: <AdminDashboardPage /> },
      {
        path: "login",
        element: <AdminLoginPage />,
      },
      {
        path: "media",
        element: <AdminMediaPage />,
      },
      {
        path: "analytics",
        element: <AdminAnalyticsPage />,
      },
      {
        path: "articles",
        element: <AdminArticlesPage />,
      },
      {
        path: "articles/new",
        element: <AdminArticleEditorPage />,
      },
      {
        path: "articles/edit/:id",
        element: <AdminArticleEditorPage />,
      },
      {
        path: "articles/categories",
        element: <AdminCategoriesPage />,
      },
      {
        path: "articles/tags",
        element: <AdminTagsPage />,
      },
      {
        path: "audit-logs",
        element: <AdminAuditLogsPage />,
      },
      {
        path: "backups",
        element: <AdminBackupsPage />,
      },
      {
        path: "pages",
        element: <AdminPagesPage />,
      },
      {
        path: "pages/new",
        element: <AdminPageEditorPage />,
      },
      {
        path: "pages/edit/:id",
        element: <AdminPageEditorPage />,
      },
      {
        path: "theme",
        element: <AdminThemePage />,
      },
      {
        path: "form-submissions",
        element: <AdminFormSubmissionsPage />,
      },
      {
        path: "menus",
        element: <AdminMenusPage />,
      },
      {
        path: "scheduled-publications",
        element: <AdminScheduledPublicationsPage />,
      },
      {
        path: "users",
        element: <AdminUsersPage />,
      },
      {
        path: "ai-settings",
        element: <AdminAISettingsPage />,
      },
      {
        path: "translation",
        element: <AdminTranslationPage />,
      },
      {
        path: "qa",
        element: <AdminQAPage />,
      },
      {
        path: "wizard",
        element: <AdminWizardPage />,
      },
      {
        path: commentModuleConfig.adminRoute.path,
        element: <AdminCommentsPage />,
      },
      {
        path: "roles",
        element: <AdminRolesPage />,
      },
      {
        path: "storage",
        element: <AdminStoragePage />,
      },
      {
        path: "api-keys",
        element: <AdminAPIKeysPage />,
      },
      {
        path: "email-settings",
        element: <AdminEmailSettingsPage />,
      },
      {
        path: "site-config",
        element: <AdminSiteConfigPage />,
      },
      {
        path: "features",
        element: <AdminFeaturesPage />,
      },
      {
        path: "migration",
        element: <AdminMigrationPage />,
      },
      {
        path: "system-status",
        element: <AdminSystemStatusPage />,
      },
      {
        path: "settings",
        element: <AdminSettingsPage />,
      },
    ],
  },
  {
    path: "/search",
    lazy: () => import("@/pages/search/page").then((m) => ({ Component: m.default })),
  },
  {
    path: "/p/*",
    element: withSiteLayout(<DynamicPage />),
  },
  {
    path: "*",
    element: <NotFound />,
  },
];
