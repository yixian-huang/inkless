import type { ArticleMetaWarning } from "@/api/ai";
import { SEO_DESC_MAX, SEO_DESC_MIN, SEO_TITLE_MAX } from "./publishChecklist";
import {
  AI_META_APPLY_KEYS,
  AI_META_LABELS,
  type AIMetaApplyKey,
} from "./applyAIMeta";

const DISPLAY_TITLE_MAX = 80;

const PLACEHOLDER_RES = [
  /^\s*(untitled|no\s*title|tbd|todo|placeholder|lorem\s+ipsum)\s*$/i,
  /^\s*(未命名|无标题|请填写|待补充|占位|测试标题)\s*$/,
  /lorem\s+ipsum/i,
  /^(这是一段?(测试|示例|占位))/,
];

export type QualityIssue = {
  code: string;
  field?: string;
  message: string;
  severity: "warn" | "info";
};

/** Detect dominant script: zh | en | mixed | unknown */
export function detectScriptLang(s: string): "zh" | "en" | "mixed" | "unknown" {
  let cjk = 0;
  let latin = 0;
  let other = 0;
  for (const ch of s) {
    const code = ch.codePointAt(0) ?? 0;
    if (code >= 0x4e00 && code <= 0x9fff) cjk += 1;
    else if (/[A-Za-z]/.test(ch)) latin += 1;
    else if (/\p{L}/u.test(ch)) other += 1;
  }
  const letters = cjk + latin + other;
  if (letters === 0) return "unknown";
  if (cjk / letters >= 0.35) {
    if (latin / letters >= 0.45) return "mixed";
    return "zh";
  }
  if (latin / letters >= 0.6) return "en";
  if (cjk > 0 && latin > 0) return "mixed";
  if (cjk > latin) return "zh";
  if (latin > 0) return "en";
  return "unknown";
}

export function extractSignificantTokens(text: string): Map<string, number> {
  const freq = new Map<string, number>();
  if (!text.trim()) return freq;

  const words = text.toLowerCase().match(/[a-z][a-z0-9]{2,}/g) || [];
  for (const w of words) freq.set(w, (freq.get(w) || 0) + 1);

  let run = "";
  const flush = () => {
    if (run.length === 1) {
      freq.set(run, (freq.get(run) || 0) + 1);
    }
    for (let i = 0; i + 1 < run.length; i += 1) {
      const bg = run.slice(i, i + 2);
      freq.set(bg, (freq.get(bg) || 0) + 1);
    }
    run = "";
  };
  for (const ch of text) {
    const code = ch.codePointAt(0) ?? 0;
    if (code >= 0x4e00 && code <= 0x9fff) run += ch;
    else flush();
  }
  flush();

  const stop = new Set([
    "the", "and", "for", "with", "this", "that", "from", "are", "was", "have", "has",
    "的", "了", "和", "是", "在", "我", "有", "也", "就", "不", "人", "都", "一", "一个",
    "我们", "可以", "没有", "什么", "这个", "那个",
  ]);
  for (const k of stop) freq.delete(k);
  return freq;
}

export function tokenOverlapRatio(field: string, bodyTokens: Map<string, number>): number {
  const ft = extractSignificantTokens(field);
  if (ft.size === 0) return 0;
  let hit = 0;
  for (const tok of ft.keys()) {
    if ((bodyTokens.get(tok) || 0) > 0) hit += 1;
  }
  return hit / ft.size;
}

function expectLangForKey(key: AIMetaApplyKey): "zh" | "en" | null {
  if (key.startsWith("zh")) return "zh";
  if (key.startsWith("en")) return "en";
  return null;
}

/**
 * Client-side Phase 1.5 quality evaluation for current preview values.
 * Re-runs when the user cycles title candidates so warnings stay accurate.
 */
