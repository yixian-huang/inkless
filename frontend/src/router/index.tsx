import { useNavigate, type RouteObject } from "react-router-dom";
import { useRoutes } from "react-router-dom";
import { useEffect, useMemo } from "react";
import type { ReactElement } from "react";
import { staticRoutes } from "./config";
import { resolveNavigate } from "./navigate";
import { useThemeManager } from "@/plugins/hooks";
import { useThemePages } from "@/contexts/ThemePagesContext";
import { useBootstrap } from "@/contexts/BootstrapContext";
import ThemePageWrapper from "@/plugins/ThemePageWrapper";
import type { ThemePageDefinition } from "@/plugins/types";
import { FeatureGate } from "@/components/feature/FeatureGate";
import { routeFeatureMap } from "@/router/featureMap";
import { resolvePublicRoutingPages } from "./publicPages";

declare global {
  interface Window {
    REACT_APP_NAVIGATE: ReturnType<typeof useNavigate>;
  }
}

export function AppRoutes() {
  const { activeTheme, isLoading: themeLoading } = useThemeManager();
  const {
    pages: themePages,
    unifiedPages,
    isLoading: themePagesLoading,
  } = useThemePages();
  const { data: bootstrapData } = useBootstrap();
  const navigate = useNavigate();
  const activeThemeId = bootstrapData?.activeTheme?.themeId ?? activeTheme?.manifest.id;

  useEffect(() => {
    window.REACT_APP_NAVIGATE = navigate;
    resolveNavigate(navigate);
  }, [navigate]);

  // Build contentKey → ThemePageDefinition lookup from the active theme's pages array
  const componentMap = useMemo(() => {
    const map = new Map<string, ThemePageDefinition>();
    for (const pageDef of activeTheme?.pages || []) {
      if (pageDef.contentKey) {
        map.set(pageDef.contentKey, pageDef);
      }
    }
    return map;
  }, [activeTheme]);

  const routingPages = useMemo(
    () => resolvePublicRoutingPages({
      unifiedPages,
      themePages,
      manifestPages: activeTheme?.pages ?? [],
      activeThemeId,
    }),
    [unifiedPages, themePages, activeTheme, activeThemeId],
  );

  const wrapWithFeatureGate = (path: string, element: ReactElement) => {
    const key = routeFeatureMap[path];
    if (!key) return element;
    return <FeatureGate feature={key}>{element}</FeatureGate>;
  };

  const routes = useMemo(() => {
    const themeRoutes: RouteObject[] = routingPages.map((page) => {
      const themeDef = componentMap.get(page.contentKey);
      const pageDef: ThemePageDefinition = themeDef || {
        slug: page.slug,
        renderMode: page.renderMode as "hardcoded" | "dynamic",
        contentKey: page.contentKey,
        nav: { label: page.title.en || page.slug, labelZh: page.title.zh || page.slug, order: page.sortOrder },
      };

      const fullPath = page.slug === "home" ? "/" : `/${page.slug}`;
      return {
        path: fullPath,
        element: wrapWithFeatureGate(fullPath, <ThemePageWrapper pageDef={pageDef} />),
      };
    });
    return [...themeRoutes, ...staticRoutes];
  }, [routingPages, componentMap]);

  const element = useRoutes(routes);

  if (themeLoading || themePagesLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center">
        <div className="text-gray-400 animate-pulse">加载中...</div>
      </div>
    );
  }

  return element;
}
