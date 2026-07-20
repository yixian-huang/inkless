/**
 * UMD entry — host loads this script and expects a register callback.
 * Built-in path uses `src/index.ts` via workspace import instead.
 */
import { blogFirstTheme } from "./index";

declare global {
  interface Window {
    __INKLESS_THEME_REGISTER__?: (theme: typeof blogFirstTheme) => void;
    __IMPRESS_THEME_REGISTER__?: (theme: typeof blogFirstTheme) => void;
  }
}

const register =
  typeof window !== "undefined"
    ? window.__INKLESS_THEME_REGISTER__ ?? window.__IMPRESS_THEME_REGISTER__
    : undefined;

if (typeof register === "function") {
  register(blogFirstTheme);
}

export {
  blogFirstTheme,
  blogFirstTokens,
  BLOG_FIRST_THEME_ID,
  BLOG_FIRST_CONTRACT_VERSION,
} from "./index";
