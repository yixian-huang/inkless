import type { DynamicPageLayout, PageConfig, SectionData } from "./types";

export type HostPagePresetId = "doc-simple" | "doc-guide" | "landing-use-cases";

export interface HostPagePresetMeta {
  id: HostPagePresetId;
  labelZh: string;
  labelEn: string;
  descriptionZh: string;
  descriptionEn: string;
  layout: DynamicPageLayout;
}

export const HOST_PAGE_PRESET_METAS: HostPagePresetMeta[] = [
  {
    id: "doc-simple",
    labelZh: "简单文档",
    labelEn: "Simple doc",
    descriptionZh: "页眉 + 富文本，适合政策与短说明",
    descriptionEn: "Page header + rich text for policies and short notes",
    layout: "reading",
  },
  {
    id: "doc-guide",
    labelZh: "上手指南",
    labelEn: "Guide",
    descriptionZh: "短 Hero + 富文本 + 步骤 + CTA",
    descriptionEn: "Compact hero + rich text + steps + CTA",
    layout: "landing",
  },
  {
    id: "landing-use-cases",
    labelZh: "用例落地",
    labelEn: "Use-cases landing",
    descriptionZh: "Hero + 文字卡片 + 富文本 + CTA",
    descriptionEn: "Hero + text cards + rich text + CTA",
    layout: "landing",
  },
];

export interface BuildPagePresetOptions {
  zhTitle?: string;
  enTitle?: string;
  zhSubtitle?: string;
  enSubtitle?: string;
  /** HTML or plain text for the primary rich-text section */
  zhBody?: string;
  enBody?: string;
}

