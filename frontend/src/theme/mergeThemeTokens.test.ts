import { describe, expect, it } from "vitest";
import { mergeThemeTokens } from "./mergeThemeTokens";
import type { ThemeTokens } from "./tokens";

const blogBase: ThemeTokens = {
  colors: {
    primary: "#44403c",
    primaryDark: "#292524",
    accent: "#57534e",
    accentHover: "#1c1917",
    surface: "#faf8f5",
    surfaceAlt: "#f3f1ec",
    onPrimary: "#faf8f5",
    onSurface: "#1c1917",
    onSurfaceMuted: "#78716c",
    border: "#e7e5e4",
  },
  fonts: {
    sans: "ui-sans-serif, system-ui, sans-serif",
    heading: "Georgia, Palatino, serif",
    mono: "ui-monospace, monospace",
  },
  fontSources: {
    sansPresetId: "system-ui",
    headingPresetId: "editorial-georgia",
    monoPresetId: "system-mono",
  },
  typography: {
    article: { bodySize: "1.0625rem", bodyLineHeight: 1.8 },
  },
  layout: {
    maxWidth: "42rem",
    borderRadius: "0.25rem",
    contentPadding: "1.5rem",
    sectionSpacing: "3.5rem",
    contentGap: "2rem",
  },
};

describe("mergeThemeTokens", () => {
  it("fills missing fontSources and typography from theme base", () => {
    const merged = mergeThemeTokens(blogBase, {
      colors: { ...blogBase.colors, primary: "#141310" },
      fonts: {
        sans: "ui-sans-serif, system-ui, sans-serif",
        heading: "ui-sans-serif, system-ui, sans-serif",
      },
      layout: { ...blogBase.layout, maxWidth: "40rem" },
    });

    expect(merged.colors.primary).toBe("#141310");
    expect(merged.layout.maxWidth).toBe("40rem");
    expect(merged.fontSources?.headingPresetId).toBe("editorial-georgia");
    expect(merged.typography?.article?.bodySize).toBe("1.0625rem");
    // Explicit overlay fonts still win (recovery happens in typography resolve).
    expect(merged.fonts.heading).toContain("ui-sans-serif");
  });

  it("keeps theme Georgia when overlay omits fonts", () => {
    const merged = mergeThemeTokens(blogBase, {
      colors: blogBase.colors,
      layout: blogBase.layout,
    } as Partial<ThemeTokens>);

    expect(merged.fonts.heading).toContain("Georgia");
  });
});
