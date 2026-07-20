import { defaultTokens, type ThemeTokens } from "./tokens";

/**
 * Deep-merge published/custom tokens onto a theme base (usually
 * `activeTheme.defaultTokens`). Framework responsibility — themes only ship
 * their default token objects; they must not reimplement merge.
 */
export function mergeThemeTokens(
  base: ThemeTokens,
  overlay: Partial<ThemeTokens> | ThemeTokens | null | undefined,
): ThemeTokens {
  if (!overlay) return base;

  return {
    colors: { ...base.colors, ...overlay.colors },
    fonts: {
      ...base.fonts,
      ...overlay.fonts,
      mono: overlay.fonts?.mono ?? base.fonts.mono,
    },
    fontSources: {
      ...base.fontSources,
      ...overlay.fontSources,
    },
    typography: {
      article: {
        ...base.typography?.article,
        ...overlay.typography?.article,
      },
    },
    layout: { ...base.layout, ...overlay.layout },
  };
}

/** Merge overlay onto the global host fallback tokens (admin / no active theme). */
export function mergeWithHostDefaults(
  overlay: Partial<ThemeTokens> | ThemeTokens | null | undefined,
): ThemeTokens {
  return mergeThemeTokens(defaultTokens, overlay);
}