export function evaluateAIMetaQuality(opts: {
  values: Partial<Record<AIMetaApplyKey, string>>;
  bodyPlain: string;
  selected?: Set<AIMetaApplyKey>;
}): QualityIssue[] {
  const issues: QualityIssue[] = [];
  const bodyTokens = extractSignificantTokens(opts.bodyPlain || "");
  const keys = opts.selected?.size
    ? AI_META_APPLY_KEYS.filter((k) => opts.selected!.has(k))
    : AI_META_APPLY_KEYS;

  for (const key of keys) {
    const value = (opts.values[key] || "").trim();
    if (!value) continue;
    const label = AI_META_LABELS[key];
    const n = value.length;

    for (const re of PLACEHOLDER_RES) {
      if (re.test(value)) {
        issues.push({
          code: "placeholder",
          field: key,
          message: `${label} 疑似占位/模板文案`,
          severity: "warn",
        });
        break;
      }
    }

    if (key === "zhSeoTitle" || key === "enSeoTitle") {
      if (n > SEO_TITLE_MAX) {
        issues.push({
          code: "length_long",
          field: key,
          message: `${label} 偏长（${n}/${SEO_TITLE_MAX}）`,
          severity: "warn",
        });
      }
    } else if (key === "zhMetaDescription" || key === "enMetaDescription") {
      if (n < SEO_DESC_MIN) {
        issues.push({
          code: "length_short",
          field: key,
          message: `${label} 偏短（${n}，建议 ≥${SEO_DESC_MIN}）`,
          severity: "warn",
        });
      }
      if (n > SEO_DESC_MAX) {
        issues.push({
          code: "length_long",
          field: key,
          message: `${label} 偏长（${n}/${SEO_DESC_MAX}）`,
          severity: "warn",
        });
      }
    } else if (key === "zhTitle" || key === "enTitle") {
      if (n > DISPLAY_TITLE_MAX) {
        issues.push({
          code: "length_long",
          field: key,
          message: `${label} 偏长（${n}，建议 ≤${DISPLAY_TITLE_MAX}）`,
          severity: "info",
        });
      }
    } else if (key === "slug") {
      if (!/^[a-z0-9]+(?:-[a-z0-9]+)*$/.test(value)) {
        issues.push({
          code: "slug_format",
          field: key,
          message: "Slug 格式不规范（应为小写英文 kebab-case）",
          severity: "warn",
        });
      }
    }

    const expect = expectLangForKey(key);
    if (expect) {
      const got = detectScriptLang(value);
      if (got !== "mixed" && got !== "unknown" && got !== expect) {
        issues.push({
          code: "language_mismatch",
          field: key,
          message: `${label} 语言与预期不符（期望 ${expect}，检测为 ${got}）`,
          severity: "warn",
        });
      }
    }

    if (bodyTokens.size > 0 && key !== "slug") {
      const ratio = tokenOverlapRatio(value, bodyTokens);
      if (ratio === 0) {
        issues.push({
          code: "low_relevance",
          field: key,
          message: `${label} 与正文关键词重叠偏低，请确认是否跑题`,
          severity: "warn",
        });
      }
    }
  }

  return dedupeIssues(issues);
}

/**
 * Merge server warnings with client re-eval.
 * Client is authoritative for primary apply keys; server keeps candidate / pre-truncate notes.
 */
export function mergeQualityIssues(
  server: ArticleMetaWarning[] | undefined,
  client: QualityIssue[],
): QualityIssue[] {
  const mapped: QualityIssue[] = (server || []).map((w) => ({
    code: w.code,
    field: w.field,
    message: w.message,
    severity: w.severity === "info" ? "info" : "warn",
  }));
  const serverExtra = mapped.filter((i) => {
    if (!i.field) return true;
    return !AI_META_APPLY_KEYS.includes(i.field as AIMetaApplyKey);
  });
  return dedupeIssues([...client, ...serverExtra]);
}

function dedupeIssues(inList: QualityIssue[]): QualityIssue[] {
  const seen = new Set<string>();
  const out: QualityIssue[] = [];
  for (const i of inList) {
    const key = `${i.code}|${i.field || ""}|${i.message}`;
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(i);
  }
  return out;
}

/** Issues for a single field (for row badges). */
export function issuesForField(issues: QualityIssue[], field: string): QualityIssue[] {
  return issues.filter((i) => i.field === field || i.field?.startsWith(`${field}[`));
}

export function hasWarnSeverity(issues: QualityIssue[]): boolean {
  return issues.some((i) => i.severity === "warn");
}
