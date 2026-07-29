import { useMemo } from "react";
import { useGlobalConfig } from "@/contexts/GlobalConfigContext";
import { useThemePages } from "@/contexts/ThemePagesContext";
import { isFeatureEnabled, routeFeatureMap } from "@/router/featureMap";
import type { NavItem } from "@/theme/layouts/types";

export interface SiteNavItem {
  label?: string;
  path?: string;
  /** Menu link target (_self / _blank / …). */
  target?: "_self" | "_parent" | "_blank" | "_top";
  children?: SiteNavItem[];
}

/** True when path is an absolute external URL (not an in-app route). */
export function isExternalNavPath(path?: string): boolean {
  return !!path && /^(https?:|mailto:|tel:)/i.test(path);
}

/** Normalize path for de-dupe (strip trailing slash except root). */
export function normalizeNavPath(path?: string): string {
  if (!path) return "";
  const p = path.trim();
  if (isExternalNavPath(p)) return p;
  if (p === "/") return "/";
  return p.replace(/\/+$/, "") || "/";
}

function filterByFeatures(
  items: SiteNavItem[],
  features: ReturnType<typeof useGlobalConfig>["features"],
): SiteNavItem[] {
  const result: SiteNavItem[] = [];
  for (const item of items) {
    const children = item.children?.length
      ? filterByFeatures(item.children, features)
      : undefined;
    const path = item.path || "/";
    // External URLs are never gated by in-app feature flags.
    if (!isExternalNavPath(path)) {
      const featureKey = routeFeatureMap[path];
      if (featureKey && !isFeatureEnabled(features, featureKey)) {
        continue;
      }
    }
    result.push({
      label: item.label,
      path: item.path,
      target: item.target,
      children: children?.length ? children : undefined,
    });
  }
  return result;
}

/**
 * Merge primary menu with theme/automatic page nav.
 * - Menu order & labels win for matching paths (operator intent).
 * - Theme/auto pages whose paths are not in the menu are **appended**
 *   so new theme IA (e.g. get-started) is not hidden by a stale menu.
 * - Menu-only items (external docs, custom links) stay.
 */
export function mergeMenuAndThemeNav(
  menuNavItems: SiteNavItem[],
  themeNavItems: SiteNavItem[],
): SiteNavItem[] {
  if (menuNavItems.length === 0) {
    return themeNavItems.map((item) => ({
      label: item.label,
      path: item.path,
      target: item.target,
      children: item.children,
    }));
  }
  if (themeNavItems.length === 0) {
    return menuNavItems.map((item) => ({
      label: item.label,
      path: item.path,
      target: item.target,
      children: item.children,
    }));
  }

  const seen = new Set<string>();
  const out: SiteNavItem[] = [];

  for (const item of menuNavItems) {
    const key = normalizeNavPath(item.path);
    if (key) seen.add(key);
    out.push({
      label: item.label,
      path: item.path,
      target: item.target,
      children: item.children,
    });
  }

  for (const item of themeNavItems) {
    const key = normalizeNavPath(item.path);
    if (!key || seen.has(key)) continue;
    seen.add(key);
    out.push({
      label: item.label,
      path: item.path,
      target: item.target,
    });
  }

  return out;
}

export function selectSiteNavigation(
  menuNavItems: SiteNavItem[],
  headerNavItems: SiteNavItem[],
  configNavigation: NavItem[] | undefined,
  legacyNavigation: Array<{ label?: string; href?: string }>,
): SiteNavItem[] {
  // Menu + theme pages merge (neither alone replaces the other).
  if (menuNavItems.length > 0 || headerNavItems.length > 0) {
    return mergeMenuAndThemeNav(menuNavItems, headerNavItems);
  }
  if (configNavigation?.length) {
    return configNavigation.map((item) => ({
      label: item.label,
      path: item.path,
      children: item.children?.map((child) => ({
        label: child.label,
        path: child.path,
        children: child.children,
      })),
    }));
  }
  return legacyNavigation.map((item) => ({
    label: item.label,
    path: item.href,
  }));
}

/** Resolve public header navigation: merge menu + theme pages, then config/legacy. */
export function useSiteNavigation(configNavigation?: NavItem[]): SiteNavItem[] {
  const { config: globalConfig, features } = useGlobalConfig();
  const { headerNavItems, menuNavItems } = useThemePages();

  return useMemo(() => {
    const navigation = selectSiteNavigation(
      menuNavItems,
      headerNavItems,
      configNavigation,
      globalConfig.nav?.items || [],
    );
    return filterByFeatures(navigation, features);
  }, [configNavigation, menuNavItems, headerNavItems, globalConfig.nav?.items, features]);
}
