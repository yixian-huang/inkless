import type { ThemePackage } from "../types";
import type { ThemeTokens } from "../../tokens";

const tokens: ThemeTokens = {
  colors: {
    primary: "#6366f1",
    primaryDark: "#4f46e5",
    accent: "#22d3ee",
    accentHover: "#06b6d4",
    surface: "#0f172a",
    surfaceAlt: "#1e293b",
    onPrimary: "#ffffff",
    onSurface: "#e2e8f0",
    onSurfaceMuted: "#94a3b8",
    border: "#334155",
  },
  fonts: {
    sans: "Inter, system-ui, -apple-system, sans-serif",
    heading: "Inter, system-ui, -apple-system, sans-serif",
  },
  layout: {
    maxWidth: "1400px",
    borderRadius: "0.75rem",
    contentPadding: "2rem",
    sectionSpacing: "5rem",
    contentGap: "2rem",
  },
};

const modernDarkTheme: ThemePackage = {
  id: "modern-dark",
  name: "现代暗色",
  description: "深色专业主题，适合科技、创新类网站",
  author: "Blotting Consultancy",
  version: "1.0.0",
  preview: "linear-gradient(135deg, #6366f1 0%, #22d3ee 100%)",
  tokens,
};
export default modernDarkTheme;
