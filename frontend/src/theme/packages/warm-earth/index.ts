import type { ThemePackage } from "../types";
import type { ThemeTokens } from "../../tokens";

const tokens: ThemeTokens = {
  colors: {
    primary: "#92400e",
    primaryDark: "#78350f",
    accent: "#d97706",
    accentHover: "#b45309",
    surface: "#fffbeb",
    surfaceAlt: "#fef3c7",
    onPrimary: "#ffffff",
    onSurface: "#451a03",
    onSurfaceMuted: "#78350f",
    border: "#fde68a",
  },
  fonts: {
    sans: "Georgia, 'Times New Roman', serif",
    heading: "Georgia, 'Times New Roman', serif",
  },
  layout: {
    maxWidth: "1400px",
    borderRadius: "0.25rem",
    contentPadding: "1.25rem",
    sectionSpacing: "4rem",
    contentGap: "1.5rem",
  },
};

const warmEarthTheme: ThemePackage = {
  id: "warm-earth",
  name: "暖色大地",
  description: "温暖的大地色调，适合文化、教育、餐饮类网站",
  author: "Blotting Consultancy",
  version: "1.0.0",
  preview: "linear-gradient(135deg, #92400e 0%, #d97706 100%)",
  tokens,
};
export default warmEarthTheme;
