import type { ArticleMetaResponse, ArticleMetaSuggested } from "@/api/ai";
import { SEO_DESC_MAX, SEO_TITLE_MAX } from "./publishChecklist";

/** Field keys the preview UI can toggle / apply. */
export type AIMetaApplyKey =
  | "zhTitle"
  | "enTitle"
  | "slug"
  | "zhSeoTitle"
  | "enSeoTitle"
  | "zhMetaDescription"
  | "enMetaDescription";

export const AI_META_APPLY_KEYS: AIMetaApplyKey[] = [
  "zhTitle",
  "enTitle",
  "slug",
  "zhSeoTitle",
  "enSeoTitle",
  "zhMetaDescription",
  "enMetaDescription",
];

export const AI_META_LABELS: Record<AIMetaApplyKey, string> = {
  zhTitle: "中文标题",
  enTitle: "英文标题",
  slug: "URL Slug",
  zhSeoTitle: "中文 SEO 标题",
  enSeoTitle: "英文 SEO 标题",
  zhMetaDescription: "中文 Meta 描述",
  enMetaDescription: "英文 Meta 描述",
};

export type AIMetaFormSnapshot = {
  zhTitle: string;
  enTitle: string;
  slug: string;
  zhSeoTitle: string;
  enSeoTitle: string;
  zhMetaDescription: string;
  enMetaDescription: string;
};

export type AIMetaFormSetters = {
  setZhTitle: (v: string) => void;
  setEnTitle: (v: string) => void;
  setSlug: (v: string) => void;
  setZhSeoTitle: (v: string) => void;
  setEnSeoTitle: (v: string) => void;
  setZhMetaDescription: (v: string) => void;
  setEnMetaDescription: (v: string) => void;
};

/** Values available to apply from a response + optional title candidate index. */
export function resolveApplyValues(
  resp: ArticleMetaResponse,
  titleIndex = 0,
): Partial<Record<AIMetaApplyKey, string>> {
  const s = resp.suggested || {};
  const zhTitles = resp.candidates?.zhTitles || [];
  const enTitles = resp.candidates?.enTitles || [];
  const zh =
    (zhTitles[titleIndex] ?? zhTitles[0] ?? s.zhTitle ?? "").trim() || undefined;
  const en =
    (enTitles[titleIndex] ?? enTitles[0] ?? s.enTitle ?? "").trim() || undefined;

  const out: Partial<Record<AIMetaApplyKey, string>> = {};
  if (zh) out.zhTitle = zh;
  if (en) out.enTitle = en;
  if (s.slug?.trim()) out.slug = s.slug.trim();
  if (s.zhSeoTitle?.trim()) out.zhSeoTitle = s.zhSeoTitle.trim();
  if (s.enSeoTitle?.trim()) out.enSeoTitle = s.enSeoTitle.trim();
  if (s.zhMetaDescription?.trim()) out.zhMetaDescription = s.zhMetaDescription.trim();
  if (s.enMetaDescription?.trim()) out.enMetaDescription = s.enMetaDescription.trim();
  return out;
}

/** Default selected keys: only those with a non-empty suggested value. */
export function defaultSelectedKeys(
  values: Partial<Record<AIMetaApplyKey, string>>,
  slugLocked: boolean,
): Set<AIMetaApplyKey> {
  const set = new Set<AIMetaApplyKey>();
  for (const key of AI_META_APPLY_KEYS) {
    if (key === "slug" && slugLocked) continue;
    if (values[key]?.trim()) set.add(key);
  }
  return set;
}

export function applyAIMetaToForm(
  selected: Iterable<AIMetaApplyKey>,
  values: Partial<Record<AIMetaApplyKey, string>>,
  setters: AIMetaFormSetters,
): number {
  let n = 0;
  for (const key of selected) {
    const v = values[key]?.trim();
    if (!v) continue;
    switch (key) {
      case "zhTitle":
        setters.setZhTitle(v);
        break;
      case "enTitle":
        setters.setEnTitle(v);
        break;
      case "slug":
        setters.setSlug(v);
        break;
      case "zhSeoTitle":
        setters.setZhSeoTitle(v);
        break;
      case "enSeoTitle":
        setters.setEnSeoTitle(v);
        break;
      case "zhMetaDescription":
        setters.setZhMetaDescription(v);
        break;
      case "enMetaDescription":
        setters.setEnMetaDescription(v);
        break;
      default:
        break;
    }
    n += 1;
  }
  return n;
}

export type MetaLengthHint = {
  length: number;
  max?: number;
  warn?: boolean;
};

export function lengthHintForKey(key: AIMetaApplyKey, value: string): MetaLengthHint | null {
  const length = value.length;
  if (key === "zhSeoTitle" || key === "enSeoTitle") {
    return { length, max: SEO_TITLE_MAX, warn: length > SEO_TITLE_MAX };
  }
  if (key === "zhMetaDescription" || key === "enMetaDescription") {
    return { length, max: SEO_DESC_MAX, warn: length > SEO_DESC_MAX };
  }
  return { length };
}

/** @deprecated Prefer evaluateAIMetaQuality — kept for simple length-only callers. */
export function qualityWarnings(
  values: Partial<Record<AIMetaApplyKey, string>>,
): string[] {
  const warns: string[] = [];
  for (const key of AI_META_APPLY_KEYS) {
    const v = values[key];
    if (!v) continue;
    const hint = lengthHintForKey(key, v);
    if (hint?.warn && hint.max) {
      warns.push(`${AI_META_LABELS[key]} 偏长（${hint.length}/${hint.max}）`);
    }
  }
  return warns;
}

export type AIMetaFeedbackKind = "useful" | "needs_edit" | "unusable";

export function suggestedHasAny(s: ArticleMetaSuggested | undefined): boolean {
  if (!s) return false;
  return !!(
    s.zhTitle ||
    s.enTitle ||
    s.slug ||
    s.zhSeoTitle ||
    s.enSeoTitle ||
    s.zhMetaDescription ||
    s.enMetaDescription
  );
}
