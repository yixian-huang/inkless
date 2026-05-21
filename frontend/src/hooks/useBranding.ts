import { useGlobalConfig } from "@/contexts/GlobalConfigContext";
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

export function useBranding(): BrandingView {
  const { config, locale } = useGlobalConfig();
  const sc: SiteConfigGlobal = config.siteConfig ?? SITE_CONFIG_GLOBAL_DEFAULT;
  const mode = sc.identity.localeMode;
  const def = sc.identity.defaultLocale;
  const cur = (locale as Locale) ?? def;

  const siteName = pickLocaleValue({ value: sc.identity.name, mode, defaultLocale: def, currentLocale: cur });

  const copyright =
    pickLocaleValue({ value: sc.footer.copyright, mode, defaultLocale: def, currentLocale: cur }) ||
    `© ${new Date().getFullYear()} ${siteName}`;

  return {
    siteName,
    tagline: pickLocaleValue({ value: sc.identity.tagline, mode, defaultLocale: def, currentLocale: cur }),
    logo: sc.brand.logo,
    favicon: sc.brand.favicon,
    primaryColor: sc.brand.primaryColor,
    author: {
      name: sc.author.name,
      avatar: sc.author.avatar,
      bio: pickLocaleValue({ value: sc.author.bio, mode, defaultLocale: def, currentLocale: cur }),
      socials: sc.author.socials,
    },
    footer: {
      copyright,
      icp: sc.footer.icp,
      extraLinks: sc.footer.extraLinks ?? [],
    },
    localeMode: mode,
    defaultLocale: def,
    currentLocale: cur,
  };
}
