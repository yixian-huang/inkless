import { lazy } from 'react';
import { RouteObject } from 'react-router-dom';
import { FeatureGate } from '@/components/feature/FeatureGate';
import { withSiteLayout } from '@/plugins/withSiteLayout';

// Admin routes
const AdminLayout = lazy(() => import('../pages/admin/AdminLayout'));
const AdminLoginPage = lazy(() => import('../pages/admin/login/page'));
// AdminContentPage and AdminContentEditorPage removed — replaced by unified page editor
const AdminMediaPage = lazy(() => import('../pages/admin/media/page'));
const AdminAnalyticsPage = lazy(() => import('../pages/admin/analytics/page'));
const AdminArticlesPage = lazy(() => import('../pages/admin/articles/page'));
const AdminArticleEditorPage = lazy(() => import('../pages/admin/articles/editor/page'));
const AdminCategoriesPage = lazy(() => import('../pages/admin/articles/categories/page'));
const AdminTagsPage = lazy(() => import('../pages/admin/articles/tags/page'));
const AdminAuditLogsPage = lazy(() => import('../pages/admin/audit-logs/page'));
const AdminBackupsPage = lazy(() => import('../pages/admin/backups/page'));
const AdminPagesPage = lazy(() => import('../pages/admin/pages/page'));
const AdminPageEditorPage = lazy(() => import('../pages/admin/pages/editor/page'));
const AdminThemePage = lazy(() => import('../pages/admin/theme/page'));
const AdminDashboardPage = lazy(() => import('../pages/admin/dashboard/page'));
const AdminFormSubmissionsPage = lazy(() => import('../pages/admin/form-submissions/page'));
const AdminMenusPage = lazy(() => import('../pages/admin/menus/page'));
const AdminScheduledPublicationsPage = lazy(() => import('../pages/admin/scheduled-publications/page'));
const AdminUsersPage = lazy(() => import('../pages/admin/users/page'));
const AdminAISettingsPage = lazy(() => import('../pages/admin/ai-settings/page'));
const AdminTranslationPage = lazy(() => import('../pages/admin/translation/page'));
const AdminQAPage = lazy(() => import('../modules/qa/admin/page'));
const AdminWizardPage = lazy(() => import('../pages/admin/wizard/page'));
const AdminRolesPage = lazy(() => import('../pages/admin/roles/page'));
const AdminStoragePage = lazy(() => import('../pages/admin/storage/page'));
const AdminEmailSettingsPage = lazy(() => import('../pages/admin/email-settings/page'));
const AdminSiteConfigPage = lazy(() => import('../pages/admin/site-config/page'));
const AdminFeaturesPage = lazy(() => import('../pages/admin/features/page'));
const AdminMigrationPage = lazy(() => import('../pages/admin/migration/page'));
const AdminSystemStatusPage = lazy(() => import('../pages/admin/system-status/page'));
import { commentModuleConfig } from '@/modules/comment';

const AdminCommentsPage = lazy(() => import('../modules/comment/admin/page'));
const SetupPage = lazy(() => import('../pages/setup/page'));
const NotFound = lazy(() => import('../pages/NotFound'));

// Public blog routes
const BlogPage = lazy(() => import('../pages/blog/page'));
const BlogDetailPage = lazy(() => import('../pages/blog/[slug]/page'));

// Public category & tag routes
const CategoriesPage = lazy(() => import('../pages/categories/page'));
const CategoryDetailPage = lazy(() => import('../pages/categories/[slug]/page'));
const TagsPage = lazy(() => import('../pages/tags/page'));
const TagDetailPage = lazy(() => import('../pages/tags/[slug]/page'));

// Dynamic page (section-based rendering)
const DynamicPage = lazy(() => import('../theme/DynamicPage'));

/**
 * Static routes — blog, admin, dynamic CMS pages, 404.
 * Theme-driven public page routes (/, /about, etc.) are generated
 * dynamically from activeTheme.pages in AppRoutes.
 */
