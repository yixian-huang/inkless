import { describe, expect, it } from "vitest";
import {
  checkPageSlug,
  normalizePageSlug,
  themePageSlugsFromManifest,
} from "./pageSlugConflict";

describe("pageSlugConflict", () => {
  it("normalizes slashes and case", () => {
    expect(normalizePageSlug(" /Get-Started/ ")).toBe("get-started");
  });

  it("blocks reserved system slugs", () => {
    const r = checkPageSlug("blog");
    expect(r.kind).toBe("reserved");
    expect(r.blocking).toBe(true);
  });

  it("blocks theme-declared slugs", () => {
    const r = checkPageSlug("features", {
      themeSlugs: ["home", "features", "contact"],
    });
    expect(r.kind).toBe("theme");
    expect(r.blocking).toBe(true);
  });

  it("allows current slug when editing", () => {
    const r = checkPageSlug("features", {
      themeSlugs: ["features"],
      allowSlug: "features",
    });
    expect(r.kind).toBe("ok");
    expect(r.blocking).toBe(false);
  });

  it("accepts valid free slugs", () => {
    const r = checkPageSlug("privacy-policy", {
      themeSlugs: ["home", "features"],
    });
    expect(r.kind).toBe("ok");
  });

  it("collects manifest slugs", () => {
    expect(
      themePageSlugsFromManifest([
        { slug: "home", contentKey: "home" },
        { slug: "features", contentKey: "features-page" },
      ]),
    ).toEqual(expect.arrayContaining(["home", "features", "features-page"]));
  });
});
