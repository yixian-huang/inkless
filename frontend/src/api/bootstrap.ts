import { http } from "./http";
import type { ThemePageItem } from "./themePages";
import type { PublicUnifiedPageItem } from "./unifiedPages";
import type { SiteConfigFeatures } from "@/types/siteConfig";

export interface ActiveThemeData {
  themeId?: string;
  source?: string;
  externalUrl?: string;
  config?: Record<string, unknown>;
}

export interface GlobalConfigData {
  pageKey?: string;
  version?: number;
  locale?: string;
  config?: Record<string, unknown>;
}

export interface PageContentData {
  pageKey?: string;
  version?: number;
  locale?: string;
  config?: Record<string, unknown>;
}

export interface BootstrapData {
  activeTheme: ActiveThemeData;
  themeTokens: Record<string, unknown>;
  themePages: ThemePageItem[];
  unifiedPages?: PublicUnifiedPageItem[];
  globalConfig: GlobalConfigData;
  pageContent?: PageContentData;
  features?: SiteConfigFeatures;
}

export async function fetchBootstrap(
  locale: string = "zh",
  pageKey?: string
): Promise<BootstrapData> {
  const params: Record<string, string> = { locale };
  if (pageKey) params.pageKey = pageKey;
  const res = await http.get<BootstrapData>("/public/bootstrap", { params });
  return res.data;
}
