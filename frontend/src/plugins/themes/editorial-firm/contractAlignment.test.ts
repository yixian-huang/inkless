import { describe, expect, it } from "vitest";
import {
  EDITORIAL_FIRM_CONTRACT_VERSION,
  EDITORIAL_FIRM_THEME_ID,
} from "@inkless/theme-editorial-firm";
import { THEME_CONTRACT_VERSION } from "@/theme-host/contract";
import { BUILTIN_THEME_IDS } from "@/plugins/builtinThemes";
import { editorialFirmTheme } from "./index";

describe("editorial-firm contract alignment", () => {
  it("manifest id matches builtin constant", () => {
    expect(EDITORIAL_FIRM_THEME_ID).toBe(BUILTIN_THEME_IDS.EDITORIAL_FIRM);
    expect(editorialFirmTheme.manifest.id).toBe(BUILTIN_THEME_IDS.EDITORIAL_FIRM);
  });

  it("targets host contract v1", () => {
    expect(EDITORIAL_FIRM_CONTRACT_VERSION).toBe(THEME_CONTRACT_VERSION);
    expect(editorialFirmTheme.contractVersion).toBe(THEME_CONTRACT_VERSION);
  });

  it("has four dynamic pages", () => {
    expect(editorialFirmTheme.pages).toHaveLength(4);
    for (const page of editorialFirmTheme.pages) {
      expect(page.renderMode).toBe("dynamic");
    }
    expect(editorialFirmTheme.pages.map((p) => p.contentKey)).toEqual([
      "home",
      "about",
      "services",
      "contact",
    ]);
  });

  it("exposes layout chrome", () => {
    expect(editorialFirmTheme.layoutChrome?.Header).toBeTruthy();
    expect(editorialFirmTheme.layoutChrome?.Footer).toBeTruthy();
  });
});
