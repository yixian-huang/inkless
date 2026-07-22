import type { ThemeTokens } from "@inkless/theme-host";

export const editorialFirmTokens: ThemeTokens = {
  colors: {
    primary: "#111111",
    primaryDark: "#000000",
    accent: "#C45C26",
    accentHover: "#A34A1C",
    surface: "#FAF8F5",
    surfaceAlt: "#F0EBE3",
    onPrimary: "#FFFFFF",
    onSurface: "#1A1A1A",
    onSurfaceMuted: "#5C5C5C",
    border: "#E5DFD6",
  },
  fonts: {
    heading:
      "\"Iowan Old Style\", \"Palatino Linotype\", Palatino, \"Songti SC\", \"Noto Serif SC\", serif",
    sans: 'system-ui, -apple-system, "PingFang SC", "Noto Sans SC", sans-serif',
  },
  layout: {
    maxWidth: "1280px",
    borderRadius: "0.125rem",
    contentPadding: "1.25rem",
    sectionSpacing: "6rem",
    contentGap: "2.5rem",
  },
};

export const noirGalleryTokens: ThemeTokens = {
  colors: {
    primary: "#F5F5F5",
    primaryDark: "#FFFFFF",
    accent: "#E8B86D",
    accentHover: "#D4A017",
    surface: "#0A0A0A",
    surfaceAlt: "#141414",
    onPrimary: "#0A0A0A",
    onSurface: "#F0F0F0",
    onSurfaceMuted: "#A3A3A3",
    border: "#2A2A2A",
  },
  fonts: editorialFirmTokens.fonts,
  layout: editorialFirmTokens.layout,
};
