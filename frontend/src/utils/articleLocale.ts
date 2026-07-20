import { pickLocaleValue, type Locale, type LocaleMode } from "@/lib/locale";
import type { Article } from "@/api/articles";

export function articleTitle(
  article: Article,
  mode: LocaleMode,
  defaultLocale: Locale,
  currentLocale: Locale,
): string {
  return (
    pickLocaleValue({ value: { zh: article.zhTitle, en: article.enTitle }, mode, defaultLocale, currentLocale }) ||
    article.zhTitle ||
    article.enTitle ||
    ""
  );
}

export function articleBody(
  article: Article,
  mode: LocaleMode,
  defaultLocale: Locale,
  currentLocale: Locale,
): string {
  return (
    pickLocaleValue({ value: { zh: article.zhBody, en: article.enBody }, mode, defaultLocale, currentLocale }) ||
    article.zhBody ||
    article.enBody ||
    ""
  );
}

export function articleMetaDescription(
  article: Article,
  mode: LocaleMode,
  defaultLocale: Locale,
  currentLocale: Locale,
): string {
  return (
    pickLocaleValue({
      value: { zh: article.zhMetaDescription, en: article.enMetaDescription },
      mode,
      defaultLocale,
      currentLocale,
    }) || ""
  );
}

/**
 * Build a list-card excerpt from body HTML or already-plain API excerpts.
 * Public list may return pre-truncated plain text in zhBody/enBody.
 */
export function articleExcerpt(body: string, maxLen = 160): string {
  if (!body) return "";
  const text = body.replace(/<[^>]*>/g, "").replace(/\s+/g, " ").trim();
  if (!text) return "";
  // Backend public list already truncates (~160 runes); avoid double ellipsis.
  if (text.endsWith("...") || text.length <= maxLen) return text;
  return text.slice(0, maxLen) + "...";
}

export function formatArticleDate(dateStr: string | null, locale: Locale): string {
  if (!dateStr) return "";
  const tag = locale === "en" ? "en-US" : "zh-CN";
  return new Date(dateStr).toLocaleDateString(tag, {
    year: "numeric",
    month: "long",
    day: "numeric",
  });
}
