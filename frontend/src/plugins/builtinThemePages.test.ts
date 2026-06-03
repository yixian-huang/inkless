import { describe, expect, it } from "vitest";
import { blogFirstTheme } from "./themes/blog-first";
import { corporateClassicTheme } from "./themes/corporate-classic";
import { BUILTIN_THEME_IDS } from "./builtinThemes";
import manifest from "../../../backend/internal/builtinthemes/pages.json";

type PageMeta = {
  slug: string;
  contentKey: string;
  renderMode: string;
  sortOrder: number;
};

function themePageMetas(themeId: string, pages: { slug: string; contentKey?: string; renderMode: string; nav: { order: number } }[]): PageMeta[] {
  return pages.map((p) => ({
    slug: p.slug,
    contentKey: p.contentKey ?? p.slug,
    renderMode: p.renderMode,
    sortOrder: p.nav.order,
  }));
}

describe("builtin theme pages SSOT", () => {
  it("corporate-classic frontend pages match embedded backend manifest", () => {
    const backend = (manifest as Record<string, PageMeta[]>)[BUILTIN_THEME_IDS.CORPORATE_CLASSIC];
    const frontend = themePageMetas(
      BUILTIN_THEME_IDS.CORPORATE_CLASSIC,
      corporateClassicTheme.pages,
    );
    expect(frontend.map((p) => p.contentKey).sort()).toEqual(
      backend.map((p) => p.contentKey).sort(),
    );
  });

  it("blog-first frontend pages match embedded backend manifest", () => {
    const backend = (manifest as Record<string, PageMeta[]>)[BUILTIN_THEME_IDS.BLOG_FIRST];
    const frontend = themePageMetas(BUILTIN_THEME_IDS.BLOG_FIRST, blogFirstTheme.pages);
    expect(frontend.map((p) => p.contentKey)).toEqual(
      backend.map((p) => p.contentKey),
    );
  });
});
