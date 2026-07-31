import { http } from "./http";
import type { SiteConfigGlobal } from "@/types/siteConfig";

/**
 * Admin site identity / branding config.
 * Backend SSOT: site_configs key "global" via /admin/site-config
 * (legacy alias /admin/global-config still accepted).
 */
export interface GlobalConfigState {
  draftConfig: SiteConfigGlobal;
  draftVersion: number;
  publishedConfig: SiteConfigGlobal;
  publishedVersion: number;
  /** "site_config" | "hydrated_from_content_document" | "empty" when provided by API */
  storageSource?: string;
}

const SITE_CONFIG_BASE = "/admin/site-config";

export async function fetchAdminGlobalConfig(): Promise<GlobalConfigState> {
  const res = await http.get<GlobalConfigState>(SITE_CONFIG_BASE);
  return res.data;
}

export async function putAdminGlobalConfigDraft(
  draftConfig: SiteConfigGlobal,
  expectedDraftVersion: number,
): Promise<{ draftVersion: number }> {
  const res = await http.put<{ draftVersion: number }>(`${SITE_CONFIG_BASE}/draft`, {
    draftConfig,
    expectedDraftVersion,
  });
  return res.data;
}

export async function publishAdminGlobalConfig(): Promise<{ publishedVersion: number }> {
  const res = await http.post<{ publishedVersion: number }>(`${SITE_CONFIG_BASE}/publish`);
  return res.data;
}
