import { describe, expect, it } from "vitest";
import type { PageConfig, SectionData } from "./types";
import {
  hasLeadingHero,
  isDocOnlySections,
  resolveDynamicPageLayout,
  shouldShowPageHeader,
} from "./dynamicPageLayout";

function sec(type: string, hidden = false): SectionData {
  return {
    id: type,
    type,
    data: {},
    settings: hidden ? { hidden: true } : {},
  };
}

describe("dynamicPageLayout", () => {
  it("detects leading hero", () => {
    expect(hasLeadingHero([sec("hero"), sec("rich-text")])).toBe(true);
    expect(hasLeadingHero([sec("rich-text")])).toBe(false);
    expect(hasLeadingHero([sec("hero", true), sec("rich-text")])).toBe(false);
  });

  it("detects doc-only stacks", () => {
    expect(isDocOnlySections([sec("rich-text")])).toBe(true);
    expect(isDocOnlySections([sec("rich-text"), sec("checklist")])).toBe(true);
    expect(isDocOnlySections([sec("rich-text"), sec("card-grid")])).toBe(false);
  });

  it("resolves layout auto / explicit", () => {
    expect(
      resolveDynamicPageLayout({ sections: [sec("hero"), sec("rich-text")] }),
    ).toBe("landing");
    expect(resolveDynamicPageLayout({ sections: [sec("rich-text")] })).toBe(
      "reading",
    );
    expect(
      resolveDynamicPageLayout({
        layout: "landing",
        sections: [sec("rich-text")],
      }),
    ).toBe("landing");
    expect(
      resolveDynamicPageLayout({
        layout: "reading",
        sections: [sec("hero")],
      }),
    ).toBe("reading");
  });

  it("shows page header for reading without hero", () => {
    const cfg: PageConfig = { sections: [sec("rich-text")] };
    expect(shouldShowPageHeader(cfg, "reading", "标题")).toBe(true);
    expect(shouldShowPageHeader(cfg, "landing", "标题")).toBe(false);
    expect(
      shouldShowPageHeader(
        { sections: [sec("hero")], showPageHeader: true },
        "landing",
        "标题",
      ),
    ).toBe(true);
    expect(shouldShowPageHeader(cfg, "reading", "")).toBe(false);
  });
});
