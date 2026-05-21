import { createContext, useContext, useState, useEffect, useCallback, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import {
  fetchPublicContent,
  normalizeConfigForLocale,
  type Locale,
} from "@/api/publicContent";
import { useBootstrap } from "@/contexts/BootstrapContext";
import { resolveLocale } from "@/utils/locale";
import type {
  SiteConfigGlobal,
  SiteConfigFeatures,
} from "@/types/siteConfig";
import { SITE_CONFIG_GLOBAL_DEFAULT, SITE_CONFIG_FEATURES_DEFAULT } from "@/types/siteConfig";

interface MediaRef {
  url?: string;
  alt?: string;
}

interface NavItem {
  label?: string;
  href?: string;
}

interface LinkItem {
  label?: string;
  href?: string;
}

export interface GlobalConfig {
  // legacy fields (kept while we transition)
  branding?: { logo?: MediaRef; companyName?: string };
  nav?: { items?: NavItem[] };
  footer?: { address?: string; phone?: string; links?: LinkItem[]; copyright?: string };
  // new typed shape — present when published config has an "identity" key
  siteConfig?: SiteConfigGlobal;
}

// Re-export to silence unused-import warnings from the type imports above.
export type { SiteConfigGlobal, SiteConfigFeatures };

interface GlobalConfigContextValue {
  config: GlobalConfig;
  loading: boolean;
  locale: Locale;
  features: SiteConfigFeatures | undefined;
  refetch: () => Promise<void>;
}

const GlobalConfigContext = createContext<GlobalConfigContextValue>({
  config: {},
  loading: true,
  locale: "zh",
  features: undefined,
  refetch: async () => {},
});

export function GlobalConfigProvider({ children }: { children: ReactNode }) {
  const { i18n } = useTranslation("common");
  const locale = resolveLocale(i18n.language);

  const { data: bootstrapData, isLoading: bootstrapLoading } = useBootstrap();
  const [config, setConfig] = useState<GlobalConfig>({});
  const [loading, setLoading] = useState(true);
  const features = bootstrapData?.features ?? undefined;

  // Use bootstrap data for initial load
  useEffect(() => {
    if (bootstrapLoading) return;

    const globalData = bootstrapData?.globalConfig;
    if (globalData?.config) {
      const rawConfig = globalData.config as Record<string, unknown>;
      const normalized = normalizeConfigForLocale(rawConfig, locale) as GlobalConfig;
      // For the new SiteConfigGlobal shape, attach the RAW config (pre-normalization) so
      // LocalizedString values like { zh, en } stay intact for pickLocaleValue downstream.
      if (rawConfig && typeof rawConfig === "object" && "identity" in rawConfig) {
        normalized.siteConfig = rawConfig as unknown as SiteConfigGlobal;
      }
      setConfig(normalized);
    }
    setLoading(false);
  }, [bootstrapData, bootstrapLoading, locale]);

  // refetch still uses the direct API for manual refresh scenarios (e.g. admin edits)
  const doFetch = useCallback(async () => {
    try {
      const data = await fetchPublicContent("global", locale);
      const rawConfig = data.config as Record<string, unknown>;
      const normalized = normalizeConfigForLocale(rawConfig, locale) as GlobalConfig;
      // For the new SiteConfigGlobal shape, attach the RAW config (pre-normalization) so
      // LocalizedString values like { zh, en } stay intact for pickLocaleValue downstream.
      if (rawConfig && typeof rawConfig === "object" && "identity" in rawConfig) {
        normalized.siteConfig = rawConfig as unknown as SiteConfigGlobal;
      }
      setConfig(normalized);
    } catch {
      // Keep previous config on error
    } finally {
      setLoading(false);
    }
  }, [locale]);

  return (
    <GlobalConfigContext.Provider value={{ config, loading, locale, features, refetch: doFetch }}>
      {children}
    </GlobalConfigContext.Provider>
  );
}

// eslint-disable-next-line react-refresh/only-export-components
export function useGlobalConfig(): GlobalConfigContextValue {
  return useContext(GlobalConfigContext);
}

export { SITE_CONFIG_GLOBAL_DEFAULT, SITE_CONFIG_FEATURES_DEFAULT };
