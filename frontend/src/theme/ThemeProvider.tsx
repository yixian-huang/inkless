import { useEffect, useRef, useState, type ReactNode } from "react";
import { defaultTokens, type ThemeTokens } from "./tokens";
import { mergeThemeTokens } from "./mergeThemeTokens";
import { ThemeContext } from "./ThemeContext";
import { useThemeManager } from "@/plugins/hooks";
import { useBootstrap } from "@/contexts/BootstrapContext";
import { resolveSerifHeadingStack } from "./typography/resolve";

function applyTokens(tokens: ThemeTokens) {
  const root = document.documentElement;
  root.style.setProperty("--color-primary", tokens.colors.primary);
  root.style.setProperty("--color-primary-dark", tokens.colors.primaryDark);
  root.style.setProperty("--color-accent", tokens.colors.accent);
  root.style.setProperty("--color-accent-hover", tokens.colors.accentHover);
  root.style.setProperty("--color-surface", tokens.colors.surface);
  root.style.setProperty("--color-surface-alt", tokens.colors.surfaceAlt);
  root.style.setProperty("--color-on-primary", tokens.colors.onPrimary);
  root.style.setProperty("--color-on-surface", tokens.colors.onSurface);
  root.style.setProperty("--color-on-surface-muted", tokens.colors.onSurfaceMuted);
  root.style.setProperty("--color-border", tokens.colors.border);
  root.style.setProperty("--font-sans", tokens.fonts.sans);
  // CSS `font-heading` should track the editorial stack used by titles, not a
  // polluted publish that copied sans into fonts.heading.
  root.style.setProperty("--font-heading", resolveSerifHeadingStack(tokens));
  if (tokens.fonts.mono) {
    root.style.setProperty("--font-mono", tokens.fonts.mono);
  }
  root.style.setProperty("--layout-max-width", tokens.layout.maxWidth);
  root.style.setProperty("--radius-card", tokens.layout.borderRadius);
  root.style.setProperty("--radius-button", "0.375rem");
  root.style.setProperty("--layout-content-padding", tokens.layout.contentPadding);
  root.style.setProperty("--layout-section-spacing", tokens.layout.sectionSpacing);
  root.style.setProperty("--layout-content-gap", tokens.layout.contentGap);
}

interface ThemeProviderProps {
  children: ReactNode;
}

export function ThemeProvider({ children }: ThemeProviderProps) {
  const { activeTheme } = useThemeManager();
  const { data: bootstrapData, isLoading: bootstrapLoading } = useBootstrap();
  // Use activeThemeId as a stable dependency instead of the object reference
  const activeThemeId = activeTheme?.manifest?.id ?? null;
  const baseTokens = activeTheme?.defaultTokens ?? defaultTokens;

  const [tokens, setTokens] = useState<ThemeTokens>(defaultTokens);
  const [isLoading, setIsLoading] = useState(true);
  const prevThemeIdRef = useRef<string | null>(null);

  // Update base tokens when active theme actually changes (by id, not reference)
  useEffect(() => {
    if (prevThemeIdRef.current !== activeThemeId) {
      prevThemeIdRef.current = activeThemeId;
      applyTokens(baseTokens);
      setTokens(baseTokens);
    }
  }, [activeThemeId, baseTokens]);

  // Apply theme tokens from bootstrap data — merge onto active theme defaults
  // (not the global host fallback) so theme fontSources / typography survive.
  useEffect(() => {
    if (bootstrapLoading) return;

    const themeTokens = bootstrapData?.themeTokens as unknown as ThemeTokens | undefined;
    if (themeTokens) {
      // The bootstrap response returns raw token data; check if it has a valid structure
      if (themeTokens.colors && themeTokens.fonts && themeTokens.layout) {
        const merged = mergeThemeTokens(baseTokens, themeTokens);
        setTokens(merged);
        applyTokens(merged);
      } else {
        // API returned token data in a different shape — keep base tokens
        applyTokens(baseTokens);
      }
    } else {
      applyTokens(baseTokens);
    }
    setIsLoading(false);
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [bootstrapData, bootstrapLoading, activeThemeId]);

  // Re-apply whenever tokens change from outside (future live-update support)
  useEffect(() => {
    applyTokens(tokens);
  }, [tokens]);

  return (
    <ThemeContext.Provider value={{ tokens, isLoading }}>
      {children}
    </ThemeContext.Provider>
  );
}