export const staticRoutes: RouteObject[] = [
  {
    path: '/setup',
    element: <SetupPage />,
  },
  {
    path: '/blog',
    element: withSiteLayout(
      <FeatureGate feature="blog">
        <BlogPage />
      </FeatureGate>,
    ),
  },
  {
    path: '/blog/:slug',
    element: withSiteLayout(
      <FeatureGate feature="blog">
        <BlogDetailPage />
      </FeatureGate>,
    ),
  },
  {
    path: '/categories',
    element: withSiteLayout(<CategoriesPage />),
  },
  {
    path: '/categories/:slug',
    element: withSiteLayout(<CategoryDetailPage />),
  },
  {
    path: '/tags',
    element: withSiteLayout(<TagsPage />),
  },
  {
    path: '/tags/:slug',
    element: withSiteLayout(<TagDetailPage />),
  },
  {
    path: '/admin',
    element: <AdminLayout />,
    children: [
      { index: true, element: <AdminDashboardPage /> },
      {
        path: 'login',
        element: <AdminLoginPage />,
      },
      // Old content editor routes removed — replaced by unified page editor at /admin/pages
      {
        path: 'media',
        element: <AdminMediaPage />,
      },
      {
        path: 'analytics',
        element: <AdminAnalyticsPage />,
      },
      {
        path: 'articles',
        element: <AdminArticlesPage />,
      },
      {
        path: 'articles/new',
        element: <AdminArticleEditorPage />,
      },
      {
        path: 'articles/edit/:id',
        element: <AdminArticleEditorPage />,
      },
      {
        path: 'articles/categories',
        element: <AdminCategoriesPage />,
      },
      {
        path: 'articles/tags',
        element: <AdminTagsPage />,
      },
      {
        path: 'audit-logs',
        element: <AdminAuditLogsPage />,
      },
      {
        path: 'backups',
        element: <AdminBackupsPage />,
      },
      {
        path: 'pages',
        element: <AdminPagesPage />,
      },
      {
        path: 'pages/new',
        element: <AdminPageEditorPage />,
      },
      {
        path: 'pages/edit/:id',
        element: <AdminPageEditorPage />,
      },
      {
        path: 'theme',
        element: <AdminThemePage />,
      },
      {
        path: 'form-submissions',
        element: <AdminFormSubmissionsPage />,
      },
      {
        path: 'menus',
        element: <AdminMenusPage />,
      },
      {
        path: 'scheduled-publications',
        element: <AdminScheduledPublicationsPage />,
      },
      {
        path: 'users',
        element: <AdminUsersPage />,
      },
      {
        path: 'ai-settings',
        element: <AdminAISettingsPage />,
      },
      {
        path: 'translation',
        element: <AdminTranslationPage />,
      },
      {
        path: 'qa',
        element: <AdminQAPage />,
      },
      {
        path: 'wizard',
        element: <AdminWizardPage />,
      },
      {
        path: commentModuleConfig.adminRoute.path,
        element: <AdminCommentsPage />,
      },
      {
        path: 'roles',
        element: <AdminRolesPage />,
      },
      {
        path: 'storage',
        element: <AdminStoragePage />,
      },
      {
        path: 'email-settings',
        element: <AdminEmailSettingsPage />,
      },
      {
        path: 'site-config',
        element: <AdminSiteConfigPage />,
      },
      {
        path: 'features',
        element: <AdminFeaturesPage />,
      },
      {
        path: 'migration',
        element: <AdminMigrationPage />,
      },
      {
        path: 'system-status',
        element: <AdminSystemStatusPage />,
      },
    ],
  },
  {
    path: '/search',
    lazy: () => import('@/pages/search/page').then(m => ({ Component: m.default })),
  },
  {
    path: '/p/*',
    element: withSiteLayout(<DynamicPage />),
  },
  {
    path: '*',
    element: <NotFound />,
  },
];
