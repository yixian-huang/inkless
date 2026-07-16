import type { PublicUnifiedPageItem } from "@/api/unifiedPages";
import type { ThemePageItem } from "@/api/themePages";
import type { ThemePageDefinition } from "@/plugins/types";

export interface PublicRoutingPage {
  id: number;
  slug: string;
  title: { zh?: string; en?: string };
  contentKey: string;
  renderMode: "hardcoded" | "dynamic";
  sortOrder: number;
  showInHeader: boolean;
  showInFooter: boolean;
  status: string;
}

interface ResolvePublicRoutingPagesInput {
  unifiedPages: PublicUnifiedPageItem[];
  themePages: ThemePageItem[];
  manifestPages: ThemePageDefinition[];
  activeThemeId?: string;
}

export interface AutomaticNavItem {
  label: string;
  path: string;
  sortOrder: number;
}

function themeRoutingPages(
  themePages: ThemePageItem[],
  manifestPages: ThemePageDefinition[],
  activeThemeId?: string,
): PublicRoutingPage[] {
  const publishedThemePages = themePages.filter(
    (page) => page.status === "published" && (!activeThemeId || page.themeId === activeThemeId),
  );
  if (publishedThemePages.length > 0) {
    return publishedThemePages.map((page) => ({
      id: page.id,
      slug: page.slug,
      title: page.title,
      contentKey: page.contentKey,
      renderMode: page.renderMode,
      sortOrder: page.sortOrder,
      showInHeader: page.navConfig?.showInHeader ?? false,
      showInFooter: page.navConfig?.showInFooter ?? false,
      status: page.status,
    }));
  }

  return manifestPages.map((page, index) => ({
    id: index,
    slug: page.slug,
    title: { zh: page.nav.labelZh, en: page.nav.label },
    contentKey: page.contentKey ?? page.slug,
    renderMode: page.renderMode,
    sortOrder: page.nav.order,
    showInHeader: page.nav.showInHeader ?? false,
    showInFooter: page.nav.showInFooter ?? false,
    status: "published",
  }));
}

/**
 * Published unified pages are the editable route truth. Theme definitions only
 * select a hardcoded component for a matching slug. During migration, legacy
 * theme routes fill only slugs that unified pages do not own yet.
 */
export function resolvePublicRoutingPages({
  unifiedPages,
  themePages,
  manifestPages,
  activeThemeId,
}: ResolvePublicRoutingPagesInput): PublicRoutingPage[] {
  const fallbackPages = themeRoutingPages(themePages, manifestPages, activeThemeId);
  const manifestBySlug = new Map(manifestPages.map((page) => [page.slug, page]));
  const publishedUnifiedPages = unifiedPages
    .filter((page) => page.status === "published")
    .map((page) => {
      const manifestPage = manifestBySlug.get(page.slug);
      return {
        id: page.id,
        slug: page.slug,
        title: page.title,
        contentKey: manifestPage?.contentKey ?? page.slug,
        renderMode: manifestPage?.renderMode ?? "dynamic",
        sortOrder: page.sortOrder,
        showInHeader: page.showInNav,
        showInFooter: page.showInNav,
        status: page.status,
      } satisfies PublicRoutingPage;
    });

  if (publishedUnifiedPages.length === 0) {
    return fallbackPages;
  }

  const unifiedSlugs = new Set(publishedUnifiedPages.map((page) => page.slug));
  return [
    ...publishedUnifiedPages,
    ...fallbackPages.filter((page) => !unifiedSlugs.has(page.slug)),
  ].sort((a, b) => a.sortOrder - b.sortOrder);
}

export function resolveAutomaticNavigation(
  unifiedPages: PublicUnifiedPageItem[],
  themePages: ThemePageItem[],
  locale: string,
  target: "header" | "footer",
): AutomaticNavItem[] {
  const publishedUnifiedPages = unifiedPages.filter((page) => page.status === "published");
  const unifiedSlugs = new Set(publishedUnifiedPages.map((page) => page.slug));
  const unifiedItems = publishedUnifiedPages
    .filter((page) => page.showInNav)
    .map((page) => ({
      label: (locale === "en" ? page.title.en : page.title.zh) || page.title.zh || page.slug,
      path: page.slug === "home" ? "/" : `/${page.slug}`,
      sortOrder: page.sortOrder,
    }));
  const themeItems = themePages
    .filter((page) =>
      page.status === "published" &&
      !unifiedSlugs.has(page.slug) &&
      (target === "header" ? page.navConfig?.showInHeader : page.navConfig?.showInFooter)
    )
    .map((page) => ({
      label: (locale === "en" ? page.title.en : page.title.zh) || page.title.zh || page.slug,
      path: page.slug === "home" ? "/" : `/${page.slug}`,
      sortOrder: page.sortOrder,
    }));

  return [...unifiedItems, ...themeItems].sort((a, b) => a.sortOrder - b.sortOrder);
}