function sid(prefix: string): string {
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return `${prefix}-${crypto.randomUUID()}`;
  }
  return `${prefix}-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}

function bilingual(zh: string, en: string): { zh: string; en: string } {
  return { zh, en };
}

function defaultBody(zhTitle: string, enTitle: string): { zh: string; en: string } {
  return {
    zh: `<h2>概述</h2><p>在此编写「${zhTitle}」的正文。支持标题、列表与链接。</p><ul><li>要点一</li><li>要点二</li></ul>`,
    en: `<h2>Overview</h2><p>Write the body for “${enTitle}” here. Headings, lists, and links are supported.</p><ul><li>Point one</li><li>Point two</li></ul>`,
  };
}

/**
 * Build a composable page config from a host preset.
 * Section content uses bilingual {zh,en} objects (resolved by SectionRenderer).
 */
export function buildHostPagePreset(
  id: HostPagePresetId,
  opts: BuildPagePresetOptions = {},
): PageConfig {
  const zhTitle = opts.zhTitle?.trim() || "未命名页面";
  const enTitle = opts.enTitle?.trim() || "Untitled page";
  const zhSubtitle = opts.zhSubtitle?.trim() || "";
  const enSubtitle = opts.enSubtitle?.trim() || "";
  const body = {
    zh: opts.zhBody?.trim() || defaultBody(zhTitle, enTitle).zh,
    en: opts.enBody?.trim() || defaultBody(zhTitle, enTitle).en,
  };

  switch (id) {
    case "doc-simple":
      return {
        layout: "reading",
        showPageHeader: true,
        sections: [
          {
            id: sid("rt"),
            type: "rich-text",
            variant: "default",
            data: { content: body, alignment: "left" },
            settings: { padding: "md", maxWidth: "reading", background: "surface" },
          },
          {
            id: sid("faq"),
            type: "faq",
            variant: "default",
            data: {
              title: bilingual("常见问题", "FAQ"),
              items: [
                {
                  question: bilingual("如何修改本页？", "How do I edit this page?"),
                  answer: bilingual(
                    "在管理端 → 页面中编辑草稿并发布。",
                    "Edit the draft under Admin → Pages, then publish.",
                  ),
                },
              ],
            },
            settings: { padding: "md", maxWidth: "reading", background: "surface" },
          },
        ],
      };

    case "doc-guide": {
      const sections: SectionData[] = [
        {
          id: sid("hero"),
          type: "hero",
          variant: "compact",
          data: {
            title: bilingual(zhTitle, enTitle),
            subtitle: bilingual(
              zhSubtitle || "分步说明与关键资源",
              enSubtitle || "Steps and related resources",
            ),
            backgroundColor: "#0f172a",
          },
          settings: { padding: "none", maxWidth: "full", background: "surface" },
        },
        {
          id: sid("rt"),
          type: "rich-text",
          variant: "default",
          data: { content: body, alignment: "left" },
          settings: { padding: "md", maxWidth: "reading", background: "surface" },
        },
        {
          id: sid("steps"),
          type: "steps",
          variant: "default",
          data: {
            title: bilingual("推荐步骤", "Suggested steps"),
            steps: [
              {
                title: bilingual("确认环境", "Prepare environment"),
                description: bilingual("准备服务器或本机开发环境", "Server or local dev environment"),
              },
              {
                title: bilingual("完成配置", "Configure"),
                description: bilingual("站点身份、主题与基础开关", "Identity, theme, and feature toggles"),
              },
              {
                title: bilingual("验证公开页", "Verify public pages"),
                description: bilingual("检查首页与关键路由", "Check home and key routes"),
              },
            ],
          },
          settings: { padding: "md", maxWidth: "layout", background: "surface-alt" },
        },
        {
          id: sid("cta"),
          type: "cta",
          variant: "default",
          data: {
            title: bilingual("下一步", "Next steps"),
            subtitle: bilingual("浏览能力页或继续完善内容", "Explore features or keep refining content"),
            primaryLabel: bilingual("查看能力", "View features"),
            primaryHref: "/features",
            secondaryLabel: bilingual("面向 Agent", "For agents"),
            secondaryHref: "/p/agents",
          },
          settings: { padding: "lg", maxWidth: "layout", background: "surface" },
        },
      ];
      return { layout: "landing", showPageHeader: false, sections };
    }

    case "landing-use-cases": {
      const sections: SectionData[] = [
        {
          id: sid("hero"),
          type: "hero",
          variant: "compact",
          data: {
            title: bilingual(zhTitle, enTitle),
            subtitle: bilingual(
              zhSubtitle || "适用场景与价值",
              enSubtitle || "Where it fits and why",
            ),
            backgroundColor: "#0f172a",
          },
          settings: { padding: "none", maxWidth: "full", background: "surface" },
        },
        {
          id: sid("cards"),
          type: "card-grid",
          variant: "default",
          data: {
            title: bilingual("典型场景", "Use cases"),
            columns: 3,
            preferTextCards: true,
            cards: [
              {
                title: bilingual("个人博客", "Personal blog"),
                description: bilingual(
                  "文章流 + 轻量页面",
                  "Article stream with light pages",
                ),
              },
              {
                title: bilingual("产品运营站", "Product site"),
                description: bilingual(
                  "主题页 + 动态页扩展",
                  "Theme IA plus dynamic pages",
                ),
              },
              {
                title: bilingual("内容团队", "Content team"),
                description: bilingual(
                  "SEO 与 Agent 协作维护",
                  "SEO and agent-assisted ops",
                ),
              },
            ],
          },
          settings: { padding: "lg", maxWidth: "layout", background: "surface" },
        },
        {
          id: sid("rt"),
          type: "rich-text",
          variant: "default",
          data: { content: body, alignment: "left" },
          settings: { padding: "md", maxWidth: "reading", background: "surface" },
        },
        {
          id: sid("cta"),
          type: "cta",
          variant: "default",
          data: {
            title: bilingual("开始搭建", "Start building"),
            primaryLabel: bilingual("快速开始", "Get started"),
            primaryHref: "/p/get-started",
            secondaryLabel: bilingual("能力一览", "Features"),
            secondaryHref: "/features",
          },
          settings: { padding: "lg", maxWidth: "layout", background: "surface" },
        },
      ];
      return { layout: "landing", showPageHeader: false, sections };
    }

    default: {
      const _exhaustive: never = id;
      throw new Error(`Unknown page preset: ${_exhaustive}`);
    }
  }
}

export function listHostPagePresets(): HostPagePresetMeta[] {
  return HOST_PAGE_PRESET_METAS;
}

export function isHostPagePresetId(value: string): value is HostPagePresetId {
  return HOST_PAGE_PRESET_METAS.some((m) => m.id === value);
}
