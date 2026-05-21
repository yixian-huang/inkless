import { http } from "./http";
import type { SiteConfigGlobal } from "@/types/siteConfig";

export interface GlobalConfigState {
  draftConfig: SiteConfigGlobal;
  draftVersion: number;
  publishedConfig: SiteConfigGlobal;
  publishedVersion: number;
}

export async function fetchAdminGlobalConfig(): Promise<GlobalConfigState> {
  const res = await http.get<GlobalConfigState>("/admin/global-config");
  return res.data;
}

export async function putAdminGlobalConfigDraft(
  draftConfig: SiteConfigGlobal,
  expectedDraftVersion: number,
): Promise<{ draftVersion: number }> {
  const res = await http.put<{ draftVersion: number }>("/admin/global-config/draft", {
    draftConfig,
    expectedDraftVersion,
  });
  return res.data;
}

export async function publishAdminGlobalConfig(): Promise<{ publishedVersion: number }> {
  const res = await http.post<{ publishedVersion: number }>("/admin/global-config/publish");
  return res.data;
}
