/**
 * UMD entry — host loads this script and expects a register callback.
 * Built-in path uses `src/index.ts` via workspace import instead.
 */
import { editorialFirmTheme } from "./index";

declare global {
  interface Window {
    __INKLESS_THEME_REGISTER__?: (theme: typeof editorialFirmTheme) => void;
    __IMPRESS_THEME_REGISTER__?: (theme: typeof editorialFirmTheme) => void;
  }
}

const register =
  typeof window !== "undefined"
    ? window.__INKLESS_THEME_REGISTER__ ?? window.__IMPRESS_THEME_REGISTER__
    : undefined;

if (typeof register === "function") {
  register(editorialFirmTheme);
}

export {
  editorialFirmTheme,
  editorialFirmTokens,
  noirGalleryTokens,
  EDITORIAL_FIRM_THEME_ID,
  EDITORIAL_FIRM_CONTRACT_VERSION,
  createEditorialFirmTheme,
} from "./index";
