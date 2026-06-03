import type { ThemePlugin } from "@/plugins/types";
import type { ThemeTokens } from "@/theme/tokens";
import { BLOG_DEFAULT_LAYOUT } from "@/theme/layouts/defaults";
import { BUILTIN_THEME_IDS } from "@/plugins/builtinThemes";
import BlogHeader from "./chrome/BlogHeader";
import BlogFooter from "./chrome/BlogFooter";

export const blogFirstTokens: ThemeTokens = {
  colors: {
    primary: "#1e40af",
    primaryDark: "#1e3a8a",
    accent: "#2563eb",
    accentHover: "#1d4ed8",
    surface: "#ffffff",
    surfaceAlt: "#f8fafc",
    onPrimary: "#ffffff",
    onSurface: "#0f172a",
    onSurfaceMuted: "#64748b",
    border: "#e2e8f0",
  },
  fonts: {
    sans: "Georgia, 'Times New Roman', serif",
    heading: "Georgia, 'Times New Roman', serif",
  },
  layout: {
    maxWidth: "48rem",
    borderRadius: "0.375rem",
    contentPadding: "1.25rem",
    sectionSpacing: "3rem",
    contentGap: "1.5rem",
  },
};

export const blogFirstTheme: ThemePlugin = {
  manifest: {
    id: BUILTIN_THEME_IDS.BLOG_FIRST,
    name: "Blog First",
    nameZh: "博客优先",
    description: "Minimal personal blog with author intro and article list at home",
    descriptionZh: "极简个人博客，首页展示作者介绍与最近文章",
    author: "Impress CMS",
    version: "1.0.0",
    type: "theme",
    preview: "linear-gradient(135deg, #1e40af 0%, #64748b 100%)",
    tags: ["blog", "minimal"],
  },
  defaultTokens: blogFirstTokens,
  settingSchema: [
    {
      group: "header",
      label: "Header",
      labelZh: "页眉",
      fields: [
        {
          name: "brandMode",
          type: "select",
          label: "Brand mark",
          labelZh: "品牌区",
          defaultValue: "text",
          options: [
            { label: "Site / author name", value: "text" },
            { label: "Logo image", value: "logo" },
            { label: "Avatar + name", value: "avatar" },
            { label: "Hidden", value: "none" },
          ],
        },
        {
          name: "showRssLink",
          type: "boolean",
          label: "Show RSS link",
          labelZh: "显示 RSS 链接",
          defaultValue: true,
        },
        {
          name: "showSocials",
          type: "boolean",
          label: "Show social links",
          labelZh: "显示社交链接",
          defaultValue: false,
        },
      ],
    },
  ],
  tokenPresets: [
    {
      id: "default",
      name: "Reading Room",
      nameZh: "阅读室",
      preview: "linear-gradient(135deg, #1e40af 0%, #64748b 100%)",
      tokens: blogFirstTokens,
    },
    {
      id: "ink",
      name: "Ink",
      nameZh: "墨黑",
      preview: "linear-gradient(135deg, #18181b 0%, #52525b 100%)",
      tokens: {
        ...blogFirstTokens,
        colors: {
          ...blogFirstTokens.colors,
          primary: "#18181b",
          primaryDark: "#09090b",
          accent: "#52525b",
          accentHover: "#3f3f46",
          surface: "#fafafa",
          surfaceAlt: "#f4f4f5",
          onSurface: "#18181b",
          onSurfaceMuted: "#71717a",
          border: "#e4e4e7",
        },
      },
    },
  ],
  pages: [
    {
      slug: "home",
      renderMode: "hardcoded",
      lazyComponent: () => import("@/pages/blog-home/page"),
      contentKey: "home",
      nav: { label: "Home", labelZh: "首页", order: 0, showInHeader: true, showInFooter: false },
    },
  ],
  defaultLayout: BLOG_DEFAULT_LAYOUT,
  layoutChrome: {
    Header: BlogHeader,
    Footer: BlogFooter,
  },
};
