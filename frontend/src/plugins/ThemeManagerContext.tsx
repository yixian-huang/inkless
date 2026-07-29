import { useEffect, useMemo, useState, useSyncExternalStore, type ReactNode } from "react";
import { themeManager } from "./ThemeManager";
import { ThemeManagerContext } from "./ThemeManagerContextDef";
import { corporateClassicTheme } from "./themes/corporate-classic";
import { blogFirstTheme } from "./themes/blog-first";
import { productFirstTheme } from "./themes/product-first";
import { minimalStarterTheme } from "./themes/minimal-starter";
import { editorialFirmTheme } from "./themes/editorial-firm";
import { useBootstrap } from "@/contexts/BootstrapContext";
import { DEFAULT_FALLBACK_THEME_ID } from "@/plugins/builtinThemes";
import { isRemoteThemeSource } from "@/plugins/themeSource";
import "@/plugins/externals";

export type { ThemeManagerContextValue } from "./ThemeManagerContextDef";
export { ThemeManagerContext } from "./ThemeManagerContextDef";

// Register built-in themes immediately
themeManager.registerBuiltIn(corporateClassicTheme);
themeManager.registerBuiltIn(blogFirstTheme);
themeManager.registerBuiltIn(productFirstTheme);
themeManager.registerBuiltIn(minimalStarterTheme);
themeManager.registerBuiltIn(editorialFirmTheme);

interface ThemeManagerProviderProps {
  children: ReactNode;
}

export function ThemeManagerProvider({ children }: ThemeManagerProviderProps) {
  const snapshot = useSyncExternalStore(
    themeManager.subscribe,
    themeManager.getSnapshot,
  );
  const [isLoading, setIsLoading] = useState(true);
  const { data: bootstrapData, isLoading: bootstrapLoading } = useBootstrap();

  useEffect(() => {
    if (bootstrapLoading) return;

    let cancelled = false;
    setIsLoading(true);

    async function init() {
      try {
        const activeTheme = bootstrapData?.activeTheme;
        const themeId = activeTheme?.themeId;
        const source = activeTheme?.source;
        const externalUrl = activeTheme?.externalUrl;

        if (!themeId) {
          themeManager.activate(DEFAULT_FALLBACK_THEME_ID);
          return;
        }

        // Remote themes (URL install or official marketplace) load UMD before activate.
        if (isRemoteThemeSource(source) && externalUrl) {
          try {
            await themeManager.loadExternal(externalUrl);
          } catch {
            // Fall through to activate(); may still hit built-in same themeId, else default fallback.
          }
        }

        if (cancelled) return;

        // Activate the theme; fallback to corporate-classic
        if (!themeManager.activate(themeId)) {
          themeManager.activate(DEFAULT_FALLBACK_THEME_ID);
        }
      } catch {
        themeManager.activate(DEFAULT_FALLBACK_THEME_ID);
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    }

    init();
    return () => { cancelled = true; };
  }, [bootstrapData, bootstrapLoading]);

  const contextValue = useMemo(() => ({
    manager: themeManager,
    ...snapshot,
    isLoading,
  }), [snapshot, isLoading]);

  return (
    <ThemeManagerContext.Provider value={contextValue}>
      {children}
    </ThemeManagerContext.Provider>
  );
}
