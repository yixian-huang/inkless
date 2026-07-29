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
    descriptionZh: "短 Hero + 富文本 + 可选清单",
    descriptionEn: "Compact hero + rich text + optional checklist",
    layout: "reading",
  },
  {
    id: "landing-use-cases",
    labelZh: "用例落地",
    labelEn: "Use-cases landing",
    descriptionZh: "Hero + 卡片网格 + 富文本说明",
    descriptionEn: "Hero + card grid + rich text body",
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
          id: sid("check"),
          type: "checklist",
          variant: "default",
          data: {
            categories: [
              {
                title: bilingual("检查清单", "Checklist"),
                items: ["确认环境就绪", "完成基础配置", "验证公开页面"],
              },
            ],
          },
          settings: { padding: "md", maxWidth: "layout", background: "surface-alt" },
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
