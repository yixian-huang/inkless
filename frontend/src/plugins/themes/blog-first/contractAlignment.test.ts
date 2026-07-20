import { describe, expect, it } from "vitest";
import { THEME_CONTRACT_VERSION } from "@/theme-host/contract";
import { blogFirstTheme, BLOG_FIRST_CONTRACT_VERSION } from "./index";

describe("blog-first contract alignment", () => {
  it("matches host THEME_CONTRACT_VERSION", () => {
    expect(BLOG_FIRST_CONTRACT_VERSION).toBe(THEME_CONTRACT_VERSION);
    expect(blogFirstTheme.contractVersion).toBe(THEME_CONTRACT_VERSION);
  });
});
