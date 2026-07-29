import { http } from "./http";

export type ThemeCatalogSource = "embedded" | "remote" | "cache";

export type ThemeInstallState =
  | "not_installed"
  | "installed"
  | "active"
  | "builtin"
  | "update_available"
  | "incompatible";

export interface ThemeCatalogVersion {
  version: string;
  umdUrl?: string;
  changelog?: string;
  sha256?: string;
  publishedAt?: string;
}

export interface ThemeCatalogItem {
  slug: string;
  themeId: string;
  name: string;
  nameZh?: string;
  description?: string;
  descriptionZh?: string;
  author?: string;
  category?: string;
  tags?: string[];
  iconUrl?: string;
  previewUrl?: string;
  repoUrl?: string;
  contractVersion?: string;
  minHostVersion?: string;
  latest?: ThemeCatalogVersion;
  versions?: ThemeCatalogVersion[];
  defaultFeaturesHint?: Record<string, unknown>;
  builtinOnly?: boolean;
  official?: boolean;
  installState: ThemeInstallState;
  installedVersion?: string;
  installedSource?: string;
  incompatibleReason?: string | null;
  updateAvailable?: boolean;
}

export interface ThemeCatalogResponse {
  schemaVersion: number;
  source: ThemeCatalogSource | string;
  updatedAt?: string;
  warning?: string | null;
  items: ThemeCatalogItem[];
}

export interface InstallThemeRequest {
  slug: string;
  version?: string;
  activate?: boolean;
}

export interface InstallThemeResponse {
  theme: {
    id: number;
    themeId: string;
    name: string;
    nameZh?: string;
    version?: string;
    source?: string;
    externalUrl?: string;
    isActive?: boolean;
    [key: string]: unknown;
  };
  created: boolean;
  activated: boolean;
  warning?: string | null;
}

/** GET /admin/extensions/themes/catalog */
export async function fetchThemeCatalog(options?: {
  refresh?: boolean;
}): Promise<ThemeCatalogResponse> {
  const res = await http.get<ThemeCatalogResponse>("/admin/extensions/themes/catalog", {
    params: options?.refresh ? { refresh: "1" } : undefined,
  });
  return res.data;
}

/** POST /admin/extensions/themes/install */
export async function installThemeFromCatalog(
  body: InstallThemeRequest,
): Promise<InstallThemeResponse> {
  const res = await http.post<InstallThemeResponse>("/admin/extensions/themes/install", body);
  return res.data;
}

/** Convenience: install catalog latest (and optionally activate). Also used for “update”. */
export async function installOrUpdateThemeFromCatalog(
  slug: string,
  opts?: { activate?: boolean; version?: string },
): Promise<InstallThemeResponse> {
  return installThemeFromCatalog({
    slug,
    version: opts?.version,
    activate: opts?.activate ?? false,
  });
}
