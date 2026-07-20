import { describe, expect, it } from "vitest";
import type { ThemeTokens } from "@/theme/tokens";
import {
  looksLikeSerifStack,
  resolveArticleTypography,
  resolveSerifHeadingStack,
} from "./resolve";

const base: ThemeTokens = {
  colors: {
    primary: "#000",
    primaryDark: "#000",
    accent: "#000",
    accentHover: "#000",
    surface: "#fff",
    surfaceAlt: "#f5f5f5",
    onPrimary: "#fff",
    onSurface: "#111",
    onSurfaceMuted: "#666",
    border: "#eee",
  },
  fonts: {
    sans: 'ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif',
    heading: "Georgia, Palatino, serif",
    mono: "ui-monospace, monospace",
  },
  layout: {
    maxWidth: "40rem",
    borderRadius: "0.25rem",
    contentPadding: "1.25rem",
    sectionSpacing: "3rem",
    contentGap: "1.75rem",
  },
};

describe("looksLikeSerifStack", () => {
  it("detects Georgia and generic serif", () => {
    expect(looksLikeSerifStack("Georgia, Palatino, serif")).toBe(true);
    expect(looksLikeSerifStack("ui-sans-serif, system-ui, sans-serif")).toBe(false);
    expect(looksLikeSerifStack('ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif')).toBe(
      false,
    );
  });
});

describe("resolveSerifHeadingStack", () => {
  it("keeps an explicit Georgia heading", () => {
    expect(resolveSerifHeadingStack(base)).toContain("Georgia");
  });

  it("recovers editorial Georgia when published heading equals sans (yx.ink case)", () => {
    const polluted: ThemeTokens = {
      ...base,
      fonts: {
        sans: 'ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif',
        heading: 'ui-sans-serif, system-ui, -apple-system, "Segoe UI", sans-serif',
        mono: "ui-monospace, monospace",
      },
      fontSources: {
        headingPresetId: "editorial-georgia",
        sansPresetId: "system-ui",
      },
    };
    expect(resolveSerifHeadingStack(polluted)).toContain("Georgia");
  });

  it("falls back to default serif preset when fontSources missing", () => {
    const polluted: ThemeTokens = {
      ...base,
      fonts: {
        sans: "system-ui, sans-serif",
        heading: "system-ui, sans-serif",
      },
    };
    expect(resolveSerifHeadingStack(polluted)).toContain("Georgia");
  });
});

describe("resolveArticleTypography", () => {
  it("uses serif body + title when bodyFontRole is serif and heading is polluted", () => {
    const tokens: ThemeTokens = {
      ...base,
      fonts: {
        sans: "ui-sans-serif, system-ui, sans-serif",
        heading: "ui-sans-serif, system-ui, sans-serif",
      },
      fontSources: { headingPresetId: "editorial-georgia" },
    };
    const cfg = resolveArticleTypography({
      tokens,
      themeSettings: { "article.bodyFontRole": "serif" },
    });
    expect(cfg.bodyFontRole).toBe("serif");
    expect(cfg.bodyFontStack).toContain("Georgia");
    expect(cfg.titleFontStack).toContain("Georgia");
    expect(cfg.uiFontStack).toContain("ui-sans-serif");
  });

  it("uses sans body when bodyFontRole is sans", () => {
    const cfg = resolveArticleTypography({
      tokens: base,
      themeSettings: { "article.bodyFontRole": "sans" },
    });
    expect(cfg.bodyFontRole).toBe("sans");
    expect(cfg.bodyFontStack).toContain("ui-sans-serif");
    expect(cfg.titleFontStack).toContain("Georgia");
  });
});
