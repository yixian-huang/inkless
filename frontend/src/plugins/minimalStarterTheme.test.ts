import { describe, expect, it, beforeAll } from "vitest";
import { themeManager } from "./ThemeManager";
import { minimalStarterTheme } from "./themes/minimal-starter";
import { BUILTIN_THEME_IDS } from "./builtinThemes";

describe("minimal-starter theme extension path", () => {
  beforeAll(() => {
    if (!themeManager.getTheme(BUILTIN_THEME_IDS.MINIMAL_STARTER)) {
      themeManager.registerBuiltIn(minimalStarterTheme);
    }
  });

  it("registers with layoutChrome without SiteLayout changes", () => {
    const theme = themeManager.getTheme(BUILTIN_THEME_IDS.MINIMAL_STARTER);
    expect(theme).toBe(minimalStarterTheme);
    expect(theme?.layoutChrome?.Header).toBeDefined();
    expect(theme?.layoutChrome?.Footer).toBeDefined();
    expect(theme?.manifest.tags).toContain("starter");
  });

  it("uses dynamic home page (CMS sections)", () => {
    expect(minimalStarterTheme.pages).toHaveLength(1);
    expect(minimalStarterTheme.pages[0]?.renderMode).toBe("dynamic");
  });
});
