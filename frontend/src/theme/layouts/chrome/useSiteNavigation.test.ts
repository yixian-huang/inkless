import { describe, expect, it } from "vitest";
import {
  mergeMenuAndThemeNav,
  normalizeNavPath,
  selectSiteNavigation,
} from "./useSiteNavigation";

describe("normalizeNavPath", () => {
  it("strips trailing slashes on in-app paths", () => {
    expect(normalizeNavPath("/get-started/")).toBe("/get-started");
    expect(normalizeNavPath("/")).toBe("/");
  });

  it("leaves external URLs intact", () => {
    expect(normalizeNavPath("https://example.com/docs/")).toBe(
      "https://example.com/docs/",
    );
  });
});

describe("mergeMenuAndThemeNav", () => {
  it("appends theme paths missing from the primary menu", () => {
    const menu = [
      { label: "首页", path: "/" },
      { label: "能力", path: "/features" },
    ];
    const theme = [
      { label: "Home", path: "/" },
      { label: "Features", path: "/features" },
      { label: "上手", path: "/get-started" },
      { label: "用例", path: "/use-cases" },
    ];
    expect(mergeMenuAndThemeNav(menu, theme)).toEqual([
      { label: "首页", path: "/" },
      { label: "能力", path: "/features" },
      { label: "上手", path: "/get-started" },
      { label: "用例", path: "/use-cases" },
    ]);
  });

  it("keeps menu label when paths collide", () => {
    const menu = [{ label: "产品能力", path: "/features" }];
    const theme = [{ label: "Features", path: "/features/" }];
    expect(mergeMenuAndThemeNav(menu, theme)).toEqual([
      { label: "产品能力", path: "/features" },
    ]);
  });

  it("returns theme only when menu empty", () => {
    const theme = [{ label: "Home", path: "/" }];
    expect(mergeMenuAndThemeNav([], theme)).toEqual(theme);
  });
});

describe("selectSiteNavigation", () => {
  const menu = [{ label: "Primary menu", path: "/menu" }];
  const themePages = [
    { label: "Home", path: "/" },
    { label: "Published page", path: "/published-page" },
  ];
  const themeLayout = [{ label: "Theme default", path: "/theme-default" }];
  const legacy = [{ label: "Legacy", href: "/legacy" }];

  it("merges primary menu with automatic theme/page navigation", () => {
    expect(selectSiteNavigation(menu, themePages, themeLayout, legacy)).toEqual([
      { label: "Primary menu", path: "/menu" },
      { label: "Home", path: "/" },
      { label: "Published page", path: "/published-page" },
    ]);
  });

  it("uses automatic published-page navigation when menu is empty", () => {
    expect(selectSiteNavigation([], themePages, themeLayout, legacy)).toEqual(
      themePages,
    );
  });

  it("falls back from theme layout navigation to legacy global navigation", () => {
    expect(selectSiteNavigation([], [], themeLayout, legacy)).toEqual(themeLayout);
    expect(selectSiteNavigation([], [], undefined, legacy)).toEqual([
      { label: "Legacy", path: "/legacy" },
    ]);
  });

  it("preserves external menu targets on primary menu items", () => {
    const withTarget = [
      { label: "Docs", path: "https://example.com/docs", target: "_blank" as const },
      { label: "Home", path: "/", target: "_self" as const },
    ];
    expect(selectSiteNavigation(withTarget, themePages, themeLayout, legacy)).toEqual([
      ...withTarget,
      { label: "Published page", path: "/published-page" },
    ]);
  });
});
