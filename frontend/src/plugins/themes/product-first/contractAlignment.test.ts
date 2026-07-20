import { describe, expect, it } from "vitest";
import { THEME_CONTRACT_VERSION } from "@/theme-host/contract";
import { productFirstTheme, PRODUCT_FIRST_CONTRACT_VERSION } from "./index";
import { BUILTIN_THEME_IDS } from "@/plugins/builtinThemes";

describe("product-first contract alignment", () => {
  it("matches host THEME_CONTRACT_VERSION", () => {
    expect(PRODUCT_FIRST_CONTRACT_VERSION).toBe(THEME_CONTRACT_VERSION);
    expect(productFirstTheme.contractVersion).toBe(THEME_CONTRACT_VERSION);
  });

  it("manifest id matches BUILTIN_THEME_IDS", () => {
    expect(productFirstTheme.manifest.id).toBe(BUILTIN_THEME_IDS.PRODUCT_FIRST);
  });
});
