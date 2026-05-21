import { useGlobalConfig } from "@/contexts/GlobalConfigContext";
import { SITE_CONFIG_GLOBAL_DEFAULT, type SiteConfigGlobal } from "@/types/siteConfig";
import type { Locale, LocaleMode } from "@/lib/locale";

export interface LocaleModeView {
  localeMode: LocaleMode;
  defaultLocale: Locale;
  currentLocale: Locale;
  available: Locale[];
  isMono: boolean;
}

export function useLocaleMode(): LocaleModeView {
  const { config, locale } = useGlobalConfig();
  const sc: SiteConfigGlobal = config.siteConfig ?? SITE_CONFIG_GLOBAL_DEFAULT;
  const mode = sc.identity.localeMode;
  const def = sc.identity.defaultLocale;
  let available: Locale[];
  let current: Locale;
  if (mode === "mono-zh") {
    available = ["zh"];
    current = "zh";
  } else if (mode === "mono-en") {
    available = ["en"];
    current = "en";
  } else {
    available = ["zh", "en"];
    current = (locale as Locale) ?? def;
  }
  return {
    localeMode: mode,
    defaultLocale: def,
    currentLocale: current,
    available,
    isMono: mode !== "bilingual",
  };
}
