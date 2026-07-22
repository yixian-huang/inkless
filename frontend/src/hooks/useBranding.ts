import { useGlobalConfig, type GlobalConfig } from "@/contexts/GlobalConfigContext";
import {
  SITE_CONFIG_GLOBAL_DEFAULT,
  type SiteConfigGlobal,
  type SiteConfigSocial,
  type SiteConfigFooterLink,
} from "@/types/siteConfig";
import { pickLocaleValue, type Locale, type LocaleMode } from "@/lib/locale";

export interface BrandingView {
  siteName: string;
  tagline: string;
  logo: { light: string; dark?: string };
  favicon: string;
  primaryColor: string;
  author: {
    name: string;
    avatar?: string;
    bio: string;
    socials: SiteConfigSocial[];
  };
  footer: {
    copyright: string;
    icp?: string;
    extraLinks: SiteConfigFooterLink[];
  };
  localeMode: LocaleMode;
  defaultLocale: Locale;
  currentLocale: Locale;
}

/** Pull a string from legacy bilingual objects or plain strings. */
function asLocalizedString(
  value: unknown,
  locale: Locale,
): string {
  if (typeof value === "string") return value.trim();
  if (value && typeof value === "object") {
    const o = value as Record<string, unknown>;
    const picked = o[locale] ?? o.zh ?? o.en;
    if (typeof picked === "string") return picked.trim();
  }
  return "";
}

/** Media ref after normalizeConfigForLocale is often `{ url, alt }`. */
function asMediaUrl(value: unknown): string {
  if (typeof value === "string") return value.trim();
  if (value && typeof value === "object") {
    const url = (value as { url?: unknown }).url;
    if (typeof url === "string") return url.trim();
  }
  return "";
}

/**
 * Map host global config (new siteConfig shape or legacy branding/header) into BrandingView.
 * Exported for unit tests.
 */
export function resolveBrandingView(
  config: GlobalConfig,
  locale: string | undefined,
): BrandingView {
  const cur = (locale as Locale) || "zh";

  // New SiteConfigGlobal shape (has identity)
  if (config.siteConfig) {
    const sc: SiteConfigGlobal = config.siteConfig;
    const mode = sc.identity.localeMode;
    const def = sc.identity.defaultLocale;
    const current = cur || def;

    const siteName = pickLocaleValue({
      value: sc.identity.name,
      mode,
      defaultLocale: def,
      currentLocale: current,
    });

    const copyright =
      pickLocaleValue({
        value: sc.footer.copyright,
        mode,
        defaultLocale: def,
        currentLocale: current,
      }) || `© ${new Date().getFullYear()} ${siteName}`;

    return {
      siteName,
      tagline: pickLocaleValue({
        value: sc.identity.tagline,
        mode,
        defaultLocale: def,
        currentLocale: current,
      }),
      logo: sc.brand.logo,
      favicon: sc.brand.favicon,
      primaryColor: sc.brand.primaryColor,
      author: {
        name: sc.author.name,
        avatar: sc.author.avatar,
        bio: pickLocaleValue({
          value: sc.author.bio,
          mode,
          defaultLocale: def,
          currentLocale: current,
        }),
        socials: sc.author.socials,
      },
      footer: {
        copyright,
        icp: sc.footer.icp,
        extraLinks: sc.footer.extraLinks ?? [],
      },
      localeMode: mode,
      defaultLocale: def,
      currentLocale: current,
    };
  }

  // Legacy impress global content: branding / header / footer (no identity key)
  const branding = config.branding as
    | { logo?: unknown; companyName?: unknown }
    | undefined;
  const header = (config as { header?: { logo?: unknown } }).header;
  const footer = config.footer as
    | { copyright?: unknown; address?: unknown; phone?: unknown }
    | undefined;

  const siteName =
    asLocalizedString(branding?.companyName, cur) ||
    SITE_CONFIG_GLOBAL_DEFAULT.identity.name.zh ||
    "Site";

  const logoUrl =
    asMediaUrl(branding?.logo) ||
    asMediaUrl(header?.logo) ||
    "";

  const copyright =
    asLocalizedString(footer?.copyright, cur) ||
    `© ${new Date().getFullYear()} ${siteName}`;

  return {
    siteName,
    tagline: "",
    logo: { light: logoUrl },
    favicon: "",
    primaryColor: SITE_CONFIG_GLOBAL_DEFAULT.brand.primaryColor,
    author: { name: "", bio: "", socials: [] },
    footer: {
      copyright,
      extraLinks: [],
    },
    localeMode: "mono-zh",
    defaultLocale: "zh",
    currentLocale: cur,
  };
}

export function useBranding(): BrandingView {
  const { config, locale } = useGlobalConfig();
  return resolveBrandingView(config, locale);
}
