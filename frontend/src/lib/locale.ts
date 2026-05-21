export type Locale = "zh" | "en";
export type LocaleMode = "mono-zh" | "mono-en" | "bilingual";

export type LocalizedString = { zh?: string; en?: string };

export interface PickLocaleValueArgs {
  value: LocalizedString | undefined | null;
  mode: LocaleMode;
  defaultLocale: Locale;
  currentLocale: Locale;
}

export function pickLocaleValue({
  value,
  mode,
  defaultLocale,
  currentLocale,
}: PickLocaleValueArgs): string {
  if (!value) return "";
  if (mode === "mono-zh") return value.zh ?? "";
  if (mode === "mono-en") return value.en ?? "";
  return (
    value[currentLocale] ??
    value[defaultLocale] ??
    value.zh ??
    value.en ??
    ""
  );
}
