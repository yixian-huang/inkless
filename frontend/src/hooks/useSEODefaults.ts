import { useGlobalConfig } from "@/contexts/GlobalConfigContext";
import { PRODUCT_DEFAULT_OG_IMAGE } from "@/config/productBrand";
import { SITE_CONFIG_GLOBAL_DEFAULT, type SiteConfigGlobal } from "@/types/siteConfig";
import { pickLocaleValue, type Locale } from "@/lib/locale";

export interface SEODefaultsView {
  defaultTitle: string;
  titleTemplate: string;
  defaultDescription: string;
  defaultOgImage: string;
  buildTitle(pageTitle: string): string;
}

export function useSEODefaults(): SEODefaultsView {
  const { config, locale } = useGlobalConfig();
  const sc: SiteConfigGlobal = config.siteConfig ?? SITE_CONFIG_GLOBAL_DEFAULT;
  const mode = sc.identity.localeMode;
  const def = sc.identity.defaultLocale;
  const cur = (locale as Locale) ?? def;

  const siteName = pickLocaleValue({ value: sc.identity.name, mode, defaultLocale: def, currentLocale: cur });
  const defaultTitle =
    pickLocaleValue({ value: sc.seo.defaultTitle, mode, defaultLocale: def, currentLocale: cur }) ||
    siteName;
  const titleTemplate = sc.seo.titleTemplate?.trim() || "{page} | {site}";
  const defaultDescription = pickLocaleValue({ value: sc.seo.defaultDescription, mode, defaultLocale: def, currentLocale: cur });
  const defaultOgImage = sc.brand.ogImage || PRODUCT_DEFAULT_OG_IMAGE;

  function buildTitle(pageTitle: string): string {
    const trimmed = (pageTitle ?? "").trim();
    const site = (siteName || defaultTitle || "").trim();
    if (!trimmed) return site || defaultTitle;
    // Avoid "Site · Site" when callers pass the site name (e.g. theme home).
    if (trimmed === site || trimmed === defaultTitle.trim()) {
      return trimmed;
    }
    return titleTemplate
      .replace("{page}", trimmed)
      .replace("{site}", site || defaultTitle);
  }

  return { defaultTitle, titleTemplate, defaultDescription, defaultOgImage, buildTitle };
}
