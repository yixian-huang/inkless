import { themeManager } from "./ThemeManager";
import type { ThemeLayoutChrome } from "./types";

export const BUILTIN_THEME_IDS = {
  CORPORATE_CLASSIC: "corporate-classic",
  BLOG_FIRST: "blog-first",
  MINIMAL_STARTER: "minimal-starter",
} as const;

export type BuiltinThemeId = (typeof BUILTIN_THEME_IDS)[keyof typeof BUILTIN_THEME_IDS];

/** Fallback when bootstrap has no active theme or activation fails. */
export const DEFAULT_FALLBACK_THEME_ID = BUILTIN_THEME_IDS.CORPORATE_CLASSIC;

/** Default theme for blank-site seed. */
export const BLANK_SITE_DEFAULT_THEME_ID = BUILTIN_THEME_IDS.BLOG_FIRST;

export function getFallbackLayoutChrome(): ThemeLayoutChrome | undefined {
  return themeManager.getTheme(DEFAULT_FALLBACK_THEME_ID)?.layoutChrome;
}
