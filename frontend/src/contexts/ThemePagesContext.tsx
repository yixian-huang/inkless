import { createContext, useContext, useState, useEffect, useMemo, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import type { ThemePageItem } from "@/api/themePages";
import type { PublicUnifiedPageItem } from "@/api/unifiedPages";
import { useBootstrap } from "@/contexts/BootstrapContext";
import { getPublicMenu } from "@/api/menus";
import type { MenuGroup, MenuItem } from "@/api/menus";
import { resolveLocale } from "@/utils/locale";
import { resolveAutomaticNavigation } from "@/router/publicPages";
import { useThemeManager } from "@/plugins/hooks";

interface NavItem {
  label: string;
  path: string;
  sortOrder: number;
  /** Menu link target (_self / _blank / …). Omitted for automatic page nav. */
  target?: "_self" | "_parent" | "_blank" | "_top";
  children?: NavItem[];
}

interface ThemePagesContextValue {
  pages: ThemePageItem[];
  unifiedPages: PublicUnifiedPageItem[];
  headerNavItems: NavItem[];
  footerNavItems: NavItem[];
  menuNavItems: NavItem[];
  isLoading: boolean;
}

const ThemePagesContext = createContext<ThemePagesContextValue>({
  pages: [],
  unifiedPages: [],
  headerNavItems: [],
  footerNavItems: [],
  menuNavItems: [],
  isLoading: true,
});

 
export function useThemePages() {
  return useContext(ThemePagesContext);
}

const TYPE_PREFIX: Record<string, string> = {
  category: "/categories",
  tag: "/tags",
  article: "/blog",
  page: "",
};

function menuItemToNavItem(item: MenuItem, locale: string): NavItem | null {
  // Skip hidden items
  if (item.visible === false) return null;
  const label = locale === "en" && item.enName ? item.enName : item.zhName;
  let path: string;
  if (item.type === "custom_link") {
    path = item.url || "/";
  } else if (item.refSlug) {
    const prefix = TYPE_PREFIX[item.type] || "";
    path = `${prefix}/${item.refSlug}`;
  } else {
    path = item.url || "/";
  }
  const children = item.children
    ?.map((c) => menuItemToNavItem(c, locale))
    .filter((c): c is NavItem => c !== null);
  return {
    label,
    path,
    sortOrder: item.sortOrder,
    target: item.target,
    children: children?.length ? children : undefined,
  };
}

export function ThemePagesProvider({ children }: { children: ReactNode }) {
  const { data: bootstrapData, isLoading: bootstrapLoading } = useBootstrap();
  const { activeTheme } = useThemeManager();
  const { i18n } = useTranslation("common");
  const [menuGroup, setMenuGroup] = useState<MenuGroup | null>(null);
  const manifestPages = useMemo(() => activeTheme?.pages ?? [], [activeTheme]);

  useEffect(() => {
    getPublicMenu().then((g) => setMenuGroup(g)).catch(() => {});
  }, []);

  const pages = useMemo(() => bootstrapData?.themePages ?? [], [bootstrapData]);
  const unifiedPages = useMemo(
    () => bootstrapData?.unifiedPages ?? [],
    [bootstrapData],
  );
  const isLoading = bootstrapLoading;
  const locale = resolveLocale(i18n.language);

  const menuNavItems = useMemo(() => {
    if (!menuGroup?.items?.length) return [];
    // Public API already returns tree-structured items (children nested under parents)
    return menuGroup.items
      .map((item) => menuItemToNavItem(item, locale))
      .filter((item): item is NavItem => item !== null);
  }, [menuGroup, locale]);

  const headerNavItems = useMemo(() => {
    return resolveAutomaticNavigation(unifiedPages, pages, locale, "header", manifestPages);
  }, [pages, unifiedPages, locale, manifestPages]);

  const footerNavItems = useMemo(() => {
    return resolveAutomaticNavigation(unifiedPages, pages, locale, "footer", manifestPages);
  }, [pages, unifiedPages, locale, manifestPages]);

  const value = useMemo(() => ({
    pages,
    unifiedPages,
    headerNavItems,
    footerNavItems,
    menuNavItems,
    isLoading,
  }), [pages, unifiedPages, headerNavItems, footerNavItems, menuNavItems, isLoading]);

  return (
    <ThemePagesContext.Provider value={value}>
      {children}
    </ThemePagesContext.Provider>
  );
}
