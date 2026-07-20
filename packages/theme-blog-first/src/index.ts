import type { ThemePlugin, ThemeTokens } from "@inkless/theme-host";
import { BLOG_DEFAULT_LAYOUT } from "@inkless/theme-host";
import BlogHeader from "./chrome/BlogHeader";
import BlogFooter from "./chrome/BlogFooter";

/** Theme id — keep in sync with host `BUILTIN_THEME_IDS.BLOG_FIRST`. */
export const BLOG_FIRST_THEME_ID = "blog-first";

/**
 * Host contract this package targets.
 * Keep in lockstep with:
 * - host `THEME_CONTRACT_VERSION` (`@inkless/theme-host`)
 * - `inkless.theme.json#contractVersion`
 * - `package.json#inkless.contractVersion`
 *
 * Hardcoded (not imported from host) so UMD init only needs layout tokens,
 * not the full contract module, from `InklessThemeHost`.
 */
export const BLOG_FIRST_CONTRACT_VERSION = "1";

export const blogFirstTokens: ThemeTokens = {
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
    sans: "ui-sans-serif, system-ui, -apple-system, 'Segoe UI', sans-serif",
    heading:
      "Georgia, 'Iowan Old Style', 'Palatino Linotype', 'Book Antiqua', Palatino, serif",
    mono: 'ui-monospace, "SF Mono", Menlo, Monaco, Consolas, monospace',
  },
  fontSources: {
    sansPresetId: "system-ui",
    headingPresetId: "editorial-georgia",
    monoPresetId: "system-mono",
  },
  typography: {
    article: {
      bodySize: "1.0625rem",
      bodyLineHeight: 1.8,
    },
  },
  layout: {
    maxWidth: "42rem",
    borderRadius: "0.25rem",
    contentPadding: "1.5rem",
    sectionSpacing: "3.5rem",
    contentGap: "2rem",
  },
};

export const blogFirstTheme: ThemePlugin = {
  manifest: {
    id: BLOG_FIRST_THEME_ID,
    name: "Blog First",
    nameZh: "博客优先",
    description: "Minimal personal blog with author intro and article list at home",
    descriptionZh: "极简个人博客，首页展示作者介绍与最近文章",
    author: "Inkless CMS",
    version: "1.0.0",
    type: "theme",
    preview: "linear-gradient(135deg, #44403c 0%, #a8a29e 100%)",
    tags: ["blog", "minimal"],
  },
  contractVersion: BLOG_FIRST_CONTRACT_VERSION,
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
          label: "Show social links in header (author page always shows them)",
          labelZh: "页眉显示社交链接（关于页始终显示）",
          defaultValue: true,
        },
      ],
    },
    {
      group: "article",
      label: "Article",
      labelZh: "文章",
      fields: [
        {
          name: "showReadingMeta",
          type: "boolean",
          label: "Show word count & reading time",
          labelZh: "显示字数与阅读时间",
          defaultValue: true,
        },
        {
          name: "wordsPerMinute",
          type: "number",
          label: "Words per minute (estimate)",
          labelZh: "每分钟阅读字数（估算）",
          defaultValue: 280,
        },
        {
          name: "showComments",
          type: "boolean",
          label: "Show comment section on articles",
          labelZh: "文章页显示评论区",
          defaultValue: true,
        },
        {
          name: "showToc",
          type: "boolean",
          label: "Auto table of contents on long articles",
          labelZh: "长文自动显示目录（短文中隐藏）",
          defaultValue: true,
        },
        {
          name: "bodyFontRole",
          type: "select",
          label: "Article body font",
          labelZh: "正文默认字体（衬线 / 无衬线）",
          defaultValue: "serif",
          options: [
            { label: "Serif (reading)", value: "serif" },
            { label: "Sans-serif", value: "sans" },
          ],
        },
      ],
    },
  ],
  tokenPresets: [
    {
      id: "default",
      name: "Reading Room",
      nameZh: "阅读室",
      preview: "linear-gradient(135deg, #44403c 0%, #a8a29e 100%)",
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
          primary: "#27272a",
          primaryDark: "#09090b",
          accent: "#52525b",
          accentHover: "#18181b",
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
      lazyComponent: () => import("./pages/home"),
      contentKey: "home",
      nav: { label: "Home", labelZh: "首页", order: 0, showInHeader: true, showInFooter: false },
    },
    {
      slug: "author",
      renderMode: "hardcoded",
      lazyComponent: () => import("./pages/author"),
      contentKey: "author",
      nav: {
        label: "About",
        labelZh: "关于",
        order: 10,
        showInHeader: true,
        showInFooter: false,
      },
    },
  ],
  defaultLayout: BLOG_DEFAULT_LAYOUT,
  layoutChrome: {
    Header: BlogHeader,
    Footer: BlogFooter,
  },
};

export { resolveBlogHomeBrandMode } from "./chrome/resolveHomeBrand";
export { default as BlogFirstHomePage } from "./pages/home";
export { default as BlogFirstAuthorPage } from "./pages/author";
export { default as BlogHeader } from "./chrome/BlogHeader";
export { default as BlogFooter } from "./chrome/BlogFooter";

