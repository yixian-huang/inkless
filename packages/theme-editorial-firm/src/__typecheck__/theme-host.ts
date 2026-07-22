/**
 * Package type-check surface for `@inkless/theme-host`.
 *
 * The real host facade re-exports runtime modules under `@/` and is resolved
 * by the frontend Vite alias. For isolated package `tsc`, map the import to
 * this thin type-only re-export so we do not type-check the entire SPA graph.
 *
 * Runtime / host bundling still uses `frontend/src/theme-host/index.ts`.
 */
export type {
  ThemePlugin,
  ThemePageDefinition,
  ThemeLayoutChrome,
  HeaderChromeProps,
  FooterChromeProps,
  ThemeSettingGroup,
  TokenPreset,
} from "@/plugins/types";
export type { ThemeTokens } from "@/theme/tokens";
export type {
  LayoutConfig,
  HeaderConfig,
  FooterConfig,
} from "@/theme/layouts/types";
