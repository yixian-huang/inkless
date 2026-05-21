import { useNavigate, type RouteObject } from "react-router-dom";
import { useRoutes } from "react-router-dom";
import { useEffect, useMemo } from "react";
import type { ReactElement } from "react";
import { staticRoutes } from "./config";
import { resolveNavigate } from "./navigate";
import { useThemeManager } from "@/plugins/hooks";
import { useThemePages } from "@/contexts/ThemePagesContext";
import ThemePageWrapper from "@/plugins/ThemePageWrapper";
import type { ThemePageDefinition } from "@/plugins/types";
import { FeatureGate } from "@/components/feature/FeatureGate";
import { routeFeatureMap } from "@/router/featureMap";

declare global {
  interface Window {
    REACT_APP_NAVIGATE: ReturnType<typeof useNavigate>;
  }
}

export function AppRoutes() {
  const { activeTheme, isLoading: themeLoading } = useThemeManager();
  const { pages: themePages, isLoading: themePagesLoading } = useThemePages();
  const navigate = useNavigate();

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

  const wrapWithFeatureGate = (path: string, element: ReactElement) => {
    const key = routeFeatureMap[path];
    if (!key) return element;
    return <FeatureGate feature={key}>{element}</FeatureGate>;
  };

  const routes = useMemo(() => {
    // If we have backend-driven theme pages, use them for routing
    if (themePages.length > 0) {
      const backendRoutes: RouteObject[] = themePages
        .filter((p) => p.status === "published")
        .map((page) => {
          // For hardcoded pages, look up the component from the theme's pages array
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
      return [...backendRoutes, ...staticRoutes];
    }

    // Fallback: use theme's hardcoded pages directly (before backend data loads)
    const themeRoutes: RouteObject[] = (activeTheme?.pages || []).map((pageDef) => {
      const fullPath = pageDef.slug === "home" ? "/" : `/${pageDef.slug}`;
      return {
        path: fullPath,
        element: wrapWithFeatureGate(fullPath, <ThemePageWrapper pageDef={pageDef} />),
      };
    });
    return [...themeRoutes, ...staticRoutes];
  }, [activeTheme, themePages, componentMap]);

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
