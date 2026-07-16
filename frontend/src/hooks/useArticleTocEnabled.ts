import { useThemeSettings } from "@/plugins/hooks";

/** Theme toggle for article table of contents (auto layout when enabled). */
export function useArticleTocEnabled(): boolean {
  const themeSettings = useThemeSettings();
  return themeSettings["article.showToc"] !== false;
}
