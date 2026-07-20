import { describe, expect, it } from "vitest";
import { THEME_CONTRACT_VERSION } from "@/theme-host/contract";
import { blogFirstTheme, BLOG_FIRST_CONTRACT_VERSION } from "./index";
// JSON import is allowed by Vite/vitest; path is monorepo-relative from frontend.
import themeManifest from "../../../../../packages/theme-blog-first/inkless.theme.json";

describe("blog-first contract alignment", () => {
  it("matches host THEME_CONTRACT_VERSION", () => {
    expect(BLOG_FIRST_CONTRACT_VERSION).toBe(THEME_CONTRACT_VERSION);
    expect(blogFirstTheme.contractVersion).toBe(THEME_CONTRACT_VERSION);
  });

  it("matches inkless.theme.json contractVersion", () => {
    expect(themeManifest.contractVersion).toBe(THEME_CONTRACT_VERSION);
  });
});
