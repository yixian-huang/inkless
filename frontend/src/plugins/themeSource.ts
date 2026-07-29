/**
 * Theme install sources returned by the host API.
 * - built-in: shipped with host registerBuiltIn
 * - external: admin URL install
 * - marketplace: official catalog one-click install (Phase A)
 */
export type ThemeInstallSource = "built-in" | "external" | "marketplace" | string;

/** Sources that ship a UMD/JS bundle URL and must be loaded before activate. */
export function isRemoteThemeSource(source: string | undefined | null): boolean {
  const s = (source || "").trim().toLowerCase();
  return s === "external" || s === "marketplace";
}

/** Whether the theme may be uninstalled from the gallery (non-active remote rows). */
export function isUninstallableThemeSource(source: string | undefined | null): boolean {
  return isRemoteThemeSource(source);
}

/** Badge label for admin gallery. */
export function themeSourceBadgeLabel(source: string | undefined | null): string | null {
  const s = (source || "").trim().toLowerCase();
  if (s === "marketplace") return "市场";
  if (s === "external") return "外部";
  return null;
}
