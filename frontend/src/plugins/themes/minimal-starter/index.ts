import type { ThemePlugin } from "@/plugins/types";
import type { ThemeTokens } from "@/theme/tokens";
import { BLOG_DEFAULT_LAYOUT } from "@/theme/layouts/defaults";
import { BUILTIN_THEME_IDS } from "@/plugins/builtinThemes";
import MinimalHeader from "./chrome/MinimalHeader";
import MinimalFooter from "./chrome/MinimalFooter";

export const minimalStarterTokens: ThemeTokens = {
  colors: {
    primary: "#374151",
    primaryDark: "#1f2937",
    accent: "#6b7280",
    accentHover: "#4b5563",
    surface: "#ffffff",
    surfaceAlt: "#f9fafb",
    onPrimary: "#ffffff",
    onSurface: "#111827",
    onSurfaceMuted: "#6b7280",
    border: "#e5e7eb",
  },
  fonts: {
    sans: "system-ui, -apple-system, sans-serif",
    heading: "system-ui, -apple-system, sans-serif",
  },
  layout: {
    maxWidth: "42rem",
    borderRadius: "0.375rem",
    contentPadding: "1rem",
    sectionSpacing: "2.5rem",
    contentGap: "1.25rem",
  },
};

/** Reference theme for third-party authors: chrome + manifest only, no custom pages. */
export const minimalStarterTheme: ThemePlugin = {
  manifest: {
    id: BUILTIN_THEME_IDS.MINIMAL_STARTER,
    name: "Minimal Starter",
    nameZh: "极简起步",
    description: "Bare-bones theme demonstrating the extension path (layoutChrome + dynamic home)",
    descriptionZh: "最简主题，演示扩展路径（layoutChrome + 动态首页）",
    author: "Impress CMS",
    version: "1.0.0",
    type: "theme",
    preview: "linear-gradient(135deg, #374151 0%, #9ca3af 100%)",
    tags: ["starter", "minimal", "blog"],
  },
  defaultTokens: minimalStarterTokens,
  pages: [
    {
      slug: "home",
      renderMode: "dynamic",
      contentKey: "home",
      nav: { label: "Home", labelZh: "首页", order: 0, showInHeader: true, showInFooter: false },
    },
  ],
  defaultLayout: BLOG_DEFAULT_LAYOUT,
  layoutChrome: {
    Header: MinimalHeader,
    Footer: MinimalFooter,
  },
};
