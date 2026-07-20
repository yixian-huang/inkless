/**
 * Frozen inventory of `@inkless/theme-host` public surface.
 *
 * Rules:
 * - Value exports MUST appear on `import * as host from "./index"` at runtime.
 * - Type-only exports are compile-time documentation (erased at runtime).
 * - Adding a value export: update this list + docs/theme-contract.md + tests.
 * - Removing/renaming a value export: bump THEME_CONTRACT_VERSION.
 */

/** Runtime value exports (components, hooks, constants, helpers). */
export const THEME_HOST_VALUE_EXPORTS = [
  // Contract lock
  "THEME_CONTRACT_VERSION",
  "THEME_CONTRACT_SUPPORTED",
  "THEME_HOST_VALUE_EXPORTS",
  "THEME_HOST_TYPE_EXPORTS",
  "normalizeThemeContractVersion",
  "isThemeContractCompatible",
  "resolveThemeContractVersion",
  "assertThemeContractCompatible",
  // Layout / chrome
  "BLOG_DEFAULT_LAYOUT",
  "BaseSiteHeader",
  "BrandMark",
  "HeaderUtilities",
  "useHeaderSettings",
  // Hooks
  "useBranding",
  "useContentMaxWidth",
  "useIsReadingLayout",
  "useIsThemeHomePath",
  "useGlobalConfig",
  "useSEODefaults",
  "useLocaleMode",
  // Public UI primitives
  "SeoHead",
  "BlogPageShell",
  "AuthorIntro",
  "ArticleList",
  "AuthorSocialLinks",
  "ArticleAdjacentNav",
  // Data / i18n helpers
  "getPublicArticles",
  "pickLocaleValue",
  "SITE_CONFIG_GLOBAL_DEFAULT",
] as const;

export type ThemeHostValueExport = (typeof THEME_HOST_VALUE_EXPORTS)[number];

/**
 * Type-only exports re-exported from theme-host (not present as runtime keys).
 * Keep in sync with `index.ts` type re-exports for theme authors / extraction.
 */
export const THEME_HOST_TYPE_EXPORTS = [
  "ThemePlugin",
  "ThemePageDefinition",
  "ThemeLayoutChrome",
  "HeaderChromeProps",
  "FooterChromeProps",
  "ThemeSettingGroup",
  "TokenPreset",
  "ThemeTokens",
  "LayoutConfig",
  "HeaderConfig",
  "FooterConfig",
  "HeaderBrandMode",
  "BrandingView",
  "ThemeContractVersion",
  "ThemeHostValueExport",
] as const;

export type ThemeHostTypeExport = (typeof THEME_HOST_TYPE_EXPORTS)[number];
