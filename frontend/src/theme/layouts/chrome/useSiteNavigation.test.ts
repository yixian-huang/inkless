import { describe, expect, it } from "vitest";
import { selectSiteNavigation } from "./useSiteNavigation";

describe("selectSiteNavigation", () => {
  const menu = [{ label: "Primary menu", path: "/menu" }];
  const unifiedPages = [{ label: "Published page", path: "/published-page" }];
  const themeLayout = [{ label: "Theme default", path: "/theme-default" }];
  const legacy = [{ label: "Legacy", href: "/legacy" }];

  it("uses the explicit primary menu before automatic page navigation", () => {
    expect(selectSiteNavigation(menu, unifiedPages, themeLayout, legacy)).toEqual(menu);
  });

  it("uses automatic published-page navigation before theme layout defaults", () => {
    expect(selectSiteNavigation([], unifiedPages, themeLayout, legacy)).toEqual(unifiedPages);
  });

  it("falls back from theme layout navigation to legacy global navigation", () => {
    expect(selectSiteNavigation([], [], themeLayout, legacy)).toEqual(themeLayout);
    expect(selectSiteNavigation([], [], undefined, legacy)).toEqual([
      { label: "Legacy", path: "/legacy" },
    ]);
  });
});
