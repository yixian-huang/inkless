import { createContext, useContext, useState, useEffect, useMemo, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import type { ThemePageItem } from "@/api/themePages";
import { useBootstrap } from "@/contexts/BootstrapContext";
import { getPublicMenu } from "@/api/menus";
import type { MenuGroup, MenuItem } from "@/api/menus";

interface NavItem {
  label: string;
  path: string;
  sortOrder: number;
  children?: NavItem[];
}

interface ThemePagesContextValue {
  pages: ThemePageItem[];
  headerNavItems: NavItem[];
  footerNavItems: NavItem[];
  menuNavItems: NavItem[];
  isLoading: boolean;
}

const ThemePagesContext = createContext<ThemePagesContextValue>({
  pages: [],
  headerNavItems: [],
  footerNavItems: [],
  menuNavItems: [],
  isLoading: true,
});

// eslint-disable-next-line react-refresh/only-export-components
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
  return { label, path, sortOrder: item.sortOrder, children: children?.length ? children : undefined };
}

export function ThemePagesProvider({ children }: { children: ReactNode }) {
  const { data: bootstrapData, isLoading: bootstrapLoading } = useBootstrap();
  const { i18n } = useTranslation("common");
  const [menuGroup, setMenuGroup] = useState<MenuGroup | null>(null);

  useEffect(() => {
    getPublicMenu().then((g) => setMenuGroup(g)).catch(() => {});
  }, []);

  const pages = useMemo(() => bootstrapData?.themePages ?? [], [bootstrapData]);
  const isLoading = bootstrapLoading;
  const locale = i18n.language === "en" ? "en" : "zh";

  const menuNavItems = useMemo(() => {
    if (!menuGroup?.items?.length) return [];
    // Public API already returns tree-structured items (children nested under parents)
    return menuGroup.items
      .map((item) => menuItemToNavItem(item, locale))
      .filter((item): item is NavItem => item !== null);
  }, [menuGroup, locale]);

  const headerNavItems = useMemo(() => {
    return pages
      .filter((p) => p.status === "published" && p.navConfig?.showInHeader)
      .sort((a, b) => a.sortOrder - b.sortOrder)
      .map((p) => ({
        label: (locale === "en" ? p.title.en : p.title.zh) || p.title.zh || p.slug,
        path: p.slug === "home" ? "/" : `/${p.slug}`,
        sortOrder: p.sortOrder,
      }));
  }, [pages, locale]);

  const footerNavItems = useMemo(() => {
    return pages
      .filter((p) => p.status === "published" && p.navConfig?.showInFooter)
      .sort((a, b) => a.sortOrder - b.sortOrder)
      .map((p) => ({
        label: (locale === "en" ? p.title.en : p.title.zh) || p.title.zh || p.slug,
        path: p.slug === "home" ? "/" : `/${p.slug}`,
        sortOrder: p.sortOrder,
      }));
  }, [pages, locale]);

  const value = useMemo(() => ({
    pages,
    headerNavItems,
    footerNavItems,
    menuNavItems,
    isLoading,
  }), [pages, headerNavItems, footerNavItems, menuNavItems, isLoading]);

  return (
    <ThemePagesContext.Provider value={value}>
      {children}
    </ThemePagesContext.Provider>
  );
}
