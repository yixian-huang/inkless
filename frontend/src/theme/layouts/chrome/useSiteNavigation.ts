import { useMemo } from "react";
import { useGlobalConfig } from "@/contexts/GlobalConfigContext";
import { useThemePages } from "@/contexts/ThemePagesContext";
import { isFeatureEnabled, routeFeatureMap } from "@/router/featureMap";
import type { NavItem } from "@/theme/layouts/types";

export interface SiteNavItem {
  label?: string;
  path?: string;
  children?: SiteNavItem[];
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
    const featureKey = routeFeatureMap[path];
    if (featureKey && !isFeatureEnabled(features, featureKey)) {
      continue;
    }
    result.push({
      label: item.label,
      path: item.path,
      children: children?.length ? children : undefined,
    });
  }
  return result;
}

export function selectSiteNavigation(
  menuNavItems: SiteNavItem[],
  headerNavItems: SiteNavItem[],
  configNavigation: NavItem[] | undefined,
  legacyNavigation: Array<{ label?: string; href?: string }>,
): SiteNavItem[] {
  if (menuNavItems.length > 0) {
    return menuNavItems.map((item) => ({
      label: item.label,
      path: item.path,
      children: item.children,
    }));
  }
  if (headerNavItems.length > 0) {
    return headerNavItems.map((item) => ({
      label: item.label,
      path: item.path,
    }));
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

/** Resolve public header navigation: menu > theme pages > layout override > legacy global nav. */
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
