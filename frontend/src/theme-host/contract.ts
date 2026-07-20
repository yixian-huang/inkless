/**
 * Inkless theme-host contract version lock.
 *
 * Bump `THEME_CONTRACT_VERSION` only when the public facade breaks themes
 * (removed/renamed exports, changed semantics of a host primitive).
 * Additive exports may stay on the same major contract version.
 *
 * Themes declare the version they were built against via
 * `ThemePlugin.contractVersion` and `inkless.theme.json#contractVersion`.
 */

/** Current host contract major version (string for JSON / UMD). */
export const THEME_CONTRACT_VERSION = "1" as const;

export type ThemeContractVersion = typeof THEME_CONTRACT_VERSION;

/**
 * Host-accepted contract versions for registration.
 * Keep this tight; only add entries after a deliberate compatibility window.
 */
export const THEME_CONTRACT_SUPPORTED: readonly string[] = ["1"];

export function normalizeThemeContractVersion(
  version: string | number | null | undefined,
): string | null {
  if (version == null) return null;
  const s = String(version).trim();
  return s.length > 0 ? s : null;
}

export function isThemeContractCompatible(
  version: string | number | null | undefined,
): boolean {
  const v = normalizeThemeContractVersion(version);
  if (!v) return false;
  return THEME_CONTRACT_SUPPORTED.includes(v);
}

/**
 * Resolve effective contract version for a theme.
 * Missing version is treated as `"1"` only while host still supports `"1"`,
 * so older UMD bundles keep loading during the first lock generation.
 */
export function resolveThemeContractVersion(
  version: string | number | null | undefined,
): string {
  const v = normalizeThemeContractVersion(version);
  if (v) return v;
  if (THEME_CONTRACT_SUPPORTED.includes("1")) return "1";
  return "";
}

export function assertThemeContractCompatible(
  version: string | number | null | undefined,
  themeId?: string,
): void {
  const effective = resolveThemeContractVersion(version);
  if (!isThemeContractCompatible(effective)) {
    const id = themeId ? ` "${themeId}"` : "";
    throw new Error(
      `Theme${id} contractVersion "${version ?? ""}" (effective "${effective}") is incompatible with host ` +
        `(supports: ${THEME_CONTRACT_SUPPORTED.join(", ")}; current: ${THEME_CONTRACT_VERSION})`,
    );
  }
}
