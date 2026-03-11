import type { ThemePlugin } from "@/plugins/types";
import { defaultTokens } from "@/theme/tokens";
import StatsCounterSection from "./StatsCounterSection";

export const corporateClassicTheme: ThemePlugin = {
  manifest: {
    id: "corporate-classic",
    name: "Corporate Classic",
    nameZh: "企业经典",
    description: "Professional corporate website with homepage, about, advantages, services, cases, experts, and contact pages",
    descriptionZh: "专业企业官网，含首页、关于、优势、服务、案例、专家、联系",
    author: "Blotting Consultancy",
    version: "1.0.0",
    type: "theme",
    preview: "linear-gradient(135deg, #1a5f8f 0%, #8bc34a 100%)",
    tags: ["corporate", "bilingual"],
  },
  defaultTokens,
  settingSchema: [
    {
      group: "homepage",
      label: "Homepage",
      labelZh: "首页设置",
      fields: [
        { name: "heroStyle", type: "select", label: "Hero Style", labelZh: "主视觉样式",
          defaultValue: "image", options: [
            { label: "背景图", value: "image" },
            { label: "纯色渐变", value: "gradient" },
          ] },
        { name: "showLatestArticles", type: "boolean", label: "Show Latest Articles", labelZh: "显示最新文章",
          defaultValue: true },
        { name: "latestArticlesCount", type: "number", label: "Article Count", labelZh: "文章数量",
          defaultValue: 3 },
      ],
    },
    {
      group: "footer",
      label: "Footer",
      labelZh: "页脚设置",
      fields: [
        { name: "showSocialLinks", type: "boolean", label: "Show Social Links", labelZh: "显示社交链接",
          defaultValue: false },
        { name: "icp", type: "text", label: "ICP Number", labelZh: "ICP 备案号",
          defaultValue: "" },
      ],
    },
  ],
  tokenPresets: [
    {
      id: "default",
      name: "经典蓝绿",
      nameZh: "经典蓝绿",
      preview: "linear-gradient(135deg, #1a5f8f 0%, #8bc34a 100%)",
      tokens: defaultTokens,
    },
    {
      id: "modern-dark",
      name: "现代暗色",
      nameZh: "现代暗色",
      preview: "linear-gradient(135deg, #6366f1 0%, #22d3ee 100%)",
      tokens: {
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
        layout: { maxWidth: "1200px", borderRadius: "0.75rem", contentPadding: "2rem", sectionSpacing: "5rem", contentGap: "2rem" },
      },
    },
    {
      id: "warm-earth",
      name: "暖色大地",
      nameZh: "暖色大地",
      preview: "linear-gradient(135deg, #92400e 0%, #d97706 100%)",
      tokens: {
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
        layout: { maxWidth: "1200px", borderRadius: "0.25rem", contentPadding: "1.25rem", sectionSpacing: "4rem", contentGap: "1.5rem" },
      },
    },
  ],
  sections: {
    "stats-counter": StatsCounterSection,
  },
  sectionMetas: [
    { type: "stats-counter", label: "Stats Counter", labelZh: "数据统计" },
  ],
  pages: [
    {
      slug: "home",
      renderMode: "hardcoded",
      lazyComponent: () => import("@/pages/home/page"),
      contentKey: "home",
      nav: { label: "Home", labelZh: "首页", order: 0, showInHeader: true, showInFooter: true },
    },
    {
      slug: "about",
      renderMode: "hardcoded",
      lazyComponent: () => import("@/pages/about/page"),
      contentKey: "about",
      nav: { label: "About", labelZh: "关于我们", order: 1, showInHeader: true, showInFooter: true },
    },
    {
      slug: "advantages",
      renderMode: "hardcoded",
      lazyComponent: () => import("@/pages/advantages/page"),
      contentKey: "advantages",
      nav: { label: "Advantages", labelZh: "我们的优势", order: 2, showInHeader: true, showInFooter: true },
    },
    {
      slug: "core-services",
      renderMode: "hardcoded",
      lazyComponent: () => import("@/pages/core-services/page"),
      contentKey: "core-services",
      nav: { label: "Services", labelZh: "核心服务", order: 3, showInHeader: true, showInFooter: true },
    },
    {
      slug: "cases",
      renderMode: "hardcoded",
      lazyComponent: () => import("@/pages/cases/page"),
      contentKey: "cases",
      nav: { label: "Cases", labelZh: "案例展示", order: 4, showInHeader: true, showInFooter: true },
    },
    {
      slug: "experts",
      renderMode: "hardcoded",
      lazyComponent: () => import("@/pages/experts/page"),
      contentKey: "experts",
      nav: { label: "Experts", labelZh: "专家团队", order: 5, showInHeader: true, showInFooter: true },
    },
    {
      slug: "contact",
      renderMode: "hardcoded",
      lazyComponent: () => import("@/pages/contact/page"),
      contentKey: "contact",
      nav: { label: "Contact", labelZh: "联系我们", order: 6, showInHeader: true, showInFooter: true },
    },
  ],
  defaultLayout: {
    type: "default",
    header: { style: "sticky" },
    footer: { style: "full" },
  },
};
