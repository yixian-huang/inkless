/**
 * Built-in registration entry for editorial-firm.
 * Implementation: github.com/yixian-huang/inkless-theme-editorial-firm
 * (pnpm pin in frontend/package.json).
 *
 * Pages are CMS section-driven (`renderMode: "dynamic"`).
 */
import { createEditorialFirmTheme } from "@inkless/theme-editorial-firm";

export {
  EDITORIAL_FIRM_THEME_ID,
  EDITORIAL_FIRM_CONTRACT_VERSION,
  EDITORIAL_PAGE_CONTENT_KEYS,
  EDITORIAL_DEFAULT_LAYOUT,
  createEditorialFirmTheme,
  editorialFirmTokens,
  noirGalleryTokens,
} from "@inkless/theme-editorial-firm";

export const editorialFirmTheme = createEditorialFirmTheme();
