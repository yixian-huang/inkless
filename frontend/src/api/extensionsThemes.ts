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

// ── Optional theme auto-update (catalog poll without host redeploy) ──

export interface ThemeAutoUpdateItem {
  themeId: string;
  slug?: string;
  from?: string;
  to?: string;
  reason?: string;
}

export interface ThemeAutoUpdateReport {
  checkedAt: string;
  catalogSource?: string;
  checked: number;
  updated: ThemeAutoUpdateItem[];
  skipped: ThemeAutoUpdateItem[];
  errors: ThemeAutoUpdateItem[];
}

export interface ThemeAutoUpdateSettings {
  enabled: boolean;
  intervalMinutes: number;
  /** Only marketplace/external rows with externalUrl (default true). */
  onlyMarketplace: boolean;
  /** Allow updating the active theme's package pointer (default true). */
  includeActive: boolean;
  /** Only check the active theme. */
  onlyActive: boolean;
  lastCheckAt?: string;
  lastApplyAt?: string;
  lastError?: string;
  lastReport?: ThemeAutoUpdateReport | null;
}

export interface ThemeAutoUpdatePutInput {
  enabled?: boolean;
  intervalMinutes?: number;
  onlyMarketplace?: boolean;
  includeActive?: boolean;
  onlyActive?: boolean;
}

/** GET /admin/extensions/themes/auto-update */
export async function fetchThemeAutoUpdateSettings(): Promise<ThemeAutoUpdateSettings> {
  const res = await http.get<ThemeAutoUpdateSettings>(
    "/admin/extensions/themes/auto-update",
  );
  return res.data;
}

/** PUT /admin/extensions/themes/auto-update */
export async function updateThemeAutoUpdateSettings(
  body: ThemeAutoUpdatePutInput,
): Promise<ThemeAutoUpdateSettings> {
  const res = await http.put<ThemeAutoUpdateSettings>(
    "/admin/extensions/themes/auto-update",
    body,
  );
  return res.data;
}

/**
 * POST /admin/extensions/themes/auto-update/run
 * dryRun=true: report only; dryRun=false: apply catalog updates (even if auto-update is disabled).
 */
export async function runThemeAutoUpdate(opts?: {
  dryRun?: boolean;
}): Promise<ThemeAutoUpdateReport> {
  const res = await http.post<ThemeAutoUpdateReport>(
    "/admin/extensions/themes/auto-update/run",
    { dryRun: opts?.dryRun ?? false },
  );
  return res.data;
}
