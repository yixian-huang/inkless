import { useMemo } from "react";
import { useTranslation } from "react-i18next";

/**
 * Recursively resolves { zh, en } objects in section data to the current locale string.
 * Passes through non-localized values unchanged.
 */
export function useLocalizedData<T extends Record<string, unknown>>(data: T): T {
  const { i18n } = useTranslation();
  const locale = i18n.language === "en" ? "en" : "zh";

  return useMemo(() => resolveLocale(data, locale) as T, [data, locale]);
}

function resolveLocale(value: unknown, locale: string): unknown {
  if (value === null || value === undefined) return value;

  if (Array.isArray(value)) {
    return value.map((item) => resolveLocale(item, locale));
  }

  if (typeof value === "object") {
    const obj = value as Record<string, unknown>;
    // Check if this is a { zh, en } localized object
    if (isLocalizedValue(obj)) {
      const localized = obj as { zh?: string; en?: string };
      return (locale === "en" ? localized.en : localized.zh) ?? localized.zh ?? "";
    }
    // Recurse into nested objects
    const result: Record<string, unknown> = {};
    for (const [key, val] of Object.entries(obj)) {
      result[key] = resolveLocale(val, locale);
    }
    return result;
  }

  return value;
}

function isLocalizedValue(obj: Record<string, unknown>): boolean {
  const keys = Object.keys(obj);
  return (
    keys.length <= 2 &&
    keys.every((k) => k === "zh" || k === "en") &&
    keys.some((k) => typeof obj[k] === "string" || obj[k] === undefined || obj[k] === null)
  );
}
